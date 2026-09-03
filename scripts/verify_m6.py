#!/usr/bin/env python3
"""Verify M6 fleet organization, concurrent authority, and exact-plan execution."""

import json
import sys
import time
import urllib.error
import urllib.request


BASE = (sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:8080").rstrip("/")
STAMP = str(time.time_ns())


def call(path: str, payload=None, status=200, method=None):
    data = None if payload is None else json.dumps(payload).encode()
    request = urllib.request.Request(
        BASE + path,
        data=data,
        headers={"Content-Type": "application/json"},
        method=method,
    )
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            code, body = response.status, json.load(response)
    except urllib.error.HTTPError as error:
        code, body = error.code, json.load(error)
    if code != status:
        raise AssertionError(f"{path}: expected {status}, got {code}: {body}")
    return body


def mutation(label: str, version: int):
    key = f"m6-{label}-{STAMP}"
    return {
        "request_id": key,
        "idempotency_key": key,
        "expected_version": version,
    }


fleet = call("/api/v2/fleet")
call(
    "/api/v2/scenarios/fleet-operations:reset",
    mutation("reset", fleet["fleet_version"]),
)
fleet = call("/api/v2/fleet")
assert len(fleet["vessels"]) == 48
assert len(fleet["groups"]) >= 8
assert len({v["id"] for v in fleet["vessels"]}) == 48
assert len({v["callsign"] for v in fleet["vessels"]}) == 48
assert fleet["environment"]["label"] == "NOAA-derived simulation fixture"

for group in fleet["groups"][:8]:
    members = [v for v in fleet["vessels"] if v["id"] in group["member_ids"]]
    mix = {name: sum(v["class"]["id"] == name for v in members) for name in ("kestrel", "mariner", "atlas")}
    assert mix == {"kestrel": 3, "mariner": 2, "atlas": 1}, (group["id"], mix)

first = fleet["vessels"][0]
reach = call(f"/api/v2/vessels/{first['id']}/reachability")
assert reach["vessel_id"] == first["id"]
assert len(reach["direct_peers"]) + len(reach["relayed_peers"]) == 5
assert reach["authority"] and reach["reachable_outside_group"]

collection_payload = {
    **mutation("collection-create", fleet["fleet_version"]),
    "name": "Interview watch",
    "member_ids": [fleet["groups"][0]["member_ids"][0], fleet["groups"][1]["member_ids"][0]],
}
collection = call("/api/v2/collections", collection_payload, 201)
collection = call(
    f"/api/v2/collections/{collection['id']}",
    {
        **mutation("collection-patch", collection["revision"]),
        "name": "Interview relay watch",
        "member_ids": collection["member_ids"],
    },
    method="PATCH",
)
assert collection["name"] == "Interview relay watch" and collection["revision"] == 2

missions = []
for index, group in enumerate(fleet["groups"][:3], 1):
    current = call("/api/v2/fleet")
    mission = call(
        "/api/v2/missions",
        {
            **mutation(f"mission-{index}", current["fleet_version"]),
            "name": f"Concurrent Mission {index}",
            "objective": "Follow the assigned corridor in formation",
            "target_ids": group["member_ids"],
        },
        201,
    )
    missions.append(mission)

conflict_fleet = call("/api/v2/fleet")
conflict = call(
    "/api/v2/missions",
    {
        **mutation("conflict", conflict_fleet["fleet_version"]),
        "name": "Conflicting authority",
        "target_ids": [missions[0]["target_ids"][0]],
    },
    409,
)
assert conflict["code"] == "MOVEMENT_AUTHORITY_CONFLICT"

mission = missions[0]
geometry = call(
    f"/api/v2/missions/{mission['id']}/geometry",
    {
        **mutation("geometry", mission["version"]),
        "included_areas": [[[-71.43, 41.45], [-71.30, 41.45], [-71.30, 41.34], [-71.43, 41.34], [-71.43, 41.45]]],
        "exclusion_areas": [[[-71.38, 41.40], [-71.36, 41.40], [-71.36, 41.38], [-71.38, 41.38], [-71.38, 41.40]]],
        "waypoints": [[-71.41, 41.43], [-71.33, 41.36]],
        "pois": [],
    },
)
draft = call(
    f"/api/v2/missions/{mission['id']}/commands:compile",
    {
        **mutation("compile", geometry["version"]),
        "text": "Proceed through the waypoints in a wedge, then hold while keeping 35 percent reserve",
        "target_ids": mission["target_ids"],
        "formation": "wedge",
    },
    201,
)
assert draft["target_ids"] == mission["target_ids"] and draft["geometry_revision"] == geometry["geometry"]["revision"]
plans = call(
    f"/api/v2/missions/{mission['id']}/plans",
    {**mutation("plans", geometry["version"]), "draft_id": draft["id"]},
    201,
)["plans"]
assert 2 <= len(plans) <= 4 and sum(plan["recommended"] for plan in plans) == 1
plan = next(plan for plan in plans if plan["recommended"])
planned = call(f"/api/v2/missions/{mission['id']}")
before = {v["id"]: v["telemetry"]["position"] for v in call("/api/v2/fleet")["vessels"]}
preview = call(
    f"/api/v2/missions/{mission['id']}/plans/{plan['id']}:preview",
    mutation("preview", planned["version"]),
)
after_preview = {v["id"]: v["telemetry"]["position"] for v in call("/api/v2/fleet")["vessels"]}
assert preview["nothing_sent"] and preview["plan_hash"] == plan["content_hash"] and before == after_preview

tampered = call(
    f"/api/v2/missions/{mission['id']}/plans/{plan['id']}:authorize",
    {
        **mutation("tampered", planned["version"]),
        "plan_hash": "sha256:tampered",
        "operator_id": "demo-operator",
    },
    409,
)
assert tampered["code"] == "PLAN_HASH_MISMATCH"
lease = call(
    f"/api/v2/missions/{mission['id']}/plans/{plan['id']}:authorize",
    {
        **mutation("authorize", planned["version"]),
        "plan_hash": plan["content_hash"],
        "operator_id": "demo-operator",
    },
    201,
)
authorized = call(f"/api/v2/missions/{mission['id']}")
started = call(
    f"/api/v2/missions/{mission['id']}/plans/{plan['id']}:start",
    {
        **mutation("start", authorized["version"]),
        "plan_hash": plan["content_hash"],
        "lease_id": lease["id"],
    },
)
assert started["status"] == "executing"

voices = call("/api/v2/voices")["voices"]
assert len(voices) == 13 and next(v for v in voices if v["default"])["id"] == "jarvis"
speech = call("/api/v2/speech/capabilities")
assert speech["tts_engine"] == "Pocket TTS" and speech["transcription_routes"][-1] == "typed-input"

print(
    json.dumps(
        {
            "status": "pass",
            "vessels": 48,
            "primary_groups": 8,
            "concurrent_missions": 3,
            "reachability_peers": len(reach["direct_peers"]) + len(reach["relayed_peers"]),
            "plan_options": len(plans),
            "formation": plan["formation"],
            "exact_hash": plan["content_hash"],
            "voices": len(voices),
            "default_voice": "jarvis",
        }
    )
)
