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

KeelMesh preserves `/api/v1` through `/api/v5`. Public contracts are versioned, mirrored across languages where applicable, and exercised through fixtures.

## API families

| Version | Domain |
|---|---|
| `/api/v1` | Mission, resilience, scale, AI incidents/evals, Quiet Fleet |
| `/api/v2` | Fleet, groups, missions, geometry, plans, speech |
| `/api/v3` | Agent actions, factions, Arena, nodes, topology |
| `/api/v4` | Assistant turns, A2UI scenes, history, catalog |
| `/api/v5` | Memory, candidates, contexts, entities, sync, replay |

Prometheus metrics are at `/metrics`. WebSocket/SSE extensions preserve earlier fields.

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

## Design records

- [Product requirements](../PRD.md)
- [Implementation plan](../IMPLEMENTATION_PLAN.md)
- [M2 resilient autonomy](../M2_IMPLEMENTATION_PLAN.md)
- [M3 infrastructure at scale](../M3_IMPLEMENTATION_PLAN.md)
- [M6 fleet operations](../M6_FLEET_OPERATIONS_PLAN.md)
- [M8 adaptive execution](../M8_ADAPTIVE_MISSION_EXECUTION_PLAN.md)
- [M9 group autonomy and MCP](../M9_GROUP_AUTONOMY_AND_EXTERNAL_MCP.md)
- [M10 AI command surface](../M10_AI_COMMAND_SURFACE.md)
- [Role alignment audit](../ROLE_ALIGNMENT_AUDIT.md)

