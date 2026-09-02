from __future__ import annotations

import io
import os
import threading
import tempfile
import time
import wave
from pathlib import Path
from typing import Any

import yaml
from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import StreamingResponse
from pydantic import BaseModel, ConfigDict, Field

VOICE_NAMES = ("anna", "vera", "charles", "paul", "jeff", "patrick", "james", "morgan", "trailer", "ian", "sam", "david")
MODEL_ROOT = Path(os.getenv("KEELMESH_SPEECH_MODEL_ROOT", "/models"))
STT_ROOT = Path(os.getenv("KEELMESH_STT_MODEL_ROOT", "/stt-model"))


class SynthesisRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")
    text: str = Field(min_length=1, max_length=1200)
    voice: str = "morgan"
    request_id: str = Field(min_length=4, max_length=160)


class Runtime:
    def __init__(self) -> None:
        self.model: Any | None = None
        self.voices: dict[str, Any] = {}
        self.lock = threading.Lock()
        self.stt_model: Any | None = None
        self.stt_lock = threading.Lock()

    def model_files(self) -> tuple[Path, Path]:
        weights = next(MODEL_ROOT.glob("**/model.safetensors"), None)
        tokenizer = next(MODEL_ROOT.glob("**/tokenizer.model"), None)
        if not weights or not tokenizer:
            raise RuntimeError("verified Pocket TTS model pack is unavailable")
        return weights, tokenizer

    def load(self) -> Any:
        if self.model is not None:
            return self.model
        from pocket_tts import TTSModel
        from pocket_tts.utils.config import CONFIGS_DIR

        weights, tokenizer = self.model_files()
        with (Path(CONFIGS_DIR) / "english.yaml").open(encoding="utf-8") as source:
            config = yaml.safe_load(source)
        config["weights_path"] = str(weights)
        config["weights_path_without_voice_cloning"] = str(weights)
        config["flow_lm"]["lookup_table"]["tokenizer_path"] = str(tokenizer)
        generated = Path("/tmp/pocket-tts-local-english.yaml")
        generated.write_text(yaml.safe_dump(config, sort_keys=False), encoding="utf-8")
        self.model = TTSModel.load_model(config=generated)
        return self.model

    def voice(self, name: str) -> Any:
        if name not in VOICE_NAMES:
            raise ValueError("unknown voice")
        if name not in self.voices:
            model = self.load()
            source = MODEL_ROOT / "built-in-voices" / f"{name}.wav"
            if not source.is_file():
                raise RuntimeError("voice prompt missing from model pack")
            self.voices[name] = model.get_state_for_audio_prompt(str(source), truncate=True)
        return self.voices[name]

    def synthesize(self, text: str, voice: str) -> bytes:
        with self.lock:
            model = self.load()
            audio = model.generate_audio(self.voice(voice), text.strip())
            samples = audio.detach().cpu().numpy().clip(-1.0, 1.0)
            pcm = (samples * 32767.0).astype("<i2")
            output = io.BytesIO()
            with wave.open(output, "wb") as wav:
                wav.setnchannels(1)
                wav.setsampwidth(2)
                wav.setframerate(int(model.sample_rate))
                wav.writeframes(pcm.tobytes())
            return output.getvalue()

    def load_stt(self) -> Any:
        if self.stt_model is not None:
            return self.stt_model
        if not (STT_ROOT / "model.bin").is_file():
            raise RuntimeError("verified local STT model is unavailable")
        from faster_whisper import WhisperModel

        self.stt_model = WhisperModel(
            str(STT_ROOT), device="cpu", compute_type="int8", cpu_threads=max(1, min(8, os.cpu_count() or 1))
        )
        return self.stt_model

    def transcribe(self, audio: bytes, suffix: str) -> dict[str, Any]:
        with self.stt_lock:
            model = self.load_stt()
            started = time.monotonic()
            with tempfile.NamedTemporaryFile(suffix=suffix) as source:
                source.write(audio)
                source.flush()
                segments, info = model.transcribe(
                    source.name,
                    language="en",
                    beam_size=3,
                    best_of=3,
                    vad_filter=True,
                    vad_parameters={"min_silence_duration_ms": 350, "speech_pad_ms": 250},
                    condition_on_previous_text=False,
                )
                parts = [segment.text.strip() for segment in segments if segment.text.strip()]
            elapsed = time.monotonic() - started
            duration = float(getattr(info, "duration", 0.0) or 0.0)
            return {
                "schema_version": 2,
                "text": " ".join(parts),
                "route": "colocated-node",
                "engine": "faster-whisper",
                "model": "distil-small.en-int8",
                "language": getattr(info, "language", "en"),
                "duration_seconds": duration,
                "latency_ms": round(elapsed * 1000, 1),
                "real_time_factor": round(elapsed / duration, 3) if duration > 0 else 0.0,
                "final": True,
            }


RUNTIME = Runtime()
app = FastAPI(title="KeelMesh Speech Node", version="0.6.0")


@app.on_event("startup")
def warm_default_voice() -> None:
    """Move the expensive model/voice load to appliance startup, not first speech."""
    try:
        RUNTIME.voice("morgan")
        RUNTIME.load_stt()
    except RuntimeError:
        # Health remains degraded and visible text stays authoritative.
        return


@app.get("/healthz")
def health() -> dict[str, Any]:
    try:
        weights, tokenizer = RUNTIME.model_files()
        ready = weights.stat().st_size == 219_029_196 and tokenizer.stat().st_size == 59_339
    except (RuntimeError, OSError):
        ready = False
    stt_ready = (STT_ROOT / "model.bin").is_file()
    return {"status": "ready" if ready else "degraded", "engine": "Pocket TTS", "version": "2.1.0", "default_voice": "morgan", "offline_ready": ready, "loaded": RUNTIME.model is not None, "stt_ready": stt_ready, "stt_engine": "faster-whisper", "stt_model": "distil-small.en-int8"}


@app.get("/v1/voices")
def voices() -> dict[str, Any]:
    return {"voices": [{"id": name, "name": "Movie Trailer Voice" if name == "trailer" else name.title(), "default": name == "morgan", "available": (MODEL_ROOT / "built-in-voices" / f"{name}.wav").is_file()} for name in VOICE_NAMES]}


@app.post("/v1/synthesize")
def synthesize(request: SynthesisRequest) -> StreamingResponse:
    try:
        audio = RUNTIME.synthesize(request.text, request.voice)
    except ValueError as exc:
        raise HTTPException(status_code=422, detail="VOICE_NOT_FOUND") from exc
    except RuntimeError as exc:
        raise HTTPException(status_code=503, detail="TTS_UNAVAILABLE") from exc
    return StreamingResponse(io.BytesIO(audio), media_type="audio/wav", headers={"X-KeelMesh-Request-ID": request.request_id, "Cache-Control": "no-store"})


@app.post("/v1/transcribe")
async def transcribe(request: Request) -> dict[str, Any]:
    audio = await request.body()
    if not 128 <= len(audio) <= 8 * 1024 * 1024:
        raise HTTPException(status_code=422, detail="AUDIO_SIZE_INVALID")
    content_type = request.headers.get("content-type", "audio/webm")
    suffix = ".wav" if "wav" in content_type else ".webm"
    try:
        result = RUNTIME.transcribe(audio, suffix)
    except RuntimeError as exc:
        raise HTTPException(status_code=503, detail="STT_UNAVAILABLE") from exc
    except Exception as exc:
        # Audio containers/codecs are untrusted input. Fail with a stable client
        # error instead of leaking decoder exceptions as a server failure.
        raise HTTPException(status_code=422, detail="AUDIO_DECODE_FAILED") from exc
    result["request_id"] = request.query_params.get("request_id", "")
    return result
