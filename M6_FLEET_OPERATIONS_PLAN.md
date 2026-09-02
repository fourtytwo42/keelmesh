# M6 — Fleet Operations Workspace

## Outcome

Turn KeelMesh from a six-vessel scenario into a persistent one-to-many operations workspace:

> Select vessels by click, search, group, or map area → inspect their reachable swarm and environment → create persistent operational groups → assign one or more groups to concurrent missions → describe the objective in plain English → compare formation and maneuver plans → change inherited constraints safely → preview and authorize the exact plan.

M6 preserves the M1–M5 authority, resilience, scale, and AI boundaries. Voice, chat, and direct controls all compile into the same typed command-draft and exact-plan authorization path.

## 0. System-wide redundancy doctrine

KeelMesh has no cloud-dependent control spine. Every operator appliance and vessel is an authenticated peer node with local compute, GPU-ready inference, policy enforcement, mission-tape execution, durable state, and at least one fleet radio. Starlink is the preferred wide-area path; Wi-Fi HaLow is the local peer underlay. KeelMesh implements authenticated routing, store-and-forward bundles, deduplication, authority checks, and reconciliation above the radios rather than assuming that HaLow alone provides a complete mobile mesh.

An operator controller within range of any authenticated HaLow peer may join the reachable mesh component and discover routes to every node that component can currently reach, subject to trust, policy, hop, bandwidth, and mission-authority limits. Joining the network grants connectivity, not command authority.

### Connectivity modes

1. **Fully connected:** Nodes use healthy direct Starlink paths for wide-area coordination, cloud services, fleet aggregation, and model/data maintenance. HaLow remains available for local traffic and redundant critical delivery.
2. **Single-node Starlink loss:** The node routes signed bundles over HaLow to an authorized peer advertising healthy Starlink egress.
3. **Fleet-wide Starlink loss:** The reachable HaLow component continues local command delivery, telemetry summaries, group coordination, browser/node speech, and node-local inference without internet or cloud services.
4. **Mesh partition:** Each connected component executes only prior authority available to its members. A partition cannot lower quorum, add members, extend geography, or mint a new lease.
5. **Isolated node:** The node executes validated unexpired mission tape, uses local perception/planning/inference within its lease, then follows the bounded communications-recovery or safe-contingency behavior.
6. **Reconnection:** Nodes exchange signed high-water marks, discard stale work, reconcile immutable events, recover a trustworthy state estimate, and bridge to future mission work without replay or position jump.

### Redundant layers

| Layer | Preferred path | Independent fallback |
| --- | --- | --- |
| Operator UI | Browser connected to preferred node | Another reachable node serving the same signed mission workspace |
| Speech | Browser WebGPU/WASM | Colocated node runtime, then one trusted mesh peer |
| Advisory LLM/RAG | Node-local GPU/model/cache | Trusted peer inference, then deterministic tools and typed operation |
| Mission planning | Connected planning service | Cached deterministic planner and policy runtime on each node |
| Command transport | Direct Starlink | HaLow multi-hop, peer Starlink egress, store-and-forward outbox |
| Mission execution | Fresh signed group plan | Sixty-second mission tape and bounded local contingency |
| Group adaptation | Connected operator coordination | Cell-local proposal, deterministic validation, quorum, all-affected arming, future commit |
| Navigation | Fused GNSS/inertial/radar/peer evidence | Uncertainty-aware dead reckoning, reduced envelope, safe hold |
| Data durability | Kafka/PostgreSQL platform plane | Append-only node journal and later idempotent reconciliation |
| Time | Signed mission activation plus monotonic clock | Local monotonic continuation; wall/GNSS time cannot revive work |

### Independent and group intelligence

- Each hardware node is designed to receive a GPU and an immutable, versioned open-source model bundle. CPU deterministic planning and safety remain available if the GPU or model fails.
- A node-local LLM may interpret local operator speech, summarize observations, retrieve cached runbooks, identify anomalies, and propose typed individual or group adaptations.
- Independent action is limited to immediate safety and behavior already inside the node's signed mission envelope.
- Coupled group decisions use the existing Quiet Fleet protocol: models propose; deterministic node planners validate; original-membership quorum and every affected node arm the exact hash; a signed commit activates at a future boundary.
- Model agreement is never treated as safety evidence because nodes may share the same model, prompt, data, or defect.
- Coordination stays cell-local and hierarchical. Large fleets do not run one all-to-all vote; cells publish compact status and accept scoped mission contracts from authorized controllers.
- Multiple controllers may be reachable, but authority has an explicit epoch, issuer, lease, and conflict rule. Network proximity or a better route cannot override the active authority chain.

### Traffic and resource priority

1. Collision, grounding, emergency stop, and PNT-integrity events.
2. Lease, mission tape, commit, acknowledgment, and clock evidence.
3. Link-state and bounded coordination deltas.
4. Operator command text and inference results.
5. Compressed audio only when local speech inference failed and peer processing is allowed.
6. Telemetry summaries.
7. Bulk telemetry, model artifacts, traces, and media.

Every queue is bounded. Lower-priority traffic is delayed, summarized, or dropped before it can starve safety or mission authority. Large model synchronization is a maintenance activity and never an automatic response to an operational partition.

## 1. Real East Coast operating picture

- Move the primary scenario to Narragansett Bay and Rhode Island Sound, with enough coastline, islands, harbor water, and open ocean to make zooming and grouping meaningful.
- Package a local low-zoom East Coast basemap and a local high-detail NOAA chart layer. Serve tiles from the Go appliance; the demo must not depend on an internet tile server.
- Prefer NOAA Chart Display Service MBTiles for the detailed maritime layer and a bounded PMTiles/vector basemap for regional context.
- Clearly label all mission, vessel, weather, current, and sensor data as simulation. The display is not represented as certified navigation software.
- Add a compact layer control for:
  - Basemap.
  - Nautical detail.
  - Mission geometry.
  - Environmental field.
  - Connectivity.
  - Vessel labels and trails.
- Initial extent: approximately `[-71.75, 40.85, -70.65, 41.90]`. Permit regional zoom-out without losing mission/group summaries.

## 2. Scalable map rendering and selection

- Replace per-vessel DOM markers with one GeoJSON source and MapLibre symbol/circle/text layers.
- Use three transparent top-down vessel sprites, rotated by heading. Render group identity as a color halo and group code, not by recoloring the hull image.
- Use feature state for `selected`, `hovered`, `alert`, `offline`, and `mission_active` without rebuilding the complete source.
- Cluster vessels at low zoom. A cluster shows vessel count, group composition, minimum reserve, and alert count.
- Selection methods:
  - Click selects one vessel.
  - Shift-click toggles additive selection.
  - Drag a selection rectangle to select every vessel inside it.
  - Double-click a vessel selects its complete operational group and zooms to its bounds.
  - Click a group chip or group row to select that group.
  - Search by human name, designation, group, class, mission, or status.
  - Table checkboxes support bulk selection and `Select all filtered`.
  - `Escape` clears selection; a visible selection breadcrumb allows removing individual targets.
- Preserve MapLibre pan/zoom conventions. Rectangle selection is an explicit toolbar mode or `Shift+drag`, so it does not make normal map navigation surprising.
- Keep UI selection local to the operator session. A command freezes the selected vessel IDs, group revisions, fleet version, and geometry references into an immutable target snapshot.

## 3. Persistent vessel identity, classes, and metrics

Use three fictional KeelMesh vessel classes:

| Class | Role | Demonstration envelope |
| --- | --- | --- |
| **Kestrel** | Agile scout and near-shore sensor | Lowest reserve capacity, highest maneuverability |
| **Mariner** | General-purpose mission vessel | Balanced endurance, sensing, and payload |
| **Atlas** | Endurance, relay, and support vessel | Highest reserve and comms capacity, widest turn radius |

- Give every vessel a stable UUID, designation, and human-readable callsign, for example `KM-214 · Gannet`.
- Generate callsigns deterministically from a curated maritime/nature dictionary, enforce uniqueness in PostgreSQL, and never regenerate a name after restart.
- Seed names such as Gannet, Aster, Brant, Fathom, Heron, Juniper, Mistral, Petrel, Quill, Sable, Tern, and Wake.
- The AI and audit log always use `Callsign (designation)` on first mention and may use the callsign thereafter.
- Per-vessel state includes:
  - Position, heading, speed, active mode, route, and mission.
  - Reserve, draw rate, projected reserve at completion, and health.
  - Wind speed/direction, current speed/direction, wave height/period, and water temperature.
  - PNT integrity and uncertainty.
  - Direct/relayed/unavailable links, selected egress, latency, loss, and tape depth.
  - Nearest-vessel and nearest-object separation.
- Environmental values come from a deterministic time-varying field sampled at each vessel position. A later optional adapter may refresh a recorded NOAA fixture, but the release remains deterministic and offline.

## 4. Swarms, operational groups, and reachability

Keep three concepts distinct:

- **Reachable swarm:** computed communications graph. It answers which peers this vessel can currently reach directly or through authenticated relays.
- **Operational group:** persistent, operator-created, exclusive primary membership used for color, formation, and group commands.
- **Mission task force:** immutable set of one or more groups plus optional individual vessels assigned to one mission revision.

A selected-vessel inspector shows reachable peers in four sections:

1. Direct peers.
2. Relayed peers, with hop path and estimated quality.
3. Same operational group but currently unreachable.
4. Reachable peers outside the group.

Never imply that network reachability grants command authority. Show connectivity and authority as separate badges.

Group operations:

- Create from current selection.
- Name, recolor, rename, dissolve, and change membership.
- Show class mix, member count, minimum reserve, worst link, formation, mission, and alert count.
- Validate group changes against active mission leases. Membership changes affecting an active mission require a preview and revised authorization.
- Use an accessible eight-color palette with a short group code and shape/pattern in addition to color.
- Support multiple selected groups and mixed targets. Deduplicate vessels before planning.

## 5. Concurrent mission workspaces

- Replace the single in-memory active mission with a mission registry persisted in PostgreSQL.
- Add tabs for active mission workspaces. `+ New mission` opens an untitled draft without disturbing running missions.
- Each tab contains its own:
  - Objective and status.
  - Assigned task force.
  - Included operating areas.
  - Exclusion areas.
  - Points of interest with `observe`, `stay_near`, or `avoid` behavior and radius.
  - Waypoints, routes, holding points, and corridors.
  - Constraints and formation policy.
  - Plans, authorization, execution, audit, and chat history.
- Tabs show compact status: draft, planning, approval required, active, degraded, paused, completed, or alert.
- Asset leases are exclusive by default. A vessel cannot receive conflicting motion authority from two missions. Conflicts produce a visible resolution sheet rather than silently stealing the vessel.
- Use separate `fleet_state_version`, `mission_state_version`, `group_revision`, and geometry revision. Telemetry updates do not invalidate an otherwise unchanged draft.
- Closing a tab closes only the view. Ending or deleting a mission is a separate explicit action.

## 6. Constraint inheritance and per-vessel overrides

Add a constraint matrix with four layers:

1. Immutable class/hardware envelope.
2. Fleet/operator defaults.
3. Operational-group policy.
4. Mission and vessel-specific overrides.

Initial typed constraints:

- Minimum reserve percentage.
- Maximum speed.
- Minimum vessel separation.
- Minimum obstacle/shore/exclusion separation.
- Maximum wave height and wind speed.
- Maximum PNT uncertainty.
- Maximum mission duration and route distance.
- Required communications/tape watermark.
- Allowed operating polygons and prohibited zones.
- Formation type, spacing, leader policy, and regroup threshold.

Safety merge rules are deterministic:

- Minimum constraints use the strictest maximum, e.g. reserve and separation.
- Maximum constraints use the strictest minimum, e.g. speed, wave height, and PNT uncertainty.
- Allowed geometry is intersected; prohibited geometry is unioned and inflated by uncertainty/separation.
- An override may make behavior safer without additional approval. Loosening an active authorized limit creates a plan diff and requires approval.
- Hardware limits and prohibited policy cannot be overridden by the AI or operator.

The inspector displays `effective value`, `source`, and `override`. A conflict explains the exact layers involved and offers safe resolutions.

## 7. Manual guidance, waypoints, formations, and maneuvers

Provide a `Guide` mode for one vessel, one group, or multiple groups:

- Set one waypoint.
- Draw an ordered multi-waypoint route.
- Set heading plus duration or distance.
- Hold position or orbit a point.
- Return to a named holding area.
- Regroup, split, merge, or rendezvous.

Group movement defaults to automatic formation maintenance. Initial formations:

- Column/trail.
- Line abreast.
- Wedge.
- Echelon left/right.
- Parallel columns.
- Dispersed screen.
- Ring/orbit.
- Search grid.

Formation controls include spacing, leader strategy, orientation, speed cap, and regroup threshold. Route generation offsets each member path from a formation reference trajectory, then independently checks shoreline, exclusion, separation, reserve, speed, PNT, and collision policy.

Manual guidance is not raw rudder/throttle control. It produces a typed, simulated trajectory plan with a preview, exact hash, lease check, and audit record. Local collision and grounding safety remain active.

## 8. Plain-English and AI planning

Voice, chat, suggestions, and typed commands produce `CommandDraftV2`:

- Frozen target vessel/group IDs and revisions.
- Referenced map geometry IDs.
- Objective and task type.
- Constraint changes.
- Formation/maneuver preference.
- Waypoints and timing.
- Ambiguities and required clarifications.

Example:

> “Send Kestrel Group and Atlas Group around Point Gull, keep 40 percent reserve, stay 150 meters apart, and give me an option optimized for coverage and one optimized for endurance.”

The LLM may resolve names, map references, and operator language into a schema. Deterministic services then validate targets and generate two to four feasible candidates. Candidate cards show:

- Formation and maneuver sequence.
- Map preview and timeline.
- Target groups and every affected vessel.
- Coverage, duration, reserve, distance, energy, link exposure, and minimum separation.
- Constraint conflicts and excluded assets.
- A state-backed recommendation with a concise reason.

The model never issues vessel commands or approves a plan. Authorization remains bound to the exact candidate hash and target snapshot.

## 9. Voice pipeline

TTS, STT, and local open-source LLM inference are node capabilities, not permanent centralized services. Every operator and vessel node uses the same portable inference-runtime contract and may eventually carry its own GPU and preloaded models. The browser is the closest execution tier on an operator device; its fallback is the colocated node runtime, followed by an explicitly trusted reachable mesh peer. VM 214 hosts the initial node runtime for the interview appliance. No path requires a cloud speech API. Inference loss must not affect mission execution or deterministic safety.

Serve the operator UI over HTTPS. Microphone capture and WebGPU require a secure browser context, so the current plain `http://192.168.50.214:8080` origin is insufficient for voice. During M6 development and the near-term interview demo, the existing free Cloudflare Quick Tunnel is the accepted HTTPS entry point; its generated hostname is explicitly temporary and must not be presented as a stable product URL. A named tunnel/domain is deferred. A later offline release must add a locally trusted hostname/certificate because the Quick Tunnel still depends on internet reachability.

### Distributed node inference fabric

- Package a portable node inference runtime with versioned STT, TTS, embedding, and local-LLM adapters. It can run on the operator appliance, a vessel computer, VM 214, or another authenticated edge node.
- Each node publishes a signed, short-lived capability advertisement containing supported tasks, model/runtime versions, acceleration class, available memory, queue depth, measured latency class, power mode, and whether it may accept remote work.
- Do not expose unrestricted hardware/process information or secrets in advertisements.
- Route each request local-first:

  `browser local → colocated node runtime → trusted reachable peer → deferred/typed fallback`

- The central/cloud provider may remain an optional additional route when policy and connectivity permit, but it is never necessary for an already provisioned node to interpret local speech, synthesize feedback, or run its bounded advisory agent.
- A node with cached models, current schemas/runbooks, and valid mission authority continues its local workflow without VM 214 or internet access. Mesh connectivity adds peer coordination, shared evidence, and optional inference failover; it does not create authority.
- Browser and node providers use one request ID and immutable input hash. Exactly one result is accepted. Late, duplicate, or conflicting results are discarded and audited.
- Peer inference requires authenticated transport, an allowed capability, bounded request size, deadline, hop limit, and explicit data classification. Raw audio is never broadcast.
- When local STT fails and peer fallback is allowed, send one encrypted, short-lived compressed audio stream to one selected peer. Return text immediately and retain no audio after the request completes.
- Select a peer using trust, task compatibility, measured latency, link cost, queue, power/reserve impact, and route stability with hysteresis. Mission-tape and safety traffic always outrank inference.
- TTS returns text plus optional compressed audio to the requesting human-interface node. Vessel nodes without a local speaker/microphone keep the capability dormant and do not generate continuous speech traffic.
- Local LLM output remains advisory. It may compile intent, explain state, retrieve cached runbooks, or propose a typed plan; it cannot sign leases, alter policy, authorize itself, or command actuators directly.
- Distribute model artifacts before missions through signed manifests, immutable hashes, resumable chunks, and storage quotas. Do not transfer multi-hundred-megabyte models over a degraded operational mesh unless an explicit maintenance policy permits it.
- The interview build may host several logical node providers on VM 214 for routing/failure demonstrations, but must label them simulated node isolation rather than claiming separate physical GPUs.

### TTS

- Reuse PPTtoVoice's pinned Pocket TTS 2.1.0 worker design and verified model revision.
- Import all twelve built-in voice assets with their existing provenance metadata.
- Make Morgan the default and expose the complete voice list in settings.
- Do not include voice training or cloning UI.
- Keep the worker warm, synthesize sentence-sized chunks, stream PCM/WAV audio to the browser, and support immediate cancel/barge-in.
- Preserve the applicable Pocket TTS and voice-source license/attribution files in the image and UI.
- Benchmark Pocket TTS first on VM 214 and later on each supported node hardware profile rather than relying on results from PPTtoVoice hardware. Record cold start, warm first-audio latency, total synthesis latency, real-time factor, CPU/GPU, and RSS for Morgan and at least two other voices.
- Initial TTS gate on VM 214: warm p95 first-audio below 300 ms and p95 real-time factor below 0.5 for short operator responses. If it misses, shorten speech chunks and prioritize first-audio latency before changing the voice engine.

### STT

Use one typed STT adapter with four measured execution paths:

1. **Browser WebGPU:** quantized Whisper/ONNX through Transformers.js in a dedicated Web Worker.
2. **Browser WASM:** sherpa-onnx streaming transducer/Zipformer with VAD and endpointing in a dedicated Web Worker.
3. **Colocated node runtime:** private local streaming to the node's selected CPU/GPU model.
4. **Trusted peer runtime:** policy-bounded encrypted streaming to one selected reachable peer when local tiers are unhealthy.

At startup, a capability probe checks secure context, microphone permission, WebGPU adapter/limits, WASM SIMD, memory, model cache, and a short local calibration clip. Route in this order only when the path passed its benchmark threshold:

`browser WebGPU → browser WASM → colocated node → trusted mesh peer → typed input`

Display the active path and measured latency, for example `STT · Browser WASM · 184 ms`. Failover occurs only between utterances; one utterance has one accepted transcript and shared request ID.

Serve versioned, checksum-verified browser models from the KeelMesh appliance and cache them locally through browser storage. Runtime inference must not download models from Hugging Face or another internet host. A model update uses a new immutable version and never silently replaces the active cached model.

Browser microphone audio is converted to 16 kHz mono PCM. Browser-local modes keep raw audio on the operator device. Node fallback sends bounded compressed frames only to the selected colocated or trusted peer provider. Every implementation emits the same partial text, final text, confidence, endpoint reason, timing, provider node, and route metadata. Maritime callsigns, formations, and place names are supplied as a bounded vocabulary/boosting list where supported.

Build the STT benchmark harness in parallel with M6A instead of waiting for the voice UI. Run browser candidates on the actual interview laptop/browser and VM candidates on VM 214. Use at least 100 command utterances plus quiet, fan, office, wind, and marine-noise fixtures. Run identical audio, chunking, VAD, warm-up conditions, and scoring against every candidate. Record:

- First partial latency.
- End-of-speech to final latency.
- Real-time factor.
- Command exact-match and slot accuracy.
- CPU, GPU, and memory.
- False endpoint and false activation rates.
- Model download/cache warm-up time and browser storage use.
- Main-thread responsiveness and dropped audio frames.

Publish machine-readable JSON and a concise Markdown comparison containing exact model/runtime versions, model hashes, thread count, VM or operator hardware, browser/version, execution backend, corpus hash, warm/cold state, and raw per-utterance timings. Select the lowest-latency path that satisfies the command-accuracy and UI-responsiveness gates, not simply the model with the lowest real-time factor.

Initial gates for both browser and node-runtime paths: p50 first partial below 250 ms, p95 endpoint-to-final below 700 ms, real-time factor below 0.25, at least 95% typed-command slot accuracy, and no map/input frame stall above 100 ms caused by browser inference. If no candidate meets every gate, choose the best measured path, show the actual latency in the UI/evidence, and continue typed/chat operation while optimizing STT. Do not claim the fastest engine until this harness produces measurements.

Run VM speech services with explicit CPU and memory limits. Run browser models only in workers with bounded audio/model memory. Keep STT and TTS queues bounded, allow only one active microphone session in the interview profile, cancel stale utterances, and ensure speech workloads cannot starve the map, Go mission authority, Kafka workers, or PostgreSQL.

## 10. Compact airline-operations visual system

- Replace the blue/teal visual identity with neutral graphite, warm off-white, signal amber, and restrained status colors.
- Target dense airline-dispatch ergonomics: 32–36 px controls, tabular numbers, compact rows, high information density, and strong hierarchy.
- Layout:
  - 36 px global status bar.
  - Compact mission-tab strip.
  - Collapsible left fleet/group rail.
  - Map as the permanent center surface.
  - Dockable right inspector/planner.
  - Thin bottom command/voice dock.
  - Minimized panels appear as labeled chips in a bottom shelf.
- Use one shared floating-window manager for every popup, inspector, planner, approval sheet, alert detail, chat, voice, and engineering panel.
- Every popup must be:
  - Movable by dragging a clearly visible title bar.
  - Minimized to a labeled chip in the bottom shelf.
  - Closable without changing or deleting the underlying mission state.
  - Restorable to its previous size and position.
  - Resizable when its content benefits from additional space.
  - Dockable or snappable to the left, right, or bottom work area.
- Clamp moved or restored windows to the visible viewport so a title bar and close control can never become unreachable after a resize or resolution change.
- Bring the active window to the front deterministically while preventing unbounded `z-index` growth. Reopening an existing singleton panel focuses it instead of creating a duplicate.
- Preserve independent window geometry per panel and operator. Reset layout returns to a compact safe default.
- Provide keyboard equivalents: focus-cycle through open windows, move/resize mode, minimize, close, restore, and reset layout. Dragging is never the only way to manage a window.
- Closing an alert popup dismisses only that presentation. An unresolved essential alert remains visible as a compact status-center item until acknowledged or resolved; it cannot be hidden by closing its floating window.
- Preserve keyboard navigation, visible focus, text-plus-color state, reduced motion, and 1280×720 usability.
- Save panel layout, density, selected voice, map layers, and last mission tabs per operator.

## 11. Contracts and APIs

Add versioned contracts mirrored in Go, Python, TypeScript, and fixtures:

- `VesselProfileV2`
- `VesselTelemetryV2`
- `EnvironmentalSampleV1`
- `SwarmReachabilityV1`
- `OperationalGroupV1`
- `GroupMembershipChangeV1`
- `SelectionTargetV1`
- `MissionWorkspaceV2`
- `MissionGeometryV1`
- `PointOfInterestV1`
- `ConstraintSetV2`
- `EffectiveConstraintV1`
- `FormationSpecV1`
- `GuidanceCommandV1`
- `CommandDraftV2`
- `FleetPlanCandidateV2`
- `VoiceCatalogV1`
- `TranscriptEventV1`
- `NodeInferenceCapabilityV1`
- `InferenceCapabilityAdvertisementV1`
- `InferenceRouteV1`
- `InferenceRequestV1`
- `InferenceAttemptV1`
- `WindowLayoutV1`

Initial APIs:

- `GET /api/v2/fleet`
- `GET /api/v2/vessels/{id}`
- `GET /api/v2/vessels/{id}/reachability`
- `GET|POST /api/v2/groups`
- `PATCH /api/v2/groups/{id}`
- `GET|POST /api/v2/missions`
- `GET|PATCH /api/v2/missions/{id}`
- `POST /api/v2/missions/{id}/geometry`
- `POST /api/v2/missions/{id}/commands:compile`
- `POST /api/v2/missions/{id}/plans`
- `POST /api/v2/missions/{id}/plans/{plan_id}:preview`
- `POST /api/v2/missions/{id}/plans/{plan_id}:authorize`
- `POST /api/v2/missions/{id}/plans/{plan_id}:start`
- `GET /api/v2/voices`
- `POST /api/v2/speech:synthesize`
- `GET /api/v2/speech/capabilities`
- `GET /api/v2/inference/routes`
- `POST /api/v2/inference/faults`
- `GET /api/v2/transcription/stream` (WebSocket)

All persistent mutations use request ID, idempotency key, and the relevant fleet/group/mission version. Existing M1–M5 v1 endpoints remain available for regression and the scripted interview drill.

## 12. Delivery slices

### M6A — Fleet picture and selection

- Local East Coast/NOAA map.
- 24 persistent named vessels across three classes and four colored groups.
- MapLibre symbol layers, clustering, click/shift-click/box/search/group selection.
- Vessel/group inspector, deterministic environment, and reachable-swarm list.
- Transparent sprites and compact non-blue shell.
- Cloudflare Quick Tunnel HTTPS for immediate microphone/WebGPU access, with the ephemeral URL shown by `scripts/keelmesh status`; named tunnel/domain and locally trusted offline HTTPS remain later hardening.
- Cross-runtime speech benchmark harness, Pocket TTS baseline, browser WebGPU/WASM results on the interview laptop, and CPU STT results on VM 214 run in parallel with the visual build.

### M6B — Groups, constraints, and multi-mission state

- Persistent group CRUD and membership validation.
- Mission tabs and PostgreSQL mission registry.
- Included/excluded areas, POIs, waypoints, and corridors.
- Constraint inheritance/editor and asset-conflict handling.

### M6C — Guidance and formation planner

- Single/group waypoint, heading, route, hold, orbit, and rendezvous guidance.
- Formation library and deterministic offset/collision planner.
- Multi-group target snapshot, candidate comparison, preview, exact authorization, and execution.

### M6D — Conversational operation

- Typed/chat `CommandDraftV2` first.
- Integrate the measured browser WebGPU/WASM winner, colocated VM 214 node runtime, signed capability advertisements, route selection, and one simulated trusted peer fallback behind the same STT adapter.
- Pocket TTS with Morgan default, all voices, cancel, and barge-in.
- Demonstrate loss of browser inference, loss of the colocated provider, trusted peer selection, deduplication of a late result, and recovery to local-first routing.
- AI formation/maneuver options using the same deterministic planning tools.

### M6E — Scale and release hardening

- Hundreds of visible vessels through clustering and filtered rendering while 1,000+ continue through M3 telemetry.
- Concurrent mission and group stress tests.
- Multi-layer fault matrix covering Starlink, HaLow routes, browser inference, node inference, GPU/model loss, operator-node loss, Kafka, PostgreSQL, and component reconnection without mission-authority coupling.
- Offline map/voice/STT proof, restart recovery, Playwright interaction matrix, evidence, and rehearsal.

## 13. Acceptance gates

- Selecting by click, additive click, box, double-click group, search, table, and group chip yields the exact same vessel IDs.
- A selected vessel's direct/relayed/unreachable list matches the M2 graph and never conflates reachability with authority.
- Names, designations, group membership, constraints, and missions survive appliance restart.
- Two missions can run concurrently with disjoint assets; conflicting assignment fails closed with a useful resolution.
- Per-vessel effective constraints exactly match deterministic inheritance and cannot exceed hardware/policy bounds.
- All manual and AI paths produce the same typed draft, policy checks, preview, exact hash, authorization, and audit lifecycle.
- Formation routes remain within included geometry, outside exclusions, within speed/reserve limits, and above separation minima.
- No vessel or route changes before authorization and the scheduled activation boundary.
- The map remains interactive at 1,000 visible features; low zoom clusters instead of rendering 1,000 labels.
- The application starts and performs map, selection, grouping, M1/M2/M5, TTS, and selected STT operations offline.
- Voice interruption stops playback immediately; partial/final transcript timing is visible and measured.
- The near-term demo uses the free Cloudflare Quick Tunnel; microphone and WebGPU feature checks pass without browser security overrides. Evidence labels the tunnel URL ephemeral and does not claim voice availability during internet loss.
- The later offline voice gate remains pending until a locally trusted HTTPS origin exists; a Quick Tunnel alone cannot satisfy it.
- Browser-local STT keeps audio on the operator device; fallback sends encrypted bounded audio only to one selected colocated or trusted peer node; outbound cloud speech access remains disabled.
- The chosen browser and node-runtime paths win reproducible same-corpus benchmarks on their actual hardware while satisfying command accuracy, or the release explicitly records which gate remains unmet.
- Disconnecting or degrading browser inference visibly fails over to the colocated node; losing that provider selects one eligible mesh peer between utterances without accepting duplicate transcripts.
- A node with preloaded models and cached policy/runbooks remains locally useful without VM 214 or internet access; mesh connectivity enables coordination and peer inference but never expands mission authority.
- Speech and LLM requests never broadcast, never preempt mission-tape/safety traffic, and never trigger operational model downloads over a degraded mesh.
- Signed capability advertisements expire, stale routes are rejected, and a late provider result cannot create a second command draft.
- Browser worker and VM speech resource limits preserve map responsiveness, M1 command/snapshot latency, and active mission execution under simultaneous TTS/STT load.
- A controller joining through one authenticated HaLow peer discovers every currently reachable authorized route in that mesh component while unreachable partitions remain explicitly unavailable.
- Fleet-wide Starlink loss preserves local HaLow command delivery, node-local planning/inference, authorized group adaptation, and cached mission execution.
- Loss of all links preserves only local authority; no isolated node or partition invents work, expands a lease, changes membership, or lowers quorum.
- GPU/model failure falls back to deterministic planning and safety without interrupting mission execution.
- Kafka, PostgreSQL, cloud, AI, speech, and operator-UI failures remain outside the hard real-time mission execution dependency chain.
- Every panel can move, resize where supported, minimize, restore, dock, and close without losing mission state.
- Reopening a singleton panel focuses the existing instance; windows remain reachable after viewport changes and never escape the screen bounds.
- Window layout survives reload, can be reset, has keyboard equivalents, and the full workflow remains usable at 1280×720.

## Scope guard

M6 is a robust simulation and autonomy-infrastructure demonstration. It does not claim certified navigation, COLREG compliance, physical radio performance, production command authority, or live operational weather accuracy. The real map provides geographic context; all vessels and environmental telemetry remain clearly fictional unless a recorded source fixture is explicitly identified.

## Research basis

- [NOAA ENC and free regional downloads](https://nauticalcharts.noaa.gov/charts/noaa-enc.html)
- [NOAA Chart Display Service and offline MBTiles](https://www.nauticalcharts.noaa.gov/data/gis-data-and-services.html)
- [NVIDIA Parakeet Unified streaming model](https://huggingface.co/nvidia/parakeet-unified-en-0.6b)
- [NVIDIA low-latency Parakeet EOU model](https://huggingface.co/nvidia/parakeet_realtime_eou_120m-v1)
- [NeMo-Speech.cpp local CPU/CUDA/Vulkan runtime](https://github.com/NVIDIA/NeMo-Speech.cpp)
- [Transformers.js browser WebGPU inference](https://huggingface.co/docs/transformers.js/en/guides/webgpu)
- [sherpa-onnx streaming ASR runtime](https://github.com/k2-fsa/sherpa-onnx)
- [whisper.cpp portable fallback](https://github.com/ggml-org/whisper.cpp)
- [Browser microphone secure-context requirement](https://developer.mozilla.org/en-US/docs/Web/API/MediaDevices/getUserMedia)
- Local PPTtoVoice source: Pocket TTS 2.1.0, pinned model revision, twelve voice assets, and Morgan provenance/integrity metadata.
