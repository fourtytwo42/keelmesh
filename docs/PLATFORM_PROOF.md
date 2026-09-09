# Cloud Platform Proof Operations

M13 presents KeelMesh as four independently failing planes: edge execution, distributed coordination, cloud data processing, and advisory AI/ML. The source plan is [M13 — Cloud Platform Proof](PLATFORM_PROOF_PLAN.md).

## Verified lab baseline

The current twelve-node release was verified on 2026-09-09. These values are evidence from the home-lab deployment, not production claims:

| Evidence | Result |
|---|---|
| Capacity workload | 1,000 logical producers; approximately 2,020 events/second; zero dropped events |
| Projection recovery | Peak lag 202; drained in approximately 19.1 seconds; replay counts and checksums matched |
| Leader recovery | 2,740 ms; six voters reconverged; zero duplicate effects |
| Radio 4/2 partition | Four-voter side committed; two-voter side did not; management/provider interruptions zero |
| Radio 3/3 partition | Both sides rejected new authority; all six voters reconverged after rollback |
| GNSS evaluation | Source-linked MinIO artifact, successful Dagster materialization, finished MLflow comparison, promotion still human-gated |
| Cross-process trace | Trace `af627f5dbae13f2d09778fa5760eb781`: core → coordination gateway → node mTLS/Raft apply, with signed quorum proof `http-2bec2891c2f89faa791ad5e6-a` |

The API supplies exact timestamps, hashes, run IDs, workload labels, and freshness. Engineer and System render that same state rather than a separate demonstration fixture.

The final trace probe created and immediately deleted an unassigned mission draft; it issued no movement. A post-deployment check also verified all twelve node exporters against the private collector, followed by one-at-a-time service restarts and full two-cell convergence.

## Evidence surfaces

- **Engineer** is the live SRE surface: plane health, SLO measurements, real OTLP traces, bounded drill controls, and capacity/cost evidence.
- **System** (Cutaway) is the reviewer surface: deployed versus production topology, state ownership, consistency guarantees, failure boundaries, and the GNSS data flywheel.
- `/api/v6/platform/*` provides the same state as versioned JSON. Unavailable evidence is reported as unavailable; projected scale and cost values are never labeled measured.

## Read-only verification

Run from VM 214 or another management-network host:

```bash
python3 scripts/verify_m13.py http://127.0.0.1:8080
```

To retain a review artifact:

```bash
python3 scripts/verify_m13.py http://127.0.0.1:8080 \
  --json evidence/m13-platform-proof.json \
  --markdown evidence/m13-platform-proof.md
```

The default command is read-only. It checks the four-plane contract, SLO provenance, real trace inventory, source-linked GNSS evaluation, human promotion gate, and measured-versus-projected labels.

## Bounded worker-recovery drill

The disruptive path requires an additional explicit flag:

```bash
python3 scripts/verify_m13.py http://127.0.0.1:8080 \
  --run-worker-drill --confirm
```

The drill resolves one currently running ingestion worker, stops only that supervised child, observes reassignment and restart, waits for lag to return to its bounded baseline, and writes a durable receipt. The worker supervisor is the rollback mechanism. It does not affect Raft, vessel execution, the simulated-radio plane, management access, or provider access.

## Raft and simulated-radio drills

Leader and partition drills run from the privileged VM 214 management shell, not from the public application container. The runner verifies the signed six-node manifest, exact management addresses, node health, passwordless access to the bounded helper, `eth1` radio addressing, and the absence of a default route on `eth1` before it changes anything. Every node-process stop or radio fault has an automatic systemd rollback armed first.

Enable the side-effect-free Raft barrier only for the bounded drill window:

```bash
export KEELMESH_PLATFORM_DRILLS_ENABLED=true
docker compose up -d core
```

Then run one profile at a time:

```bash
python3 scripts/run_m13_node_drill.py --type leader-recovery --cell A --confirm
python3 scripts/run_m13_node_drill.py --type radio-4-2 --cell A --confirm
python3 scripts/run_m13_node_drill.py --type radio-3-3 --cell A --confirm
```

The leader drill stops only the current `keelmesh-node` process after arming a 35-second restart. The partition profiles invoke `/usr/local/sbin/m7-radio-fault` only on `eth1`; that helper rejects any other interface and independently restores the qdisc after 60 seconds. The runner also checks management health and direct provider HTTPS while the radio plane is impaired, explicitly restores every target, waits for six-node state convergence, and requires a fresh four-signature no-op barrier proof.

Receipts are written atomically to ignored `evidence/runtime/`. Core mounts that directory read-only, validates each receipt's canonical SHA-256 hash, and exposes accepted records through the normal drill/evidence APIs. Disable the barrier after the drill window:

```bash
unset KEELMESH_PLATFORM_DRILLS_ENABLED
docker compose up -d core
```

The barrier mutates only the Raft audit projection. It never creates a mission, changes a group, emits an Arena effect, or grants authority to the runner.

## GNSS evaluation profile

The optional `memory-lab` Compose profile runs private MinIO, Dagster, and MLflow services. Dagster materializes a deterministic GNSS episode with exact source event IDs and a checksum, stores the bounded artifact in MinIO, and compares baseline and candidate policies in MLflow using the same episode and seed. The candidate remains `awaiting_privileged_human_decision`; the pipeline cannot promote itself.

```bash
docker compose --profile memory-lab up -d minio minio-init mlflow dagster
docker compose --profile memory-lab exec dagster \
  dagster asset materialize -f /opt/keelmesh-memory-lab/memorylab/definitions.py \
  --select gnss_operational_episode,gnss_dataset_artifact,gnss_policy_evaluation
```

Only port 8080 is public. OTLP, Kafka, PostgreSQL, MinIO, Dagster, MLflow, node management, and Raft remain private.

## Safety boundaries

- Trace export is bounded and nonblocking; collector loss cannot stop mission execution.
- Radio drills may affect only `10.77.0.0/24`; they must never target management or provider traffic.
- The public drill barrier is disabled by default and remains a side-effect-free quorum probe when explicitly enabled.
- New authority requires a four-voter cell quorum. Cached edge execution remains independent of Kafka, PostgreSQL, AI, and the optional MLOps profile.
- No Proxmox snapshot or GitHub-hosted workflow is part of M13 verification.
