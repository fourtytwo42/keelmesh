# Implementation Plan

Status: Friday interview build plan  
Target demo date: 2026-09-04  
Product name: KeelMesh

## 1. Delivery Principle

Build one deterministic end-to-end mission path first, then add AI assistance,
durability, and scale around it. Every milestone must leave the application in a
demonstrable state. The simulator, policy gateway, mission lease, mission tape,
PNT arbiter, and executor remain usable when Python, Kafka, PostgreSQL, voice,
or every model provider is unavailable.

The Friday build tells one coherent story:

> Define intent, compare plans, authorize a bounded mission, lose connectivity,
> continue from validated onboard work, reject bad navigation evidence, recover
> safely, and inspect the same events flowing through the platform.

## 2. Friday Scope Contract

### Must be real and live

- One browser URL served by the Go core.
- Six individually visible vessels and at least 1,000 aggregate background vessels.
- Typed mission intent, two computed candidate plans, recommendation reasons,
  route preview, authorization, and simulated execution.
- Deterministic mission lease and policy boundary.
- Direct Starlink, HaLow relay through a peer, full partition, 60-second mission
  tape, stale-segment rejection, and bridge-to-future rejoin.
- GNSS-spoof injection, source exclusion, uncertainty growth, and a deterministic
  safe response.
- Operator-to-Cutaway transition backed by current runtime state.
- Kafka telemetry path, PostgreSQL audit/projection path, deduplication, and a
  visible ingestion-worker failure/recovery.
- At least two real Kafka consumer processes in one group during the scale segment,
  with visible partition assignment, lag, worker loss, and rebalance.
- Python AI service using restricted tools, with deterministic fixtures available
  for every model-generated result.
- Cloud/local/mock provider routing with a visible provider-state audit event.
- One Autonomy Engineer incident workflow that uses MCP to inspect telemetry,
  logs, simulation state, and policy evidence; retrieves cited runbook/history
  context through pgvector; replays the incident; and promotes an approved,
  versioned evaluation case.
- Versioned prompts, tool schemas, evaluation fixtures, scenario manifests, and
  dataset-lineage metadata stored in Git and exercised by CI.
- Trace IDs and OpenTelemetry spans across model, MCP tool, planner, policy,
  bundle, ingestion, storage, and evaluation boundaries. A separate collector
  is optional; the live cutaway must consume real span/state data.
- One scripted four-vessel Quiet Fleet adaptation.
- One-command startup and one verification command.

### Best-effort Friday polish

- Push-to-talk through local STT.
- A small visible agent-evaluation panel.
- Named Cloudflare tunnel and stable hostname.
- Spoken TTS confirmation.

### Explicit Friday cuts

- Kubernetes deployment, AWS resources, MLflow, Dagster, MinIO, and Grafana.
- Real marine physics, physical radios, physical vessel GPUs, and hardware-in-loop.
- Full-duplex voice, general-purpose multi-agent collaboration, or model training.
- More than one scripted Quiet Fleet coordination case.

If time is lost, cut in this order: TTS, live STT, live local model, Quiet Fleet
animation detail. Never cut plan-before-action, policy, mission tape, PNT
degradation, fault injection, the incident-to-eval autonomy workflow, measured
horizontal scaling, cutaway, or verification.

## 3. Runtime Architecture

The release consists of four required containers on one private Compose network:

| Container | Responsibility | Failure behavior |
|---|---|---|
| `core` | Embedded React UI, API/WebSocket, simulator, planner, policy, leases, mesh, tape, PNT, Quiet Fleet, metrics, load and faults | Mission path remains deterministic and self-contained |
| `ai` | Intent assistance, MCP client, retrieval, explanation, STT adapter, eval runner | UI falls back to typed deterministic intent templates |
| `kafka` | Durable telemetry and asynchronous platform events | Mission control continues; edge events queue locally |
| `postgres` | Missions, approvals, audit projection, incidents, pgvector retrieval | Active mission continues from memory/local journal; UI shows degraded persistence |

Only `core:8080` is published. The browser never connects directly to AI, Kafka,
PostgreSQL, a model provider, or signing material.

### Request and event flow

1. The UI submits text, confirmed speech, or map geometry to `core`.
2. `core` creates a request/trace ID and captures the current fleet-state version.
3. `ai` may compile language into a typed intent through restricted read/propose
   tools. A deterministic compiler fixture is always available.
4. The Go planner expands two candidates, computes assignments and routes, runs
   the simulator, applies policy, scores candidates, and hashes each result.
5. The UI previews immutable candidates. Nothing is executable yet.
6. Authorization signs a short-lived lease bound to the exact candidate hash.
7. The executor creates signed mission-tape segments and sends them through the
   simulated peer overlay using idempotency keys.
8. Logical vessel actors validate, arm, execute, and journal segments locally.
9. Telemetry and lifecycle events publish asynchronously through Kafka; workers
   deduplicate and project them into PostgreSQL.
10. WebSocket state drives the map, timeline, tape, PNT, route, metrics, and cutaway.

The Autonomy Engineer path branches from step 9: select an incident, use scoped
MCP resources/tools to gather evidence, retrieve cited runbooks and similar
scenarios, run deterministic replay, propose an evaluation case, require human
approval to promote it, and run the case against the configured providers. This
is an internal engineering workflow; it never grants fleet execution authority.

## 4. Repository Shape

```text
cmd/core/                    Go application entry point
internal/api/                HTTP, WebSocket, validation, error contract
internal/domain/             Canonical IDs, intents, plans, events, leases
internal/simulator/          Tick engine, visible vessels, scenarios, load
internal/planner/            Candidate expansion, routes, simulation, scoring
internal/policy/             Lease checks and deterministic reason codes
internal/edge/               Logical vessel actor and local journal
internal/mesh/               Link graph, routes, bundles, relay, partition
internal/tape/               Segments, lifecycle, watermarks, rejoin bridge
internal/pnt/                Source checks, fusion fixture, uncertainty policy
internal/quietfleet/         Proposal, decision, quorum, arm, future commit
internal/events/             Kafka adapter and in-memory fallback
internal/store/              PostgreSQL adapter, migrations, memory fallback
internal/provider/           Cloud, local, and deterministic provider routing
internal/metrics/            Runtime measurements and snapshots
internal/trace/              OpenTelemetry spans and live trace projection
web/                         React, TypeScript, MapLibre, tests
ai/                          Python MCP client, retrieval, incident agent, STT, evals
contracts/                   Versioned JSON Schema and shared fixtures
scenarios/                   Deterministic demo and failure scripts
prompts/                     Versioned prompt templates and metadata
datasets/                    Reviewed eval manifests and lineage, never raw secrets
deploy/                      Compose configuration and container assets
scripts/                     Start, status, verify, reset, evidence export
evidence/                    Generated reports; ignored except examples
```

KeelMesh is the canonical product and repository identifier for this structure.

## 5. Canonical Contracts First

Implement these versioned contracts before broad UI work:

1. `EventEnvelopeV1`
2. `FleetSnapshotV1`
3. `MissionIntentV1`
4. `PlanCandidateV1`
5. `PolicyDecisionV1`
6. `MissionLeaseV1`
7. `PeerBundleV1`
8. `MissionTapeSegmentV1`
9. `PntEstimateV1`
10. `GroupAdaptationProposalV1` and `GroupDecisionV1`

Go owns canonical validation. JSON Schema lets TypeScript and Python validate
fixtures without hand-maintaining three incompatible model definitions. Contract
tests must reject unknown action kinds, stale state versions, invalid hashes,
expired leases, out-of-order authority epochs, and malformed provider output.

## 6. API Surface

### Browser-facing HTTP

| Method and path | Purpose |
|---|---|
| `GET /api/v1/bootstrap` | Scenario, fleet snapshot, capabilities, runtime health |
| `POST /api/v1/intents:compile` | Text/map input to typed intent |
| `POST /api/v1/plans` | Generate immutable candidate plans |
| `POST /api/v1/plans/{id}:preview` | Run/retrieve deterministic preview |
| `POST /api/v1/plans/{id}:authorize` | Create lease bound to candidate hash |
| `POST /api/v1/missions/{id}:start` | Start an authorized mission idempotently |
| `POST /api/v1/faults` | Apply allow-listed demo fault |
| `POST /api/v1/scenarios/{id}:run` | Run a deterministic scripted sequence |
| `GET /api/v1/audit/{trace_id}` | Explain one request or incident |
| `GET /api/v1/evaluations/latest` | Latest compact eval result |
| `POST /api/v1/incidents/{id}:investigate` | Start scoped autonomy investigation |
| `POST /api/v1/incidents/{id}:replay` | Run deterministic simulation replay |
| `POST /api/v1/incidents/{id}:promote-eval` | Human-approved eval-case promotion |
| `GET /api/v1/metrics/snapshot` | Measured platform state for cutaway |
| `GET /healthz` and `GET /readyz` | Process and dependency readiness |

### Live stream

Use `GET /api/v1/stream` as one WebSocket carrying versioned messages:

- `fleet.snapshot`
- `mission.updated`
- `plan.preview.updated`
- `link.route.changed`
- `tape.watermark.changed`
- `segment.lifecycle.changed`
- `pnt.integrity.changed`
- `quietfleet.round.changed`
- `platform.metrics.changed`
- `audit.event.appended`
- `investigation.updated`
- `evaluation.run.updated`

Each message includes server sequence and state version. A reconnecting browser
fetches a fresh bootstrap snapshot instead of assuming it received every message.

### Restricted AI boundary

Expose only the PRD allow-list through MCP or an equivalent versioned internal
HTTP adapter during the first slice. Never expose authorization, deployment,
lease signing, policy mutation, audit deletion, or vessel commands. Provider
output can request a typed proposal only; Go revalidates every argument.

## 7. Deterministic Simulation Model

- Fixed 100 ms simulation tick; UI updates at 5-10 Hz.
- Seeded scenario clock separated from wall time.
- Six visible actor-isolated vessels with pose, reserve, link state, PNT state,
  lease, tape, local event sequence, and mailbox.
- Background fleet uses aggregate actors/batches and the same event envelope; it
  does not render 1,000 individual WebGL symbols unless zoom requires it.
- Six immutable ten-second tape segments represent the 60-second lookahead.
- Faults are commands recorded in the audit log, not UI-only animations.
- All displayed metrics are derived from counters/histograms/state, never success
  strings hardcoded in React.

The main scripted scenario is a state machine with manual controls for each step:

`idle -> intent_ready -> candidates_ready -> previewed -> authorized -> executing
-> direct_link_failed -> peer_relay -> partitioned -> pnt_suspect -> contingency
-> reconnected -> bridged -> quietfleet_adapting -> recovered`

Manual stepping makes the interview controllable. An auto-run mode provides a
repeatable fallback and powers the verification script.

## 8. Milestones and Acceptance Gates

### M0 - Identity and baseline (30-60 minutes)

- Choose an independent product/repository name.
- Rename modules, paths, service labels, UI copy, VM label if desired, and docs.
- Create the private GitHub repository from the VM and push the baseline.
- Add CI for Go tests, Python tests, TypeScript checks, and Compose validation.

Gate: clean checkout builds the bootstrap image and `/healthz` passes.

### M1 - Golden mission loop

- Create canonical contracts and deterministic fixtures.
- Scaffold React/TypeScript UI embedded by Go.
- Render local MapLibre style, six vessels, search polygon, exclusion zone, and
  two candidate route layers.
- Implement typed intent, candidate generation, scoring, preview, policy, exact
  plan-hash authorization, lease creation, start, and vessel movement.
- Show audit events and “Nothing has been sent yet” before authorization.

Gate: a fresh observer can create, compare, preview, authorize, and watch one
mission without Python, Kafka, PostgreSQL, internet, or voice.

### M2 - Resilient edge wow path

- Add peer bundle envelope, simulated link graph, route scoring, hop receipts,
  content deduplication, and fault controls.
- Add six-segment mission tape and full/low/critical/empty watermarks.
- Implement Starlink failure -> HaLow peer relay -> complete partition.
- Add monotonic expiry, local buffering, execution high-water mark, stale-segment
  rejection, deterministic bridge, and future refill.
- Add PNT sources, spoof fault, source exclusion, uncertainty halo, speed reduction,
  and safe contingency.

Gate: the scripted incident passes with no duplicate execution and no stale replay,
and the UI can explain the route, tape, PNT, and policy decisions.

### M3 - Durable platform and cutaway

- Expand Compose to Kafka KRaft and PostgreSQL/pgvector with health checks.
- Add migrations and deterministic seed data.
- Publish telemetry/audit events; validate, deduplicate, and project them.
- Add local queue fallback when Kafka is down.
- Add configurable background load using the same event path.
- Run ingestion through a separate role of the same Go image and at least two
  actual consumer processes during the scale segment; expose Kafka partition
  ownership and rebalance state.
- Add worker kill/pause/scale fault and real throughput, p50/p95/p99 latency, lag,
  duplicate, quarantine, recovery-time, and provider metrics.
- Build the Operator/Cutaway transition and animate one selected trace through
  actual subsystem states.

Gate: 1,000 background vessels run without starving mission control; worker
failure creates lag, Kafka reassigns partitions, and recovery drains it without
logical duplicates. The evidence report also includes events/second and resource
use, so fleet count is not used as a substitute for measured scale.

### M4 - AI assistance and provider resilience

- Scaffold Python AI service and restricted MCP client.
- Add deterministic intent/explanation provider first.
- Add configured cloud and local OpenAI-compatible adapters with deadlines,
  circuit breaker, request IDs, schema validation, and one winning response.
- Implement a scoped incident-investigation agent with MCP resources for telemetry,
  logs, audit, simulation replay, policy, runbooks, and similar incidents.
- Add runbook/history retrieval through pgvector with citation IDs, context limits,
  document trust labels, and prompt-injection test fixtures.
- Add human-approved incident-to-eval promotion with scenario seed, source event
  IDs, expected tool calls, assertions, prompt/tool versions, and provenance.
- Add compact evaluation fixtures for valid schema, tool allow-list, stale state,
  approval boundary, injection refusal, provider failover, tool reliability,
  task success, model quality, and replay determinism.
- Instrument model and MCP calls with OpenTelemetry trace/span IDs and expose one
  trace in the live cutaway and evaluation evidence.
- Add push-to-talk only after the typed path is fully reliable.

Gate: cloud failure visibly selects local or deterministic fallback, creates one
proposal/audit trace, and cannot duplicate or authorize an action. An autonomy
engineer can investigate the Vessel 4 incident, retrieve cited evidence, replay
it, approve promotion to a versioned eval case, and run that regression. Stopping
AI does not stop the mission.

### M5 - Quiet Fleet and release hardening

- Implement the one four-vessel scripted adaptation: Vessel 4 slows, first
  assignment rejects, revised assignment reaches quorum/all-affected arm, and
  activates at a future tick.
- Measure bytes and rounds against the configured budget.
- Add start, status, verify, reset-demo, and export-evidence scripts.
- Run responsive/browser checks, restart tests, offline test, and full rehearsal.
- Snapshot the VM only after verification passes and the repository is pushed.

Gate: one command starts four healthy containers and one verification command
produces a timestamped evidence bundle with pass/fail results and measurements.

## 9. Three-Day Execution Order

### Tuesday, September 1 - Make the product real

1. Resolve naming and push the baseline.
2. Complete M1 end to end.
3. Start M2 with link-state and mission-tape contracts.
4. Record a screen capture of the working golden path before adding infrastructure.

End-of-day requirement: a polished deterministic map demo from intent through
authorized vessel motion.

### Wednesday, September 2 - Make it resilient

1. Complete M2, including reconnect and GNSS spoofing.
2. Add Kafka/PostgreSQL Compose services and event adapters.
3. Implement cutaway against real subsystem state.
4. Establish the first reproducible 1,000-vessel measurement.

End-of-day requirement: the complete edge-resilience story works even if AI and
the durable platform are disabled.

### Thursday, September 3 - Make it intelligent and bulletproof

1. Complete M3 measurements and worker recovery.
2. Add M4 Python/provider path and minimal eval set.
3. Add Quiet Fleet scripted case.
4. Add voice only if every prior gate is green.
5. Run clean-start, offline, restart, responsive, and full-demo rehearsals.
6. Export evidence, push, tag the release, and snapshot VM 214.

End-of-day requirement: two consecutive six-minute rehearsals from a clean reset
with no manual terminal repair.

### Friday, September 4 - Freeze and present

- No feature work unless it fixes a rehearsal-blocking defect.
- Start from the verified snapshot, pre-warm optional models, and verify the URL.
- Keep deterministic scenario, typed input, and LAN URL ready as fallbacks.
- Present Operator mode first; reveal architecture only after product value is clear.

## 10. Verification Matrix

| Claim | Automated evidence |
|---|---|
| Nothing executes before approval | Plan/executor contract test and browser flow test |
| Approval binds exact plan | Modified plan hash is rejected |
| Relay does not duplicate action | Dual-path bundle test yields one lifecycle |
| Partition is bounded | Tape reaches empty and enters signed contingency |
| Reconnect never replays stale work | Expired segments remain terminal; bridge starts in future |
| Spoofed GNSS cannot reset pose | Arbiter excludes source and threshold triggers contingency |
| AI has bounded capability | Tool allow-list and forbidden-tool tests |
| Provider failover is idempotent | Timeout race yields one accepted proposal |
| Mission survives AI/platform loss | Mission tick progresses with AI/Kafka/Postgres unavailable |
| Worker failure recovers | Produced count equals stored unique plus quarantined |
| Quiet Fleet cannot split authority | No commit without original quorum and all affected arms |
| Appliance is portable | Clean Compose start and verify on VM/laptop profile |

`scripts/verify` should emit JSON plus a short Markdown report containing image
version, git commit, scenario seed, hardware summary, dependency health, test
results, event counts, latency percentiles, lag peak, recovery time, duplicate
count, quarantined count, and provider path. Never pre-fill performance numbers.

## 11. Demo Rehearsal Script

1. Ask the fleet to search the polygon while preserving 30% reserve.
2. Compare two plans and explain the recommendation with measured projections.
3. Preview the plan and point out that nothing has been sent.
4. Authorize and start the exact hashed plan.
5. Fail Vessel 4 Starlink; show relay through Vessel 3.
6. Partition Vessel 4; show its onboard tape draining.
7. Inject GNSS spoofing; show exclusion, uncertainty, and safe degradation.
8. Reconnect; show missed segments expire and a future bridge refill the tape.
9. Request an out-of-envelope extension; show permission required.
10. Open Cutaway and follow the incident through the platform.
11. Fail cloud AI, then all AI; show assistance degradation without mission loss.
12. Kill an ingestion worker under background load; show lag and deduplicated recovery.
13. Run the Quiet Fleet adaptation if time remains.

The primary presentation should fit six minutes. Quiet Fleet and detailed scale
evidence are optional interviewer-driven branches after the core story lands.

## 12. Definition of Done

The Friday build is done only when:

- A clean reset followed by start and verify succeeds twice consecutively.
- Every required map/cutaway value comes from live state or measured metrics.
- Internet loss does not break the core scenario.
- AI loss does not stop an active mission.
- No secret is present in Git history, images, browser bundles, logs, or evidence.
- The UI clearly labels simulated radios, simulated navigation sensors, logical
  edge agents, and scenario assumptions.
- The README gives one start command, one URL, one verification command, and the
  six-minute demo path.
- The release commit is pushed and VM 214 has a post-verification snapshot.
