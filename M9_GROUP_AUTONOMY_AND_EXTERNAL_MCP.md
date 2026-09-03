# M9 — Guarded Group Autonomy and External MCP

## Authority model

- A group elects the lexicographically lowest reachable `decision_capable` vessel ID. The result and epoch are visible and deterministic.
- The decision node coordinates advisory replans for the whole group. It does not gain lease-signing or policy authority.
- Every proposed adjustment is checked independently on every affected vessel against the signed mission envelope: geography, separation, speed, reserve, PNT, time, equipment, and communications limits.
- Inside-envelope collision, weather, energy, and schedule corrections may execute and are journaled with their decision node and scope.
- Outside-envelope conditions fail closed to the contract's pre-authorized contingency and emit `instruction_requested` with the exact violated constraints.
- A disconnected vessel uses its own inference route and deterministic planner inside cached authority. Signal seeking and return-home are executable only when their corridors and limits were included in the approved contract; otherwise the vessel holds safely and requests instructions.
- Radio faults never target the management/inference plane. Model availability is helpful but never required for safety enforcement.

## External agent boundary

The private Streamable HTTP MCP endpoint is `/mcp/control` on port `8081`. It uses a separate runtime-generated bearer identity and typed JSON schemas.

Available semantic capabilities include fleet/vessel/reachability reads, mission and trajectory reads, mission draft/compile/plan/preview, filtered Arena state, infrastructure inspection, and presentation-only workspace actions. All mutations use the same state-version and idempotency checks as the UI.

The external identity intentionally has no shell, filesystem, secret, arbitrary-network, mission-authorize, mission-start, or effect-application tool. `effect.request_approval` returns a durable human-approval pause carrying the exact proposal hash. Approval and execution continue through the canonical operator boundary, preventing an external model from granting itself authority.

## Next integration increments

1. Persist explicit GPU/inference capabilities and simulated reachability per node, then re-elect on membership or link changes.
2. Add signed group-adaptation proposals, all-affected arming, synchronized future activation, and execution receipts to normal M6 missions.
3. Add pre-authorized signal-seek and return-home corridors to mission contracts and exercise isolated-vessel contingencies.
4. Add scoped external MCP identities (`observer`, `operator-assistant`, `diagnostic`) with per-tool rate limits, expiring capability leases, audit export, and token rotation.
5. Expose the control MCP through an authenticated management-plane ingress when an external client is selected; do not place it on the public player URL by default.
6. Add protocol fuzzing, stale-version/idempotency tests, prompt-injection tests, hidden-knowledge tests, and approval pause/resume conformance tests.

