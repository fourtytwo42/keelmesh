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
| M2 | Tape incident, relay/partition, PNT rejection, safe hold/rejoin |
| M3 | Kafka/PostgreSQL scale path, workers, quarantine, replay, metrics |
| M4 | MCP investigation, retrieval, replay, approval, provider regression |
| M5 | Quiet Fleet rejection/quorum/arming/future commit; release commands |
| M6 | Map-first twelve-VM Fleet/Mission workspace, optional groups, formations, voice, touch |
| M7 | Arena vertical slice and twelve-VM node fabric |
| M8 | Long signed trajectory programs and rolling hot buffers |
| M9 | Read/draft-only external MCP control boundary |
| M10 | Trusted A2UI scenes, live bindings, assistant tools, critical scenes |
| M11 | Central/node memory, Kafka learning, replay, optional MLOps profile |
| M12 | Real two-cell Raft runtime, separate-role mTLS PKI, four-signature proofs, follower forwarding, cross-cell activation, and signed leader discovery |

Recent product hardening adds first-class multi-turn provider input for the shared voice/text assistant, deterministic follow-up resolution across controlled vessels, groups, and neutral contacts, and live battery-flow telemetry. Fleet reserve percentages now distinguish net charging from discharge, while each vessel inspector exposes solar input and net power.

## Deployment

- VM 214 hosts the core appliance and LAN endpoint.
- Twelve vessel VMs run the same healthy Go/API/UI node binary and map one-to-one to the twelve persistent operating vessels.
- The default operating picture has zero groups and scatters all twelve vessels across shoreline-validated open-water positions.
- VM 214 probes each real node management health endpoint; simulated Starlink/HaLow and GNSS state is projected onto the matching Fleet vessel.
- PostgreSQL/pgvector, Kafka, workers, AI, and speech remain private.
- Builds and verification run on VM 214/nodes.
- GitHub-hosted workflows remain disabled.
- No snapshot is authorized by documentation or deployment operations.

## Limitations

- Radio behavior is simulated; no physical HaLow radios are attached.
- M12 Raft and mTLS software passes the local twelve-process fault suite, and the twelve production voters are distributed 2/2/2 per cell across the three Proxmox hosts. Cell A Raft promotion is paused while a leader-restart epoch regression is validated; Cell B and VM 214 remain in shadow mode.
- Radio-plane memory bundle exchange remains separate follow-up work.
- Separate GPU LLM/STT/TTS services do not run on every node.
- Kafka is single-broker and does not demonstrate broker HA.
- Map, environment, contacts, navigation, and effects are simulation-only.
- Some legacy Playwright tests still target removed Mission chat.
- Cloudflare Quick Tunnel URLs are ephemeral.

## Next investments

1. Migrate legacy browser tests to global Assistant plus manual Mission semantics.
2. Complete the guarded M12 per-cell cutover after the leader-restart regression and live quorum drills pass.
3. Complete radio-plane memory synchronization on the authenticated M12 transport.
4. Finish browser/node STT benchmarks and trusted-peer routing.
5. Replace remaining fixture retrieval with bundled ONNX indexing.
6. Complete private OTLP ingestion and cross-process traces.
7. Add stable named HTTPS ingress after domain/account selection.
