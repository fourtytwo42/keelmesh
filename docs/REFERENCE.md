# API and repository reference

## Contents

- [Compatibility](#compatibility)
- [API families](#api-families)
- [Service roles](#service-roles)
- [Ports](#ports)
- [Mutation contract](#mutation-contract)
- [Repository map](#repository-map)
- [Design records](#design-records)

## Compatibility

KeelMesh preserves `/api/v1` through `/api/v6`. Public contracts are versioned, mirrored across languages where applicable, and exercised through fixtures.

## API families

| Version | Domain |
|---|---|
| `/api/v1` | Mission, resilience, scale, AI incidents/evals, Quiet Fleet |
| `/api/v2` | Fleet, groups, missions, geometry, plans, speech |
| `/api/v3` | Agent actions, factions, Arena, nodes, topology |
| `/api/v4` | Assistant turns, A2UI scenes, history, catalog |
| `/api/v5` | Memory, candidates, contexts, entities, sync, replay |
| `/api/v6` | Coordination cells, logs, quorum proofs, cross-cell state, security |

Prometheus metrics are at `/metrics`. WebSocket/SSE extensions preserve earlier fields.

`VesselTelemetryV2` includes reserve/projected reserve plus `solar_input_kw`, `power_draw_kw`, `net_power_kw`, and `energy_state`. `/api/v4/assistant/turns` assembles the latest twelve exact actor/session turns as ordered provider messages and retains a deterministic entity-reference fallback for inspector actions.

## Service roles

| Role | Runtime | Responsibility |
|---|---|---|
| `core` | Go | Public UI/API, authority, simulation, MCP/OTLP |
| `loadgen` | Go | Deterministic background telemetry |
| `worker-supervisor` | Go | Supervised Kafka consumer child |
| `memory-worker` | Go | Memory candidate/event projection |
| `migrate` | Go | PostgreSQL migrations |
| `topic-init` | Go | Kafka topics |
| `ai` | Python | Agent, providers, MCP client, retrieval/evals |
| `ai-index` | Python | Corpus/model initialization |
| `speech` | Python | VM-local STT/TTS |
| `player-b-ingress` | Go | Coordinator-aware faction ingress |

## Ports

| Port | Exposure | Purpose |
|---:|---|---|
| 8080 | Published | Core UI/public API |
| 8081 | Private | MCP and bounded trace intake |
| 8082 | Loopback | Player B ingress before tunnel |
| 8090 | Private | Python AI |
| 8091 | Private | Speech |
| 9092 | Private | Kafka |
| 5432 | Private | PostgreSQL |

Optional MinIO, MLflow, and Dagster endpoints remain private.

## Mutation contract

Mutations require request ID, idempotency key, actor identity where applicable, and the expected domain version. Consequential effects additionally bind an exact hash and approval identity. Stale versions fail rather than silently merging authority.

## Repository map

| Path | Ownership |
|---|---|
| `cmd/keelmesh-core` | Executable roles and embedded UI |
| `internal/fleetops` | Fleet, groups, contacts, missions, energy |
| `internal/resilience` | Tape, PNT, communication incidents |
| `internal/platform` | Kafka/PostgreSQL scale path |
| `internal/ai` | AI contracts, incidents, scenes, traces |
| `internal/memory` | Scoped memory and node stores |
| `web/src` | Workspace and trusted scene rendering |
| `web/e2e` | Browser workflows |
| `ai/keelmesh_ai` | Python agent/provider/MCP |
| `contracts/fixtures` | Compatibility fixtures |
| `scripts` | Verification and release operations |
| `infrastructure` | VM/node and network safeguards |
| `deploy/kubernetes` | Production-shape manifests only |

## Product records

- [Product requirements](../PRD.md)
- [Delivery status](STATUS.md)
- [Architecture](ARCHITECTURE.md)
- [Verification strategy](VERIFICATION.md)

Superseded implementation plans remain available through Git history. They are not current requirements or active operator documentation.

