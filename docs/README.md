# KeelMesh documentation

This is the maintained documentation portal for operators, engineers, reviewers, and platform owners.

## Choose your path

| Audience | Recommended path |
|---|---|
| Executive reviewer | [Project overview](../README.md) → [Delivery status](STATUS.md) → [Security](SECURITY.md) |
| Mission operator | [Operator guide](OPERATOR_GUIDE.md) → [Operations](OPERATIONS.md) |
| Application engineer | [Architecture](ARCHITECTURE.md) → [Reference](REFERENCE.md) → [Verification](VERIFICATION.md) |
| AI/ML engineer | [AI and memory](AI_AND_MEMORY.md) → [Security](SECURITY.md) → [Verification](VERIFICATION.md) |
| Platform/SRE owner | [Operations](OPERATIONS.md) → [Architecture](ARCHITECTURE.md) → [Verification](VERIFICATION.md) |

## Catalog

- [Operator guide](OPERATOR_GUIDE.md): workspace, missions, Fleet, AI, voice, and touch.
- [Architecture](ARCHITECTURE.md): topology, roles, data flow, authority, and failure behavior.
- [AI and memory](AI_AND_MEMORY.md): providers, MCP, A2UI, RAG, memory, and node caches.
- [Operations](OPERATIONS.md): secrets, deployment, health, reset, recovery, and evidence.
- [Security](SECURITY.md): trust boundaries, authorization, networks, credentials, and abuse resistance.
- [Verification](VERIFICATION.md): test layers, gates, evidence, and known coverage debt.
- [Reference](REFERENCE.md): APIs, service roles, ports, compatibility, and source layout.
- [Status](STATUS.md): delivered milestones, limitations, and roadmap.

## Source-of-truth hierarchy

1. Current source code and contracts.
2. Fresh verification output and evidence.
3. This documentation set.
4. Superseded plans retained in Git history.

The [PRD](../PRD.md) records product intent. Local agent handoff memory is intentionally excluded from version control; shared state belongs in this maintained documentation set.

## Documentation standards

- Label capabilities as implemented, simulated, optional, or planned.
- Tie performance claims to a workload and measured environment.
- Never include credentials, raw secrets, private keys, or runtime tokens.
- Update operator, architecture, security, and verification pages together when contracts change.
- Keep links relative for GitHub and offline clones.

