#!/usr/bin/env python3
"""Verify the M15 fictional maritime-combat surface.

The default check is read-only. ``--exercise`` creates and then stops one
pending exact-hash engagement without authorizing fire or changing hull state.
"""

import json
import sys
import time
import urllib.error
import urllib.request


BASE = next((arg for arg in sys.argv[1:] if not arg.startswith("--")), "http://127.0.0.1:8080").rstrip("/")
EXERCISE = "--exercise" in sys.argv[1:]
STAMP = str(time.time_ns())


def call(path: str, payload=None, status=200, method=None):
    data = None if payload is None else json.dumps(payload).encode()
    request = urllib.request.Request(BASE + path, data=data, headers={"Content-Type": "application/json"}, method=method)
    try:
        with urllib.request.urlopen(request, timeout=45) as response:
            code, body = response.status, json.load(response)
    except urllib.error.HTTPError as error:
        code, body = error.code, json.load(error)
    if code != status:
        raise AssertionError(f"{path}: expected {status}, got {code}: {body}")
    return body


combat = call("/api/v8/combat")
entities = combat["entities"]
blackwake = next(entity for entity in entities if entity["entity_id"] == "HOSTILE-0001")
controlled = [entity for entity in entities if entity["profile"]["controlled"]]
assert combat["schema_version"] == 1
assert "fictional" in combat["disclaimer"].lower()
assert len(controlled) == 12
assert len(entities) == 45
assert blackwake["name"] == "Blackwake"
assert blackwake["profile"]["class"] == "pirate-raider"
assert blackwake["profile"]["hostility"] == "hostile"
assert blackwake["profile"]["hull_maximum"] == 170
assert 0 <= blackwake["speed_mps"] <= 3.0
weapons = {weapon["id"]: weapon for weapon in blackwake["profile"]["weapons"]}
assert weapons["twin-deck-cannons"]["effective_range_m"] == 750
assert weapons["limited-rockets"]["effective_range_m"] == 1400
assert all(entity["damage"]["hull"] <= entity["profile"]["hull_maximum"] for entity in entities)

if not EXERCISE:
    print(json.dumps({"status": "pass", "mode": "read_only", "entities": len(entities), "controlled": len(controlled), "hostile": blackwake["name"], "state_version": combat["state_version"]}))
    raise SystemExit(0)

armed = next(entity for entity in controlled if entity["profile"]["weapons"] and not entity["damage"]["disabled"])
key = f"m15-engagement-{STAMP}"
program = call(
    "/api/v8/combat/engagements",
    {
        "request_id": key,
        "idempotency_key": key,
        "expected_version": combat["state_version"],
        "target_id": blackwake["entity_id"],
        "participant_ids": [armed["entity_id"]],
        "duration_seconds": 120,
        "maximum_effects": 4,
    },
    201,
    "POST",
)
assert program["status"] == "pending_approval" and program["content_hash"]
error = call(
    f"/api/v8/combat/engagements/{program['id']}:authorize",
    {"request_id": key + "-bad", "idempotency_key": key + "-bad", "expected_version": combat["state_version"], "plan_hash": "wrong-hash", "operator_id": "m15-verifier"},
    409,
    "POST",
)
assert error["code"] == "COMBAT_STATE_STALE"
stopped = call(
    f"/api/v8/combat/engagements/{program['id']}:stop",
    {"request_id": key + "-stop", "idempotency_key": key + "-stop", "expected_version": combat["state_version"]},
    200,
    "POST",
)
assert stopped["status"] == "stopped"
print(json.dumps({"status": "pass", "mode": "exercise", "engagement": program["id"], "exact_hash_rejected": True, "effects_authorized": 0}))
