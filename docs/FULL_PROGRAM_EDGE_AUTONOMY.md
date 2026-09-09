# Full-program edge autonomy

## Contents

- [Authority model](#authority-model)
- [Program lifecycle](#program-lifecycle)
- [Group and local decisions](#group-and-local-decisions)
- [Failure behavior](#failure-behavior)
- [Interfaces](#interfaces)
- [Rollout and rollback](#rollout-and-rollback)
- [Verification](#verification)

## Authority model

M14 separates command authority from operational adaptation. An operator-approved plan becomes a canonical `TrajectoryProgramV2`, is quorum-committed by the applicable four-of-six Raft cell, and is durably installed on every node in that cell before full-program mode reports readiness. AI can interpret intent or suggest an adaptation, but it cannot sign, extend, or approve authority.

Each program binds the mission, plan, lease, revision set, complete ten-second trajectory, geographic and safety envelope, authorization-expiration tick, completion policy, and terminal contingency. The immutable content hash is verified before installation. Installation and bounded adaptations produce inspectable receipts.

## Program lifecycle

1. Deterministic planning creates complete per-vessel trajectories.
2. Exact-plan confirmation creates a signed lease and Raft command.
3. The full program is stored in each relevant node's BoltDB execution store.
4. Nodes activate the same revision at a shared future tick.
5. Execution cursors advance monotonically; completed path graphics are consumed.
6. A revision reconciles from fused current position rather than replaying missed movement.
7. Completion or expiry atomically transitions to the signed terminal behavior.

Finite routes normally hold at their final point. Looping routes require an explicit expiry and cannot renew themselves. A new expiry is a new exact, quorum-backed authorization.

## Group and local decisions

The lowest reachable decision-capable group member is selected deterministically. Its `decision_epoch` changes whenever leadership changes. It may coordinate only bounded collision avoidance, formation correction, schedule/energy/current compensation, contact-follow refresh, regrouping, and communication recovery.

Every receiving vessel independently validates the adaptation against its locally stored program. If group communication disappears, an isolated vessel applies the same validator with `decision_scope: local`; isolation does not widen its permissions. Model-generated replans remain advisory until converted into a typed adaptation and accepted by the deterministic validator.

## Failure behavior

| Condition | Behavior |
|---|---|
| Starlink lost, HaLow available | Group continues the complete approved program and shares bounded adaptations |
| Shore authority unavailable | Existing unexpired program remains valid; no new authority can be created |
| Vessel isolated | Local continuation inside the same envelope |
| Unsafe PNT, reserve, grounding, boundary, or separation | Signed contingency and instruction request |
| Authorization expiry | Atomic contingency; looping stops |
| Reconnection | Exchange revision, watermark, epoch, and fused pose; reject stale/conflicting state |

Kafka, PostgreSQL, memory, speech, AI, and VM 214 are not in the immediate onboard execution or safety loop. Their loss may reduce observability or planning but does not invalidate already committed work.

## Interfaces

Public read interfaces:

- `GET /api/v7/missions/{id}/program`
- `GET /api/v7/vessels/{id}/execution`
- `GET /api/v7/groups/{id}/decision-state`

Private mTLS node interfaces:

- `POST /internal/v1/execution/programs:install`
- `GET /internal/v1/execution/programs/{id}`
- `POST /internal/v1/execution/adaptations`
- `POST /internal/v1/execution/reconcile`

Legacy execution-buffer fields remain serialized as zero for one compatibility release. They do not grant authority and are absent from canonical v7 contracts.

## Rollout and rollback

`KEELMESH_EXECUTION_MODE` accepts:

- `tape`: bounded compatibility executor for emergency rollback.
- `full_program_shadow`: central execution remains authoritative while node installation differences are recorded.
- `full_program`: program installation and validation are required before readiness.

Rollout proceeds through local tests, two local six-node cells, VM 214 shadow comparison, one vessel at a time, Cell A, then Cell B. The old executor is removed only after live acceptance. No rollout step requires a Proxmox snapshot or network-topology change.

## Verification

The deterministic suite covers complete-program hashing and persistence, execution beyond one minute, loop expiry, revision activation, idempotent installation, signed receipt verification, deterministic decision-node selection, out-of-envelope rejection, isolated-vessel fallback, PNT contingency, and stale-safe reconciliation. Engineer and System expose program installation, remaining authority, decision scope, and terminal behavior from live state.

Use `scripts/keelmesh edge-autonomy-status` for a read-only compatibility check. During an explicitly controlled verification window, `scripts/keelmesh edge-autonomy-verify` creates one Cell A mission, proves all six nodes installed the same program, advances beyond one minute, checks decision state and deprecated zero-value fields, then resets Fleet state.
