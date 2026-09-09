#!/usr/bin/env python3
"""Verify M14 complete-program authority and bounded edge continuation.

The default mode is read-only. Pass ``--exercise`` only in a controlled demo
window; it resets Fleet state, creates one Cell A mission, advances beyond one
minute, validates v7/legacy projections, and restores an idle fleet.
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
            code, body, headers = response.status, json.load(response), response.headers
    except urllib.error.HTTPError as error:
        code, body, headers = error.code, json.load(error), error.headers
    if code != status:
        raise AssertionError(f"{path}: expected {status}, got {code}: {body}")
    return body, headers


def mutation(label: str, version: int):
    value = f"m14-{label}-{STAMP}"
    return {"request_id": value, "idempotency_key": value, "expected_version": version}


fleet, _ = call("/api/v2/fleet")
assert fleet["execution_mode"] in {"tape", "full_program_shadow", "full_program"}
assert all(vessel["telemetry"]["tape_depth_seconds"] == 0 for vessel in fleet["vessels"])
assert all(not mission.get("trajectory") or mission["trajectory"]["hot_tape_horizon_seconds"] == 0 for mission in fleet["missions"])

if not EXERCISE:
    print(json.dumps({"status": "pass", "mode": "read_only", "execution_mode": fleet["execution_mode"], "vessels": len(fleet["vessels"]), "legacy_tape_depth": 0}))
    raise SystemExit(0)

if fleet["execution_mode"] not in {"full_program_shadow", "full_program"}:
    raise AssertionError(f"mutating M14 verification requires a complete-program mode, got {fleet['execution_mode']}")

mission_id = ""
try:
    fleet, _ = call("/api/v2/scenarios/fleet-operations:reset", mutation("reset", fleet["fleet_version"]), method="POST")
    cell_a = sorted((vessel for vessel in fleet["vessels"] if vessel["node_faction"] == "A"), key=lambda vessel: vessel["node_id"])
    if len(cell_a) != 6:
        raise AssertionError(f"expected six Cell A vessels, got {len(cell_a)}")
    members = [vessel["id"] for vessel in cell_a[:3]]
    group, _ = call("/api/v2/groups", {**mutation("group", fleet["fleet_version"]), "name": "M14 Edge Proof", "color": "#e9a93f", "pattern": "solid", "member_ids": members}, 201, "POST")
    fleet, _ = call("/api/v2/fleet")
    mission, _ = call("/api/v2/missions", {**mutation("mission", fleet["fleet_version"]), "name": "Full Program Continuity", "objective": "Transit east and continue without shore authority", "target_ids": members}, 201, "POST")
    mission_id = mission["id"]
    anchor = cell_a[0]["telemetry"]["position"]
    geometry, _ = call(f"/api/v2/missions/{mission_id}/geometry", {**mutation("geometry", mission["version"]), "included_areas": [], "exclusion_areas": [], "waypoints": [[anchor[0] + 0.025, anchor[1]], [anchor[0] + 0.03, anchor[1] + 0.005]], "pois": []}, method="POST")
    draft, _ = call(f"/api/v2/missions/{mission_id}/commands:compile", {**mutation("compile", geometry["version"]), "text": "Transit through both markers in column and hold at the final marker", "target_ids": members, "planning_mode": "manual", "formation": "column"}, 201, "POST")
    mission, _ = call(f"/api/v2/missions/{mission_id}")
    plans, _ = call(f"/api/v2/missions/{mission_id}/plans", {**mutation("plans", mission["version"]), "draft_id": draft["id"]}, 201, "POST")
    plan = next(candidate for candidate in plans["plans"] if candidate["policy_status"] != "prohibited")
    mission, _ = call(f"/api/v2/missions/{mission_id}")
    lease, _ = call(f"/api/v2/missions/{mission_id}/plans/{plan['id']}:authorize", {**mutation("authorize", mission["version"]), "plan_hash": plan["content_hash"], "operator_id": "demo-operator"}, 201, "POST")
    mission, _ = call(f"/api/v2/missions/{mission_id}")
    started, _ = call(f"/api/v2/missions/{mission_id}/plans/{plan['id']}:start", {**mutation("start", mission["version"]), "plan_hash": plan["content_hash"], "lease_id": lease["id"]}, method="POST")
    assert started["status"] == "executing"

    view, _ = call(f"/api/v7/missions/{mission_id}/program")
    summary = view["summary"]
    assert summary["complete_program_onboard"] and summary["installed_node_count"] == 6
    assert summary["total_segments"] > 18 and summary["authorization_expiry_tick"] > 60
    authority, _ = call(f"/api/v7/vessels/{members[0]}/execution")
    assert authority["complete_program_onboard"] and authority["status"] == "active"
    decision, _ = call(f"/api/v7/groups/{group['id']}/decision-state")
    assert decision["decision_node_id"] == min(members) and decision["decision_scope"] == "group"

    fleet, _ = call("/api/v2/fleet")
    call("/api/v2/simulation/rate", {**mutation("rate", fleet["fleet_version"]), "rate": 500}, method="POST")
    deadline = time.time() + 5
    while time.time() < deadline:
        view, _ = call(f"/api/v7/missions/{mission_id}/program")
        if view["summary"]["mission_tick"] > 60:
            break
        time.sleep(0.2)
    assert view["summary"]["mission_tick"] > 60
    mission, _ = call(f"/api/v2/missions/{mission_id}")
    assert mission["status"] == "executing"
    legacy, headers = call(f"/api/v2/missions/{mission_id}/trajectory")
    assert headers.get("Deprecation") == "true"
    assert legacy["summary"]["hot_tape_horizon_seconds"] == 0 and legacy["hot_tape"] == {}

    print(json.dumps({"status": "pass", "mode": "exercise", "program_id": summary["program_id"], "installed_nodes": summary["installed_node_count"], "segments": summary["total_segments"], "continued_to_tick": view["summary"]["mission_tick"], "decision_node": decision["decision_node_id"], "legacy_tape_depth": 0}))
finally:
    current, _ = call("/api/v2/fleet")
    call("/api/v2/scenarios/fleet-operations:reset", mutation("cleanup", current["fleet_version"]), method="POST")
