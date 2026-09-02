# M2 — Resilient Autonomy Implementation Plan

Status: implementation-ready  
Depends on: completed M1 golden mission loop  
Primary incident vessel: Vessel 4  

## 1. Outcome

M2 extends the authorized M1 mission into one deterministic incident drill:

> Start with a healthy six-vessel mission → fail Vessel 4's Starlink → relay its
> signed traffic over HaLow through Vessel 3 → fully partition Vessel 4 → consume
> only its six validated tape segments → reject a spoofed GNSS jump while
> uncertainty grows → enter the lease-defined safe contingency → reconnect,
> discard stale work, and bridge from actual state to a safe future mission point.

This remains one offline `keelmesh-core` appliance. M2 does not require physical
radios, real GNSS receivers, Kafka, PostgreSQL, Python, an LLM, internet access,
or a production navigation claim.

## 2. Non-negotiable invariants

- Link availability never grants authority. Every destination validates the
  bundle, lease, immutable content hash, state version, and idempotency key.
- Relays route opaque signed content; they cannot alter plans or mint leases.
- A partitioned vessel may consume only validated, unexpired work already on its
  tape and only while its mission lease remains active.
- Segment timing uses mission-relative ticks mapped to a local monotonic clock.
  GNSS time and wall time cannot extend a lease or revive expired work.
- Duplicate delivery across Starlink and HaLow produces one logical receipt and
  one execution transition.
- A vessel never catches up by executing missed segments faster.
- Reconnection starts from actual fused state and execution high-water mark. It
  expires stale segments and targets a future synchronization point.
- PNT consumers receive an integrity-bearing fused estimate, never a bare GNSS
  coordinate. A suspect source cannot reset position or expand authority.
- Collision/geofence policy, PNT limits, and reserve limits always outrank
  communications recovery or mission progress.

## 3. Package boundaries

Refactor the M1 engine only as needed; preserve all M1 public behavior and tests.

```text
internal/clock/       Mission-relative monotonic clock and deterministic test clock
internal/edge/        Six logical node runtimes, local journals, deduplication
internal/mesh/        Link graph, egress advertisements, routing, bundles, receipts
internal/tape/        Segment construction, validation, lifecycle, watermarks
internal/pnt/         Observations, deterministic arbiter, uncertainty state machine
internal/rejoin/      Watermark reconciliation and bridge-to-future planning
internal/faults/      Allow-listed deterministic fault commands and incident script
internal/core/        Orchestration and M1-compatible aggregate snapshot
```

Logical vessels remain actor-like state machines inside one Go process. Do not
create one container or goroutine graph per simulated radio service.

## 4. Versioned contracts

Add canonical Go types, mirrored TypeScript types, JSON fixtures, and contract
tests for:

- `NodeSnapshotV1`: node identity, mission state, active lease, tape summary,
  active route, buffered bundle count, PNT state, and local event sequence.
- `LinkStateV1`: directed source/destination, underlay (`starlink` or `halow`),
  reachability, latency, loss, capacity, trust, and last transition.
- `EgressAdvertisementV1`: authenticated node, internet reachability, expiry,
  capacity class, and sequence.
- `PeerBundleV1`: immutable bundle/idempotency IDs, origin/destination, message
  class, priority, authority epoch, mission/plan references, lifetime, hop limit,
  payload schema/hash, and origin signature.
- `HopReceiptV1`: bundle ID, relay, ingress/egress links, observed monotonic tick,
  and forwarding result. Forwarding metadata stays outside signed content.
- `MissionTapeSegmentV1`: mission/lease/revision, sequence, activation and expiry
  ticks, predecessor hash, route corridor, speed envelope, expected state,
  reserve/PNT preconditions, failure behavior, content hash, and signature.
- `SegmentEventV1`: node, segment hash, lifecycle state, local sequence, actual
  state summary, monotonic tick, and reason code.
- `PntObservationV1`: source, observed pose/velocity/heading, source sequence,
  age, uncertainty, and integrity signals.
- `PntEstimateV1`: fused pose, velocity, heading, uncertainty, integrity state,
  contributing/excluded sources, source ages, reason codes, lease threshold, and
  selected deterministic behavior.
- `RejoinBridgeV1`: actual-state hash, execution watermark, discarded sequences,
  future target segment/tick, bridge route, policy decision, hash, and whether
  operator approval is required.
- `FaultCommandV1`: allow-listed kind, target, scenario tick, request/idempotency
  IDs, expected state version, and deterministic parameters.
- `ResilienceSnapshotV1`: incident phase, link graph, node/tape/PNT summaries,
  active bundle path, queued work, bridge state, and scenario assumptions.

All signed structures use immutable map-free representations and the M1
ephemeral HMAC authority. A later production shape may replace HMAC with node
keys without changing payload semantics.

## 5. Deterministic scenario and clock

Use a test-injectable mission clock that advances from the signed mission
activation. The production demo still ticks every 100 ms and publishes at 5 Hz,
but scenario time may run faster for a short interview demonstration.

M2 uses fixed, visibly labeled simulation assumptions rather than hardware
claims. Initial suggested thresholds:

| Assumption | Demo value |
|---|---:|
| Tape segment duration | 10 seconds |
| Initial lookahead | 6 segments / 60 seconds |
| Full / low / critical / empty | 45–60 / 15–44 / 1–14 / 0 seconds |
| Bundle hop limit | 3 |
| Egress-advertisement lifetime | 15 seconds |
| Route-switch hysteresis | 2 consecutive scoring intervals |
| Trusted / suspect uncertainty | ≤10 m / ≤25 m |
| Denied dead-reckoning budget | ≤45 m |
| Unsafe threshold | >45 m or lease-specific lower limit |

The exact values become scenario configuration and contract fixtures. They are
not presented as real HaLow range, maritime sensor accuracy, or certified limits.

## 6. Staged build

### Stage 1 — Clock, contracts, and M1-safe refactor

- Introduce the deterministic mission clock and explicit mission activation tick.
- Add the M2 contracts and shared Go/TypeScript fixtures.
- Extend bootstrap and WebSocket snapshots with an optional `resilience` object;
  keep every existing M1 field stable.
- Move audit and simulation responsibilities out of `core.Engine` only where the
  new state machines require it.
- Keep the M1 verifier and browser workflows green before continuing.

Gate: identical scenario seeds and commands produce byte-identical M2 fixtures;
all M1 tests and API behavior still pass.

### Stage 2 — Edge nodes and six-segment mission tape

- Create six node runtimes with isolated local state, deduplication sets, local
  append-only event journals, active lease reference, and execution watermark.
- Compile each authorized route into six immutable ten-second trajectory
  envelopes per vessel. Chain segments by predecessor hash and sign them.
- Validate signature, lease, plan revision, predecessor, activation/expiry,
  reserve, PNT integrity, corridor, and idempotency before arming.
- Drive vessel movement through active tape segments instead of directly through
  the complete M1 route while preserving the same visible route and mission result.
- Implement lifecycle states: `received`, `validated`, `armed`, `started`, then
  `completed`, `skipped`, `expired`, `rejected`, or `preempted`.
- Compute full/low/critical/empty watermarks from validated future work.

Gate: every vessel starts with six valid segments; corrupt, out-of-order,
expired, or lease-invalid segments fail closed; M1 movement still completes.

### Stage 3 — Starlink/HaLow routing and store-and-forward

- Model directed Starlink and HaLow links independently from mission authority.
- Advertise authenticated peer Starlink egress with expiry.
- Score eligible routes by reachability, trust, latency, loss, capacity, energy,
  and message priority with bounded hysteresis and a three-hop limit.
- Route critical tape/refill/receipt traffic before telemetry.
- Permit redundant critical delivery over direct and relayed paths; deduplicate
  at Vessel 4 by immutable bundle and idempotency IDs.
- Buffer destination-addressed bundles when no route exists and expose queue age,
  depth, and drop/expiry reasons.

Gate: disabling Vessel 4's direct Starlink visibly selects
`operator → Vessel 3 Starlink → Vessel 3 HaLow → Vessel 4`, produces hop receipts,
and causes exactly one destination transition.

### Stage 4 — Complete partition and safe tape depletion

- Add an allow-listed fault that disables every Starlink and HaLow edge incident
  to Vessel 4 without changing any other vessel's link state.
- Continue Vessel 4 only through validated onboard tape while other vessels
  continue normally.
- Stop tape refill, expose falling watermarks, suppress bulk telemetry at low,
  prepare contingency at critical, and refuse invented work at empty.
- Buffer local execution/PNT/audit events for later reconciliation.
- Enter a bounded safe hold at tape empty unless the existing lease authorizes a
  safe communications-recovery corridor and all PNT/reserve policies permit it.
  M2 defaults to hold; active signal-seeking motion remains a later enhancement.

Gate: a 30-second partition shows continued bounded execution; a partition beyond
60 seconds reaches empty and safe hold without lease expansion or stale motion.

### Stage 5 — GNSS spoof detection and uncertainty behavior

- Generate deterministic observations from GNSS, INS/motion, local
  radar/shoreline match, and authenticated relative peer evidence.
- Inject a 650 m northeast GNSS jump plus inconsistent velocity/clock evidence
  for Vessel 4.
- Use innovation and source-consistency gates to quarantine GNSS. Do not average
  the bad fix into the fused estimate.
- Transition `trusted → suspect → denied → unsafe` as corroboration disappears and
  dead-reckoning uncertainty grows.
- Inflate exclusion/geofence margins by current uncertainty; reduce speed and
  optional maneuver scope at suspect/denied; preempt mission motion at unsafe.
- Keep lease timing on the monotonic mission clock when GNSS time is invalid.

Gate: the spoofed raw marker jumps, the fused vessel marker does not, GNSS appears
in excluded sources with reason codes, and the exact configured uncertainty
threshold triggers the safe contingency.

### Stage 6 — Reconnection and bridge to future

- Restore one authenticated HaLow contact, exchange node state and execution
  high-water marks, and reconcile buffered immutable events.
- Reject queued bundles whose lifetime or referenced segment has expired.
- Mark missed segment sequences expired/skipped; never return them to armed state.
- Select the first future segment with enough lead time and a compatible plan
  revision. Build a route from Vessel 4's actual fused state to its future entry.
- Re-run boundary, exclusion, reserve, lease, speed, PNT, and collision-envelope
  policy on the bridge.
- Auto-activate only when the bridge stays inside the existing lease and current
  PNT state permits it. Otherwise show a diff and require operator authorization.
- Refill six future segments from the accepted synchronization point.

Gate: reconnection reports discarded sequences and one bridge hash, executes no
stale movement, restores tape depth, and rejoins without a position jump.

### Stage 7 — Operator presentation and scripted drill

- Add a compact **Resilience drill** stepper with one primary action at a time:
  `Fail Starlink`, `Partition Vessel 4`, `Inject GNSS spoof`, and `Restore contact`.
- Keep the M1 map primary. Overlay direct/relayed link paths, queued bundle motion,
  selected-vessel tape cells, and the bridge route.
- Add a Vessel 4 detail card containing:
  - current route: direct, relayed, or isolated;
  - six tape cells and lifecycle/watermark;
  - execution high-water mark and buffered-event count;
  - PNT integrity, uncertainty, contributing/excluded sources, and reason;
  - active behavior: mission, reduced speed, dead reckoning, safe hold, or rejoin.
- Render raw spoofed GNSS as a red ghost marker and fused position as the vessel
  marker with a growing uncertainty halo.
- Animate stale segments fading into `expired`; render bridge-to-future in a
  distinct color and show why it was auto-approved or requires approval.
- Add a collapsible explanation panel backed entirely by current simulator state
  and audit events—no hardcoded success narrative.

Gate: a new observer can understand the incident from the map and four-step drill
without opening logs, while the cutaway/infrastructure view remains deferred to M3.

## 7. Public API additions

- `GET /api/v1/resilience` — current link, tape, PNT, buffer, and bridge state.
- `POST /api/v1/faults` — apply one allow-listed fault with `request_id`,
  `expected_state_version`, and `idempotency_key`.
- `POST /api/v1/scenarios/resilient-edge:reset` — reset to a fresh authorized M2
  fixture; demo-only and explicitly labeled.
- `POST /api/v1/scenarios/resilient-edge:advance` — advance exactly one permitted
  scripted incident phase; useful for UI and deterministic Playwright tests.

Add WebSocket kinds:

- `link.route.changed`
- `bundle.lifecycle.changed`
- `tape.watermark.changed`
- `segment.lifecycle.changed`
- `pnt.integrity.changed`
- `contingency.changed`
- `rejoin.bridge.changed`

Stable errors:

- `INVALID_FAULT`
- `FAULT_CONFLICT`
- `ROUTE_UNAVAILABLE`
- `BUNDLE_EXPIRED`
- `SEGMENT_HASH_MISMATCH`
- `SEGMENT_EXPIRED`
- `PREDECESSOR_MISMATCH`
- `LEASE_INACTIVE`
- `PNT_UNSAFE`
- `BRIDGE_POLICY_REJECTED`
- `BRIDGE_REQUIRES_APPROVAL`

## 8. Scripted incident timeline

The UI advances phases manually so interview timing remains controllable. An
auto-run mode uses the same commands and powers CI.

| Phase | Visible result |
|---|---|
| Healthy | Six tapes full; Vessel 4 uses direct Starlink; PNT trusted |
| Direct failure | Signed refill bundle reroutes through Vessel 3 HaLow/Starlink |
| Partition | Vessel 4 becomes isolated; tape drains and local events queue |
| Spoof | Raw GNSS jumps; fused pose rejects it; uncertainty halo grows |
| Critical | Tape traffic priority and reduced-speed preparation are visible |
| Empty/unsafe | No new mission work; Vessel 4 enters signed safe hold |
| Contact restored | Watermarks exchange; stale segments expire visibly |
| Bridge | Policy-valid route targets a future segment; tape refills |
| Rejoined | Vessel 4 resumes at the future point; audit proves no stale replay |

## 9. Test and delivery gates

- Shared Go/TypeScript fixtures validate every new contract.
- Same seed and fault schedule produce identical routes, segment hashes, PNT
  transitions, bridge, watermarks, and audit ordering.
- Direct-link failure selects the expected HaLow peer-egress path after hysteresis.
- Redundant direct/relay delivery causes one logical destination transition.
- Relay payload mutation, excessive hops, stale advertisements, and bad signatures
  are rejected.
- Six segments provide exactly 60 seconds of initial validated lookahead.
- Segment lifecycle is monotonic; terminal states never return to active states.
- GNSS/wall-clock manipulation cannot extend lease or segment expiry.
- Complete partition continues only cached authorized work and reaches safe hold.
- Spoofed GNSS is excluded before fused position changes; uncertainty transitions
  and speed/contingency behavior occur at exact fixture thresholds.
- Reconnection discards expired segments, reconciles by high-water mark, and never
  executes stale work.
- Bridge route stays in bounds, avoids exclusion geometry, meets reserve/PNT
  constraints, and targets a genuinely future activation tick.
- All existing M1 Go, API, Vitest, and Playwright gates remain green.
- New Playwright coverage runs the complete resilience drill and asserts the
  visible route, tape, PNT, safe hold, stale expiration, and rejoin states.
- `scripts/verify_m2.py` runs the incident through the API and emits compact
  measured evidence: route changes, duplicate count, tape depth, expired segment
  count, maximum uncertainty, contingency tick, bridge target, and completion.
- Clean Compose deployment on VM 214 passes health, readiness, M1, and M2 flows.

No M2 Proxmox snapshot is created unless thin-pool data/metadata utilization,
free capacity, and VM 214's existing snapshots are inspected first and capacity
is safe. A snapshot still requires explicit user authorization.

## 10. Recommended execution order

1. Clock and contract fixtures.
2. Node runtime and tape-backed execution.
3. Mesh graph, bundle routing, receipts, and deduplication.
4. Partition and watermarks.
5. PNT arbiter and spoof fixture.
6. Contingency and bridge-to-future reconciliation.
7. UI drill, verifier, visual QA, Compose deployment, and CI.

Do not start with the fault buttons. The tape executor and integrity-bearing PNT
contracts must be real before their failure states are animated.

