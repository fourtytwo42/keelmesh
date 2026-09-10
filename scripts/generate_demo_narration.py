#!/usr/bin/env python3
"""Render the checked-in guided-demo script through the VM Pocket TTS service.

Generated MP3 files are release assets: the browser never synthesizes narration
during a demo. Run this only when the script or custom voices intentionally change.
"""

from __future__ import annotations

import argparse
import json
import re
import shutil
import subprocess
import tempfile
import urllib.request
import uuid
from pathlib import Path


BEAT = re.compile(
    r'id: "(?P<id>[^"]+)".*?transcript:\s*\{\s*'
    r'navy: (?P<navy>"(?:[^"\\]|\\.)*"),\s*'
    r'pirate: (?P<pirate>"(?:[^"\\]|\\.)*"),\s*\}',
    re.DOTALL,
)


def synthesize(base_url: str, voice: str, text: str, request_id: str) -> bytes:
    payload = json.dumps(
        {"request_id": request_id, "voice": voice, "text": text}
    ).encode()
    request = urllib.request.Request(
        f"{base_url.rstrip('/')}/api/v2/speech:synthesize",
        data=payload,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(request, timeout=180) as response:
        return response.read()


def narration_chunks(text: str, max_chars: int) -> list[str]:
    """Keep cloned-voice generations short enough to avoid phrase looping."""
    sentences = re.split(r"(?<=[.!?])\s+", text.strip())
    chunks: list[str] = []
    current = ""
    for sentence in sentences:
        if not sentence:
            continue
        candidate = f"{current} {sentence}".strip()
        if current and len(candidate) > max_chars:
            chunks.append(current)
            current = sentence
        else:
            current = candidate
    if current:
        chunks.append(current)
    return chunks


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-url", default="http://192.168.50.214:8080")
    parser.add_argument("--source", type=Path, default=Path("web/src/guidedDemo.ts"))
    parser.add_argument("--output", type=Path, default=Path("web/public/assets/demo"))
    parser.add_argument(
        "--beat",
        action="append",
        default=[],
        help="Render only this beat ID; repeat for multiple beats.",
    )
    parser.add_argument(
        "--persona",
        action="append",
        choices=("navy", "pirate"),
        default=[],
        help="Render only this persona; repeat to render both. Defaults to both.",
    )
    parser.add_argument(
        "--max-chars",
        type=int,
        default=280,
        help="Maximum text per synthesis request; shorter chunks prevent cloned-voice phrase looping.",
    )
    args = parser.parse_args()

    ffmpeg = shutil.which("ffmpeg")
    if not ffmpeg:
        raise SystemExit("ffmpeg is required to create compact prerecorded assets")
    source = args.source.read_text(encoding="utf-8")
    beats = list(BEAT.finditer(source))
    if not beats:
        raise SystemExit(f"No narration beats found in {args.source}")
    if args.beat:
        requested = set(args.beat)
        known = {match.group("id") for match in beats}
        unknown = sorted(requested - known)
        if unknown:
            raise SystemExit(f"Unknown narration beat(s): {', '.join(unknown)}")
        beats = [match for match in beats if match.group("id") in requested]
    personas = args.persona or ["navy", "pirate"]
    persona_settings = {
        "navy": ("jarvis", "1.00"),
        "pirate": ("barbossa", "1.00"),
    }

    with tempfile.TemporaryDirectory(prefix="keelmesh-demo-") as temporary:
        temp = Path(temporary)
        for match in beats:
            beat_id = match.group("id")
            # Preserve the cloned voices' natural cadence. The guided tour is
            # intentionally allowed to run longer than five minutes so its
            # technical content remains comfortable to follow in an interview.
            for persona in personas:
                voice, tempo = persona_settings[persona]
                text = json.loads(match.group(persona))
                destination = args.output / persona / f"{beat_id}.mp3"
                destination.parent.mkdir(parents=True, exist_ok=True)
                chunks = narration_chunks(text, args.max_chars)
                wavs: list[Path] = []
                for index, chunk in enumerate(chunks, start=1):
                    wav = temp / f"{persona}-{beat_id}-{index:02d}.wav"
                    request_id = f"guided-demo-{persona}-{beat_id}-{index}-{uuid.uuid4().hex}"
                    wav.write_bytes(synthesize(args.base_url, voice, chunk, request_id))
                    wavs.append(wav)
                temporary_mp3 = temp / f"{persona}-{beat_id}.mp3"
                inputs = [item for wav in wavs for item in ("-i", str(wav))]
                streams = "".join(f"[{index}:a]" for index in range(len(wavs)))
                subprocess.run(
                    [
                        ffmpeg,
                        "-hide_banner",
                        "-loglevel",
                        "error",
                        "-y",
                        *inputs,
                        "-filter_complex",
                        f"{streams}concat=n={len(wavs)}:v=0:a=1,atempo={tempo}[out]",
                        "-map",
                        "[out]",
                        "-ac",
                        "1",
                        "-b:a",
                        "64k",
                        str(temporary_mp3),
                    ],
                    check=True,
                )
                shutil.move(temporary_mp3, destination)
                print(f"rendered {destination} from {len(chunks)} synthesis chunks")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
