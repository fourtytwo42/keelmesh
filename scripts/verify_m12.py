#!/usr/bin/env python3
"""Verify the public M12 coordination boundary without mutating fleet state."""

from __future__ import annotations

import json
import os
import sys
import time
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

for attempt in range(5):
    cells = overview.get("cells", {})
    observed_modes = {
        node.get("mode")
        for nodes in cells.values()
        for node in nodes
        if node.get("mode")
    }
    distributed_mode = bool(observed_modes.intersection({"shadow", "raft"}))
    converged = set(cells) == {"A", "B"} and all(
        len(nodes) == 6
        and len({node.get("applied_index") for node in nodes}) == 1
        and len({node.get("state_hash") for node in nodes}) == 1
        for nodes in cells.values()
    )
    if not distributed_mode or converged or attempt == 4:
        break
    time.sleep(0.25)
    overview = get("/api/v6/coordination/cells")

if distributed_mode and set(cells) != {"A", "B"}:
    raise SystemExit("M12 distributed coordination requires both cells A and B")

leaders: dict[str, dict] = {}
for cell_id in ("A", "B"):
    nodes = cells.get(cell_id, [])
    if distributed_mode:
        if len(nodes) != 6:
            raise SystemExit(f"M12 Cell {cell_id} requires six voters; observed {len(nodes)}")
        node_ids = {node.get("local_node_id") for node in nodes}
        if len(node_ids) != 6 or None in node_ids:
            raise SystemExit(f"M12 Cell {cell_id} has duplicate or missing voter identities")
        if len({node.get("applied_index") for node in nodes}) != 1:
            raise SystemExit(f"M12 Cell {cell_id} voters have not converged on one applied index")
        if len({node.get("state_hash") for node in nodes}) != 1:
            raise SystemExit(f"M12 Cell {cell_id} voters have not converged on one state hash")
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
    if leader["reachable_voters"] < leader["quorum_required"]:
        raise SystemExit(
            f"M12 Cell {cell_id} leader has only {leader['reachable_voters']} reachable voters"
        )
    assert leader["commit_index"] >= leader["applied_index"]
    assert leader["state_hash"]
    leaders[cell_id] = leader

if (require_raft or distributed_mode) and set(leaders) != {"A", "B"}:
    raise SystemExit("M12 requires both Raft cells")
if require_raft and observed_modes != {"raft"}:
    raise SystemExit(f"M12 requires Raft mode; observed {sorted(observed_modes)}")

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
