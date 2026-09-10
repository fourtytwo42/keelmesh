# Delivery status

## Contents

- [Implemented](#implemented)
- [Deployment](#deployment)
- [Limitations](#limitations)
- [Next investments](#next-investments)

## Implemented

| Milestone | Capability |
|---|---|
| M1 | Deterministic planning, preview, hash authorization, execution |
| M2 | Full-program continuation, relay/partition, PNT rejection, signed contingency/rejoin |
| M3 | Kafka/PostgreSQL scale path, workers, quarantine, replay, metrics |
| M4 | MCP investigation, retrieval, replay, approval, provider regression |
| M5 | Quiet Fleet rejection/quorum/arming/future commit; release commands |
| M6 | Map-first twelve-VM Fleet/Mission workspace, optional groups, formations, voice, touch |
| M7 | Arena vertical slice and twelve-VM node fabric |
| M8 | Long signed trajectory programs and future-boundary revisions |
| M9 | Read/draft-only external MCP control boundary |
| M10 | Trusted A2UI scenes, live bindings, assistant tools, critical scenes |
| M11 | Central/node memory, Kafka learning, replay, optional MLOps profile |
| M12 | Real two-cell Raft runtime, separate-role mTLS PKI, four-signature proofs, follower forwarding, cross-cell activation, and signed leader discovery |
| M13 | Cross-process OTLP tracing, SLO-backed recovery drills, measured capacity evidence, and a governed GNSS incident-to-evaluation flywheel |
| M14 | Complete finite onboard mission programs, explicit expiry, node-local durable install/validation, and group/local decision scope |
| M15 | Persistent fictional maritime combat, exact-hash engagement programs, deterministic effects, world-time repair, sinking, and respawn |

M13 unifies the four operational planes in Engineer and System: edge mission execution, quorum-backed coordination, Kafka/PostgreSQL data processing, and advisory AI/ML. The deployed evidence includes a real multi-service OTLP trace, rollback-protected leader and radio-partition drills, supervised worker recovery, immutable GNSS evaluation artifacts, and a measured 1,000-producer capacity run.

M14 is deployed in authoritative `full_program` execution mode on VM 214 and all twelve vessel nodes. A live quorum-backed proof installed one exact 3,663-segment program on all six Cell A nodes and continued to mission tick 108 without a rolling authority refill. The compatibility sweep passed twice; every legacy tape-depth projection is zero.

M15 adds the persistent Blackwake hostile and a fictional combat workload to Fleet Operations. The software implementation and local deterministic verification are complete; deployment evidence and the final all-node binary hash are recorded here only after rollout.

## Deployment

- VM 214 hosts the core appliance and LAN endpoint.
- Twelve vessel VMs run the same healthy Go/API/UI node binary and map one-to-one to the twelve persistent operating vessels.
- The default operating picture has zero groups and scatters all twelve vessels across shoreline-validated open-water positions.
- VM 214 probes each real node management health endpoint; simulated Starlink/HaLow and GNSS state is projected onto the matching Fleet vessel.
- PostgreSQL/pgvector, Kafka, workers, AI, and speech remain private.
- Builds and verification run on VM 214/nodes.
- M13's private OTLP collector, MinIO, Dagster, and MLflow services are deployed without publishing additional ports.
- M14 complete-program storage and validation are active on all twelve nodes; the identical node binary SHA-256 is `03f522dad5ca7e3430a268e5e805d892c35f87ea418cfc40bec9f5370df06879`.
- Latest measured scale run: 1,000 logical producers, approximately 2,020 events/second, zero dropped events, and matching replay checksums on VM 214.
- GitHub-hosted workflows remain disabled.
- No snapshot is authorized by documentation or deployment operations.

## Limitations

- Radio behavior is simulated; no physical HaLow radios are attached.
- M12 is authoritative in production Raft mode. The twelve voters are distributed 2/2/2 per cell across the three Proxmox hosts; live leader restart, 4/2 majority, 3/3 no-quorum, follower convergence, signed proof, and cross-cell activation drills pass.
- Radio-plane memory bundle exchange remains separate follow-up work.
- Node-local full-program execution currently provides durable installation and deterministic adaptation validation; physical propulsion and radio/GNSS inputs remain simulated.
- Separate GPU LLM/STT/TTS services do not run on every node.
- Kafka is single-broker and does not demonstrate broker HA.
- Map, environment, contacts, navigation, and effects are simulation-only.
- Some legacy Playwright tests still target removed Mission chat.
- Cloudflare Quick Tunnel URLs are ephemeral.
- The 100- and 1,000-asset capacity extrapolations and all cloud-cost rows are planning projections, not production benchmarks or vendor quotes.

## Next investments

1. Migrate legacy browser tests to global Assistant plus manual Mission semantics.
2. Complete radio-plane memory synchronization on the authenticated M12 transport.
3. Finish browser/node STT benchmarks and trusted-peer routing.
4. Replace remaining fixture retrieval with bundled ONNX indexing.
5. Add stable named HTTPS ingress after domain/account selection.
6. Validate the production-shape Kubernetes design in a disposable local cluster without replacing the stable interview appliance.
