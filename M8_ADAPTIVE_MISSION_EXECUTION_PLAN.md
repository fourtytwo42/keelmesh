# M8 — Adaptive Mission Execution and Conversational Planning

## Product outcome

KeelMesh missions become durable trajectory programs instead of fixed one-minute demonstrations. An operator can converse with the mission AI, compare bounded strategies, authorize an exact trajectory revision, and let each vessel execute and locally adapt inside signed limits. Groups can station-keep in configurable formations at persistent assembly points, including an explicit unassigned state.

## Authority model

- A **trajectory program** is the complete, arbitrarily long, signed mission revision.
- A **hot tape** is a rolling, bounded execution window materialized from that program. Sixty seconds remains the default disconnected-autonomy reserve, not the mission-length limit.
- Every segment contains timestamped position, speed, corridor, energy/reserve, separation, PNT, communications, and contingency envelopes plus predecessor and revision hashes.
- Mid-mission edits create a new immutable revision. They activate at a future segment boundary after every affected node validates and arms the exact hash. Completed history is never rewritten and stale segments never replay.
- The local deterministic controller may alter heading and speed inside the active segment envelope for collision avoidance, weather/current compensation, formation recovery, and energy management.
- A node LLM may diagnose a complex interruption and propose a typed replan. It receives bounded knowledge only. Deterministic navigation, policy, lease, reserve, collision, geography, and PNT checks must accept the proposal before it can be armed.
- Management networking and configured model-provider networking remain out of band during simulated radio faults.

## Delivery stages

### M8A — Workspace and group semantics

- Map click selects a vessel and reveals/scrolls Fleet / Groups; only the eye icon opens the vessel inspector.
- Context windows grow to their content up to the available viewport before scrolling.
- Remove the global bottom command box. Mission creation remains in the `+` mission control and Fleet / Groups selection workflow.
- Add the unassigned vessel section. Deleting a group atomically unassigns its members and releases its identity; active mission membership remains fail-closed.
- Add group formation, formation spacing, and assembly point to the group inspector.
- Use an existing same-color mission waypoint as a group's initial assembly point; otherwise use the first member's current water-safe position.
- Deleting the linked waypoint clears the assembly point. The next created same-color waypoint, or an explicit `Use first member`, re-establishes it.

### M8B — Long trajectory programs

- Add versioned `TrajectoryProgramV1`, `TrajectoryRevisionV1`, `TrajectorySegmentV2`, `ExecutionCursorV1`, `LocalAdjustmentV1`, and `ReplanProposalV1` contracts.
- Compile full plan assignments into timestamped ten-second segments for the complete computed route and duration.
- Materialize a configurable 60–300 second hot tape per node and refill it ahead of the execution cursor.
- Expose total program duration, hot-tape depth, execution cursor, revision, and refill status independently.
- Preserve the existing six-segment M2 fixture as a compatibility drill.

### M8C — Local adaptive control and formation keeping

- Add deterministic encounter tracks and closest-point-of-approach prediction.
- Resolve routine interruptions locally with bounded lateral corridor offsets and speed adjustments.
- Recompute arrival-error and projected reserve continuously; prefer schedule recovery only when reserve, speed, separation, and maneuver limits remain satisfied.
- On idle/mission completion, navigate groups to their assembly point and station-keep in the selected formation and spacing.
- Model station-keeping power, current/wind disturbance, solar charging, and day/night cycle in the same deterministic energy accounting.

### M8D — Node-agent replan escalation

- Add a typed node-agent tool to request a replan for interruptions that deterministic maneuvers cannot resolve.
- Include current fused pose, local contact tracks, environmental field, remaining program, constraints, and authority envelope; exclude secrets and hidden world truth.
- Require two to four bounded alternatives with machine-readable assumptions. Go computes all final paths and rejects arbitrary coordinates outside offered maneuver corridors.
- Auto-arm only changes demonstrably inside existing authority. Route/target/constraint expansion requires explicit operator approval of the new revision hash.

### M8E — Full conversational mission planner

- Add persistent per-mission conversations with operator, agent, tool, approval, and system messages.
- Render sanitized Markdown, autoscroll unless the user deliberately reads history, stream progress chips, support cancellation/barge-in, and retain durable tool receipts.
- Allow the agent to reference mission state, explain choices, request clarification, focus map layers, and insert state-backed map/route preview cards into chat.
- Compact planner: three collapsed strategy cards with recommendation, policy, and headline metrics.
- Expanded planner: viewport-safe two-pane layout with full chat left and detailed strategy comparison right; map/menu/mission/status bars remain unobscured.
- A compact plan selection never hides exact preview, authorization, lease, or revision state.

## Acceptance gates

- A route longer than one minute produces a complete immutable program and a rolling hot tape without a fixed six-segment ceiling.
- Mid-mission revision preserves completed history, activates only at a future boundary, and never teleports or replays stale work.
- Routine obstacle, current, schedule, and reserve changes remain inside the signed envelope; out-of-envelope cases stop or request a new exact authorization.
- LLM outage never blocks routine avoidance, safe hold, tape execution, or group station keeping.
- Group deletion leaves every former member visible under Unassigned and does not delete vessels.
- Group formation, spacing, and assembly point survive restart.
- A map click never opens an inspector; it selects and scrolls the matching Fleet / Groups row into view.
- Windows use available vertical space before introducing themed internal scrollbars.
- The global command dock is absent; the planner chat supports Markdown, progress/tool chips, route preview cards, and compact/expanded layouts.
- Existing M1–M7 compatibility, authority, resilience, infrastructure, AI, Arena, and node-health gates remain green.

