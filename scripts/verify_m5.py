#!/usr/bin/env python3
"""Verify Quiet Fleet authority, rejection, future commit, and activation."""
import hashlib, json, sys, time, urllib.error, urllib.request

base=(sys.argv[1] if len(sys.argv)>1 else "http://127.0.0.1:8080").rstrip("/")
stamp=str(time.time_ns())
def call(path,payload=None,status=200):
    data=None if payload is None else json.dumps(payload).encode()
    request=urllib.request.Request(base+path,data=data,headers={"Content-Type":"application/json"})
    try:
        with urllib.request.urlopen(request,timeout=10) as response: code,body=response.status,json.load(response)
    except urllib.error.HTTPError as error: code,body=error.code,json.load(error)
    if code!=status: raise AssertionError(f"{path}: expected {status}, got {code}: {body}")
    return body
def mutation(kind,version,proposal_hash=None):
    key=f"m5-{kind}-{stamp}"
    body={"schema_version":1,"kind":kind,"request_id":key,"idempotency_key":key,"expected_state_version":version}
    if proposal_hash: body["proposal_hash"]=proposal_hash
    return body

boot=call("/api/v1/bootstrap"); version=boot["snapshot"]["state_version"]
call("/api/v1/scenarios/demo:reset",{"request_id":"m5-reset-"+stamp,"idempotency_key":"m5-reset-"+stamp,"expected_state_version":version})
boot=call("/api/v1/bootstrap"); version=boot["snapshot"]["state_version"]
intent=call("/api/v1/intents:compile",{"request_id":"m5-intent-"+stamp,"expected_state_version":version,"text":"Search with six vessels, maintain 30% reserve, avoid the exclusion zone.","area":boot["suggested_area"]["geometry"]},201)
plans=call("/api/v1/plans",{"request_id":"m5-plans-"+stamp,"expected_state_version":intent["source_state_version"],"intent_id":intent["id"]},201)["plans"]
plan=next(item for item in plans if item["recommended"])
lease=call(f"/api/v1/plans/{plan['id']}:authorize",{"request_id":"m5-authorize-"+stamp,"expected_state_version":intent["source_state_version"],"plan_hash":plan["content_hash"],"operator_id":"demo-operator"},201)
call(f"/api/v1/missions/{lease['mission_id']}:start",{"request_id":"m5-start-"+stamp,"idempotency_key":"m5-start-"+stamp,"expected_state_version":intent["source_state_version"],"lease_id":lease["id"],"plan_hash":plan["content_hash"]})
quiet=call("/api/v1/quiet-fleet")
contract_hash=quiet["contract"]["content_hash"]
quiet=call("/api/v1/quiet-fleet/commands",mutation("enter_mode",quiet["state_version"]))
quiet=call("/api/v1/quiet-fleet/commands",mutation("inject_slowdown",quiet["state_version"]))
assert quiet["metrics"]["quorum_count"]==3 and quiet["metrics"]["affected_armed"]==3
assert next(d for d in quiet["decisions"] if d["node_id"]=="vessel-02")["reason_code"]=="SPEED_ENVELOPE_EXCEEDED"
rejected=call("/api/v1/quiet-fleet/commands",mutation("commit_proposal",quiet["state_version"],quiet["proposal"]["content_hash"]),422)
assert rejected["code"]=="AFFECTED_NODE_NOT_ARMED"
quiet=call("/api/v1/quiet-fleet/commands",mutation("submit_revision",quiet["state_version"]))
assert quiet["metrics"]["quorum_count"]==4 and quiet["metrics"]["affected_armed"]==4
assert max(a["speed_mps"] for a in quiet["proposal"]["assignments"])<=1.6
tampered=call("/api/v1/quiet-fleet/commands",mutation("commit_proposal",quiet["state_version"],quiet["proposal"]["content_hash"]+"x"),422)
assert tampered["code"]=="COMMIT_HASH_MISMATCH"
active_before=hashlib.sha256(json.dumps(quiet["active_assignments"],sort_keys=True).encode()).hexdigest()
quiet=call("/api/v1/quiet-fleet/commands",mutation("commit_proposal",quiet["state_version"],quiet["proposal"]["content_hash"]))
assert active_before==hashlib.sha256(json.dumps(quiet["active_assignments"],sort_keys=True).encode()).hexdigest()
assert quiet["commit"]["activation_tick"]-quiet["commit"]["commit_tick"]>=20 and quiet["commit"]["activation_tick"]%10==0
quiet=call("/api/v1/quiet-fleet/commands",mutation("advance_to_activation",quiet["state_version"]))
assert quiet["phase"]=="activated" and active_before!=hashlib.sha256(json.dumps(quiet["active_assignments"],sort_keys=True).encode()).hexdigest()
assert quiet["contract"]["content_hash"]==contract_hash
assert quiet["metrics"]["rounds"]<=3 and all(w["bytes_used"]<=w["byte_budget"] for w in quiet["windows"])
print(json.dumps({"status":"pass","phase":quiet["phase"],"proposal_rejection":"AFFECTED_NODE_NOT_ARMED","quorum":"4/4","activation_tick":quiet["mission_tick"],"coordination_bytes":quiet["metrics"]["bytes_sent"],"rounds":quiet["metrics"]["rounds"],"contract_hash":contract_hash}))
