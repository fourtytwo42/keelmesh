from __future__ import annotations

import hashlib
import json
import os
import urllib.request

import dagster as dg
import mlflow


def memory_snapshot() -> dict[str, object]:
    with urllib.request.urlopen("http://core:8080/api/v5/memory", timeout=5) as response:
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


defs = dg.Definitions(assets=[document_inventory, embedding_index, memory_consolidation])
