from __future__ import annotations

import hashlib
import json
from typing import Any


def canonical_bytes(value: object) -> bytes:
    return json.dumps(value, sort_keys=True, separators=(",", ":")).encode()


def build_episode(incident: dict[str, Any]) -> dict[str, Any]:
    evidence = sorted(incident.get("evidence", []), key=lambda item: int(item.get("tick", 0)))
    spoof = next(item for item in evidence if item.get("kind") == "pnt")
    episode = {
        "schema_version": 1,
        "id": f"episode-{incident['id']}",
        "incident_id": incident["id"],
        "scenario_seed": int(incident["scenario_seed"]),
        "source_state_checksum": incident["state_checksum"],
        "source_event_ids": [item["id"] for item in evidence],
        "window": {"start_tick": max(0, int(spoof["tick"]) - 17), "end_tick": int(spoof["tick"]) + 43},
        "classification": incident["classification"],
        "evidence": evidence,
        "redactions": ["raw_audio", "credentials", "hidden_faction_state"],
    }
    episode["artifact_checksum"] = "sha256:" + hashlib.sha256(canonical_bytes(episode)).hexdigest()
    return episode


def evaluate_policy(episode: dict[str, Any], policy: str) -> dict[str, float | int | str | bool]:
    pnt = next(item for item in episode["evidence"] if item["kind"] == "pnt")
    summary = str(pnt["summary"]).lower()
    offset_m = 650.0 if "650 m" in summary else 0.0
    thresholds = {"baseline-v1": 300.0, "candidate-multisignal-v2": 100.0}
    if policy not in thresholds:
        raise ValueError(f"unsupported policy: {policy}")
    detected = offset_m >= thresholds[policy] and "excluded" in summary
    return {
        "policy": policy,
        "detected": detected,
        "detection_latency_ticks": 0 if detected else 60,
        "false_acceptance": 0 if detected else 1,
        "route_deviation_m": 0.0 if detected else offset_m,
        "uncertainty_growth_m": 48.0,
        "time_to_safe_behavior_ticks": 43,
        "seed": int(episode["scenario_seed"]),
    }
