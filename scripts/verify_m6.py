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
assert len(fleet["vessels"]) == 12
assert len(fleet["groups"]) == 0
assert len({v["id"] for v in fleet["vessels"]}) == 12
assert len({v["callsign"] for v in fleet["vessels"]}) == 12
assert all(v["node_id"] and v["vm_id"] and not v["group_id"] for v in fleet["vessels"])
assert fleet["environment"]["label"] == "NOAA-derived simulation fixture"
contacts = fleet["surface_contacts"]
assert len(contacts) == 16
assert len({contact["boat_id"] for contact in contacts}) == 16
assert len({contact["color_name"] for contact in contacts}) == 16
assert {contact["class"] for contact in contacts} == {"container", "tanker", "ferry", "trawler", "patrol", "yacht"}
contact = call(f"/api/v2/surface-contacts/{contacts[0]['id']}")
assert contact["boat_id"] == "NPC-4101" and contact["looping"] and len(contact["route"]) >= 2

group_specs = [
    ("North Watch", "#e9a93f", fleet["vessels"][0:4]),
    ("Sound Watch", "#62c5a8", fleet["vessels"][4:8]),
    ("South Watch", "#d86f5f", fleet["vessels"][8:12]),
]
groups = []
for index, (name, color, vessels) in enumerate(group_specs, 1):
    current = call("/api/v2/fleet")
    groups.append(call("/api/v2/groups", {**mutation(f"group-{index}", current["fleet_version"]), "name": name, "color": color, "pattern": "solid", "member_ids": [v["id"] for v in vessels]}, 201))

fleet = call("/api/v2/fleet")
follow_mission = call(
    "/api/v2/missions",
    {
        **mutation("follow-mission", fleet["fleet_version"]),
        "name": "Surface Contact Watch",
        "objective": "Follow a selected fictional contact",
        "target_ids": groups[0]["member_ids"],
    },
    201,
)
follow_draft = call(
    f"/api/v2/missions/{follow_mission['id']}/commands:compile",
    {
        **mutation("follow-compile", follow_mission["version"]),
        "text": "Have amber team follow NPC-4101 at a safe distance",
        "target_ids": follow_mission["target_ids"],
    },
    201,
)
assert follow_draft["follow_contact_id"] == "surface-01"
assert follow_draft["guidance_kind"] == "follow_contact"
assert len(follow_draft["waypoints"]) == 12 and not follow_draft["unresolved_ambiguities"]
follow_current = call(f"/api/v2/missions/{follow_mission['id']}")
call(
    f"/api/v2/missions/{follow_mission['id']}",
    mutation("follow-delete", follow_current["version"]),
    method="DELETE",
)
fleet = call("/api/v2/fleet")

first = fleet["vessels"][0]
reach = call(f"/api/v2/vessels/{first['id']}/reachability")
assert reach["vessel_id"] == first["id"]
assert len(reach["direct_peers"]) + len(reach["relayed_peers"]) == 3
assert reach["authority"]

collection_payload = {
    **mutation("collection-create", fleet["fleet_version"]),
    "name": "Interview watch",
    "member_ids": [groups[0]["member_ids"][0], groups[1]["member_ids"][0]],
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
for index, group in enumerate(groups, 1):
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
        "included_areas": [[[-71.55, 41.08], [-71.15, 41.08], [-71.15, 41.16], [-71.55, 41.16], [-71.55, 41.08]]],
        "exclusion_areas": [[[-71.37, 41.11], [-71.35, 41.11], [-71.35, 41.10], [-71.37, 41.10], [-71.37, 41.11]]],
        "waypoints": [[-71.34, 41.12], [-71.32, 41.14]],
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
compiled_mission = call(f"/api/v2/missions/{mission['id']}")
plans = call(
    f"/api/v2/missions/{mission['id']}/plans",
    {**mutation("plans", compiled_mission["version"]), "draft_id": draft["id"]},
    201,
)["plans"]
assert 2 <= len(plans) <= 4 and sum(plan["recommended"] for plan in plans) == 1
plan = next(plan for plan in plans if plan["recommended"])
planned = call(f"/api/v2/missions/{mission['id']}")
before_preview = call(f"/api/v2/missions/{mission['id']}")
preview = call(
    f"/api/v2/missions/{mission['id']}/plans/{plan['id']}:preview",
    mutation("preview", planned["version"]),
)
after_preview = call(f"/api/v2/missions/{mission['id']}")
assert preview["nothing_sent"] and preview["plan_hash"] == plan["content_hash"]
assert before_preview["version"] == after_preview["version"] and before_preview["status"] == after_preview["status"]

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
assert len(voices) == 14 and next(v for v in voices if v["default"])["id"] == "jarvis"
assert next(v for v in voices if v["id"] == "barbossa")["name"] == "Captain Barbossa"
speech = call("/api/v2/speech/capabilities")
assert speech["tts_engine"] == "Pocket TTS" and speech["transcription_routes"][-1] == "typed-input"

final_fleet = call("/api/v2/fleet")
call("/api/v2/scenarios/fleet-operations:reset", mutation("final-reset", final_fleet["fleet_version"]))
final_fleet = call("/api/v2/fleet")
assert len(final_fleet["vessels"]) == 12 and len(final_fleet["groups"]) == 0 and len(final_fleet["missions"]) == 0

print(
    json.dumps(
        {
            "status": "pass",
            "vessels": 12,
            "surface_contacts": len(contacts),
            "primary_groups": 0,
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
