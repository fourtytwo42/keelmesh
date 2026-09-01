#!/usr/bin/env python3
"""Exercise the complete M1 API contract against a running appliance."""

import json
import sys
import time
import urllib.error
import urllib.request

base = (sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:8080").rstrip("/")


def request(path, payload=None, expected=200):
    data = None if payload is None else json.dumps(payload).encode()
    req = urllib.request.Request(base + path, data=data, headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=5) as response:
            status, body = response.status, json.load(response)
    except urllib.error.HTTPError as error:
        status, body = error.code, json.load(error)
    if status != expected:
        raise AssertionError(f"{path}: expected HTTP {expected}, got {status}: {body}")
    return body


boot = request("/api/v1/bootstrap")
version = boot["snapshot"]["state_version"]
initial_positions = {v["id"]: v["position"] for v in boot["snapshot"]["vessels"]}
intent = request("/api/v1/intents:compile", {
    "request_id": "verify-intent", "expected_state_version": version,
    "text": "Search this area with six vessels. Maintain 30% reserve and finish in 20 minutes.",
    "area": boot["suggested_area"]["geometry"],
}, 201)
plans = request("/api/v1/plans", {
    "request_id": "verify-plans", "expected_state_version": intent["source_state_version"], "intent_id": intent["id"],
}, 201)["plans"]
assert len(plans) == 2 and sum(bool(plan["recommended"]) for plan in plans) == 1
plan = next(plan for plan in plans if plan["recommended"])
preview = request(f"/api/v1/plans/{plan['id']}:preview", {
    "request_id": "verify-preview", "expected_state_version": intent["source_state_version"],
})
assert len(preview["samples"]) == preview["duration_seconds"] + 1
after_preview = request("/api/v1/bootstrap")
assert initial_positions == {v["id"]: v["position"] for v in after_preview["snapshot"]["vessels"]}

problem = request(f"/api/v1/plans/{plan['id']}:authorize", {
    "request_id": "verify-tamper", "expected_state_version": intent["source_state_version"],
    "plan_hash": plan["content_hash"] + "tampered", "operator_id": "demo-operator",
}, 422)
assert problem["code"] == "PLAN_HASH_MISMATCH"
lease = request(f"/api/v1/plans/{plan['id']}:authorize", {
    "request_id": "verify-authorize", "expected_state_version": intent["source_state_version"],
    "plan_hash": plan["content_hash"], "operator_id": "demo-operator",
}, 201)
start_payload = {
    "request_id": "verify-start", "expected_state_version": intent["source_state_version"],
    "lease_id": lease["id"], "plan_hash": plan["content_hash"], "idempotency_key": "verify-start-once",
}
mission = request(f"/api/v1/missions/{lease['mission_id']}:start", start_payload)
assert mission["phase"] == "executing"
assert request(f"/api/v1/missions/{lease['mission_id']}:start", start_payload)["id"] == mission["id"]
conflict = dict(start_payload, plan_hash=plan["content_hash"] + "different")
assert request(f"/api/v1/missions/{lease['mission_id']}:start", conflict, 422)["code"] == "IDEMPOTENCY_CONFLICT"

for _ in range(30):
    current = request("/api/v1/bootstrap")["snapshot"]
    if any(v["position"] != initial_positions[v["id"]] for v in current["vessels"]):
        break
    time.sleep(0.1)
else:
    raise AssertionError("authorized vessels did not move")

events = request(f"/api/v1/audit/{intent['trace_id']}")["events"]
required = {"intent.compiled", "plans.generated", "plan.previewed", "mission.authorized", "mission.started"}
assert required.issubset({event["kind"] for event in events})
print(json.dumps({"status": "pass", "plans": len(plans), "preview_samples": len(preview["samples"]), "audit_events": len(events)}))

