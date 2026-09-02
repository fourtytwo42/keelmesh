#!/usr/bin/env python3
"""Exercise and export the bounded KeelMesh M4 incident-to-evaluation workflow."""
from __future__ import annotations

import argparse
import json
import subprocess
import time
import urllib.error
import urllib.request
from pathlib import Path


def call(base: str, path: str, method: str = "GET", body: dict | None = None, timeout: int = 45):
    data = None if body is None else json.dumps(body).encode()
    request = urllib.request.Request(base + path, data=data, method=method, headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            return json.load(response)
    except urllib.error.HTTPError as error:
        detail = error.read().decode()
        raise RuntimeError(f"{method} {path}: {error.code} {detail}") from error


def mutation(version: int, label: str, **extra):
    stamp = time.time_ns()
    return {"request_id": f"verify-m4-{label}-{stamp}", "idempotency_key": f"verify-m4-{label}-{stamp}", "expected_ai_state_version": version, **extra}


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("base", nargs="?", default="http://127.0.0.1:8080")
    parser.add_argument("--json", dest="json_path")
    parser.add_argument("--markdown", dest="markdown_path")
    args = parser.parse_args()
    base = args.base.rstrip("/")

    state = call(base, "/api/v1/ai")
    if not state["available"]:
        raise RuntimeError("AI service is not available")
    call(base, "/api/v1/scenarios/ai-tooling:reset", "POST", mutation(state["state_version"], "reset"))
    state = call(base, "/api/v1/ai")
    incident = state["incidents"][0]
    investigation = call(base, f"/api/v1/incidents/{incident['id']}:investigate", "POST", mutation(state["state_version"], "investigate"))
    if len(investigation["tool_receipts"]) != 8 or not investigation["citations"]:
        raise RuntimeError("Investigation did not produce eight scoped receipts and citations")
    state = call(base, "/api/v1/ai")
    replay = call(base, f"/api/v1/investigations/{investigation['id']}:replay", "POST", mutation(state["state_version"], "replay"))
    if not replay["matches"] or replay["live_state_changed"]:
        raise RuntimeError("Isolated replay failed determinism or changed live state")
    state = call(base, "/api/v1/ai")
    candidate = state["candidate"]
    tamper = mutation(state["state_version"], "tamper", candidate_hash="sha256:tampered", operator_identity="demo-engineer")
    try:
        call(base, f"/api/v1/eval-candidates/{candidate['id']}:approve", "POST", tamper)
        raise RuntimeError("Tampered candidate hash was accepted")
    except RuntimeError as error:
        if "EVAL_HASH_MISMATCH" not in str(error):
            raise
    state = call(base, "/api/v1/ai")
    approval = call(base, f"/api/v1/eval-candidates/{candidate['id']}:approve", "POST", mutation(state["state_version"], "approve", candidate_hash=candidate["candidate_hash"], operator_identity="demo-engineer"))
    state = call(base, "/api/v1/ai")
    evaluation = call(base, "/api/v1/evaluations/runs", "POST", mutation(state["state_version"], "eval", candidate_id=candidate["id"]))
    final = call(base, "/api/v1/ai")
    trace = call(base, f"/api/v1/traces/{investigation['trace_id']}")
    evidence = call(base, f"/api/v1/evidence/ai/{evaluation['id']}")
    health = call(base, "/healthz")
    platform = call(base, "/api/v1/platform")

    unauthorized = subprocess.run(["docker", "exec", "keelmesh-core-1", "wget", "-qO-", "--post-data={}", "http://127.0.0.1:8081/mcp"], capture_output=True, text=True, check=False)
    if unauthorized.returncode == 0:
        raise RuntimeError("Unauthenticated private MCP call unexpectedly succeeded")

    report = {
        "incident_id": incident["id"], "state_checksum": incident["state_checksum"],
        "tool_receipt_count": len(investigation["tool_receipts"]), "tools": [item["tool"] for item in investigation["tool_receipts"]],
        "citations": investigation["citations"], "provider_attempts": investigation["provider_attempts"],
        "replay": replay, "candidate_hash": candidate["candidate_hash"], "approval": approval,
        "evaluation": evaluation, "trace_id": trace["trace_id"], "span_count": len(trace["spans"]),
        "mcp_unauthorized_denied": True, "core_health": health, "platform_available": platform["available"],
        "evidence": evidence,
    }
    rendered = json.dumps(report, indent=2)
    print(rendered)
    if args.json_path:
        Path(args.json_path).write_text(rendered + "\n", encoding="utf-8")
    if args.markdown_path:
        providers = " → ".join(f"{x['provider']}:{x['model']} ({x['state']})" for x in investigation["provider_attempts"])
        Path(args.markdown_path).write_text(
            f"# KeelMesh M4 evidence\n\n- Incident: `{incident['id']}`\n- MCP tools: **{len(investigation['tool_receipts'])}/8** with immutable receipts\n- Provider route: {providers}\n- Replay: **{'MATCH' if replay['matches'] else 'DIVERGED'}**, live state changed: `{replay['live_state_changed']}`\n- Candidate: `{candidate['candidate_hash']}`\n- Human approval: **{approval['state']}** by `{approval['approved_by']}`\n- Regression: **{evaluation['state']}** ({len(evaluation['results'])} provider results)\n- Trace: `{trace['trace_id']}` with **{len(trace['spans'])} spans**\n- Unauthenticated MCP request: **DENIED**\n- M3 platform remained available: `{platform['available']}`\n",
            encoding="utf-8",
        )


if __name__ == "__main__":
    main()
