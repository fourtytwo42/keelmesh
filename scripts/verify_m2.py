#!/usr/bin/env python3
"""Run M1 authority setup and the complete deterministic M2 incident."""

import json
import sys
import time
import urllib.error
import urllib.request

base = (sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:8080").rstrip("/")
run_id = str(time.time_ns())


def key(label):
    return f"m2-{label}-{run_id}"


def request(path, payload=None, expected=200):
    data = None if payload is None else json.dumps(payload).encode()
    req = urllib.request.Request(base + path, data=data, headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=8) as response:
            status, body = response.status, json.load(response)
    except urllib.error.HTTPError as error:
        status, body = error.code, json.load(error)
    if status != expected:
        raise AssertionError(f"{path}: expected HTTP {expected}, got {status}: {body}")
    return body


boot = request("/api/v1/bootstrap")
request("/api/v1/scenarios/demo:reset", {
    "request_id": key("reset"), "idempotency_key": key("reset"),
    "expected_state_version": boot["snapshot"]["state_version"],
})
boot = request("/api/v1/bootstrap")
version = boot["snapshot"]["state_version"]
intent = request("/api/v1/intents:compile", {
    "request_id": key("intent"), "expected_state_version": version,
    "text": "Search this area with six vessels. Maintain 30% reserve and avoid the exclusion zone.",
    "area": boot["suggested_area"]["geometry"],
}, 201)
plans = request("/api/v1/plans", {"request_id": key("plans"), "expected_state_version": intent["source_state_version"], "intent_id": intent["id"]}, 201)["plans"]
plan = next(candidate for candidate in plans if candidate["recommended"])
lease = request(f"/api/v1/plans/{plan['id']}:authorize", {
    "request_id": key("authorize"), "expected_state_version": intent["source_state_version"],
    "plan_hash": plan["content_hash"], "operator_id": "demo-operator",
}, 201)
request(f"/api/v1/missions/{lease['mission_id']}:start", {
    "request_id": key("start"), "expected_state_version": intent["source_state_version"],
    "lease_id": lease["id"], "plan_hash": plan["content_hash"], "idempotency_key": key("start"),
})

state = request("/api/v1/resilience")
initial_fused = next(node for node in state["nodes"] if node["id"] == "vessel-04")["pnt"]["position"]
assert state["phase"] == "ready" and next(node for node in state["nodes"] if node["id"] == "vessel-04")["tape"]["depth_seconds"] == 60

schedule = ["fail_starlink", "partition_vessel4", "inject_gnss_spoof", "restore_contact"]
snapshots = [state]
for index, kind in enumerate(schedule):
    state = request("/api/v1/faults", {
        "schema_version": 1, "kind": kind, "target_id": "vessel-04", "scenario_tick": state["mission_tick"],
        "request_id": key(f"fault-{index}"), "idempotency_key": key(f"fault-{index}"),
        "expected_state_version": state["state_version"],
    })
    snapshots.append(state)

relay, partitioned, held, rejoined = snapshots[1:]
assert relay["active_path"] == ["operator", "vessel-03", "vessel-04"]
assert relay["duplicate_deliveries"] == 1 and len(relay["hop_receipts"]) == 2
partition_node = next(node for node in partitioned["nodes"] if node["id"] == "vessel-04")
assert partitioned["active_path"] == [] and partition_node["tape"]["depth_seconds"] == 30
held_node = next(node for node in held["nodes"] if node["id"] == "vessel-04")
assert held_node["behavior"] == "safe_hold" and held_node["tape"]["watermark"] == "empty"
assert held["raw_gnss_position"] != held_node["pnt"]["position"]
assert held_node["pnt"]["position"] == initial_fused and held_node["pnt"]["uncertainty_m"] > 45
final_node = next(node for node in rejoined["nodes"] if node["id"] == "vessel-04")
assert rejoined["phase"] == "rejoined" and rejoined["discarded_sequences"] == [6, 7, 8]
assert rejoined["bridge"]["route"][0] == final_node["pnt"]["position"]
assert rejoined["bridge"]["target_sequence"] == 9 and not rejoined["bridge"]["requires_approval"]
assert [segment["sequence"] for segment in final_node["tape"]["segments"]] == [9, 10, 11, 12, 13, 14]
assert "gnss" in final_node["pnt"]["excluded_sources"] and final_node["pnt"]["integrity"] == "trusted"
assert [transition["integrity"] for transition in rejoined["pnt_transitions"]] == ["trusted", "suspect", "denied", "unsafe", "trusted"]

print(json.dumps({
    "status": "pass", "route_changes": 2, "duplicate_count": relay["duplicate_deliveries"],
    "tape_depths": [next(node for node in item["nodes"] if node["id"] == "vessel-04")["tape"]["depth_seconds"] for item in snapshots],
    "expired_segments": len(rejoined["discarded_sequences"]),
    "maximum_uncertainty_m": max(next(node for node in item["nodes"] if node["id"] == "vessel-04")["pnt"]["uncertainty_m"] for item in snapshots),
    "contingency_tick": held["mission_tick"], "bridge_target": rejoined["bridge"]["target_sequence"], "final_state": rejoined["phase"],
}))
