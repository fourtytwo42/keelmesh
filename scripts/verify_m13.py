#!/usr/bin/env python3
"""Verify M13 platform-proof contracts and optionally run one bounded worker drill."""

from __future__ import annotations

import argparse
import json
import time
import urllib.error
import urllib.request
import uuid
from pathlib import Path
from typing import Any


def call(base: str, path: str, method: str = "GET", body: dict[str, Any] | None = None) -> dict[str, Any]:
    payload = None if body is None else json.dumps(body).encode()
    request = urllib.request.Request(base + path, data=payload, method=method, headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(request, timeout=15) as response:
            return json.load(response)
    except urllib.error.HTTPError as error:
        detail = error.read().decode(errors="replace")
        raise RuntimeError(f"{method} {path} returned {error.code}: {detail}") from error


def mutation(prefix: str, state_version: int) -> dict[str, Any]:
    token = f"m13-{prefix}-{uuid.uuid4()}"
    return {"request_id": token, "idempotency_key": token, "expected_platform_state_version": state_version}


def run_worker_drill(base: str) -> dict[str, Any]:
    platform = call(base, "/api/v1/platform")
    target = next((item for item in platform["workers"] if item["id"] == "worker-2" and item["state"] == "running"), None)
    if target is None:
        target = next((item for item in platform["workers"] if item["state"] == "running"), None)
    if target is None:
        raise RuntimeError("No running worker is available for the bounded recovery drill.")
    request = {
        **mutation("worker-recovery", int(platform["state_version"])),
        "type": "data_pipeline_worker_recovery",
        "target_id": target["id"],
        "actor_identity": "m13-verifier",
        "confirmed": True,
    }
    receipt = call(base, "/api/v6/platform/drills", "POST", request)
    deadline = time.monotonic() + 90
    while receipt["state"] == "running" and time.monotonic() < deadline:
        time.sleep(1)
        receipt = call(base, f"/api/v6/platform/drills/{receipt['id']}")
    if receipt["state"] == "running":
        raise RuntimeError("Worker drill did not reach a terminal state within 90 seconds.")
    return receipt


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("base", nargs="?", default="http://127.0.0.1:8080")
    parser.add_argument("--json", dest="json_path")
    parser.add_argument("--markdown", dest="markdown_path")
    parser.add_argument("--run-worker-drill", action="store_true")
    parser.add_argument("--confirm", action="store_true", help="Required with --run-worker-drill because it terminates one supervised worker child.")
    args = parser.parse_args()
    if args.run_worker_drill and not args.confirm:
        parser.error("--run-worker-drill requires --confirm")
    base = args.base.rstrip("/")
    summary = call(base, "/api/v6/platform/summary")
    capacity = call(base, "/api/v6/platform/capacity")
    costs = call(base, "/api/v6/platform/cost-model")
    evaluation = call(base, "/api/v6/platform/evaluations/episode-incident-vessel4-resilient-edge")
    traces = call(base, "/api/v6/platform/traces?limit=5")
    drills = call(base, "/api/v6/platform/drills")
    assert {item["id"] for item in summary["planes"]} == {"edge", "coordination", "data", "ai"}
    assert all("source" in item and "measured" in item for item in summary["planes"])
    duplicate_slo = next(item for item in summary["slos"] if item["id"] == "duplicate-effects")
    assert duplicate_slo["measured"] is False or any(item["outcome"] == "pass" for item in drills["drills"])
    assert capacity["capacity"][0]["asset_count"] == 12 and capacity["capacity"][0]["evidence_class"] == "measured"
    assert all(item["evidence_class"] == "projected" for item in capacity["capacity"][1:])
    assert all(item["evidence_class"] == "projected" for item in costs["cost"])
    assert evaluation["source_event_ids"] and evaluation["artifact_checksum"].startswith("sha256:")
    assert evaluation["promotion_state"] == "awaiting_privileged_human_decision"
    drill = run_worker_drill(base) if args.run_worker_drill else None
    if drill is not None:
        assert drill["outcome"] == "pass", drill
        assert drill["evidence_hash"].startswith("sha256:")
    result = {"schema_version": 1, "verified_at": summary["sampled_at"], "base_url": base, "commit": summary["commit"], "planes": summary["planes"], "slos": summary["slos"], "capacity": capacity, "cost_model": costs, "evaluation": evaluation, "recent_trace_count": len(traces["traces"]), "latest_worker_drill": drill, "status": "pass"}
    encoded = json.dumps(result, indent=2, sort_keys=True)
    if args.json_path:
        Path(args.json_path).write_text(encoded + "\n", encoding="utf-8")
    if args.markdown_path:
        measured = capacity["capacity"][0]
        Path(args.markdown_path).write_text(
            "# KeelMesh M13 platform proof\n\n"
            f"- Commit: `{summary['commit']}`\n"
            f"- Four planes: **{len(summary['planes'])}/4 evidenced**\n"
            f"- Recent real traces: **{len(traces['traces'])}**\n"
            f"- Measured 12-node event rate: **{measured['events_per_second']:.0f} events/s**\n"
            f"- GNSS episode: `{evaluation['episode_id']}` (`{evaluation['artifact_checksum']}`)\n"
            f"- Promotion: **{evaluation['promotion_state']}**\n"
            f"- Worker drill: **{drill['outcome'] if drill else 'not run'}**\n"
            "- 100/1,000-asset and cloud-cost rows are labeled projections, not benchmarks.\n",
            encoding="utf-8",
        )
    print(encoded)


if __name__ == "__main__":
    main()
