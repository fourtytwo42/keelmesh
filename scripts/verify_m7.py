#!/usr/bin/env python3
import argparse, json, sys, urllib.error, urllib.request, uuid

def call(base, path, method="GET", body=None, expected=200):
    data=None if body is None else json.dumps(body).encode()
    req=urllib.request.Request(base+path,data=data,method=method,headers={"Content-Type":"application/json"})
    try:
        with urllib.request.urlopen(req,timeout=10) as response:
            payload=json.load(response)
            if response.status!=expected: raise AssertionError((response.status,payload))
            return payload
    except urllib.error.HTTPError as error:
        payload=json.load(error)
        if error.code!=expected: raise AssertionError((error.code,payload))
        return payload

def mutation(snapshot,label):
    return {"request_id":f"verify-{label}-{uuid.uuid4().hex[:8]}","idempotency_key":f"verify-key-{label}-{uuid.uuid4().hex[:8]}","expected_version":snapshot["state_version"],"actor_id":"A"}

def main():
    parser=argparse.ArgumentParser();parser.add_argument("--base-url",default="http://127.0.0.1:8080");args=parser.parse_args();base=args.base_url.rstrip("/")
    state=call(base,"/api/v3/arena?faction=A")
    state=call(base,"/api/v3/scenarios/fleet-arena:reset","POST",mutation(state,"reset"))
    assert len(state["nodes"])==6 and state["viewer_faction"]=="A"
    b=call(base,"/api/v3/arena?faction=B"); assert len(b["nodes"])==6 and b["viewer_faction"]=="B"
    assert not ({n["id"] for n in state["nodes"]}&{n["id"] for n in b["nodes"]})
    infra=call(base,"/api/v3/infrastructure");assert len(infra["nodes"])==12
    state=call(base,"/api/v3/matches","POST",mutation(state,"start"),201)
    denied=call(base,f'/api/v3/matches/{state["match_id"]}/faults',"POST",{**mutation(state,"protected"),"faction":"A","kind":"fail_management"},403);assert denied["code"]=="PROTECTED_PLANE"
    state=call(base,f'/api/v3/matches/{state["match_id"]}/faults',"POST",{**mutation(state,"starlink"),"faction":"A","kind":"fail_starlink"});assert state["radio_plane"]=="HaLow-only"
    state=call(base,f'/api/v3/matches/{state["match_id"]}/faults',"POST",{**mutation(state,"election"),"faction":"A","kind":"partition_coordinator"});coord=state["coordinators"][0];assert coord["node_id"]=="node-a-02" and coord["epoch"]==2 and coord["votes"]==5
    assert all(n["management_connected"] and n["inference_connected"] for n in state["nodes"])
    session=call(base,"/api/v3/agent/sessions","POST",{**mutation(state,"session"),"faction":"A","text":""},201)
    state=call(base,"/api/v3/arena?faction=A")
    turn=call(base,f'/api/v3/agent/sessions/{session["id"]}/messages',"POST",{**mutation(state,"message"),"faction":"A","text":"frame my fleet and show radar contacts"});assert len(turn["actions"])>=4
    state=call(base,"/api/v3/arena?faction=A")
    plan=call(base,f'/api/v3/matches/{state["match_id"]}/engagements:plan',"POST",{**mutation(state,"plan"),"faction":"A","friendly_node_ids":["node-a-05"],"target_track_ids":[state["knowledge"]["contacts"][0]["id"]],"equipment":["light_kinetic"],"maximum_effects":1},201)
    state=call(base,"/api/v3/arena?faction=A")
    lease=call(base,f'/api/v3/matches/{state["match_id"]}/engagements/{plan["id"]}:authorize',"POST",{**mutation(state,"authorize"),"plan_hash":plan["content_hash"],"operator_id":"verify-player-a"},201)
    state=call(base,"/api/v3/arena?faction=A")
    effect=call(base,f'/api/v3/matches/{state["match_id"]}/effects',"POST",{**mutation(state,"effect"),"lease_id":lease["id"],"target_track_id":plan["target_track_ids"][0],"equipment":plan["equipment"][0]},202)
    result={"status":"pass","physical_nodes":len(infra["nodes"]),"knowledge_isolated":True,"protected_fault_denied":True,"coordinator_after_failover":coord["node_id"],"epoch":coord["epoch"],"workspace_actions":len(turn["actions"]),"exact_plan_hash":plan["content_hash"],"effect_receipt":effect["receipt_hash"]}
    print(json.dumps(result,indent=2))
if __name__=="__main__":
    try: main()
    except Exception as error: print(json.dumps({"status":"fail","error":str(error)},indent=2));sys.exit(1)
