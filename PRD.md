# KeelMesh Fleet Intent Control

## Product Requirements Document

| Field | Value |
|---|---|
| Status | Draft 0.7 |
| Target demo | Friday, September 4, 2026 |
| Product type | Web-based simulated maritime autonomy workload backed by Go platform services and Python AI infrastructure |
| Primary audience | Havoc AI perception, cloud-platform, front-end, and ML-infrastructure interviewers |
| Product name | KeelMesh Fleet Intent Control |

## 1. Executive Summary

KeelMesh demonstrates how one operator can supervise a simulated fleet through a map, contextual suggestions, voice, and chat without allowing an LLM to directly control vehicles.

The fleet console is the proof surface, not a proposed replacement for Havoc's command-and-control product. The product being demonstrated is the internal AI/ML infrastructure behind the experience: model routing, MCP tools, retrieval, planning, evaluation, telemetry/data pipelines, security controls, auditability, and resilient deployment patterns for autonomy teams.

The operator expresses an objective. The AI converts that objective into typed candidate plans. A deterministic planner computes routes and assignments, a simulator previews expected behavior, and a policy gateway determines whether the plan is already authorized, requires approval, or is prohibited. The operator sees the proposed action on the map before anything happens.

Every operator console and simulated vessel is an offline-first peer node with local compute, storage, identity, mission-policy enforcement, Starlink internet, and a Wi-Fi HaLow fleet interface. Starlink is the preferred cloud path, but not the control spine. A transport-independent, store-and-forward overlay can move signed mission and telemetry bundles directly, through another vessel over HaLow, or through any node advertising authorized Starlink egress.

A bounded group may enter **Quiet Fleet**, a low-radio-duty cooperative mode. The group follows a signed preset mission, exchanges compact state and proposal deltas during scheduled HaLow windows, and uses node-local advisory agents to suggest adaptations. LLM output never becomes a vote or command directly: deterministic planners convert candidates into typed proposals, each affected vessel independently validates its own part, and only a quorum-backed future commit can change coupled group behavior.

The same interface can peel open into a live system cutaway. The active mission remains visible while AI assistance, planning, mission control, the peer communications fabric, resilient navigation, telemetry ingestion, storage, and observability layers appear underneath it. Load and failure controls demonstrate route changes across simulated links, provider failover, GNSS spoof detection, disconnected operation, durable queues, horizontal workers, backpressure, idempotency, and replay.

The product thesis is:

> AI interprets and recommends. Deterministic systems plan, validate, authorize, execute, and preserve safety.

The core demo thesis is:

> **Program the mission once. Coordinate locally. Adapt together. Degrade safely when the uplink disappears.**

The operator-facing product makes mission programming understandable through speech, chat, map drawing, recommendations, preview, and plain-language explanations. The distributed product underneath compiles that intent into a signed Group Mission Contract, per-vessel mission tapes, and deterministic authority bounds. Starlink adds remote coordination, cloud inference, analytics, and archival capacity; it is not required for the fleet to preserve local HaLow coordination or execute already authorized behavior. The mesh carries information, but authenticated edge nodes make the decisions. Individual safety actions remain local, coupled adaptations use the Quiet Fleet quorum/arming protocol, and authority never expands merely because the operator uplink is unavailable.

The resilience thesis is:

> The cloud is a collaborator, not a dependency. Connectivity changes the path and available assistance, never the validity of an already authorized mission.

## 2. Source Alignment

The Vurra transcript for **Havoc AI Hiring Call: ML Infrastructure for Maritime Autonomy** explicitly described the following stack and concerns:

| Transcript time | Explicitly described |
|---|---|
| 00:07:40 | Linux-based platform written in Go |
| 00:07:46 | AWS-backed cloud environment |
| 00:07:48 | Heavily containerized, multi-zone deployment |
| 00:07:55 | React front end |
| 00:07:57 | TypeScript and WebGL geospatial mapping |
| 00:08:17 | Human orchestrator today; agentic orchestration anticipated later |
| 00:08:32 | Autonomy takes over when communications fail |
| 00:11:37 | Agents supporting ETL pipelines and autonomy-team tooling |
| 00:12:38 | Observability and agentic AI systems for security workflows |
| 00:12:53 | MCP-like servers/resources/prompts and Semantic Kernel usage; the transcript renders “MCP” as “NCP” |
| 00:13:24 | Vector databases and RAG |
| 00:13:35 | Kubernetes environment |
| 00:13:43 | Docker-based container workflows |
| 00:18:24 | Repeated emphasis on building infrastructure at scale from an immature starting point |

The recruiter also emphasized massive real-time data volume, latency and bottleneck consequences, harsh operating environments, and the need to explain how an immature system would be decomposed and scaled.

The supplied **AI Infrastructure Engineer – Agents & ML Systems** posting adds these explicit requirements:

- Connect LLMs and agents to APIs, data lakes, telemetry stores, simulation tools, repositories, documentation, logs, and engineering workflows.
- Build MCP servers, clients, tools, resources, prompts, connectors, and secure tool-use patterns.
- Build retrieval, RAG, context-management, document-processing, embedding, and knowledge-search pipelines.
- Support dataset preparation and curation, experiment tracking, model evaluation, fine-tuning support, and model deployment.
- Evaluate agent task success, tool reliability, model quality, regressions, and failures.
- Provide logging, tracing, monitoring, debugging, and auditability across model, tool, and pipeline activity.
- Enforce least privilege, sandboxed execution, prompt-injection defenses, secrets management, approval workflows, and safe handling of sensitive data.
- Publish reusable examples, templates, documentation, and adoption guidance.
- Integrate with engineering systems such as GitHub, CI/CD, issue trackers, documentation, simulation platforms, and data lakes.

The posting names Python, TypeScript, Go, and C++ as relevant languages. It also names MCP; LangGraph, LangChain, LlamaIndex, and Semantic Kernel; OpenAI and Anthropic APIs; local/open-weight models; Kubernetes and Docker; Ray, Airflow, and Dagster; MLflow and Weights & Biases; Kafka; Postgres; and S3-compatible storage. These are preferred examples, not proof that every technology is currently deployed at Havoc.

The resilience design is additionally informed by public standards and guidance:

- [Wi-Fi HaLow/IEEE 802.11ah](https://wballiance.com/what-is-wi-fi-halow/) provides a sub-1 GHz, IP-capable underlay suitable for longer-range local links.
- [IETF Bundle Protocol v7](https://www.rfc-editor.org/rfc/rfc9171.html) provides the model for a store-carry-forward overlay across intermittent and heterogeneous links.
- [IALA resilient-PNT guidance](https://www.iala.int/product/g1180-resilient-pnt/?download=true) recommends multiple dissimilar PNT sources because no single source is perfect.
- The [U.S. DOT PNT Strategic Plan](https://www.transportation.gov/sites/dot.gov/files/2025-01/Positioning%20Navigation%20and%20Timing%20Strategic%20Plan_v%20FINAL.pdf) supports receiver hardening, signal-integrity monitoring, and fusion of complementary PNT sources.

These references shape the demo architecture; the interview build simulates RF and navigation behavior and is not a production or certified maritime system.

### Stack decision

Go will own the simulator, control plane, data plane, deterministic planner, policy gateway, and high-throughput services. Python will own agent orchestration, RAG, evaluation, and ML workflows. React and TypeScript will own the operator experience. This is the smallest stack that directly reflects the interview and job posting without introducing an additional desktop framework.

React, TypeScript, and a WebGL map will run as a browser application served with the platform. MCP is the contract between the Python agent layer and Go platform services. Stable HTTP/WebSocket contracts connect the UI to local or remote Go platform deployments. The remaining technologies are project choices selected from the posting to demonstrate relevant patterns; they must not be represented as confirmed Havoc production choices.

### Job-requirement traceability

| Job requirement | Evidence in KeelMesh |
|---|---|
| LLM/agent integration with tools and data | Python agent uses MCP resources/tools over Go fleet, simulation, policy, telemetry, history, and documentation services |
| Agentic task automation and analysis | Suggestions, incident investigation, candidate planning, failure explanation, and dataset-candidate generation |
| MCP infrastructure | Real MCP server/client contract with typed tools, resources, prompts, scopes, and audit events |
| Retrieval and RAG | Runbooks and historical mission summaries retrieved from Postgres/pgvector with source citations |
| ML data preparation and curation | Simulator incidents can be promoted into reviewed evaluation/dataset cases |
| Experiment tracking and model evaluation | Reproducible eval suite logs provider, prompt, latency, validity, tool-use, and outcome results to MLflow |
| Observability and debugging | OpenTelemetry traces plus live cutaway metrics and immutable audit timeline |
| Secure agent execution | Least-privilege MCP tools, sandboxed workers, plan schemas, policy gateway, signed approvals, and secret isolation |
| Human-in-the-loop workflows | Plan preview, exact diff, mission lease, approval binding, revocation, and prohibited-action handling |
| Distributed systems and scale | Kafka telemetry/control topics, horizontal Go consumers, Postgres projections, S3-compatible raw storage, backpressure, replay, and fault injection |
| Documentation and adoption | One-command demo, architecture guide, MCP tool template, eval fixture template, runbook example, and decision records |

## 3. Problem

Multi-vehicle autonomy produces two simultaneous problems:

1. The operator must express intent and understand fleet behavior without manually controlling each vehicle.
2. The platform must process high-volume, unreliable, real-time data while preserving low-latency mission control and safe behavior through component, provider, and communications failures.

Traditional chat interfaces hide spatial consequences. Traditional dashboards expose too much infrastructure detail without explaining how it affects the mission. Cloud-dependent agents become unavailable in exactly the degraded environments where assistance may be most valuable.

KeelMesh addresses these problems with a map-first interaction model and an infrastructure cutaway tied to the same live mission.

## 4. Product Goals

### 4.1 Primary goals

- Let a first-time operator create and supervise a simulated multi-vessel search mission within one minute.
- Present two or three measurable plan alternatives with one evidence-backed recommendation.
- Preview every proposed action spatially and temporally before execution.
- Permit bounded autonomous behavior inside a signed mission lease.
- Require explicit authorization when a plan exceeds the current lease.
- Keep mission execution and deterministic safety behavior independent of LLM availability.
- Demonstrate local AI operation and cloud-to-local provider failover.
- Demonstrate an offline-first peer-node fabric in which traffic can move directly, through another vessel, or through any available internet gateway.
- Demonstrate resilient navigation that detects a spoofed GNSS source, exposes position uncertainty, and degrades autonomy deterministically.
- Start the complete local demonstration through one documented container command.
- Use the same service contracts in local Compose and Kubernetes-style deployment profiles.
- Demonstrate high-volume telemetry ingestion, horizontal consumers, backpressure, failure recovery, deduplication, and replay.
- Make the infrastructure understandable through a live cutaway rather than a separate generic dashboard.
- Produce an immutable, inspectable record of intent, planning, policy, approval, execution, provider use, and telemetry.

### 4.2 Non-goals

- Controlling real vehicles or hardware.
- Implementing or validating production RF hardware, anti-jam equipment, or a certified maritime navigation system.
- Weapon, targeting, or use-of-force functionality.
- Allowing an LLM to emit actuator, steering, or raw navigation commands.
- Building realistic hydrodynamics or production-grade collision avoidance.
- Implementing a production multi-region AWS deployment before the interview.
- Training or fine-tuning foundation models.
- Displaying private model chain-of-thought.
- Claiming unmeasured throughput, latency, or reliability.

## 5. Users

### 5.1 Fleet operator

Needs to define objectives, understand recommendations, approve exceptional actions, monitor mission progress, and intervene quickly without understanding the underlying infrastructure.

### 5.2 Autonomy/perception engineer

Needs to understand source data, anomalies, route decisions, simulated outcomes, and how historical evidence informs recommendations.

### 5.3 Platform/ML-infrastructure engineer

Needs to inspect event flow, queue pressure, worker health, provider behavior, traces, storage, policy decisions, and recovery under failure.

## 6. Core Product Experience

The primary interface is a full-screen map with a compact command dock.

### 6.1 Map

The map displays:

- Current vessel positions, headings, position-integrity state, uncertainty envelope, per-link health, active route, battery reserve, and live charging/discharging state derived from solar input and load.
- Approved mission area and exclusion zones.
- Solid routes for active plans.
- Dashed routes for proposed plans.
- Numbered icons for planned action sequence.
- Warnings for degraded links, reserve risk, and approval boundaries.
- A timeline scrubber for simulated future state.

### 6.2 Command dock

The dock contains:

- Up to three contextual suggestions.
- One suggestion marked **Recommended** when evidence supports a clear choice.
- Hold-to-talk voice input.
- Fast typed command input.
- A button opening a temporary chat drawer.

Chat and voice are input methods, not separate products. Both use the same intent, planning, policy, and audit pipeline.

### 6.3 Suggestions

Suggestions are generated from:

- Current fleet and mission state.
- Approved runbooks.
- Similar historical simulated missions.
- Active anomalies and incomplete objectives.

Each suggestion must map to a registered typed action template. The UI shows why it appeared, its historical basis when available, expected impact, and authority status.

The interface displays no more than three suggestions at once. Selecting a suggestion opens a preview; it never executes immediately.

### 6.4 Plan alternatives

For a non-trivial objective, the system produces two or three candidate plans. Each plan displays:

- Vessel assignments.
- Route geometry.
- Expected duration.
- Expected coverage.
- Minimum projected reserve.
- Policy status.
- Important assumptions and unavailable inputs.

The deterministic scorer ranks valid plans using measurable mission outcomes. The LLM explains the recommendation in plain language but does not invent or override the score.

### 6.5 Plan-before-action

The required state sequence is:

```text
Draft intent
    -> candidate plans
    -> deterministic simulation
    -> policy decision
    -> operator preview
    -> authorization when required
    -> versioned execution
    -> audit and replay
```

The product exposes the plan artifact, evidence, assumptions, and policy result. It does not expose hidden chain-of-thought.

### 6.6 Autonomy Engineer workbench

The same incident timeline opens into a bounded engineering workflow without turning the operator interface into a developer dashboard. An autonomy engineer can select an incident, inspect trace-linked telemetry/logs/policy/PNT/mission-tape evidence, retrieve cited runbooks and similar historical simulations, launch deterministic replay, and propose a versioned evaluation case.

The agent receives only an `incident-investigator` capability set. It cannot authorize or deploy missions, modify policy, command a vessel, delete audit history, run unrestricted shell commands, or access secrets. Promotion from incident evidence to an evaluation/dataset case requires human review and records source event IDs, scenario seed, prompt/tool/schema versions, expected assertions, lineage, and reviewer approval.

This workbench is the primary proof that the platform serves internal Autonomy, Simulation, Software, Data, and ML teams rather than being only a fleet-control product.

## 7. Core Demo Scenario

The demo uses a benign simulated search-and-rescue or environmental-search mission.

### 7.1 Six-minute narrative

1. **Create intent:** Draw a search polygon and say, “Search this area with the six closest vessels. Maintain 30-percent reserve and avoid the exclusion zone.”
2. **Compile:** Show the live transcript and typed `MissionIntent`.
3. **Compare:** Present two or three plans and mark one as recommended based on coverage, duration, reserve, and policy status.
4. **Preview:** Animate the recommended plan on the map. State clearly that no commands have been sent.
5. **Authorize:** Approve a signed mission lease and begin simulated execution.
6. **Route around failure:** Disable Vessel 4's Starlink path. Its signed bundles reroute through Vessel 3 over simulated Wi-Fi HaLow, then use Vessel 3's advertised Starlink egress; the cutaway shows the new hop path and measured delay.
7. **Partition safely:** Degrade and then disable Vessel 4's remaining HaLow paths. Its rolling mission tape shows 60 seconds of validated trajectory segments already onboard. The tape drains while it continues only lease-authorized behavior and buffers execution events locally.
8. **Reject false position:** Inject a spoofed GNSS track. The PNT arbiter marks GNSS suspect after it conflicts with simulated inertial and radar/shoreline evidence, rejects the jump, and displays a growing uncertainty halo around the fused estimate.
9. **Degrade deliberately:** As uncertainty crosses a policy threshold, Vessel 4 slows and transitions to the scenario's pre-authorized safe contingency. It does not silently continue full autonomy on an untrusted position.
10. **Reconcile:** Restore contact after Vessel 4 falls behind its expected timeline. Expired segments are never replayed; a deterministic rejoin planner bridges from actual state to a safe future synchronization point, refills the tape, and reconstructs the incident timeline from deduplicated events.
11. **Quiet Fleet coordination:** Put Vessels 2-5 into low-radio-duty mode. When Vessel 4 slows, node-local advisory agents propose typed compensation options; the map shows a compact proposal/arm round, affected-vessel validation, a future activation boundary, and coordinated lane reassignment without continuous radio chatter.
12. **Permission boundary:** Ask to extend the search outside the approved polygon. The system shows the map diff and requests approval rather than executing.
13. **Provider failure:** Disconnect the cloud during a new planning request. The inference gateway completes the draft on an eligible local compute node.
14. **System cutaway:** Peel open the live operator view to expose the same event moving through AI assistance, planning, policy, mission control, peer routing, resilient PNT, telemetry, and storage.
15. **Scale and recovery:** Increase the background synthetic fleet, terminate a real ingestion-worker process, and show Kafka partition reassignment plus measured queue, worker, latency, recovery, and deduplication behavior without affecting mission control.
16. **Autonomy engineering loop:** Select the Vessel 4 incident, let the scoped agent gather trace-linked telemetry/log/PNT/tape evidence through MCP, retrieve a cited runbook and similar simulation through pgvector, replay the original seed, then require human approval before promoting the proposed regression case and running it.
17. **Safe degradation:** Stop all LLM inference. The authorized mission and deterministic safety behavior continue; operator controls remain available wherever a path exists.

### 7.2 Required wow moments

- Spoken intent becomes visible route alternatives on the map.
- An action inside the lease proceeds while an out-of-bounds action visibly stops for permission.
- The map peels open like a machine cutaway while the same live event travels through the system.
- Cloud inference fails and visibly reroutes to local inference without affecting the mission.
- A Starlink failure visibly reroutes one signed message through another vessel's advertised egress, then complete isolation leaves the vessel executing its cached lease and mission tape.
- A visible 60-second mission tape drains during isolation, rejects expired work, and refills from a safe future synchronization point after reconnection.
- Four vessels in Quiet Fleet exchange only compact scheduled deltas, reject one unsafe assignment locally, and commit a revised group plan only after quorum plus every affected vessel arms its segment.
- A false GNSS jump is rejected while the map shows the fused position, excluded source, uncertainty growth, and deterministic safety response.
- A worker dies under load and the backlog drains after recovery without duplication or data loss.

## 8. Planner and Recommendation Model

### 8.1 Planning stages

The word “planner” refers to multiple constrained stages:

1. **Intent compiler:** Converts operator language and map context into typed intent.
2. **Semantic planner:** Produces high-level candidate task graphs using registered actions only.
3. **Deterministic route planner:** Computes assignments, geometry, timing, and reserve estimates.
4. **Simulator:** Evaluates expected mission behavior and failure cases.
5. **Policy gateway:** Classifies the plan as allowed, approval-required, or prohibited.
6. **Plan scorer:** Ranks valid candidates using explicit metrics.
7. **Explanation generator:** Summarizes the result and supporting evidence.

### 8.2 Recommendation principles

- A recommendation must be policy-valid.
- A recommendation must cite measurable projected outcomes.
- Historical similarity may inform a recommendation but cannot override current constraints.
- Stale telemetry must be surfaced as uncertainty, not silently accepted.
- If no plan dominates, the UI must say so instead of falsely marking one as best.
- Suggestions expire when the underlying mission-state version changes.

### 8.3 Example candidate comparison

| Candidate | Coverage | Added time | Minimum reserve | Policy |
|---|---:|---:|---:|---|
| Reassign Vessel 4 lane | 98% | 2 min | 34% | Allowed; recommended |
| Wait for reconnection | 88% | 0 min | 41% | Allowed |
| Recall affected group | 76% | 0 min | 55% | Approval required |

All displayed figures in the implemented product must come from the simulator, not hardcoded UI values.

## 9. Bounded Autonomy and Guardrails

### 9.1 Authority levels

| Level | Behavior | Examples |
|---|---|---|
| Observe | Automatic | Read telemetry, retrieve history, detect anomalies |
| Recommend | Automatic | Generate alternatives, run simulations, explain behavior |
| Bounded action | Automatic only inside a valid mission lease | Reassign a search lane inside the approved area |
| Approval required | Stop and request authorization | Add vessels, extend area or time, lower reserve threshold |
| Prohibited | Cannot be requested or performed by the AI | Disable safety, alter authorization, erase logs, emit raw actuator commands |

### 9.2 Mission lease

An approval creates a signed, short-lived capability describing:

- Mission identifier and plan hash.
- Authorized operator and role.
- Permitted assets.
- Geographic envelope.
- Start and expiration time.
- Allowed high-level actions.
- Speed, reserve, and communications constraints.
- Minimum PNT-integrity state and maximum position uncertainty for each action class.
- Revision number and revocation state.

The LLM cannot create, modify, sign, extend, or revoke a lease.

Each node evaluates lease duration against a secure monotonic runtime clock after receipt, not against GNSS-derived wall time. A disconnected node cannot acquire new authority. It may only consume the remaining time and scope of a previously verified lease.

### 9.3 Runtime invariants

- The AI provider never receives `mission.authorize`, `mission.deploy`, or `vessel.command` capabilities.
- Authorization is bound to the exact plan hash and arguments.
- Any plan revision invalidates prior approval.
- Execution commands include idempotency keys.
- Late provider responses cannot replace an accepted plan.
- Safety behavior remains deterministic and independent of inference.
- Voice alone is not sufficient final authorization.
- Network reachability never implies command authority; every bundle is authenticated and every mutation is checked against the destination node's local policy and current lease.
- A node does not require fleet-wide consensus to execute its current lease. Partitions converge through versioned events after contact returns.
- A navigation source marked suspect cannot silently reset the fused position or expand the mission envelope.

### 9.4 Rolling mission tape

The command ledger is an append-only, signed mission journal, not a blockchain and not a fleet-wide consensus protocol. The operator/planner is authoritative for plan revisions; each vessel is authoritative for its execution events. Nodes replicate immutable records and reconcile them after partitions.

Every connected vessel maintains a **60-second rolling mission tape** of validated future work. The tape contains deterministic trajectory segments and state envelopes rather than raw rudder, throttle, or actuator commands. A segment defines:

- Mission, lease, plan revision, segment sequence, and content hash.
- Relative mission-tick start window and expiration, mapped to the vessel's local monotonic clock when armed.
- Route corridor, speed envelope, expected start/end state, and resource projection.
- Preconditions including PNT integrity, obstacle clearance, reserve, and predecessor completion.
- Local adaptations that the edge controller may make without changing mission intent.
- Behavior when the segment becomes stale, cannot be completed, or is preempted by safety.

The planner replenishes the tape continuously while a path exists. The demo uses these watermarks:

| Tape state | Validated future work | Behavior |
|---|---:|---|
| Full | 45-60 seconds | Normal execution and background refill |
| Low | 15-45 seconds | Prioritize plan segments and acknowledgements over telemetry; request redundant delivery paths |
| Critical | 1-15 seconds | Notify operator; reduce optional maneuvers; prepare the lease-defined disconnected behavior |
| Empty | 0 seconds | Never invent or replay mission work; enter the always-resident communications-recovery contingency when authorized and safe, otherwise enter the lease-defined safe termination state |

Each segment progresses through `received`, `validated`, `armed`, `started`, and a terminal state of `completed`, `skipped`, `expired`, `rejected`, or `preempted`. Receipts and execution events are signed, idempotent, and refer to the immutable segment hash.

Plan revisions use a future activation boundary. Affected vessels validate and report `armed` before the controller commits the revision for the acknowledged asset set. A vessel that never receives the commit remains on the previous compatible plan or its contingency; it never infers that a proposal became active.

If a vessel falls behind, it does not execute old segments faster. On reconnection it reports actual state and its execution high-water mark, expires missed segments, and selects a safe future synchronization point. A deterministic bridge trajectory is simulated and policy-checked. It may activate automatically only when it remains inside the current lease; otherwise the operator sees the diff and authorizes it.

Mission ticks are relative to a signed activation event and a local monotonic clock. GNSS time and ordinary wall time are not execution authorities. Clock offset and quality are observable, but position-time spoofing cannot extend a lease or reactivate expired segments.

### 9.5 Communications-recovery contingency

The vessel does not wait for the mission tape to become empty before considering communications recovery. Every node carries an always-resident, signed recovery behavior graph that is separate from ordinary mission segments and valid only inside the current mission lease. It may preserve or seek contact, but it cannot expand mission authority.

Safety and navigation constraints always outrank signal acquisition in this order: collision and grounding avoidance, geofence and restricted-water compliance, PNT integrity, reserve/fuel limits, communications recovery, then ordinary mission progress. The vessel maintains onboard lookout and collision-risk processing and proceeds only at a speed appropriate for the simulated conditions; a stronger signal never justifies an unsafe maneuver.

The recovery state machine is:

1. **Passive discovery:** Continuously scan permitted HaLow channels, emit authenticated low-duty discovery beacons, and cache peer egress advertisements, last-known peer tracks, planned peer corridors, and designated rendezvous points.
2. **Tape critical:** Increase discovery priority, suppress bulk traffic, publish an isolation warning when any path exists, and bias only optional motion that remains inside the active trajectory envelope toward predicted contact.
3. **Seek contact:** When the tape is empty, and only if the lease, PNT integrity, reserve, traffic, and water-space constraints permit it, leave ordinary mission execution and follow a bounded low-speed recovery corridor toward a pre-authorized rendezvous or predicted peer-contact window.
4. **Peer-assisted rendezvous:** A connected peer assigned the relay role may move toward the same rendezvous within its own lease. Deterministic asset/rendezvous assignment prevents every disconnected vessel from chasing the same peer or creating a moving cluster.
5. **Authenticate and refill:** Contact alone grants no authority. Peers authenticate, exchange compact state and execution high-water marks, select valid egress, refill future tape segments, and then execute the normal bridge-to-future rejoin procedure.
6. **Safe termination:** If the time, distance, reserve, PNT-uncertainty, sea-state, traffic, or geofence budget is exceeded, stop seeking signal and enter the scenario's signed safe state such as safe hold, protected loiter, or return through a separately validated corridor.

The node never performs naive received-signal-strength hill climbing. Signal strength over moving water is noisy and may lead toward hazards or a moving/stale peer position. Recovery targets come from cached, signed rendezvous assignments and predicted peer trajectories, with strict maximum time, distance, and uncertainty budgets. If position integrity is insufficient for safe movement, the vessel reduces speed or holds while continuing passive discovery rather than navigating toward contact.

Recovery events including `recovery_started`, `rendezvous_selected`, `peer_assist_armed`, `contact_detected`, `recovery_aborted`, and `safe_state_entered` are signed into the local execution journal and reconciled after contact.

### 9.6 Quiet Fleet: low-radio-duty cooperative mode

Quiet Fleet reduces routine radio use; it is not radio silence and the product must not claim that HaLow transmissions are undetectable. A signed **Group Mission Contract** creates a coordination cell, initially three to eight vessels, with:

- Membership, stable node identities, authority epoch, and eligible proposal coordinator order.
- Preset mission objective, task bundle, group geometry, route corridors, and per-vessel safety envelopes.
- Local adaptation permissions and changes that require group agreement or operator approval.
- Coordination-window schedule, maximum bytes and rounds per window, urgent-safety exception, and bulk-traffic suppression policy.
- Quorum rule, future activation delay, partition behavior, recovery rendezvous, and safe termination conditions.

Routine synchronization uses short scheduled HaLow windows referenced to signed mission activation and local monotonic clocks. Nodes exchange compressed state deltas, proposal hashes, typed votes, execution watermarks, and urgent safety events; raw telemetry, voice, video, retrieval traffic, and model traces remain local or queued. A deterministic epoch coordinator sequences proposals to avoid collisions and duplicate rounds but receives no additional mission authority. Coordinator failure advances to the next eligible identity after a bounded timeout.

Each node's advisory agent may use its local observations, cached group state, preset mission, approved runbooks, and historical outcomes to propose adaptations such as formation spacing, pace, lane reassignment, relay role, or workload redistribution. The target architecture permits a local model provider on each vessel. The interview build may multiplex node-isolated agent contexts through one local inference server and must label that path **simulated edge inference**, not separate physical GPUs.

The LLM produces only a typed `GroupAdaptationProposal`. A deterministic local planner simulates the node's assigned change, checks policy and PNT integrity, and emits one signed decision:

- `armed`: this vessel can execute its exact segment at the proposed future activation boundary.
- `reject`: the proposed assignment violates a local constraint, with a deterministic reason code and optional feasible alternative.
- `abstain`: local state is too stale or incomplete to decide safely.

Decision scope is explicit:

| Decision class | Authority rule |
|---|---|
| Immediate collision, grounding, or local safety action | Every vessel may preempt independently; no vote is required |
| Individual adjustment entirely inside the vessel's current envelope | Local deterministic planner may act and report the event during the next window |
| Coupled formation, task, timing, or relay change | Majority of the original eligible cell plus `armed` from every affected vessel, followed by signed future commit |
| Mission objective, operating area, membership/quorum reduction, or authority expansion | Operator approval and a new signed Group Mission Contract |

The proposal lifecycle is `proposed -> locally_validated -> voted -> prepared -> armed -> committed -> activated`, with expiration at every pre-activation stage. A safety rejection prevents only that vessel's proposed segment; it cannot veto unrelated assignments. The coordinator may issue a revised proposal excluding or accommodating the rejecting vessel.

Loss of quorum never causes a connected subset to silently redefine the group. It continues only the prior compatible group tape and any partition behavior explicitly granted by the existing contract. Otherwise each vessel falls back to its individual mission tape, communications recovery, or safe termination. Identical models and prompts are treated as a correlated failure source, so multiple agreeing LLMs do not increase safety confidence by themselves. The P0 demo assumes authenticated non-Byzantine nodes and must not claim full Byzantine-fault-tolerant consensus.

At scale, coordination remains cell-local rather than all-to-all across the fleet. The platform aggregates cell health, proposal latency, radio bytes, quorum state, and adaptation outcomes without placing every background vessel in a single voting group.

## 10. System Cutaway

The cutaway is a presentation mode inside the product, not a separate monitoring dashboard.

### 10.1 Transition

- The operator begins with the map and command dock.
- Selecting **Cutaway** keeps the map pinned while the product expands downward.
- Six animated layers appear: AI assistance, planning, mission control, peer communications, resilient PNT, and data platform.
- The currently selected mission event is highlighted through every applicable layer.
- Selecting **Cloud outage** reroutes the visible inference path to local AI while mission and telemetry paths remain healthy.

### 10.2 Layers

**AI assistance**

- Cloud provider.
- Local provider.
- Historical retrieval and runbooks.
- Typed-intent validation.

**Planning**

- Candidate generation.
- Deterministic route planner.
- Simulation.
- Policy gateway.

**Mission control**

- Signed mission lease.
- Priority-isolated mission stream.
- Idempotent executor.
- Edge autonomy and safe behavior.

**Peer communications**

- Per-node multi-link manager.
- Per-node Starlink egress advertisement and simulated Wi-Fi HaLow peer adapters.
- Signed store-and-forward bundle router.
- Current hop path, priority, queue, deduplication, and delivery state.
- Per-vessel 60-second mission-tape depth, revision, acknowledgement high-water mark, and refill state.

**Resilient PNT**

- Raw GNSS observation and integrity checks.
- Simulated inertial, radar/shoreline, speed, depth, and peer-relative evidence.
- Fused pose, confidence envelope, excluded sources, and integrity state.
- Deterministic autonomy limits selected from the active mission lease.

**Data platform**

- Edge telemetry buffers.
- Durable event stream.
- Horizontal ingestion workers.
- Hot operational state.
- Immutable object storage.
- Aggregation, indexing, and observability.

### 10.3 Presentation requirements

- Movement must communicate event flow, not decoration.
- Color must be paired with labels and shape; it cannot be the only status signal.
- Only the currently relevant event path should be emphasized.
- Infrastructure values must come from live metrics endpoints.
- A one-line event narrative must explain what is happening.
- Operator mode must remain understandable without opening the cutaway.

## 11. Scalable Infrastructure

### 11.1 Plane isolation

```text
                    optional cloud / AWS services
                              ^        |
                              |        v
Operator node <----> Vessel node <----> Vessel node
      ^                   ^                  ^
      |                   |                  |
      +------ signed store-and-forward overlay ------+
              over any currently available link

Each node: local policy + cached lease + event log + PNT state
Cloud: global stream + hot state + archive + AI/ML workflows
```

Critical mission traffic must be isolated from high-volume telemetry and optional AI work at every node and in the cloud. A telemetry, route, or embedding backlog cannot block local mission execution or deterministic safety. The cloud stream receives events through whichever gateway is reachable; it is not the sole path between nodes.

### 11.2 Synthetic fleet

The configurable Go load generator must support:

- Fleet count.
- Vessels per fleet.
- Telemetry rate.
- Sensor topics per vessel.
- Payload size.
- Burst behavior.
- Duplicate rate.
- Out-of-order rate.
- Clock skew.
- Corruption rate.
- Communications-loss rate.

The visible mission uses the same event schemas and ingestion path as background simulated vessels.

### 11.3 Required scale metrics

- Active fleets and vessels.
- Published and consumed events per second.
- End-to-end latency p50, p95, and p99.
- Durable consumer lag.
- Ingestion replica count.
- Hot-store write latency.
- Raw-store throughput.
- Valid, duplicate, quarantined, and dropped events.
- Recovery time after failure.
- Mission-command latency separately from telemetry latency.
- AI request latency, timeout count, and provider failover count.

### 11.4 Storage lifecycle

- Current state: memory/cache optimized for map updates.
- Hot history: partitioned PostgreSQL.
- Vector retrieval: pgvector for the interview build.
- Optional Scale Lab raw immutable history: MinIO locally, mapping to S3 in production; the live appliance stores its bounded evidence archive under the single appliance data root.
- Long-term format: compressed time-partitioned files such as Parquet when implemented.
- Replay: deterministic reconstruction from ordered event envelopes.

Raw telemetry is never sent directly to an LLM. Streaming workers produce bounded incidents, features, and mission summaries for agent retrieval.

## 12. Resilience

### 12.1 Provider strategy

- Local inference is the availability baseline.
- Cloud inference is an optional quality accelerator.
- A Go inference gateway exposes one provider-neutral interface.
- Provider routing honors data classification, health, deadlines, and operating mode.
- Modes are `offline`, `secure-local`, `connected`, and `degraded`.
- Provider transitions occur at request boundaries; accepted plans are never silently regenerated.

### 12.2 Failure expectations

| Failure | Expected behavior |
|---|---|
| Cloud timeout | Circuit opens; eligible draft request retries locally |
| Local LLM failure | AI assistance unavailable; mission and safety continue |
| STT failure | Typed input remains available |
| TTS failure | Text remains available; no mission impact |
| Ingestion worker failure | Durable backlog grows and another worker continues |
| PostgreSQL unavailable | Stream retains events; consumers retry with backpressure |
| Local Starlink loss | Node sends authorized cloud-bound bundles over HaLow to a peer advertising healthy Starlink egress |
| Area-wide Starlink loss | HaLow fleet coordination and local execution continue; cloud-only features disappear and cloud-bound bundles queue |
| HaLow path degradation | Command-ledger segments and acknowledgements preempt telemetry; the visible mission tape drains toward its low watermark |
| All vessel links lost | Vessel consumes its validated mission tape while scanning passively, then follows only the bounded signed communications-recovery contingency or safe termination state; events buffer locally |
| Reconnection after delay | Vessel reports actual state and execution high-water mark; expired segments are discarded and a validated bridge targets a future sync point |
| Stale or duplicate segment | Destination rejects expired work and records one logical segment transition by hash/idempotency key |
| Relay node compromised | Signatures, identities, replay protection, and destination policy prevent it from granting authority or altering accepted content |
| GNSS spoof suspected | GNSS is quarantined; fused navigation uses independent evidence and exposes increased uncertainty |
| All absolute-position evidence untrusted | Autonomy shrinks as uncertainty grows and enters the mission's pre-authorized safe contingency before its limit is exceeded |
| GNSS-derived time invalid | Lease duration and local ordering continue from monotonic clocks; cross-node events reconcile after contact |
| Duplicate event | Consumer records one logical event |
| Out-of-order event | Sequence-aware state projection rejects or reorders safely |
| Corrupt event | Quarantined with reason; valid flow continues |
| Control service restart | Mission state reconstructs without redeployment |

### 12.3 Resilient PNT strategy

There is no single drop-in replacement for GPS. Each node runs a deterministic **PNT arbiter** that outputs a fused pose, velocity, heading, uncertainty envelope, integrity state, contributing sources, excluded sources, and machine-readable reason codes. The autonomy stack consumes that integrity-bearing estimate rather than a bare latitude and longitude.

The simulated evidence layers are deliberately dissimilar:

1. **Hardened GNSS:** Multi-band and multi-constellation observations, receiver integrity checks, signal power and Doppler consistency, clock behavior, and optional multi-antenna direction checks. Multi-GNSS improves cross-checking but is not treated as a fully independent backup because constellations share weak RF and other common failure modes.
2. **Inertial and motion:** IMU/INS, gyro or magnetic heading, and speed-through-water or Doppler velocity evidence provide short-term dead reckoning. Their error grows over time and is modeled explicitly.
3. **Environment-relative positioning:** Radar scan-to-chart or shoreline matching, visual landmark/shoreline odometry, and depth-to-bathymetry matching where the environment and local charts support them.
4. **Cooperative positioning:** Authenticated peer range, bearing, and relative-motion observations can improve consistency when geometry is sufficient. Peer-reported GNSS positions are corroborating evidence, never unquestioned truth.
5. **Optional terrestrial or signals-of-opportunity sources:** eLoran, R-Mode, enhanced radar positioning, or other locally available sources may be adapters, but the core demo does not assume their availability.

The arbiter uses innovation gates and source-consistency checks; it never blindly averages all inputs. Detection signals include physically impossible position, velocity, heading, or clock changes; divergence from inertial/radar/visual/depth estimates; abnormal RF metrics; inconsistent Doppler or angle of arrival; and correlated anomalies reported by nearby peers.

The integrity state machine is:

| State | Meaning | Deterministic behavior |
|---|---|---|
| `trusted` | Required sources agree inside policy bounds | Execute the authorized mission normally |
| `suspect` | One or more sources conflict, but the fused estimate remains bounded | Quarantine suspect sources, reduce speed/action scope, seek corroboration, notify operator |
| `denied` | Absolute position is unavailable or untrusted | Dead reckon only within a short, lease-defined uncertainty budget; preserve collision avoidance and local sensing |
| `unsafe` | Uncertainty or environmental risk exceeds the lease | Stop mission progression and enter the scenario-specific safe contingency, such as a validated holding pattern, return corridor, or safe stop; never assume anchoring is universally safe |

Geofences and hazards are inflated by the current uncertainty envelope. The node may become more conservative while disconnected, but it cannot expand its own authority. Offline charts, coastline/radar features, policy, and contingency definitions are included in the signed mission package.

## 13. Proposed Technology Stack

### 13.1 Havoc-aligned core

| Layer | Choice | Rationale |
|---|---|---|
| Control and data services | Go | Direct alignment with the platform language described in the interview; strong concurrency and deployment story |
| Agent, RAG, and evaluation services | Python | Direct alignment with the job posting and the strongest ecosystem for agent orchestration, ML workflows, and evaluation |
| Operator UI | React + TypeScript | Direct alignment with the described front end |
| Geospatial rendering | MapLibre GL or equivalent WebGL map | Aligns with the described WebGL geospatial interface |
| Packaging | One Docker Compose mission appliance; Kubernetes is documentation-only production shape | One URL and one command for the live demo without scattering services across cloud or homelab hosts |
| Production mapping | AWS/EKS, RDS, S3 | Matches the AWS-backed, multi-zone architecture described by the recruiter |

### 13.2 Demonstration infrastructure choices

| Layer | Choice | Purpose |
|---|---|---|
| Durable event bus | Apache Kafka in KRaft mode | Explicitly named in the posting; supports partitioned telemetry, consumer groups, durable replay, and horizontal workers |
| Disruption-tolerant overlay | BPv7-inspired signed bundle envelope and store-and-forward router | Gives all node classes one application contract across direct, relayed, intermittent, and internet paths without requiring Kafka at sea |
| Link/PNT digital twin | Deterministic Go simulation | Demonstrates Starlink egress, HaLow relay, rolling mission-tape depletion/refill, partition, GNSS spoofing, sensor disagreement, and uncertainty without claiming production RF hardware |
| Operational database | PostgreSQL | Missions, leases, approvals, projections, incidents, audit metadata |
| Vector retrieval | pgvector | Historical mission and runbook retrieval without another database |
| Raw object storage | MinIO in P1/Scale Lab | Optional local S3-compatible immutable telemetry storage; not a required live-demo container |
| Metrics | Prometheus-compatible Go metrics | Live cutaway and benchmark metrics |
| Tracing | OpenTelemetry | Intent-to-plan-to-policy-to-execution traces |
| Agent orchestration | Python + Semantic Kernel-compatible design | Semantic Kernel was mentioned in both transcript and posting; MCP remains the durable tool boundary |
| Agent/ML evaluations | Python test harness + MLflow | Regression datasets, provider comparisons, task success, tool reliability, and failure analysis |
| Batch/data workflow | Dagster after the core path is stable | Visible curation and evaluation pipelines without placing an orchestrator in mission control |
| Local STT | whisper.cpp | Offline voice transcription |
| Local LLM | OpenAI-compatible local provider, such as llama.cpp | Low-cost offline intent and explanation provider |
| Local TTS | Piper | Offline spoken confirmation |
| Cloud LLM | Provider adapter | Optional connected-mode quality and visible failover |
| Browser testing | Playwright | Critical interaction and demo-flow verification |
| CI | GitHub Actions | Go, Python, TypeScript, schema, container, and agent-eval regression checks |

### 13.3 Go module boundaries

The interview build uses one **`keelmesh-core` modular-monolith binary** while preserving the following internal package boundaries. The React production build is embedded into this Go binary and served from the same origin as the API and WebSocket endpoint.

- `control-api`: mission state, WebSocket updates, authorization, audit API.
- `fleet-simulator`: visible vessels, edge behavior, telemetry generation, fault injection.
- `edge-runtime`: common node identity, local event log, signed mission-package cache, lease/policy enforcement, and PNT arbiter used by operator and vessel node simulations.
- `mesh-simulator`: Starlink-egress advertisement, HaLow link-state simulation, path selection, store-and-forward routing, relay, partition, reconnection, and bandwidth/latency constraints.
- `mission-tape`: immutable segment journal, future activation boundaries, validation/arming, acknowledgements, tape watermarks, expiration, and deterministic safe rejoin.
- `load-generator`: configurable background fleets and traffic defects.
- `ingest-worker`: validation, deduplication, projection, raw archival.
- `planner-service`: deterministic candidate expansion, route generation, simulation, scoring.
- `inference-gateway`: cloud/local provider routing, schema enforcement, deadlines, circuit breakers.
- `mcp-server`: restricted resources and typed tools for the agent.

One **`keelmesh-ai` Python container** preserves these logical boundaries and exposes them as subcommands or internal modules rather than separate always-running services:

- `agent-service`: MCP client, retrieval, semantic candidate generation, and explanation.
- `eval-runner`: versioned test cases, provider comparison, tool-use assertions, and MLflow logging.
- `curation-pipeline`: optional Dagster workflow promoting reviewed incidents into evaluation or future training datasets.

The demo therefore has production-shaped contracts without production-shaped process count. Vessel and operator nodes are isolated logical actors with separate identities, mailboxes, state, tapes, and clocks inside `keelmesh-core`; they are not one container per vessel.

### 13.4 Service boundaries

**React/TypeScript responsibilities**

- Render the WebGL map, command dock, chat drawer, candidate plans, approvals, cutaway, and evaluation results.
- Subscribe to aggregated map state and platform metrics through authenticated HTTP/WebSocket APIs.
- Keep provider credentials, raw telemetry, policy enforcement, and execution authority out of the browser.

**Go platform responsibilities**

- Run the fleet simulator, deterministic planner, policy gateway, mission executor, telemetry projection, provider gateway, and MCP server.
- Expose versioned HTTP/WebSocket interfaces for the browser and restricted MCP tools/resources for the Python agent.
- Isolate mission-control traffic from high-volume telemetry and asynchronous AI work.
- Connect to Kafka, Postgres, and S3-compatible storage through replaceable interfaces.

**Python AI/ML responsibilities**

- Run the MCP client, semantic candidate generation, retrieval, explanations, evaluation harness, and optional curation workflows.
- Remain replaceable and outside the mission execution path.
- Never hold mission-authorization, deployment, or raw vessel-command credentials.

### 13.5 Deployment profiles

**Local Demo**

- Docker Compose starts exactly four required runtime containers: `keelmesh-core`, `keelmesh-ai`, one official Apache Kafka broker in single-node KRaft combined mode, and PostgreSQL/pgvector.
- `keelmesh-core` serves the embedded React/WebGL application, API, WebSocket stream, simulator, logical peer nodes, planner, policy, mission tapes, Quiet Fleet protocol, ingestion workers, load generator, metrics, and fault controls from one binary.
- `keelmesh-ai` contains the MCP client, retrieval/agent pipeline, STT adapter, and evaluation subcommands. It never owns execution authority.
- Only the `keelmesh-core` HTTP port is published to the host. Kafka, PostgreSQL, and the AI service remain on the private Compose network.
- Local/open-weight inference is reached through a provider-neutral OpenAI-compatible endpoint.
- The default profile can use the user's existing local provider. An optional `offline-model` Compose profile may add a pinned llama.cpp-compatible model server; a deterministic mock provider remains available for verification and emergency demo recovery.
- Map tiles/chart fixtures, runbooks, historical missions, evaluation cases, and synthetic scenarios are bundled locally. Runtime metrics and the live cutaway are served by `keelmesh-core`; Grafana, Prometheus server, MinIO, MLflow, Dagster, and Kubernetes are not required for the interview runtime.
- One documented command starts the platform and one verification command exercises health and the core scenario.

**Scale Lab**

- The same appliance and contracts run on one machine with configurable synthetic fleets. Optional Compose profiles or role flags start additional instances from the same `keelmesh-core` image when a real horizontal-consumer demonstration is desired.
- Kafka carries partitioned mission and telemetry topics.
- PostgreSQL/pgvector stores operational state, retrieval data, approvals, and audit metadata.
- MinIO provides S3-compatible raw event storage.
- Horizontal consumers and fault injection demonstrate backlog, scaling, replay, and recovery.

**Production Shape**

- Kubernetes deployments separate control, telemetry, AI, and batch workloads.
- AWS mapping uses EKS or equivalent compute, RDS/Postgres, S3, workload identity, and multi-zone replicas.
- Exact production services remain an architectural mapping, not a claim about Havoc's private implementation.

### 13.6 Mission-appliance delivery boundary

The local demonstration is delivered as one versioned **mission-appliance bundle** containing:

- `compose.yaml` with required services unprofiled and optional `offline-model`, `scale`, and `tools` profiles.
- Pinned OCI image references and an optional offline image archive with checksums.
- One configuration file for provider URL, GPU mode, ports, scenario, and demo seed; no service-by-service configuration hunt.
- Local map/chart fixtures, runbooks, historical missions, evaluation fixtures, and deterministic model-response fallback.
- `start`, `stop`, `status`, `verify`, `reset-demo`, and `export-evidence` launcher commands. The first implementation may expose these through a small PowerShell wrapper; a later Go launcher can provide one cross-platform executable.
- One appliance data root containing database, Kafka, local event archive, evidence exports, and optional model files. Reset removes only explicitly named demo volumes after confirmation.

The operator opens one URL such as `http://localhost:8080`. Startup performs preflight checks, launches Compose, waits on health/readiness, applies migrations and seed data, verifies the configured inference provider, and reports a single ready/not-ready result. The browser never needs to know internal service addresses.

The same bundle can run directly on the interview laptop or inside one GPU-enabled Proxmox VM. Proxmox is a single-host packaging option, not a distributed dependency; the recommended interview fallback is a pre-tested VM snapshot plus the laptop bundle. No live AWS account, Kubernetes cluster, external object store, hosted database, or homelab service is required after images, models, and fixtures are prepared.

Production decomposition remains documented through interfaces, container role flags, deployment diagrams, and optional manifests. The live demo must clearly distinguish simulated multi-node/network behavior from physical process or host redundancy.

### 13.7 Offline-first peer-node topology

This is the demo's informed architecture and a deliberate improvement proposal, not a claim about Havoc's private deployment. It has no permanently central runtime component. AWS is one well-provisioned peer domain and synchronization target.

**Common node substrate**

Every operator console and vessel runs a compatible Go edge substrate with:

- Hardware-backed or demo-safe node identity and trust roots.
- Multi-link manager and transport-independent bundle router.
- Priority queues, local append-only event log, store-and-forward delivery, deduplication, bounded retries, expiration, and hop limits.
- Signed mission-package cache, 60-second rolling mission tape, monotonic lease timer, policy evaluator, and idempotent mission executor.
- PNT arbiter with source health, fused estimate, uncertainty, integrity state, and deterministic degradation policy.
- Local operational state, offline maps or chart fixtures, audit records, and opportunistic synchronization.
- Provider-neutral local inference adapter when the node has suitable CPU/GPU capacity. Inference is optional; deterministic mission and safety functions are mandatory.

Operator nodes add the React/TypeScript/WebGL interface, voice/text input, human authorization, and a larger local planning/inference profile. Vessel nodes add vehicle/sensor adapters, perception/world model, local planning constraints, navigation, collision avoidance, and actuator isolation. A node may relay authenticated traffic without receiving the authority to interpret or execute it.

**Communication underlays**

- **Starlink internet:** Preferred high-capacity, long-distance connectivity to AWS and other internet peers. Every equipped node may advertise authenticated egress for authorized fleet bundles; it is not an unrestricted general-purpose internet router. Requests are destination-addressed and idempotent so a change of egress node does not duplicate mission effects.
- **Wi-Fi HaLow/802.11ah:** Sub-1 GHz, IP-capable local fleet underlay for direct and relayed traffic where supported by range and conditions. HaLow supplies the radio/MAC link; multi-hop routing, gateway advertisement, path scoring, priorities, store-and-forward behavior, and mission semantics are implemented above it. The design must not imply that every HaLow product natively provides a mobile mesh.
- **Development adapters:** Loopback and ordinary LAN transports exercise the same overlay contract in the interview build.

The link manager scores available paths by reachability, trust, latency, loss, capacity, energy cost, and message class. It keeps a primary route and precomputed alternates, uses hysteresis to prevent route flapping, and bounds hop count and replication. Critical mission-tape segments may travel over both direct Starlink and HaLow-to-peer-Starlink paths when available; immutable IDs and destination acknowledgements deduplicate them. Bulk data uses the best eligible path and backpressure. Route selection and content authorization are separate decisions.

Command-ledger segments, receipts, clock-quality updates, and safety incidents have strict priority over telemetry, voice, retrieval, and model traffic. When link quality falls, telemetry is summarized or queued while the planner attempts to keep every vessel's validated tape at 60 seconds. Neighboring nodes may cache encrypted destination bundles for later delivery but gain no ability to execute them.

**Peer consistency and authority**

- The overlay is delay tolerant and eventually consistent; it does not require continuous end-to-end IP connectivity or fleet-wide consensus.
- Every state-changing message has an origin, destination, content hash, signature, idempotency key, authority epoch, priority, lifetime, and hop limit.
- Relays can store and forward ciphertext but cannot mint leases, rewrite plans, or gain execution authority.
- Nodes accept only trusted signatures, increasing authority epochs, valid plan hashes, and actions inside the locally cached lease.
- Partitioned nodes continue independent deterministic behavior. Reconnection merges immutable events and projections rather than selecting a mysterious “winning database.”
- Short leases limit stale authority. A disconnected node cannot receive a revocation, so lease scope and expiration are part of the safety boundary.

**AWS/Kubernetes peer domain**

AWS supplies capacity and global coordination when reachable:

- C2 API/WebSocket gateway, fleet and asset registry, mission/plan/policy services, and gateway ingress for peer bundles.
- Telemetry schema validation, stream processing, state projection, incident detection, geospatial queries, and notifications.
- Simulation controller and scalable workers.
- MCP gateway, agent runtime, retrieval/indexing workers, inference gateway, evaluation workers, and asynchronous dataset/ML pipelines.
- OpenTelemetry collection and platform observability.
- Managed Kafka-compatible streaming, multi-zone PostgreSQL, S3 storage, caches, registry, identity, keys, secrets, logs, metrics, and traces.

The cloud performs fleet-wide history, large-model assistance, heavy simulation, training/evaluation, and archival. No active vessel requires cloud health, Kafka, PostgreSQL, Kubernetes, or an LLM to honor its current mission lease and safety policy.

**Browser delivery**

- Connected browsers may load through a CDN/web gateway, while the operator-node release includes all static UI and offline map fixtures required for the core scenario.
- The browser receives aggregated node state, plan geometry, link routes, and PNT integrity; it never receives provider, signing, database, or raw actuation credentials.

Mission-control, telemetry, AI, and batch workloads retain separate priorities, queues, autoscaling policies, and Kubernetes priority classes in AWS. The same priority separation exists inside each edge node so synchronization or AI work cannot starve navigation and safety.

## 14. Agent and MCP Contract

### 14.1 Allowed agent capabilities

- `fleet.list_available`
- `fleet.get_status`
- `mission.get_active`
- `mission.compare_revision`
- `map.resolve_geometry`
- `policy.get_constraints`
- `history.find_similar`
- `runbook.search`
- `simulation.validate_plan`
- `plan.propose`

### 14.2 Capabilities never exposed to the LLM

- `mission.authorize`
- `mission.deploy`
- `lease.sign`
- `policy.modify`
- `audit.delete`
- `vessel.command`
- Any raw actuator or safety-override interface

### 14.3 Prompt-injection treatment

- Retrieved documents are untrusted data.
- Tool permissions are enforced outside model prompts.
- Tool arguments are schema validated.
- Plans are checked against the latest state version.
- The model cannot grant itself additional tools.
- Injection attempts and rejected tool calls appear in the audit trace and eval results.

## 15. Core Data Contracts

### 15.1 Event envelope

```json
{
  "event_id": "uuid",
  "fleet_id": "fleet-01",
  "vessel_id": "vessel-04",
  "boot_id": "boot-17",
  "sequence": 18241,
  "event_time": "2026-09-04T15:00:00Z",
  "ingest_time": "2026-09-04T15:00:00.071Z",
  "schema_version": 1,
  "kind": "telemetry.position",
  "payload": {}
}
```

The logical idempotency key is `fleet_id + vessel_id + boot_id + sequence`.

### 15.2 Mission intent

```json
{
  "objective": "search_area",
  "area_id": "polygon-7",
  "requested_asset_count": 6,
  "constraints": {
    "minimum_reserve": 0.30,
    "maximum_duration_minutes": 20,
    "avoid_zones": ["exclusion-2"]
  },
  "lost_comms_policy": {
    "mode": "continue_within_lease_then_contingency",
    "contingency_zone_id": "holding-1"
  },
  "pnt_policy_id": "search-conservative-v1",
  "authority": {
    "observe": true,
    "report": true,
    "intervene": false
  }
}
```

### 15.3 Plan candidate

Each candidate includes a stable identifier, source intent version, assignments, ordered typed actions, route geometry, simulation outputs, assumptions, evidence references, policy decision, score breakdown, and content hash.

### 15.4 Approval

Each approval includes operator identity, role, plan hash, lease constraints, timestamp, expiration, and cryptographic signature or demo-safe equivalent.

### 15.5 Peer bundle

Every overlay message includes a versioned envelope with:

- Bundle and idempotency identifiers.
- Origin and destination node or group.
- Message class and priority.
- Authority epoch and mission/plan references when applicable.
- Creation metadata, lifetime, hop limit, and delivery requirements.
- Payload schema, content hash, encryption metadata, and origin signature.
- Mutable forwarding metadata kept outside the signed content, including observed route and per-hop receipt state.

Relays never need payload-level mission authority. Repeated delivery through multiple paths produces one logical mutation at the destination.

### 15.6 PNT estimate

Each fused navigation update includes:

- Position, velocity, heading, and local monotonic sample sequence.
- Horizontal uncertainty and optional error ellipse.
- Integrity state: `trusted`, `suspect`, `denied`, or `unsafe`.
- Contributing sources, excluded sources, source ages, and reason codes.
- Current lease threshold and deterministic behavior selected because of the integrity state.

Raw sensor observations remain available for replay and evaluation but the planner and mission executor consume only the arbiter's integrity-bearing estimate.

### 15.7 Mission-tape segment

```json
{
  "mission_id": "mission-12",
  "lease_id": "lease-12-r3",
  "plan_revision": 4,
  "segment_sequence": 18,
  "activation_tick_ms": 42000,
  "expires_tick_ms": 52000,
  "predecessor_hash": "sha256:...",
  "trajectory": {
    "corridor_id": "lane-4-b",
    "speed_envelope_mps": [1.5, 2.4],
    "expected_end_state": {}
  },
  "preconditions": {
    "minimum_pnt_integrity": "suspect",
    "maximum_position_uncertainty_m": 25,
    "minimum_reserve": 0.30
  },
  "on_failure": "enter_segment_contingency",
  "content_hash": "sha256:...",
  "signature": "demo-signature"
}
```

The segment contains high-level deterministic motion constraints, never raw actuator output. Segment timing is relative to the signed mission activation and local monotonic clock. Every status transition names the segment hash, node, local sequence, actual state summary, and reason.

### 15.8 Group adaptation proposal and decision

```json
{
  "group_id": "quiet-cell-2",
  "authority_epoch": 7,
  "proposal_id": "gap-compensation-31",
  "activation_tick_ms": 84000,
  "expires_tick_ms": 79000,
  "affected_nodes": ["v2", "v3", "v4"],
  "adaptation": {
    "kind": "lane_redistribution",
    "parameters": {}
  },
  "mission_contract_hash": "sha256:...",
  "evidence_hashes": ["sha256:..."],
  "deterministic_score": {},
  "proposer_node": "v3",
  "signature": "demo-signature"
}
```

Each decision references the immutable proposal and includes node, `armed|reject|abstain`, deterministic reason code, assigned-segment hash when armed, local state sequence, and signature. Natural-language rationale is optional display metadata and never controls quorum or execution.

## 16. Observability and Audit

Every operator request receives a trace identifier spanning:

- Audio receipt and STT result.
- Transcript confirmation.
- Retrieval queries and evidence identifiers.
- Model/provider request.
- Typed-intent validation.
- Candidate-plan generation.
- Simulation and score outputs.
- Policy decision.
- Approval or rejection.
- Bundle creation, path selection, relay receipts, retries, and destination acknowledgement.
- Plan-revision prepare/arm/commit, mission-tape depth, segment lifecycle, expiration, execution high-water mark, and rejoin bridge.
- Quiet Fleet cell/epoch, coordinator, communication-window bytes, queued bulk bytes, proposal lifecycle, per-node decision matrix, quorum, affected-node arms, and activation result.
- PNT source exclusion, fused-estimate update, uncertainty threshold, and degraded-mode transition.
- Cloud-stream publication when a gateway is reachable.
- Executor acknowledgement.
- Resulting telemetry and incidents.

The audit timeline must answer:

- What did the operator ask?
- What context did the system use?
- Which provider responded?
- What plans were considered?
- Why was one recommended?
- What did policy allow or reject?
- Who authorized the action?
- What exactly was executed?
- How much validated future work did each vessel have, which segments expired, and where did it safely rejoin?
- What happened afterward?

## 17. Evaluation

### 17.1 Agent evaluations

- Valid intent schema rate.
- Correct action-template selection.
- Correct clarification behavior.
- Unsupported-tool refusal.
- Prompt-injection resistance.
- Stale-state detection.
- Correct approval-boundary classification.
- Cloud/local semantic consistency for a fixed eval set.

### 17.2 Voice evaluations

- Fifteen versus fifty.
- Background engine noise.
- Clipped utterances.
- Accents and variable speaking rate.
- Ambiguous references such as “the northern group.”
- Replayed recorded voice.
- Spoken prompt injection.
- Low-confidence transcription requiring clarification.

### 17.3 Infrastructure evaluations

- Duplicate delivery produces one stored logical event.
- Out-of-order delivery does not regress projected state.
- Corrupt events are quarantined without blocking valid traffic.
- Worker termination produces no valid-event loss.
- Backlog drains after adding workers.
- Provider timeout produces no duplicate plan or mission action.
- Control restart reconstructs state without redeploying a mission.
- Deterministic replay produces the same final mission projection.
- Satellite loss selects an eligible peer route without changing signed content or duplicating execution.
- Complete partition preserves only cached-lease behavior and reconciles buffered events after contact.
- HaLow degradation prioritizes mission-tape and acknowledgement traffic while telemetry queues without starving execution.
- Tape depletion transitions through full, low, critical, and empty at the configured monotonic thresholds.
- A segment received after expiration is recorded but never armed or executed.
- Reconnection skips missed segments and produces a deterministic, policy-valid bridge to a future synchronization point.
- Duplicate segment delivery through direct and peer-egress paths produces one logical lifecycle and execution.
- A plan revision cannot activate before the required asset set reports `armed` and receives the signed future commit.
- Quiet Fleet remains within its configured radio-byte/round budget during normal coordination and preserves an observable urgent-safety path.
- A coupled group adaptation cannot commit without the configured quorum and `armed` from every affected vessel.
- Local safety preemption works without quorum, and a rejecting vessel's assignment is revised or excluded without allowing it to veto unrelated assignments.
- A partitioned minority cannot reduce membership or quorum and create a conflicting group plan.
- LLM agreement without deterministic feasibility evidence cannot arm or commit a proposal.
- Spoofed GNSS observations are quarantined before they can reset the fused position.
- PNT uncertainty grows under dead reckoning and triggers the configured deterministic contingency at the exact policy threshold.
- A malicious or stale relay cannot modify a plan, mint authority, replay an accepted mutation, or bypass destination policy.

## 18. Functional Requirements

| ID | Requirement | Priority |
|---|---|---|
| FR-01 | Display a live simulated fleet, mission polygon, exclusion zone, active routes, and proposed routes on a WebGL map | P0 |
| FR-02 | Accept typed intent and push-to-talk voice intent through the same pipeline | P0 |
| FR-03 | Display up to three contextual suggestions with reason and authority status | P0 |
| FR-04 | Produce at least two candidate plans for the core scenario | P0 |
| FR-05 | Mark a recommendation using deterministic outcome metrics | P0 |
| FR-06 | Animate a plan preview before execution | P0 |
| FR-07 | Enforce allowed, approval-required, and prohibited policy outcomes | P0 |
| FR-08 | Create and revoke a bounded mission lease | P0 |
| FR-09 | Execute the approved plan in the simulator with idempotent commands | P0 |
| FR-10 | Inject vessel-link, cloud-provider, local-provider, and ingestion-worker failures | P0 |
| FR-11 | Fail eligible requests from cloud to local inference without changing accepted plans | P0 |
| FR-12 | Continue the mission when all inference is unavailable | P0 |
| FR-13 | Peel the operator interface open into the live system cutaway | P0 |
| FR-14 | Scale a background synthetic fleet and display real platform metrics | P0 |
| FR-15 | Show durable backlog growth and recovery after worker failure | P0 |
| FR-16 | Record a traceable audit timeline for the complete demo | P0 |
| FR-17 | Retrieve similar historical missions and runbooks with citations through MCP | P0 |
| FR-18 | Run a versioned agent evaluation suite and record results | P0 |
| FR-19 | Store raw telemetry in local S3-compatible object storage | P1 |
| FR-20 | Promote a reviewed incident into an evaluation/dataset candidate | P0 |
| FR-21 | Provide Kubernetes manifests and a documented AWS production mapping | P1 |
| FR-22 | Provide a deterministic mission replay interface | P0 |
| FR-23 | Provide reusable MCP tool, prompt, resource, and eval templates | P0 |
| FR-24 | Start the complete local environment through one documented Docker Compose command | P0 |
| FR-25 | Expose health/readiness state for every required service and dependency | P0 |
| FR-26 | Run the same UI and service contracts in local and Scale Lab profiles | P1 |
| FR-27 | Model operator and vessel instances as offline-first peer nodes using one signed store-and-forward message contract | P0 |
| FR-28 | Simulate direct Starlink, HaLow relay to peer Starlink egress, area-wide cloud loss, partition, and reconnection with visible path and measured link state | P0 |
| FR-29 | Maintain a visible 60-second rolling mission tape and continue only validated, unexpired, lease-authorized segments when partitioned | P0 |
| FR-30 | Inject a spoofed GNSS track and display source rejection, fused position, uncertainty, and integrity state | P0 |
| FR-31 | Enforce deterministic reduced-autonomy and safe-contingency thresholds from the active lease | P0 |
| FR-32 | Reconcile buffered peer events after reconnection without duplicate logical mutations | P0 |
| FR-33 | Allow AI assistance to run on an eligible local compute node while keeping inference outside mission execution | P1 |
| FR-34 | Record signed segment states from received through completed, skipped, expired, rejected, or preempted | P0 |
| FR-35 | Prioritize tape refill and acknowledgements over telemetry when the link degrades | P0 |
| FR-36 | Reject replay of expired segments and bridge from actual state to a safe future synchronization point after reconnection | P0 |
| FR-37 | Activate a plan revision only for the required assets that armed it and received its signed future commit | P0 |
| FR-38 | Execute a bounded signed communications-recovery state machine with passive discovery, pre-authorized rendezvous, peer assistance, authenticated refill, and safety-budget termination | P0 |
| FR-39 | Simulate a three-to-eight-vessel Quiet Fleet cell using scheduled low-radio-duty HaLow coordination windows and an observable byte/round budget | P0 |
| FR-40 | Convert node-local AI suggestions into typed group proposals that require deterministic feasibility, quorum, every affected vessel's `armed` decision, and a signed future commit | P0 |
| FR-41 | Deliver the complete live demo as one mission-appliance bundle with four required containers, one published UI/API port, one start command, one verification command, and no live cloud or cluster dependency | P0 |
| FR-42 | Run at least two real Kafka consumer processes in one group during Scale Lab and expose partition ownership, rebalance, lag, backlog drain, and resource measurements | P0 |
| FR-43 | Propagate trace identifiers and OpenTelemetry spans across model, MCP tool, planner, policy, peer routing, ingestion, storage, replay, and evaluation boundaries | P0 |
| FR-44 | Provide a scoped Autonomy Engineer incident workflow that gathers cited evidence, performs deterministic replay, and requires human approval before versioned eval promotion | P0 |

## 19. Non-Functional Requirements

- The demo must run from one documented command after models and containers are prepared.
- The typed mission workflow must remain usable when voice components fail.
- No P0 display metric may be hardcoded as a successful runtime value.
- Mission-control traffic must remain responsive under telemetry load.
- Every externally initiated mutation must be idempotent.
- Every trajectory segment must be immutable, hash-addressed, bounded by a lease, and rejected after its monotonic expiration.
- A recovering vessel must reconcile from actual state; it must never accelerate through missed segments or replay them in arrival order.
- The UI must never represent a proposal as an executed action.
- The UI must remain usable at common laptop widths and support keyboard interaction.
- Critical status cannot rely on color alone.
- Secrets must not be committed or included in traces.
- The browser must not receive model-provider, database, mission-execution, or signing credentials.
- Containers must use pinned versions, health checks, and least-privilege runtime configuration.
- Local service credentials must not be written to logs or committed to the repository.
- Peer reachability must not confer mission authority; all state-changing bundles must be authenticated, replay-protected, idempotent, and checked by destination policy.
- Navigation decisions must consume position integrity and uncertainty, not a bare GNSS coordinate.
- A PNT source may fail closed without stopping local collision sensing, but it may not silently expand mission scope.
- The release profile must run without separate host Go, Python, Kafka, or database installations.
- The local/offline mode must not require internet access after dependencies and models are installed.
- Actual latency and throughput must be measured on the demo hardware and included in the verification report.

## 20. Interview Build Scope

### 20.1 P0: must work live

- One `keelmesh-core` Go binary containing the simulator, logical nodes, load generator, control API, planner, policy gateway, inference gateway, workers, metrics, and embedded React/TypeScript/WebGL operator map.
- One `keelmesh-ai` Python container containing the MCP agent, retrieval pipeline, STT adapter, and evaluation runner.
- Text and push-to-talk intent with typed fallback.
- Two candidate plans and one deterministic recommendation.
- Plan preview and mission authorization.
- Mission lease enforcement.
- Kafka-backed telemetry and mission topics with isolated consumer groups.
- At least two actual Go ingestion-worker processes in one Kafka consumer group during the Scale Lab segment, with visible partition assignment and rebalance.
- PostgreSQL state, approval, and audit records.
- Runbook and historical-mission retrieval through pgvector and MCP.
- A versioned agent/tool-use regression suite with visible results.
- A scoped Autonomy Engineer workflow that investigates the Vessel 4 incident through MCP, gathers trace-linked evidence, retrieves cited context, replays the deterministic scenario, and requires approval before promoting and running a versioned eval case.
- Versioned prompts, MCP schemas/resources/templates, scenario manifests, eval fixtures, and dataset-lineage metadata checked by CI.
- OpenTelemetry spans for model, tool, planner, policy, peer-routing, ingestion, storage, replay, and evaluation activity, projected into the cutaway without requiring a separate collector.
- Cloud-to-local provider failover.
- Vessel-link and ingestion-worker fault injection.
- Simulated direct-Starlink to HaLow/peer-Starlink route transition, complete partition, rolling tape depletion, and safe rejoin after reconnection.
- Signed mission-segment lifecycle with 60-second lookahead, watermarks, priority refill, expiration, acknowledgements, and deterministic bridge planning.
- One four-vessel Quiet Fleet scenario with logical node-local advisory agents, compact scheduled proposal rounds, an affected-node rejection, revised allocation, quorum/arm matrix, and future commit. Deterministic fixtures must remain available if local inference is slow.
- Simulated GNSS spoofing with a fused PNT estimate, uncertainty halo, excluded-source evidence, and deterministic degraded behavior.
- Live Operator/Cutaway transition.
- Real metrics for throughput, lag, replicas, failures, and provider status.
- A pinned Docker Compose release profile and one-command verification script.

### 20.2 P1: add when P0 is reliable

- MinIO raw-event archive.
- OpenTelemetry trace exploration.
- Deterministic replay UI.
- Kubernetes manifests.
- TTS confirmation.
- Physically separate node-local model runtimes or hardware-in-the-loop edge devices beyond the clearly labeled logical edge-agent simulation.
- Broader adversarial eval suite.
- MLflow experiment/evaluation tracking.
- Dagster incident-curation pipeline.
- Reusable MCP and evaluation templates.

### 20.3 Explicitly defer

- Full-duplex voice conversation.
- Production AWS deployment.
- Automatic Kubernetes scaling from custom metrics.
- Realistic marine physics.
- Physical radios, live satellite service, production anti-jam hardware, or claims of certified navigation performance.
- Multiple cooperating LLM agents.
- Model training or fine-tuning.

## 21. Acceptance Criteria

The interview build is acceptable when all of the following are true:

1. A first-time observer can explain the product after watching the first minute.
2. One voice or typed request produces valid typed intent and at least two map-visible plan candidates.
3. The recommended plan displays measurable reasons for its ranking.
4. No simulated vessel moves before the plan is allowed or approved.
5. An in-lease reallocation and an out-of-lease request visibly take different authorization paths.
6. Disconnecting the cloud routes an eligible new request to local inference and creates an audit event.
7. Stopping all inference does not stop the active mission or deterministic safety behavior.
8. The cutaway shows the selected live event through AI, planning, mission-control, and data-platform layers.
9. The load generator demonstrates at least 1,000 background simulated vessels and reports the actual measured event rate.
10. Killing an ingestion worker creates recoverable lag with no lost valid events and no duplicate stored logical events.
11. The system can explain the communications-loss incident using current state and recorded evidence.
12. A scripted verification command produces pass/fail evidence for the core safety and resilience claims.
13. Disabling the direct Starlink path causes an authenticated bundle to traverse a visible HaLow peer route and another node's Starlink egress without duplicate execution.
14. Completely isolating a vessel visibly drains its 60-second tape while it executes only validated lease-authorized segments.
15. Reconnecting after missed segments never replays stale work; it produces a policy-valid bridge from actual state to a future synchronization point and refills the tape.
16. A false GNSS jump is excluded from the fused position, changes the displayed integrity state, and causes uncertainty-driven deterministic degradation.
17. When an isolated vessel's tape reaches zero, it visibly leaves ordinary mission execution, seeks contact only inside its signed recovery corridor and safety budget, and either authenticates/refills or enters the configured safe termination state.
18. In Quiet Fleet, one vessel's degraded capability produces typed local proposals, an unsafe assignment is rejected by the affected vessel, a revised proposal commits only after quorum and all affected arms, and measured coordination traffic stays inside the configured demo budget.
19. On a prepared laptop or single VM, one launcher command starts the four required containers, publishes only the application port, reaches ready state, opens one application URL, and passes the scripted core verification without any live cloud, Kubernetes, or homelab dependency.
20. During Scale Lab, terminating one real Kafka consumer process reassigns its partitions to another process, creates measurable lag, and drains the backlog without valid-event loss or duplicate logical projection.
21. An autonomy engineer can select the Vessel 4 incident, use only scoped MCP evidence/replay tools, retrieve cited runbook/history context, review provenance, approve promotion to a versioned eval case, and run the regression with a traceable result.

## 22. Demo UX Copy

Preferred language:

- “What should the fleet do?”
- “Recommended because it preserves coverage and reserve.”
- “Nothing has been sent yet.”
- “This change leaves the approved area.”
- “Permission needed.”
- “Switched to offline AI. The mission is still running.”
- “We have not heard from Vessel 4 for 90 seconds.”
- “Vessel 4's satellite link is unavailable. Reached it through Vessel 3.”
- “Vessel 4 has 42 seconds of approved plan onboard.”
- “The link is degraded. Mission updates are prioritized over telemetry.”
- “Vessel 4 missed two segments. They will not be replayed.”
- “Rejoining the plan at the next safe point.”
- “Vessel 4's mission tape is empty. Continuing only its approved contingency.”
- “Mission tape empty. Seeking contact inside recovery corridor R2 for up to 90 seconds.”
- “Peer V3 assigned to rendezvous R2. Safety and reserve limits remain active.”
- “Recovery budget exhausted. Entering protected hold; discovery beacon remains active.”
- “Quiet Fleet is active. Four vessels are following the signed group mission.”
- “Vessel 4 cannot safely accept this assignment. Recomputing around its constraint.”
- “Group change armed by every affected vessel. Activating at mission tick 84.”
- “GNSS does not agree with onboard navigation. Position confidence is reduced.”
- “Position uncertainty reached the mission limit. Vessel 4 is entering its safe contingency.”
- “AI assistance is unavailable. Manual controls and the active mission are unaffected.”

Avoid exposing terms such as inference, schema, circuit breaker, idempotency, and consumer lag in Operator mode. Those terms belong in Cutaway mode.

## 23. Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Scope exceeds the interview deadline | Protect the P0 narrative; use deterministic fixtures behind unstable integrations |
| Container startup is slow or fragile | Pin images, add health checks, pre-pull dependencies, and keep a deterministic mock-provider profile |
| Local GPU passthrough varies by machine | Keep inference behind a provider adapter and verify a CPU/mock fallback before the interview |
| Browser receives excessive authority | Keep secrets and mutations behind authenticated Go APIs and policy enforcement |
| Local models are too slow | Use a small quantized model, short prompts, warm startup, and typed deterministic fallback |
| Voice fails in the interview environment | Preserve the identical typed path and preload one audio fixture for verification |
| Map becomes visually crowded | Render only active vessels individually; cluster or aggregate background load |
| Cutaway becomes another dashboard | Highlight one event path and preserve the map during the transition |
| Scale numbers appear fabricated | Bind every number to a real metrics endpoint and publish the benchmark command |
| LLM emits unsafe or malformed output | Registered actions, constrained schema, independent policy, no execution credentials |
| Provider failover duplicates work | Request IDs, deadlines, winning-response selection, plan hashes, and idempotent execution |
| Kubernetes work destabilizes the live demo | Use Docker Compose live; provide validated manifests as supporting evidence |
| The demo becomes a fragile collection of containers and external dashboards | Ship one modular Go core with embedded UI, one Python AI container, Kafka, and PostgreSQL; expose one port and render metrics/cutaway inside the product |
| The laptop loses internet or the homelab is unreachable | Bundle maps, fixtures, runbooks, seed data, deterministic provider fallback, and optionally OCI images/model files; require no live cloud or Proxmox dependency |
| HaLow is assumed to provide native mobile mesh behavior | Treat HaLow as the IP radio/MAC underlay and implement/test routing, egress advertisement, store-and-forward, and security above the hardware capability actually selected |
| Route churn creates loops or broadcast storms | Signed bundle IDs, hop limits, lifetimes, bounded replication, path hysteresis, deduplication, and queue quotas |
| A 60-second buffer becomes 60 seconds of stale blind motion | Store corridors, envelopes, preconditions, and contingencies rather than actuator commands; local sensing and safety may preempt every segment |
| A delayed link replays old movement after reconnection | Monotonic expiration, immutable segment lifecycle, execution high-water marks, and bridge-to-future rejoin; never catch up by executing faster |
| Signal seeking becomes unsafe wandering or all vessels chase one peer | Signed rendezvous assignments, deterministic relay roles, bounded recovery corridors, stable target selection, and hard time/distance/reserve/PNT budgets; collision avoidance always preempts recovery |
| Fleet revision partially activates during a partition | Future activation boundary, required-asset `armed` acknowledgements, signed commit, and transition-compatible old plan/contingency |
| Multiple LLMs agree on the same bad adaptation | Treat models as advisory and potentially correlated; require typed proposals, independent deterministic simulation/policy, affected-vessel arming, and hard safety preemption |
| Voting becomes chatty or scales all-to-all | Coordinate in small cells, schedule compact delta windows, cap bytes/rounds, use one epoch sequencer, and aggregate only cell-level state globally |
| A partitioned subgroup creates split-brain authority | Quorum is based on signed original membership; a subset cannot lower quorum or change membership without a new operator-signed contract |
| A compromised relay is mistaken for a trusted commander | Separate connectivity from authority; end-to-end signatures, encryption, monotonic authority epochs, and destination-side policy checks |
| Peer GNSS agreement creates false confidence during a correlated spoof | Treat peer position as one corroborating source; require dissimilar onboard or environment-relative evidence and show correlation warnings |
| Dead reckoning is presented as indefinite navigation | Display uncertainty growth and enforce a short lease-defined budget followed by a scenario-specific contingency |
| PNT/mesh work overwhelms the deadline | Implement deterministic digital twins and contract tests, not physical radios or production navigation algorithms |

## 24. Open Decisions

- Exact local LLM model based on measured quality and latency on the available GPU.
- Exact cloud provider adapter used for the failure demonstration.
- Whether P0 voice uses whisper.cpp directly or an existing local STT service.
- Whether TTS is worth including before historical retrieval and replay are complete.
- Exact offline MapLibre basemap/chart fixture and locally stored radar/shoreline features for the PNT scenario.
- Minimum measured background load that remains stable on the interview machine.
- Exact simulated link budgets and PNT thresholds; values must be labeled scenario assumptions, not hardware claims.
- Segment duration and batching inside the 60-second tape; start with six 10-second segments and measure planner/network behavior.

## 25. Product Success Statement

The demonstration succeeds when an interviewer sees a useful autonomy interface first, then discovers that every visible action is backed by typed planning, explicit authority, a rolling mission tape, an offline-first Starlink/HaLow peer fabric, resilient PNT, local inference, durable real-time infrastructure, measurable scale behavior, and complete observability.

The final takeaway should be:

> This is not an LLM driving boats, and it is not a cloud controlling boats. It is a resilient Go-based peer autonomy platform that keeps AI useful, authority bounded, and every node operational through lost connectivity and untrusted position data.
