#!/usr/bin/env python3
"""Exercise the public M11 memory boundary without invoking a paid model."""

from __future__ import annotations

import hashlib
import json
import sys
import time
import urllib.request


BASE = (sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:8080").rstrip("/")


def get(path: str) -> dict:
    with urllib.request.urlopen(BASE + path, timeout=15) as response:
        return json.load(response)


def post(path: str, payload: dict) -> dict:
    request = urllib.request.Request(
        BASE + path,
        data=json.dumps(payload).encode(),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(request, timeout=30) as response:
        return json.load(response)


snapshot = get("/api/v5/memory")
assert snapshot["available"], snapshot
assert snapshot["retrieval_mode"] in {"hybrid", "keyword"}, snapshot
assert snapshot["embedding_state"] in {"ready", "degraded"}, snapshot

search = post(
    "/api/v5/memory/search",
    {
        "query": "GNSS anomaly safe hold stale reconnection",
        "actor_identity": "demo-operator",
        "session_id": "m11-verifier",
        "limit": 4,
    },
)
assert search["receipt"]["mode"] in {"hybrid", "keyword"}, search
assert len(search["hits"]) >= 1, search
assert all(hit["source_ids"] for hit in search["hits"]), search

version = get("/api/v5/memory")["state_version"]
stamp = time.time_ns()
replay = post(
    "/api/v5/memory/replays",
    {
        "request_id": f"verify-memory-replay-{stamp}",
        "idempotency_key": f"verify-memory-replay-{stamp}",
        "actor_identity": "demo-operator",
        "expected_memory_state_version": version,
    },
)
assert replay["matches"], replay
assert replay["live_checksum"] == replay["replay_checksum"], replay
assert len(replay["live_checksum"]) == 64, replay

canonical = json.dumps(
    {
        "mode": search["receipt"]["mode"],
        "hit_ids": [hit["item_id"] for hit in search["hits"]],
        "replay_checksum": replay["replay_checksum"],
    },
    sort_keys=True,
).encode()
result = {
    "milestone": "M11",
    "available": True,
    "embedding_state": snapshot["embedding_state"],
    "retrieval_mode": search["receipt"]["mode"],
    "committed_items": snapshot["committed_items"],
    "retrieval_hits": len(search["hits"]),
    "replay_items": replay["item_count"],
    "replay_checksum": replay["replay_checksum"],
    "evidence_checksum": hashlib.sha256(canonical).hexdigest(),
}
print(json.dumps(result, sort_keys=True))
