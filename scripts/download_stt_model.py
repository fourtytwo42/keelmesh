#!/usr/bin/env python3
"""Stage the pinned offline STT model; run only during appliance provisioning."""

import hashlib
import json
import sys
from pathlib import Path

from huggingface_hub import snapshot_download


MODEL_ID = "Systran/faster-whisper-tiny.en"
MODEL_REVISION = "0d3d19a32d3338f10357c0889762bd8d64bbdeba"


target = Path(sys.argv[1] if len(sys.argv) > 1 else "/out")
snapshot_download(MODEL_ID, revision=MODEL_REVISION, local_dir=target)
digest = hashlib.sha256()
files = []
for path in sorted(item for item in target.rglob("*") if item.is_file() and ".cache" not in item.parts):
    relative = path.relative_to(target).as_posix()
    data = path.read_bytes()
    digest.update(relative.encode() + b"\0" + data)
    files.append({"path": relative, "bytes": len(data), "sha256": hashlib.sha256(data).hexdigest()})
(target / "keelmesh-manifest.json").write_text(
    json.dumps({"model_id": MODEL_ID, "revision": MODEL_REVISION, "content_sha256": digest.hexdigest(), "files": files}, indent=2) + "\n",
    encoding="utf-8",
)
print(json.dumps({"model_id": MODEL_ID, "revision": MODEL_REVISION, "files": len(files), "content_sha256": digest.hexdigest()}))
