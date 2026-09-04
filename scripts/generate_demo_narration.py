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
from pathlib import Path


BEAT = re.compile(
    r'id: "(?P<id>[^"]+)".*?transcript:\s*\{\s*'
    r'navy: (?P<navy>"(?:[^"\\]|\\.)*"),\s*'
    r'pirate: (?P<pirate>"(?:[^"\\]|\\.)*"),\s*\}',
    re.DOTALL,
)


def synthesize(base_url: str, voice: str, text: str) -> bytes:
    payload = json.dumps(
        {"request_id": f"guided-demo-{voice}", "voice": voice, "text": text}
    ).encode()
    request = urllib.request.Request(
        f"{base_url.rstrip('/')}/api/v2/speech:synthesize",
        data=payload,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(request, timeout=180) as response:
        return response.read()


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-url", default="http://192.168.50.214:8080")
    parser.add_argument("--source", type=Path, default=Path("web/src/guidedDemo.ts"))
    parser.add_argument("--output", type=Path, default=Path("web/public/assets/demo"))
    args = parser.parse_args()

    ffmpeg = shutil.which("ffmpeg")
    if not ffmpeg:
        raise SystemExit("ffmpeg is required to create compact prerecorded assets")
    source = args.source.read_text(encoding="utf-8")
    beats = list(BEAT.finditer(source))
    if not beats:
        raise SystemExit(f"No narration beats found in {args.source}")

    with tempfile.TemporaryDirectory(prefix="keelmesh-demo-") as temporary:
        temp = Path(temporary)
        for match in beats:
            beat_id = match.group("id")
            # Preserve the cloned voices' natural cadence. The guided tour is
            # intentionally allowed to run longer than five minutes so its
            # technical content remains comfortable to follow in an interview.
            for persona, voice, tempo in (("navy", "jarvis", "1.00"), ("pirate", "barbossa", "1.00")):
                text = json.loads(match.group(persona))
                destination = args.output / persona / f"{beat_id}.mp3"
                destination.parent.mkdir(parents=True, exist_ok=True)
                wav = temp / f"{persona}-{beat_id}.wav"
                wav.write_bytes(synthesize(args.base_url, voice, text))
                subprocess.run(
                    [ffmpeg, "-hide_banner", "-loglevel", "error", "-y", "-i", str(wav), "-filter:a", f"atempo={tempo}", "-ac", "1", "-b:a", "64k", str(destination)],
                    check=True,
                )
                print(f"rendered {destination}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
