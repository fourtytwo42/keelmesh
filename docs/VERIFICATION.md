# Verification strategy

## Contents

- [Quality model](#quality-model)
- [Test layers](#test-layers)
- [Scenario verifiers](#scenario-verifiers)
- [Resilience gates](#resilience-gates)
- [Performance and evidence](#performance-and-evidence)
- [Current status](#current-status)

## Quality model

KeelMesh verifies deterministic authority first, compatibility second, integrated failures third, and presentation last. UI success never substitutes for API/state evidence.

## Test layers

| Layer | Tools | Purpose |
|---|---|---|
| Go | `go test`, `go vet` | Mission, simulation, pipeline, memory, APIs |
| TypeScript | TypeScript, Vitest | Contracts, route projection, A2UI, helpers |
| Python | Ruff, mypy, pytest, Bandit | Agent, providers, MCP, schemas, security |
| Fixtures | Go/TypeScript/Python | Cross-language compatibility and hashes |
| Browser | Playwright | Mouse, touch, keyboard, responsive workflows |
| Appliance | Compose and verifiers | Services, faults, replay, metrics, evidence |
| Nodes | Service/API/hash checks | Binary parity, local persistence, network planes |
| Combat | Go determinism plus API/browser checks | Exact-hash authority, range/reload, damage, repair, respawn, replay, bounded rendering |

## Scenario verifiers

```bash
python3 scripts/verify_m1.py http://127.0.0.1:8080
python3 scripts/verify_m2.py http://127.0.0.1:8080
python3 scripts/verify_m3.py http://127.0.0.1:8080
python3 scripts/verify_m4.py http://127.0.0.1:8080
python3 scripts/verify_m5.py http://127.0.0.1:8080
python3 scripts/verify_m6.py http://127.0.0.1:8080
python3 scripts/verify_m7.py --base-url http://127.0.0.1:8080
python3 scripts/verify_m11.py http://127.0.0.1:8080
python3 scripts/verify_m12.py http://127.0.0.1:8080
python3 scripts/verify_m13.py http://127.0.0.1:8080
```

The consolidated command is `scripts/keelmesh verify`.

## Resilience gates

- AI and speech loss do not stop mission authority.
- Kafka outage spools bounded outboxes.
- PostgreSQL outage prevents offset commits and recovers idempotently.
- Worker termination causes real process loss and rebalance.
- Radio faults affect only simulated-radio paths.
- Spoofed GNSS cannot move the fused marker.
- Reconnection rejects stale work and bridges forward.
- Restarts report actual persistence boundaries.
- Controlled combat cannot begin without exact-hash approval. Automatic defense defaults off; attacked missionless vessels remain station-keeping and produce one actionable notification. Mission-level notify, retreat, or retaliation policy overrides vessel defaults. World-time repair, regeneration, disablement, and respawn remain deterministic from 1× through 500×.

## Performance and evidence

Performance reports include commit, image digest, seed/profile, VM hardware, counts, latency percentiles, lag peak, recovery, and resource use. Interview-load results are lab evidence, not production capacity claims.

Evidence exports are bounded JSON, Markdown, and checksum manifests. They exclude secrets, raw environment values, and raw voice audio.

## Current status

The frontend passes strict TypeScript, 18 Vitest assertions, production Vite/Docker builds, and the complete 34-scenario deployed Playwright matrix across combat, guided demonstration, Mission/Fleet workflows, A2UI scenes, desktop, tablet, phone, keyboard, mouse, and touch. Rapid Fleet-to-Mission membership changes additionally pass five consecutive end-to-end repetitions. M15 adds Go coverage for balance, deterministic effects, exact-hash approval, repair, regeneration, sinking, recovery, respawn, assistant/MCP boundaries, and restart-safe combat persistence. The post-M15 rules-of-engagement release additionally passes its deployed public combat suite, including Raft-routed arm/disarm, neutral-contact engagement planning, AI target resolution, and exact-hash rejection before effects.

The complete M1–M15 compatibility chain passes twice against the authoritative VM 214 deployment. Both runs include measured 1,000-producer Kafka processing with zero dropped events and matching replay checksums, twelve-node health, two converged six-voter Raft cells, zero legacy tape depth, and the live 45-entity combat projection. The current release was rechecked read-only against M12, M14, and M15 after sequential rollout; all twelve services are healthy on binary SHA-256 `5c62407b05f73c217e948603d6c23add0b7039fd0643b466ff3d8b4ba4924066`. Pursuit regression coverage proves that Blackwake selects a feasible route-crossing intercept, rejects a faster same-bearing escape, abandons infeasible live pursuits, and repairs degraded propulsion, sensors, and weapons before resuming. Speed-distribution coverage proves the 1.0–3.4 m/s NPC spread and the 4.1/3.8/3.6 m/s controlled-class overtaking margins. Energy regression coverage verifies that nighttime station keeping draws exactly 25% of the previous base load while underway calibration remains unchanged. Group-deletion coverage proves that former members stop, clear stale routes, and return to station keeping, while the runtime self-heals already-persisted orphan motion before energy accounting.

Coastline safety coverage forces both controlled-vessel and Blackwake movement toward an endpoint across Block Island. Every swept step remains in water, the navigator reaches the opposite side by routing around the rendered land polygon, and persisted land positions reconcile to safe water. The full Go suite and vet pass with the exact map fixture embedded in the runtime.

GitHub-hosted workflows are intentionally unused. Verification runs on VM 214 and twelve vessel nodes to avoid hosted cost and measure the real environment.

