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

## Performance and evidence

Performance reports include commit, image digest, seed/profile, VM hardware, counts, latency percentiles, lag peak, recovery, and resource use. Interview-load results are lab evidence, not production capacity claims.

Evidence exports are bounded JSON, Markdown, and checksum manifests. They exclude secrets, raw environment values, and raw voice audio.

## Current status

The frontend passes strict TypeScript, 13 Vitest assertions, production Vite/Docker builds, and focused deployed Playwright workflows for manual Mission editing and corrected phone gestures.

The broad legacy Playwright suite still contains assertions for the removed embedded Mission chat. The last full run recorded 13 passing and 11 failing tests. This is visible migration debt; superseded UI must not be restored to hide it.

GitHub-hosted workflows are intentionally unused. Verification runs on VM 214 and twelve vessel nodes to avoid hosted cost and measure the real environment.

