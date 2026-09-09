# M13 — Cloud Platform Proof

**Status:** Implemented and verified on the twelve-node lab deployment
**Purpose:** Interview-facing platform evidence and production-readiness hardening  
**Depends on:** M10 command scenes, M11 memory/data plane, and M12 Raft/mTLS authority  
**Deployment rule:** Preserve the stable VM 214 and twelve-node deployment while this work is developed and verified

## Contents

- [Outcome](#outcome)
- [Why this milestone exists](#why-this-milestone-exists)
- [Architecture narrative](#architecture-narrative)
- [Scope and priorities](#scope-and-priorities)
- [Workstream 1 — End-to-end platform tracing](#workstream-1--end-to-end-platform-tracing)
- [Workstream 2 — SLO-driven failure drills](#workstream-2--slo-driven-failure-drills)
- [Workstream 3 — Incident-to-evaluation data flywheel](#workstream-3--incident-to-evaluation-data-flywheel)
- [Workstream 4 — Capacity, recovery, and cost evidence](#workstream-4--capacity-recovery-and-cost-evidence)
- [Frontend presentation](#frontend-presentation)
- [Guided demonstration](#guided-demonstration)
- [Interfaces and records](#interfaces-and-records)
- [Delivery sequence](#delivery-sequence)
- [Verification matrix](#verification-matrix)
- [Acceptance criteria](#acceptance-criteria)
- [Explicitly deferred work](#explicitly-deferred-work)
- [Safety and rollout constraints](#safety-and-rollout-constraints)

## Outcome

M13 turns KeelMesh's existing capabilities into one inspectable platform proof:

> Operator intent → AI interpretation → deterministic policy → Raft quorum → edge execution → telemetry ingestion → controlled failure and recovery → replay/evaluation → measured reliability and cost evidence.

The milestone does not add another broad product surface. It makes the existing system easier to inspect, operate, verify, and explain to platform, SRE, cloud, autonomy, and ML-infrastructure engineers.

The primary interview claim is:

> KeelMesh separates edge execution, distributed authority, cloud data processing, and AI assistance. Every important transition is observable, failure-tested, and tied to real evidence, while the cloud and language model remain outside the vessel's safety-critical control loop.

## Why this milestone exists

KeelMesh already demonstrates twelve VM-backed nodes, independent Raft cells, mTLS identities, quorum-backed effects, local execution, Kafka/PostgreSQL processing, scoped memory, and controlled AI tools. The remaining weakness is that these capabilities are presented as adjacent features rather than one measurable operational system.

M13 optimizes for the questions a cloud-platform reviewer is likely to ask:

- Which components and network paths are real, simulated, optional, or planned?
- What state is authoritative, where is it stored, and what consistency model applies?
- What continues when a leader, worker, broker, database, provider, or network path fails?
- How are retries, stale versions, replay, and duplicate effects handled?
- How are deployments built, promoted, observed, rolled back, and recovered?
- What are the measured latency, capacity, recovery, and cost boundaries?
- How does operational data become an evaluation and then a controlled improvement?
- How would the laboratory deployment map to a multi-zone cloud environment without moving safety-critical control into the cloud?

## Architecture narrative

The frontend and documentation must consistently describe four separate planes.

| Plane | Responsibility | Authoritative state | Failure behavior |
|---|---|---|---|
| Edge mission plane | Navigation, local safety, and execution of committed work | Node journal, mission program, and cached authority | Continues within its bounded lease; cannot invent new authority |
| Coordination plane | Mission/group mutations, epochs, approvals, and cross-cell preparation | Cell-local Raft log, FSM, and quorum proofs | Rejects new authority without a four-voter quorum |
| Cloud data plane | Telemetry ingestion, projections, replay, search, and operational history | Kafka events plus PostgreSQL projections | May lag or spool without invalidating committed edge execution |
| AI/ML plane | Intent interpretation, retrieval, engineering assistance, and evaluation | Versioned prompts, context receipts, datasets, and experiment records | Advisory and replaceable; unavailable models do not stop safety or manual operation |

### State ownership

| State | Owner | Durable store | Required guarantee |
|---|---|---|---|
| Mission command and authority | Raft cell | Raft/BoltDB FSM | Quorum commit and deterministic apply |
| Vessel execution progress | Vessel node | Local journal and execution watermark | Monotonic, restart-safe execution |
| Telemetry and operational events | Event producer | Bounded outbox and Kafka | At-least-once delivery with logical deduplication |
| Query projections | Platform workers | PostgreSQL | Deterministically rebuildable |
| Conversation and semantic memory | Go memory authority | PostgreSQL/pgvector plus bounded node cache | Scoped retrieval with provenance; never mission authority |
| Evaluation datasets and artifacts | MLOps pipeline | MinIO and MLflow metadata | Immutable source linkage and reproducible comparison |

## Scope and priorities

M13 has four required workstreams, in priority order:

1. Real cross-process OpenTelemetry tracing.
2. Safe, SLO-driven failure drills with evidence receipts.
3. One complete GNSS incident-to-evaluation workflow.
4. Reproducible capacity, recovery, and cost evidence.

The first three workstreams form the minimum interview-ready release. Capacity/cost evidence completes the platform narrative. Kubernetes deployment is a later proof and must not destabilize the live demonstration.

## Workstream 1 — End-to-end platform tracing

### Goal

Allow an interviewer to select a mission command and follow it from operator request through vessel execution and durable projection.

### Required trace path

1. Voice or typed assistant request.
2. Context assembly and provider attempt.
3. Typed semantic action.
4. Deterministic policy and version validation.
5. Raft proposal, commit, and FSM apply.
6. Four-signature quorum-proof construction.
7. Referee/gateway effect acceptance.
8. Node receipt and execution watermark.
9. Kafka publication and worker consumption.
10. PostgreSQL projection and optional memory/evaluation candidate.

### Trace propagation

- Use W3C `traceparent`/`tracestate` where applicable.
- Carry trace and correlation IDs through HTTP, internal mTLS calls, Kafka headers, outbox records, worker transactions, and evidence records.
- Preserve request ID, idempotency key, actor, mission, node, cell, term, epoch, log index, state version, and command hash as typed span attributes.
- Record retry and deduplication outcomes without treating expected retries as failures.
- Record provider/model latency, token accounting, fallback, cancellation, and tool budgets without exposing prompts or secrets by default.
- Redact credentials, private keys, raw voice, hidden faction state, and sensitive model context before export.

### Runtime shape

- Add a private OTLP receiver and bounded trace store to the platform deployment.
- Publish no new external observability port.
- Retain a bounded interview window and documented retention policy.
- Mission execution must remain independent of the collector.
- Export failure must be non-blocking and observable through a drop/error counter.

### Frontend result

Engineer receives a real trace waterfall. Selecting a span shows identifiers, timing, result, retries, evidence links, and redacted structured attributes. Trace data must originate from backend telemetry rather than UI timers.

## Workstream 2 — SLO-driven failure drills

### Goal

Convert existing resilience behavior into safe, repeatable experiments with expected outcomes, measured recovery, and immutable evidence.

### Initial SLOs

Initial targets are configuration, not marketing claims. Baselines must be measured before final thresholds are adopted.

| Indicator | Initial objective |
|---|---|
| Duplicate applied mission effects | Exactly zero |
| Healthy-path command commit latency | P95 below 500 ms in the lab |
| Cell leader recovery | Below 10 seconds in the lab |
| Minority-partition mutations | 100% rejected |
| 3/3-partition mutations | 100% rejected |
| Worker recovery | Backlog drains without duplicate projections |
| Projection replay | Counts and checksums match the source event set |
| Committed mission continuity | Unaffected by central AI/data-service loss |

Every SLO view must identify its workload, sample window, environment, and whether the value is measured or projected.

### Drill A — Coordinator failure

- Resolve the current leader from signed coordination state.
- Stop only the intended leader process/node service through the existing bounded mechanism.
- Observe `authority-ready → electing → authority-ready`.
- Require the new leader to commit `epoch_advance`.
- Verify follower convergence, proof validity, mission continuity, and zero duplicate effects.
- Restore the stopped member and verify applied-index/state-hash convergence.

### Drill B — Data-platform interruption

- Stop one worker first; optionally run a separate bounded Kafka interruption after worker recovery is proven.
- Show producer outbox or Kafka lag increasing.
- Verify committed node execution continues.
- Restore the component and measure backlog drain time.
- Verify PostgreSQL projection counts/checksums and zero duplicate logical effects.

### Drill C — Radio quorum partition

- Affect only the simulated-radio interface and preserve management/provider connectivity.
- Prove that a 4/2 split permits only the majority to commit.
- Prove that a 3/3 split permits neither side to commit.
- Show stale/read-only labeling on minority nodes.
- Remove the fault through the rollback watchdog and verify convergence.

### Evidence receipt

Every drill writes a `PlatformDrillReceiptV1` containing:

- Drill ID and type.
- Initiating actor and confirmation receipt.
- Deployed commit, image, and node-binary hashes.
- Initial topology and state versions.
- Expected invariants.
- Timestamped observations.
- Measured degradation and recovery intervals.
- Relevant trace, Raft, Kafka, PostgreSQL, and node evidence IDs.
- Final indexes and state checksums.
- Pass, fail, partial, or aborted outcome with reasons.

## Workstream 3 — Incident-to-evaluation data flywheel

### Goal

Demonstrate how real operational evidence becomes a governed autonomy evaluation without allowing an AI model to promote itself or change mission authority.

### First vertical slice: GNSS spoof rejection

1. Start from a recorded clean navigation baseline.
2. Inject an impossible GNSS displacement through the existing simulation boundary.
3. Capture GNSS, inertial/dead-reckoning estimate, expected motion, PNT confidence, policy decision, vessel behavior, and operator-visible result.
4. Create an immutable `OperationalEpisodeV1` with exact source event ranges and checksums.
5. Run Dagster validation, normalization, redaction, and dataset assembly.
6. Store bounded replay/dataset artifacts in MinIO.
7. Run baseline and candidate detection policies through the same deterministic episode.
8. Record the comparison in MLflow.
9. Evaluate detection latency, false acceptance, route deviation, uncertainty growth, and time to safe behavior.
10. Require an exact-hash human promotion decision before any candidate becomes a released policy.

### Guardrails

- Raw audio is never retained.
- Model-generated labels remain candidates until deterministically validated or explicitly approved.
- Evaluation data carries source IDs, software/model versions, parameters, seed, and checksums.
- Mission authority does not depend on Dagster, MinIO, MLflow, PostgreSQL, Kafka, or a model provider.
- Promotion is separate from mission approval and requires a privileged human capability.
- A failed or incomplete evaluation cannot silently replace the baseline.

### Frontend result

Cutaway shows the episode flowing through capture, validation, artifact storage, experiment comparison, and promotion. Engineer shows current pipeline health and the exact run/result records. All animation is state-backed.

## Workstream 4 — Capacity, recovery, and cost evidence

### Goal

Replace general scalability statements with reproducible measurements and clearly labeled projections.

### Workloads

- Twelve real vessel VMs.
- One hundred logical vessels.
- One thousand telemetry producers or visible operational tracks.
- Healthy operation, bounded degradation, and reconnection burst profiles.

### Measurements

- Command latency P50/P95/P99.
- Raft commit, proof, and election latency.
- Events produced and consumed per second.
- Kafka consumer lag and recovery rate.
- PostgreSQL transaction rate and projection delay.
- Outbox depth, capacity, and oldest-event age.
- Duplicate submissions and duplicate applied effects.
- CPU, memory, disk, and network use per role.
- Trace ingestion overhead and dropped spans.
- AI calls, input/output tokens, latency, fallback rate, and estimated cost.
- Storage growth by telemetry, trace, memory, and artifact retention class.
- Measured backup/restore RPO and RTO when that drill is enabled.

### Cost model

- Record current laboratory resources separately from cloud projections.
- Publish assumptions for compute, storage, data transfer, retention, and model usage.
- Provide profiles for 12, 100, and 1,000 assets.
- Mark extrapolated values visibly and never present them as benchmarks.
- Document controls such as batching, compression, downsampling, tiered retention, deterministic AI bypass, bounded context, and priority shedding.

## Frontend presentation

M13 reuses existing windows instead of adding another top-level product mode.

### Engineer: live platform operations

Engineer becomes the operational SRE view.

#### Health ribbon

Show four compact, expandable health cells:

- Edge mission plane.
- Coordination plane.
- Cloud data plane.
- AI/ML plane.

Each cell reports current state, last transition, stale age, and evidence source.

#### SLO strip

Show commit latency, leader recovery, ingestion lag, duplicate effects, and recovery objective. Use:

- Green for within objective.
- Gold for degraded but safe.
- Red for violated.
- Gray for unavailable or not measured.

#### Live topology

Show the operator/gateway, two Raft cells, twelve nodes, Kafka/workers/PostgreSQL, and optional MLOps services. Edges represent actual health and traffic state. The view must make the management, simulated-radio, data, and provider paths visually distinct.

#### Trace explorer

Allow selection by mission, command, vessel, trace ID, or recent error. Present a compact waterfall first and details on demand.

#### Drill runner

Provide Coordinator Failure, Data Pipeline Recovery, and Radio Quorum Partition cards. Each card explains affected components, protected components, invariants, rollback behavior, and expected duration before requesting confirmation.

#### Capacity and cost drawer

Keep resource and cost details available but collapsed by default so they do not obscure active health and traces.

### Cutaway: architecture and evidence

Cutaway becomes the explanatory reviewer view.

#### Deployment-state toggle

- **Deployed lab:** three Proxmox hosts, VM 214, twelve vessel VMs, two Raft cells, real software services, and simulated physical inputs.
- **Production shape:** proposed multi-zone cloud services with independently operating edge nodes.

Every component carries exactly one status label:

- `DEPLOYED`
- `SIMULATED PHYSICAL CONDITION`
- `DESIGN VALIDATED`
- `FUTURE HARDENING`

#### Consistency and ownership matrix

Expose the state-ownership table from this plan and allow a row to highlight the relevant services, stores, and failure boundary.

#### Evidence chain

Selecting a capability claim opens the test or drill receipt supporting it. Example:

> Coordinator failover preserves committed missions.

The evidence view then shows the initial leader, failure time, new leader, election duration, indexes, state checksums, proof signatures, duplicate count, and result.

#### Data-flywheel view

Show the GNSS episode moving through Kafka capture, Dagster processing, MinIO storage, MLflow comparison, and human promotion. Do not animate inactive or unavailable services as healthy.

### Responsive behavior

- Desktop may show topology and trace/evidence detail side by side.
- Tablet shows one primary panel with collapsible details.
- Phone uses a full-screen primary view with a compact stage selector.
- Touch targets remain at least 44 CSS pixels where practical.
- Dense tables provide card/list alternatives on narrow layouts.
- Keyboard and screen-reader navigation must reach every drill, trace, evidence, and disclosure control.

## Guided demonstration

### Duration

Target five minutes at a natural speaking cadence. The demonstration may extend modestly when a real election or recovery takes longer; it must never falsify progress to preserve timing.

### Sequence

| Time | Beat | Visible proof |
|---|---|---|
| 0:00–0:35 | Explain the four planes | Cutaway deployed-lab topology and real/simulated labels |
| 0:35–1:15 | Ask the assistant about a vessel and create a mission | Live context, typed action, policy, exact confirmation, mission start |
| 1:15–1:50 | Follow the command | Engineer trace waterfall from assistant through node and PostgreSQL |
| 1:50–2:45 | Fail the active coordinator | Election, epoch advance, browser recovery, mission continuity, measured duration |
| 2:45–3:30 | Interrupt a data worker | Backlog growth, uninterrupted edge execution, drain, matching projection |
| 3:30–4:20 | Inspect the GNSS episode | Rejection, safe response, dataset, baseline/candidate evaluation, promotion gate |
| 4:20–4:50 | Review scale and cost | Measured 12-node results and labeled 100/1,000-asset profiles |
| 4:50–5:00 | State the boundary | Deployed versus simulated versus future-hardening summary |

### Demo orchestration rules

- Use prerecorded Jarvis narration in Navy mode and Captain Barbossa narration in Pirate mode only after the technical sequence is stable.
- Demo actions invoke the same public/internal APIs and confirmation boundaries used by an operator.
- Failure stages wait on real state transitions and use bounded timeouts.
- Stop Demo cancels orchestration, restores simulation speed, removes demo-owned scenes/focus, and runs the applicable fault rollback.
- A failed drill remains visible as a failure; the demo does not silently skip or replace it.
- Existing missions and operator state are preserved or restored according to a documented demo fixture.

### Closing statement

> KeelMesh does not claim that every production autonomy problem has been solved in a home lab. It demonstrates that authority boundaries, edge behavior, cloud recovery, observability, and the learning loop can be made explicit, testable, and scalable without placing the cloud or a language model inside the vessel's safety-critical control path.

## Interfaces and records

Prefer extending existing `/api/v6` coordination and platform interfaces rather than introducing another broad API version solely for the UI. Add versioned contracts where no existing record is suitable.

### Proposed read interfaces

- `GET /api/v6/platform/summary`
- `GET /api/v6/platform/slo`
- `GET /api/v6/platform/traces`
- `GET /api/v6/platform/traces/{trace_id}`
- `GET /api/v6/platform/drills`
- `GET /api/v6/platform/drills/{drill_id}`
- `GET /api/v6/platform/evidence/{evidence_id}`
- `GET /api/v6/platform/capacity`
- `GET /api/v6/platform/cost-model`
- `GET /api/v6/platform/evaluations/{episode_id}`

### Proposed mutations

- `POST /api/v6/platform/drills`
- `POST /api/v6/platform/drills/{drill_id}:abort`
- `POST /api/v6/platform/evaluations/{episode_id}:run`
- `POST /api/v6/platform/evaluations/{episode_id}:promote`

All mutations require request ID, idempotency key, actor identity, expected platform state version, capability authorization, exact target scope, and confirmation where disruption or promotion is involved.

### Proposed records

- `PlatformPlaneHealthV1`
- `ServiceLevelObjectiveV1`
- `ServiceLevelMeasurementV1`
- `PlatformTraceSummaryV1`
- `PlatformTraceSpanV1`
- `PlatformDrillRequestV1`
- `PlatformDrillReceiptV1`
- `OperationalEpisodeV1`
- `EvaluationDatasetReceiptV1`
- `PolicyEvaluationRunV1`
- `PolicyPromotionRequestV1`
- `CapacityProfileV1`
- `CostProjectionV1`
- `DeploymentEvidenceV1`

## Delivery sequence

### Phase 0 — Baseline and contracts

- Freeze and record the current deployed commit and binary hashes.
- Capture healthy-path latency/resource baselines.
- Define trace attributes, redaction rules, SLO records, drill receipts, and evidence contracts.
- Add Go/TypeScript fixtures and compatibility tests.
- Create no infrastructure fault during this phase.

### Phase 1 — Tracing

- Add OTLP receiver/storage behind the private network boundary.
- Propagate context through HTTP, mTLS, Raft receipts, outboxes, Kafka, workers, and PostgreSQL.
- Build the Engineer trace waterfall.
- Verify collector loss does not affect mission execution.

### Phase 2 — SLOs and drill evidence

- Implement state-backed SLO aggregation.
- Wrap existing leader, worker, Kafka, and radio fault mechanisms with strict target validation and automatic rollback.
- Add drill preflight, confirmation, progress, abort, and evidence receipts.
- Implement Engineer health, SLO, topology, and drill surfaces.

### Phase 3 — GNSS data flywheel

- Define immutable episode boundaries and source checksums.
- Add Dagster ingestion/validation assets.
- Store bounded artifacts in MinIO.
- Record baseline/candidate runs in MLflow.
- Add deterministic comparison and privileged promotion boundaries.
- Implement the Cutaway data-flywheel and evidence views.

### Phase 4 — Capacity and cost

- Define reproducible 12, 100, and 1,000-source workloads.
- Run healthy, degraded, and reconnect profiles.
- Publish machine-readable and Markdown results.
- Add measured/projected labeling and cost assumptions.
- Add the collapsed Engineer capacity/cost drawer.

### Phase 5 — Guided demo and optional deployment proof

- Add the Platform Proof guided sequence using the established demo runtime.
- Record narration only after all waits and failure transitions are stable.
- Run desktop, tablet, phone, Navy, Pirate, stop/restart, and failure-path rehearsals.
- Optionally validate central services on a disposable three-node k3s environment with manifests, probes, resource limits, anti-affinity, and network policy.
- Do not replace the stable interview appliance with Kubernetes as part of M13 acceptance.

## Verification matrix

| Area | Required verification |
|---|---|
| Trace propagation | Unit fixtures, cross-process integration, Kafka-header propagation, redaction, dropped-export behavior |
| Raft drill | Leader stop, epoch advance, follower convergence, state hash, zero duplicates |
| Partition drill | 4/2 majority, 2-node minority rejection, 3/3 rejection, rollback convergence |
| Data interruption | Worker restart, optional Kafka interruption, outbox/lag growth, backlog drain, projection checksum |
| GNSS episode | Source integrity, deterministic replay, baseline/candidate comparison, promotion authorization |
| Capacity | Repeatable workload seed, resource capture, percentile calculations, reconnect burst |
| Cost | Versioned assumptions, measured/projected distinction, arithmetic fixtures |
| Frontend | Desktop/tablet/phone, keyboard, touch, screen reader, Navy/Pirate, stale/live state |
| Demo | Complete, stop at every stage, timeout handling, fault rollback, no residual focus/scenes |
| Compatibility | Existing M1–M12 Go, Python, TypeScript, API, browser, node, and coordination gates |

## Acceptance criteria

- [ ] One mission command produces a complete cross-service trace from operator request through durable projection.
- [ ] Every displayed trace stage corresponds to a real span, receipt, command, or event.
- [ ] Trace collection failure cannot stop or delay committed mission execution.
- [ ] The leader-failure drill recovers without duplicate effects and records measured recovery time.
- [ ] A 4/2 partition permits only the majority; a 3/3 partition rejects all new authority.
- [ ] Radio drills cannot target management or intentional provider connectivity.
- [ ] Worker/Kafka interruption does not stop committed edge execution.
- [ ] Backlog recovery produces matching projection counts and checksums without duplicate logical effects.
- [ ] A GNSS spoof incident becomes a source-linked, replayable evaluation artifact.
- [ ] Baseline and candidate policies run against the same immutable episode and seed.
- [ ] Policy promotion requires a privileged exact-hash human decision.
- [ ] SLO values identify source, workload, window, environment, and freshness.
- [ ] Twelve-node capacity results are measured; 100/1,000-asset extrapolations are visibly labeled.
- [ ] Cost projections expose every material assumption.
- [ ] Engineer provides live health, SLO, topology, traces, drills, and capacity/cost evidence.
- [ ] Cutaway explains state ownership, failure boundaries, deployed versus production shape, and evidence chains.
- [ ] The guided Platform Proof completes without manual repair and leaves no residual fault, focus, speed, or scene state.
- [ ] Existing M1–M12 behavior remains compatible.
- [ ] No GitHub-hosted workflow or Proxmox snapshot is created.

## Explicitly deferred work

The following are valuable but are not required to complete M13:

- Physical Wi-Fi HaLow radios or measured maritime RF performance.
- Physical GNSS receivers, inertial sensors, propulsion, or sea trials.
- A complete AWS deployment.
- Migration of the stable interview appliance to Kubernetes.
- Multi-broker Kafka and managed PostgreSQL high availability.
- Hardware-backed production PKI/HSM integration.
- Dynamic Raft membership.
- Additional Arena mechanics, visual themes, vessel art, or C2 controls.
- Multi-domain PX4/ArduPilot/ROS 2 adapters; these should follow once the platform proof is complete.

## Safety and rollout constraints

- Preserve `KEELMESH_COORDINATION_MODE=simulated|shadow|raft` throughout development.
- Develop tracing and UI against local fixtures before deploying to VM 214.
- Add drills incrementally: worker, leader, 4/2 partition, then 3/3 partition.
- Validate exact targets before every fault and install rollback before applying it.
- Never impair the Proxmox management network or deliberately interrupt node provider access during radio drills.
- Do not expose OTLP, Kafka, PostgreSQL, MinIO, Dagster, MLflow, or node-management ports publicly.
- Do not store credentials, private keys, raw audio, hidden faction state, or unredacted provider context in traces or evidence.
- Treat Proxmox storage as constrained; perform fresh read-only preflight before any storage-affecting operation.
- Create no Proxmox snapshot without a separate explicit authorization and safe-capacity verification.
- Run builds and verification only on the approved local/VM 214/node environment; do not run GitHub-hosted workflows.
- Retain the current stable binary and configuration as the immediate rollback target until M13 acceptance passes.
