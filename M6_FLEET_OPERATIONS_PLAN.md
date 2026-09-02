# M6 — Fleet Operations Workspace

## Outcome

Turn KeelMesh from a six-vessel scenario into a persistent one-to-many operations workspace:

> Select vessels by click, search, group, or map area → inspect their reachable swarm and environment → create persistent operational groups → assign one or more groups to concurrent missions → describe the objective in plain English → compare formation and maneuver plans → change inherited constraints safely → preview and authorize the exact plan.

M6 preserves the M1–M5 authority, resilience, scale, and AI boundaries. Voice, chat, and direct controls all compile into the same typed command-draft and exact-plan authorization path.

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

Both TTS and STT run as private services on VM 214. Browser clients send microphone audio to the appliance and receive transcripts/audio from it; they do not run speech models locally and do not call cloud speech APIs. Speech loss must not affect mission execution or deterministic safety.

### TTS

- Reuse PPTtoVoice's pinned Pocket TTS 2.1.0 worker design and verified model revision.
- Import all twelve built-in voice assets with their existing provenance metadata.
- Make Morgan the default and expose the complete voice list in settings.
- Do not include voice training or cloning UI.
- Keep the worker warm, synthesize sentence-sized chunks, stream PCM/WAV audio to the browser, and support immediate cancel/barge-in.
- Preserve the applicable Pocket TTS and voice-source license/attribution files in the image and UI.
- Benchmark Pocket TTS on VM 214 during M6A rather than relying on results from PPTtoVoice hardware. Record cold start, warm first-audio latency, total synthesis latency, real-time factor, CPU, and RSS for Morgan and at least two other voices.
- Initial TTS gate on VM 214: warm p95 first-audio below 300 ms and p95 real-time factor below 0.5 for short operator responses. If it misses, shorten speech chunks and prioritize first-audio latency before changing the voice engine.

### STT

Use an adapter and benchmark on VM 214 before freezing a model. The current 8-vCPU VM is the required baseline; no GPU is assumed. A future passthrough GPU may be measured as an optional profile but cannot be required for the release:

- Primary CPU candidate: a sherpa-onnx streaming transducer/Zipformer model with endpointing.
- Native-runtime candidate: a quantized streaming model supported by NeMo-Speech.cpp on CPU.
- Compatibility candidates: quantized `whisper.cpp` `tiny.en` and `base.en` with VAD.
- Optional GPU candidates, only if VM 214 later receives GPU access: `nvidia/parakeet_realtime_eou_120m-v1` and `nvidia/parakeet-unified-en-0.6b`.

Browser microphone audio is converted to 16 kHz mono PCM and sent over a private WebSocket. The server emits partial text, final text, confidence, endpoint reason, and timing. Maritime callsigns, formations, and place names are supplied as a bounded vocabulary/boosting list where the selected engine supports it.

Build the STT benchmark harness in parallel with M6A instead of waiting for the voice UI. Use at least 100 command utterances plus quiet, fan, office, wind, and marine-noise fixtures. Run identical audio, chunking, VAD, thread limits, and warm-up conditions against every candidate. Record:

- First partial latency.
- End-of-speech to final latency.
- Real-time factor.
- Command exact-match and slot accuracy.
- CPU, GPU, and memory.
- False endpoint and false activation rates.

Publish machine-readable JSON and a concise Markdown comparison containing exact model/runtime versions, model hashes, thread count, VM hardware, corpus hash, warm/cold state, and raw per-utterance timings. Select the lowest-latency model that satisfies the command-accuracy gate, not simply the model with the lowest real-time factor.

Initial VM 214 gates: p50 first partial below 250 ms, p95 endpoint-to-final below 700 ms, real-time factor below 0.25, and at least 95% typed-command slot accuracy. If no candidate meets every gate, choose the best measured engine, show the actual latency in the UI/evidence, and continue typed/chat operation while optimizing STT. Do not claim the fastest engine until this harness produces measurements.

Run speech services with explicit CPU and memory limits. Keep STT and TTS queues bounded, allow only one active microphone session in the interview profile, cancel stale utterances, and ensure speech workloads cannot starve the Go mission authority, Kafka workers, or PostgreSQL.

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
- `GET /api/v2/transcription/stream` (WebSocket)

All persistent mutations use request ID, idempotency key, and the relevant fleet/group/mission version. Existing M1–M5 v1 endpoints remain available for regression and the scripted interview drill.

## 12. Delivery slices

### M6A — Fleet picture and selection

- Local East Coast/NOAA map.
- 24 persistent named vessels across three classes and four colored groups.
- MapLibre symbol layers, clustering, click/shift-click/box/search/group selection.
- Vessel/group inspector, deterministic environment, and reachable-swarm list.
- Transparent sprites and compact non-blue shell.
- VM 214 speech benchmark harness, Pocket TTS baseline, and first CPU STT candidate results run in parallel with the visual build.

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
- Integrate the VM 214 benchmark winner behind the streaming STT adapter and retain the next-best CPU engine as a packaged fallback.
- Pocket TTS with Morgan default, all voices, cancel, and barge-in.
- AI formation/maneuver options using the same deterministic planning tools.

### M6E — Scale and release hardening

- Hundreds of visible vessels through clustering and filtered rendering while 1,000+ continue through M3 telemetry.
- Concurrent mission and group stress tests.
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
- TTS and STT execute on VM 214 with outbound cloud speech access disabled; browser clients remain thin audio capture/playback clients.
- The selected STT engine wins a reproducible same-corpus benchmark on VM 214 while satisfying command accuracy, or the release explicitly records which latency/accuracy gate remains unmet.
- Speech CPU and memory limits preserve M1 command/snapshot latency and active mission execution under simultaneous TTS/STT load.
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
- [sherpa-onnx streaming ASR runtime](https://github.com/k2-fsa/sherpa-onnx)
- [whisper.cpp portable fallback](https://github.com/ggml-org/whisper.cpp)
- Local PPTtoVoice source: Pocket TTS 2.1.0, pinned model revision, twelve voice assets, and Morgan provenance/integrity metadata.
