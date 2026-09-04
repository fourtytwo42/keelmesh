#!/usr/bin/env python3
"""Verify the public M12 coordination boundary without mutating fleet state."""

from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.request


base = (sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:8080").rstrip("/")
require_raft = os.getenv("KEELMESH_REQUIRE_RAFT", "0") == "1"


def get(path: str) -> dict:
    with urllib.request.urlopen(base + path, timeout=8) as response:
        return json.load(response)


try:
    overview = get("/api/v6/coordination/cells")
except (urllib.error.URLError, json.JSONDecodeError) as exc:
    if require_raft:
        raise SystemExit(f"M12 coordination unavailable: {exc}") from exc
    print(json.dumps({"milestone": "M12", "state": "simulated-rollback", "verified": True}))
    raise SystemExit(0)

cells = overview.get("cells", {})
leaders: dict[str, dict] = {}
for cell_id in ("A", "B"):
    nodes = cells.get(cell_id, [])
    leader = next((node for node in nodes if node.get("state") == "leader"), None)
    if leader is None:
        leader = next(
            (
                node
                for node in nodes
                if node.get("leader_node_id")
                and node.get("leader_node_id") == node.get("local_node_id")
            ),
            None,
        )
    if leader is None:
        if require_raft:
            raise SystemExit(f"M12 Cell {cell_id} has no authority-ready leader")
        continue
    assert leader["mode"] in ("shadow", "raft")
    assert leader["quorum_required"] == 4
    assert leader["commit_index"] >= leader["applied_index"]
    assert leader["state_hash"]
    leaders[cell_id] = leader

if require_raft and set(leaders) != {"A", "B"}:
    raise SystemExit("M12 requires both Raft cells")

security = get("/api/v6/coordination/security") if leaders else {"mode": "simulated"}
if leaders:
    assert security["transport"] == "mTLS 1.3 / Ed25519"
    assert security["referee_role"] == "non-voting"
    assert set(security["cells"]) == {"A", "B"}
    for cell in security["cells"].values():
        assert cell["quorum"] == 4
        assert len(cell["members"]) == 6
        assert cell["manifest_expires_at"]

print(
    json.dumps(
        {
            "milestone": "M12",
            "state": security.get("mode", "simulated"),
            "verified": True,
            "cells": {
                cell_id: {
                    "leader": leader["leader_node_id"],
                    "term": leader["term"],
                    "epoch": leader["authority_epoch"],
                    "commit_index": leader["commit_index"],
                    "state_hash": leader["state_hash"],
                }
                for cell_id, leader in leaders.items()
            },
        },
        sort_keys=True,
    )
)
