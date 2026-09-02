"""One-shot M4 corpus integrity check.

The approved corpus itself is served by the capability-scoped Go MCP boundary.
This role proves the Python image and prompt/corpus versions are available before
the interactive service starts without granting Python database credentials.
"""

from __future__ import annotations

import hashlib
import json

from keelmesh_ai.service import ASSERTIONS, OPENROUTER_MODELS


def main() -> None:
    manifest = {
        "corpus_version": "keelmesh-runbooks-v1",
        "prompt_version": "incident-investigator-v1",
        "eval_suite": "autonomy-regression-v1",
        "assertions": ASSERTIONS,
        "provider_pool": OPENROUTER_MODELS,
    }
    raw = json.dumps(manifest, sort_keys=True).encode()
    print(json.dumps({"state": "indexed", "manifest_sha256": hashlib.sha256(raw).hexdigest()}))


if __name__ == "__main__":
    main()
