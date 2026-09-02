#!/usr/bin/env python3
"""Run and record the real KeelMesh M3 failure/recovery drill."""
from __future__ import annotations
import argparse, json, statistics, subprocess, time, urllib.error, urllib.parse, urllib.request
from pathlib import Path

def call(base, path, method="GET", body=None, timeout=70):
    data=None if body is None else json.dumps(body).encode()
    request=urllib.request.Request(base+path,data=data,method=method,headers={"Content-Type":"application/json"})
    try:
        with urllib.request.urlopen(request,timeout=timeout) as response:return json.load(response)
    except urllib.error.HTTPError as error:
        raise RuntimeError(f"{method} {path}: {error.code} {error.read().decode()}") from error

def mutation(version, label, **extra):
    stamp=time.time_ns();return {"request_id":f"verify-{label}-{stamp}","idempotency_key":f"verify-{label}-{stamp}","expected_platform_state_version":version,**extra}

def wait_for(base, predicate, seconds, label):
    deadline=time.time()+seconds;last=None
    while time.time()<deadline:
        last=call(base,"/api/v1/platform")
        if predicate(last):return last
        time.sleep(1)
    raise RuntimeError(f"Timed out waiting for {label}; last={last and last.get('summary')}")

def container_resources():
    result=subprocess.run(["docker","stats","--no-stream","--format","{{json .}}"],capture_output=True,text=True,check=False)
    resources=[]
    for line in result.stdout.splitlines():
        try:
            item=json.loads(line)
            if item.get("Name","").startswith("keelmesh-"):resources.append({"name":item.get("Name"),"cpu":item.get("CPUPerc"),"memory":item.get("MemUsage"),"memory_percent":item.get("MemPerc"),"network_io":item.get("NetIO"),"block_io":item.get("BlockIO")})
        except json.JSONDecodeError:pass
    return sorted(resources,key=lambda item:item["name"])

def build_metadata():
    def capture(command):
        completed=subprocess.run(command,capture_output=True,text=True,check=False)
        return completed.stdout.strip() if completed.returncode==0 else "unknown"
    return {"commit":capture(["git","rev-parse","HEAD"]),"image_digest":capture(["docker","image","inspect","keelmesh/core:m3","--format","{{.Id}}"])}

def main():
    parser=argparse.ArgumentParser();parser.add_argument("base",nargs="?",default="http://127.0.0.1:8080");parser.add_argument("--json",dest="json_path");parser.add_argument("--markdown",dest="markdown_path");parser.add_argument("--resources-only",action="store_true");args=parser.parse_args();base=args.base.rstrip("/")
    if args.resources_only:
        if not args.json_path:raise RuntimeError("--resources-only requires --json")
        path=Path(args.json_path);report=json.loads(path.read_text());report.update(build_metadata());report["container_resources"]=container_resources();path.write_text(json.dumps(report,indent=2)+"\n")
        if args.markdown_path:
            md=Path(args.markdown_path);text=md.read_text();metadata_heading="## Build provenance";text=text.split(metadata_heading)[0].rstrip()+"\n\n"+metadata_heading+f"\n\n- Commit: `{report['commit']}`\n- Image: `{report['image_digest']}`\n\n";heading="## Container resources (post-drill)";text+=heading+"\n\n"+"\n".join(f"- {item['name']}: CPU {item['cpu']}, memory {item['memory']} ({item['memory_percent']})" for item in report["container_resources"])+"\n";md.write_text(text)
        print(json.dumps(report["container_resources"],indent=2));return
    platform=wait_for(base,lambda p:p["available"] and len([w for w in p["workers"] if w["state"]=="running"])==3,30,"three live workers")
    if platform.get("active_run",{}).get("state")=="running":
        run=platform["active_run"]
    else:
        run=call(base,"/api/v1/load/runs","POST",mutation(platform["state_version"],"start",profile="interview",seed=424242))
    command_latencies=[]
    for _ in range(10):
        started=time.perf_counter();call(base,"/api/v1/bootstrap");command_latencies.append((time.perf_counter()-started)*1000);time.sleep(1)
    baseline=call(base,"/api/v1/platform");worker2=next(w for w in baseline["workers"] if w["id"]=="worker-2");old_pid=worker2["pid"];baseline_lag=baseline["metrics"]["current_lag"]
    call(base,"/api/v1/platform/faults","POST",mutation(baseline["state_version"],"fault",kind="terminate_worker",target_id="worker-2"))
    down=wait_for(base,lambda p:not any(w["id"]=="worker-2" and w["state"]=="running" for w in p["workers"]),20,"Worker 2 down")
    recovered=wait_for(base,lambda p:any(w["id"]=="worker-2" and w["state"]=="running" and w["pid"]!=old_pid for w in p["workers"]) and p["metrics"]["current_lag"]<=max(100,baseline_lag),60,"Worker 2 recovery and lag drain")
    recovered=wait_for(base,lambda p:p["metrics"]["attempted"]>=120000,65,"120,000 attempted events")
    stopped=call(base,f"/api/v1/load/runs/{run['id']}:stop","POST",mutation(recovered["state_version"],"stop"))
    final=wait_for(base,lambda p:p["metrics"]["current_lag"]==0 and p["metrics"]["attempted"]==p["metrics"]["produced"],30,"zero Kafka lag and flushed producer callbacks")
    accounting_metrics=dict(final["metrics"])
    pending=next((q for q in (final.get("quarantine") or []) if q["repair_state"]=="pending"),None);redrive=None
    if pending:
        redrive=call(base,f"/api/v1/quarantine/{pending['id']}:redrive","POST",mutation(final["state_version"],"redrive"));time.sleep(2);final=call(base,"/api/v1/platform")
    replay=call(base,"/api/v1/platform/replays","POST",mutation(final["state_version"],"replay",source_run_id=run["id"]),timeout=70)
    retrieval=call(base,"/api/v1/retrieval/similar?q="+urllib.parse.quote("consumer worker rebalance lag recovery"))["hits"]
    evidence=call(base,f"/api/v1/evidence/{run['id']}")
    metrics=call(base,"/api/v1/platform")["metrics"]
    accounting=accounting_metrics["unique_inserted"]+accounting_metrics["duplicates_suppressed"]+accounting_metrics["quarantined"]+accounting_metrics["dropped"]
    report={**build_metadata(),"run":stopped,"attempted":metrics["attempted"],"accounted":accounting,"accounting_matches":abs(metrics["attempted"]-accounting)<=3,"events_per_second":baseline["metrics"]["events_per_second"],"latency_p50_ms":metrics["latency_p50_ms"],"latency_p95_ms":metrics["latency_p95_ms"],"latency_p99_ms":metrics["latency_p99_ms"],"mission_api_p95_ms":statistics.quantiles(command_latencies,n=20)[18],"lag_baseline":baseline_lag,"lag_peak":metrics["peak_lag"],"worker_2_old_pid":old_pid,"worker_2_down_workers":len([w for w in down["workers"] if w["state"]=="running"]),"worker_2_new_pid":next(w["pid"] for w in recovered["workers"] if w["id"]=="worker-2"),"rebalance_count":metrics["rebalance_count"],"recovery_seconds":metrics["recovery_seconds"],"duplicates_suppressed":metrics["duplicates_suppressed"],"out_of_order":metrics["out_of_order"],"quarantined":metrics["quarantined"],"redrive":redrive,"replay":replay,"retrieval_top_hit":retrieval[0] if retrieval else None,"dropped":metrics["dropped"],"workers":evidence["workers"],"container_resources":container_resources(),"hardware":evidence["hardware"]}
    if report["attempted"]<120000:raise RuntimeError(f"Interview workload attempted only {report['attempted']} events")
    if not report["accounting_matches"]:raise RuntimeError(f"Accounting mismatch: attempted={report['attempted']} accounted={report['accounted']}")
    if report["dropped"]!=0:raise RuntimeError(f"Interview workload dropped {report['dropped']} events")
    if not replay["matches"]:raise RuntimeError("Shadow replay does not match live projection")
    rendered=json.dumps(report,indent=2);print(rendered)
    if args.json_path:Path(args.json_path).write_text(rendered+"\n")
    if args.markdown_path:
        Path(args.markdown_path).write_text(f"# KeelMesh M3 evidence\n\n- Run: `{run['id']}` (seed `{run['seed']}`)\n- Attempted/accounted: **{report['attempted']:,} / {report['accounted']:,}**\n- Throughput: **{report['events_per_second']:.0f} events/s**\n- Ingest p95/p99: **{report['latency_p95_ms']:.1f} / {report['latency_p99_ms']:.1f} ms**\n- Worker 2 PID: `{old_pid}` → terminated → `{report['worker_2_new_pid']}`\n- Peak lag: **{report['lag_peak']:,}**; recovery: **{report['recovery_seconds']:.1f}s**\n- Duplicate / out-of-order / quarantined / dropped: **{report['duplicates_suppressed']:,} / {report['out_of_order']:,} / {report['quarantined']:,} / {report['dropped']:,}**\n- Kafka shadow replay: **{'MATCH' if replay['matches'] else 'MISMATCH'}** (`{replay['shadow_checksum']}`)\n- Hardware: {report['hardware']}\n")

if __name__=="__main__":main()
