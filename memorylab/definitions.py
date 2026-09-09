from __future__ import annotations

import hashlib
import json
import os
import urllib.request

import boto3
import dagster as dg
import mlflow

from memorylab.gnss_eval import build_episode, canonical_bytes, evaluate_policy


def memory_snapshot() -> dict[str, object]:
    with urllib.request.urlopen("http://core:8080/api/v5/memory", timeout=5) as response:
        return json.load(response)


def api_json(path: str) -> dict[str, object]:
    with urllib.request.urlopen(f"http://core:8080{path}", timeout=5) as response:
        return json.load(response)


@dg.asset(group_name="memory_ingestion")
def document_inventory() -> dg.MaterializeResult:
    snapshot = memory_snapshot()
    return dg.MaterializeResult(metadata={"committed_items": int(snapshot["committed_items"]), "embedding_version": str(snapshot["embedding_version"])})


@dg.asset(deps=[document_inventory], group_name="memory_ingestion")
def embedding_index() -> dg.MaterializeResult:
    snapshot = memory_snapshot()
    return dg.MaterializeResult(metadata={"state": str(snapshot["embedding_state"]), "dimensions": 384})


@dg.asset(deps=[embedding_index], group_name="memory_learning")
def memory_consolidation() -> dg.MaterializeResult:
    snapshot = memory_snapshot()
    payload = json.dumps(snapshot, sort_keys=True).encode()
    mlflow.set_tracking_uri(os.environ.get("MLFLOW_TRACKING_URI", "http://mlflow:5000"))
    mlflow.set_experiment("keelmesh-memory")
    with mlflow.start_run(run_name="memory-consolidation"):
        mlflow.log_params({"embedding": snapshot["embedding_version"], "retrieval": snapshot["retrieval_mode"]})
        mlflow.log_metrics({"committed_items": float(snapshot["committed_items"]), "conversation_turns": float(snapshot["conversation_turns"])})
        mlflow.set_tag("projection_checksum", hashlib.sha256(payload).hexdigest())
    return dg.MaterializeResult(metadata={"projection_checksum": hashlib.sha256(payload).hexdigest()})


@dg.asset(group_name="gnss_data_flywheel")
def gnss_operational_episode() -> dg.Output[dict[str, object]]:
    incident = api_json("/api/v1/incidents/incident-vessel4-resilient-edge")
    episode = build_episode(incident)
    return dg.Output(
        episode,
        metadata={
            "episode_id": str(episode["id"]),
            "source_state_checksum": str(episode["source_state_checksum"]),
            "artifact_checksum": str(episode["artifact_checksum"]),
            "source_events": len(episode["source_event_ids"]),
        },
    )


@dg.asset(deps=[gnss_operational_episode], group_name="gnss_data_flywheel")
def gnss_dataset_artifact(gnss_operational_episode: dict[str, object]) -> dg.MaterializeResult:
    payload = canonical_bytes(gnss_operational_episode)
    key = f"evaluations/gnss/{gnss_operational_episode['id']}.json"
    client = boto3.client(
        "s3",
        endpoint_url=os.environ.get("MLFLOW_S3_ENDPOINT_URL", "http://minio:9000"),
        aws_access_key_id=os.environ["AWS_ACCESS_KEY_ID"],
        aws_secret_access_key=os.environ["AWS_SECRET_ACCESS_KEY"],
    )
    client.put_object(Bucket="keelmesh-memory", Key=key, Body=payload, ContentType="application/json")
    return dg.MaterializeResult(metadata={"s3_key": key, "bytes": len(payload), "sha256": hashlib.sha256(payload).hexdigest()})


@dg.asset(deps=[gnss_dataset_artifact], group_name="gnss_data_flywheel")
def gnss_policy_evaluation(gnss_operational_episode: dict[str, object]) -> dg.MaterializeResult:
    baseline = evaluate_policy(gnss_operational_episode, "baseline-v1")
    candidate = evaluate_policy(gnss_operational_episode, "candidate-multisignal-v2")
    mlflow.set_tracking_uri(os.environ.get("MLFLOW_TRACKING_URI", "http://mlflow:5000"))
    mlflow.set_experiment("keelmesh-gnss-integrity")
    with mlflow.start_run(run_name=str(gnss_operational_episode["id"])):
        mlflow.log_params({"episode_id": gnss_operational_episode["id"], "seed": gnss_operational_episode["scenario_seed"], "source_checksum": gnss_operational_episode["source_state_checksum"], "promotion": "human_exact_hash_required"})
        for prefix, result in (("baseline", baseline), ("candidate", candidate)):
            mlflow.log_metrics({f"{prefix}.{key}": float(value) for key, value in result.items() if isinstance(value, (int, float, bool))})
        mlflow.set_tag("artifact_checksum", str(gnss_operational_episode["artifact_checksum"]))
        mlflow.set_tag("candidate_state", "evaluation_only_not_promoted")
    comparison = {"baseline": baseline, "candidate": candidate, "promotion_state": "awaiting_privileged_human_decision"}
    return dg.MaterializeResult(metadata={"comparison_checksum": hashlib.sha256(canonical_bytes(comparison)).hexdigest(), "baseline_detected": bool(baseline["detected"]), "candidate_detected": bool(candidate["detected"]), "promotion_state": "awaiting_privileged_human_decision"})


defs = dg.Definitions(assets=[document_inventory, embedding_index, memory_consolidation, gnss_operational_episode, gnss_dataset_artifact, gnss_policy_evaluation])
