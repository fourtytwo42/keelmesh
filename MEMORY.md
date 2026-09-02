# Project Memory

Durable project context lives here. Update this file whenever information should survive chat context compression or future agent handoff.

## Active Handoff

- Current task: M7 Symmetric Fleet Arena is deployed across twelve physical vessel VMs and VM 214; preserve M1-M6 while hardening the explicitly documented distributed-runtime follow-ups.
- Last meaningful change: The map now has distinct vessel and navigable-water context menus. Water right-click supports preview-safe go-to guidance, persistent numbered/color-coded waypoints, color-scoped planning, clearing one color or all waypoints, map centering, and fleet selection; right-clicking a waypoint deletes it directly. The deterministic compiler resolves requests such as “navigate through red waypoints” in sequence.
- Next step: rehearse Player B at the current Quick Tunnel `/?arena=1`. Later hardening should replace the central deterministic coordination model with real per-faction replicated consensus and add node mTLS plus independent node-local Python/speech runtimes.
- Blockers: the current Quick Tunnel hostname is ephemeral. No M7 snapshot is authorized or needed.

## Current State

- Project concept: an offline-first one-operator/many-vessel peer-autonomy demonstration for a Havoc AI infrastructure interview.
- Core scenario: the operator defines a simulated search mission through the map, suggestions, voice, or chat; the system previews and simulates a constrained plan before authorization.
- Core demo thesis: “Program the mission once. Coordinate locally. Adapt together. Degrade safely when the uplink disappears.” The mesh transports signed state and proposals; authenticated edge nodes make bounded decisions under the Group Mission Contract.
- `PRD.md` is the product source of truth. The M1 React/TypeScript/MapLibre interface and deterministic Go mission authority are embedded in one offline Compose appliance running on VM 214.
- M2 is live in the same appliance. It adds a mission-relative clock, six signed 10-second tape segments per vessel, authenticated/deduplicated peer bundles, deterministic Starlink/HaLow faults, PNT evidence arbitration, bounded safe hold, and stale-safe bridge rejoin.
- M3 is live in the same appliance as a degraded-optional scale plane: Apache Kafka KRaft, PostgreSQL/pgvector with eight telemetry hash partitions, deterministic load generation, three supervised real consumer children, bounded bbolt producer outbox, quarantine/redrive, actual Kafka lag, Prometheus metrics, earliest-offset shadow replay, pgvector fixture retrieval, and an Operator/Cutaway UI. Only port 8080 is exposed.
- M4 is live as a degraded-optional AI plane. A private Go Streamable HTTP MCP server exposes ten allow-listed evidence/replay/draft tools to a non-root Python agent; the Engineer UI shows actual tool receipts, citations, provider attempts, isolated replay, immutable candidate hash, human approval, provider regression results, and trace timing. OpenRouter uses a rotating ranked pool of free models, then optional local and deterministic mock fallbacks. Mission authority and M1–M3 remain independent of AI health.
- M5 is live. Quiet Fleet coordinates Vessels 2–5 under a signed fixed-membership contract: the unsafe first proposal reaches quorum but cannot commit after Vessel 2 rejects it; the safe revision arms all affected nodes and activates atomically at a future tape boundary. VM-local release commands cover startup, status, reset, verification, offline proof, restart recovery, evidence, rehearsal, and stop.
- M6 is live on VM 214. It serves a graphite MapLibre workspace with a packaged NOAA NCDS chart extract, 48 persistent named vessels in eight exclusive operational groups, overlapping saved collections, scalable symbol layers, fleet/group/search/box/map selection, vessel/environment/reachability inspection, movable/dockable windows, PostgreSQL-backed mission workspaces, deterministic formation candidates, and exact-plan authorization. Pocket TTS exposes all twelve voices with Morgan default; pinned faster-distil-whisper-small.en int8 provides VM-local HTTP/WebSocket transcription with typed fallback. Browser WebGPU/WASM and trusted-peer speech routing are accurately labeled future tiers, not completed redundancy.
- M7 is live as a deterministic Arena vertical slice plus a physical node fabric. VM 214 exposes neutral/referee state and coordinator-aware Player B ingress; twelve Ubuntu vessel VMs run the same Go/UI node binary with faction-pinned APIs across separate management (`192.168.50.0/24`) and simulated-radio (`10.77.0.0/24`) interfaces. The UI demonstrates knowledge filtering, protected planes, 20× time, coordinator failover, semantic workspace actions, and human-approved fictional engagement effects. Real distributed Raft replication, node mTLS enforcement, and twelve independent Python/TTS/STT runtimes remain follow-ups and must not be claimed complete.
- Visual system, last verified 2026-09-02: `lucide-react` 1.39.0 provides coherent UI icons. Navy mode shows a skull control to enter Pirate mode; Pirate mode shows an anchor control to return. The selected mode persists in local storage, changes workspace/Arena nomenclature and palette, and sends a typed persona to Morgan. Persona changes never grant additional tools or bypass exact-plan/effect approval. The default map uses the pinned Natural Earth 1:10m land dataset revision `ca96624a56bd078437bca8184e78163e5039ad19`, clipped locally by `scripts/generate_narragansett_map.py`; the incomplete NOAA raster extract is not rendered by default.
- `IMPLEMENTATION_PLAN.md` is the Friday delivery plan. It sequences the work as visible vertical slices with acceptance gates, a three-day schedule, contingency cuts, API/contracts, verification evidence, and a six-minute rehearsal.
- `ROLE_ALIGNMENT_AUDIT.md` is the coverage contract against the recruiter transcript and job posting. It requires both an Operator workflow and a scoped Autonomy Engineer incident-to-eval workflow.

## User Preferences

- Keep `MEMORY.md` updated consistently with durable context that should survive chat compression.
- Do not run builds, tests, or workflows on GitHub-hosted Actions because they may cost money. Run release verification on VM 214 instead; repository pushes must not trigger GitHub workflows.
- Agents should read `AGENTS.md` at startup and check `MEMORY.md` whenever context is missing, stale, or uncertain.
- Use uppercase filenames `AGENTS.md` and `MEMORY.md`.
- The user likes Rust/Tauri portable applications but explicitly decided not to force Rust or Tauri into this demo; prefer the stack most strongly supported by Havoc's disclosures.

## Source Of Truth

When sources conflict, use this order:

1. Current files in the workspace.
2. Latest command output or freshly verified tool output.
3. This `MEMORY.md` file.
4. Chat history.

## Freshness Rules

- Treat browser state, email state, authentication state, and external service results as stale unless verified in the current session.
- Treat documented repo structure, user preferences, and decisions as durable unless current files or fresh verification contradict them.
- Use `Last verified YYYY-MM-DD` for facts that can drift.

## Privacy Rules

- Do not store secrets, tokens, passwords, private keys, or sensitive personal data.
- Store only the minimum information needed for future handoff, especially when external systems or personal data are involved.

## Setup And Commands

- Dedicated environment, last verified 2026-09-01: Proxmox VMID `214`, name `keelmesh-demo`, Ubuntu Server 24.04.4 LTS, static `192.168.50.214/24`, gateway/DNS `192.168.50.1`, 8 vCPU, 16 GiB RAM with 8 GiB balloon floor, and 96 GiB disk.
- VM application root: `/srv/keelmesh`; LAN bootstrap URL: `http://192.168.50.214:8080`.
- Access uses the `keelmesh` account and a dedicated key-only SSH identity kept outside the repository. Password SSH authentication is disabled. Do not store the private key or any supplied host credentials in the repository or memory.
- Bootstrap host packages: Docker Engine 29.7.2, Compose 5.5.0, cloudflared 2026.8.3, GitHub CLI 2.45.0, and QEMU guest agent. These versions can drift.
- Re-run the idempotent host bootstrap with `sudo bash /srv/keelmesh/infrastructure/bootstrap-vm.sh` after the repository exists, or from the copied script under `/home/keelmesh/infrastructure` before Git setup.
- Start/build: `docker compose -f /srv/keelmesh/compose.yaml --project-directory /srv/keelmesh up -d --build`.
- Verify LAN health: `curl --fail http://127.0.0.1:8080/healthz` on the VM or `curl http://192.168.50.214:8080/healthz` on the LAN.
- The currently tested Cloudflare endpoint is an account-less Quick Tunnel and is ephemeral; replace it with a named tunnel before relying on a stable demo hostname.
- Player B Quick Tunnel, last verified 2026-09-02: `https://exhibits-prominent-quizzes-fold.trycloudflare.com/?arena=1`. It terminates at the loopback-only faction ingress on VM 214 and follows Player B's advertised coordinator.
- Player A/main Quick Tunnel, last verified 2026-09-02: `https://frequently-kde-bureau-ware.trycloudflare.com/`. It terminates at the main VM 214 workspace. Both Quick Tunnel hostnames are ephemeral.
- M0 recovery point: Proxmox snapshot `m0-keelmesh-baseline`, created 2026-09-01 after commit `d91201b` was pushed and `/healthz` passed.
- Private repository: `https://github.com/fourtytwo42/keelmesh`; default branch `main`.

## Project Map

- Project purpose: Demonstrate resilient AI infrastructure, bounded autonomy, human authorization, observability, and local/cloud model failover for a simulated maritime fleet.
- Main workflow: operator intent -> typed mission intent -> deterministic planner -> policy/mission-lease validation -> map simulation preview -> authorization -> simulated execution -> audit/replay.
- External systems: Planned local STT, local TTS, local LLM provider, and optional cloud LLM provider behind a provider gateway; exact integrations are not yet implemented.
- Important paths:
  - `AGENTS.md`: repository-local agent workflow and memory rules.
  - `MEMORY.md`: durable context store.
  - `PRD.md`: product requirements, stack alignment, demo narrative, priorities, and acceptance criteria.
  - `IMPLEMENTATION_PLAN.md`: implementation architecture, Friday scope contract, milestones, schedule, verification matrix, and rehearsal order.
  - `ROLE_ALIGNMENT_AUDIT.md`: transcript/posting traceability, scale proof contract, autonomy-tooling proof contract, live/supporting tool choices, and remaining gaps.
  - `M6_FLEET_OPERATIONS_PLAN.md`: post-M5 fleet-operations expansion for scalable selection, groups, concurrent missions, constraints, formations, real maps, and voice.
  - `README.md`: repository overview and bootstrap quick start.
  - `infrastructure/bootstrap-vm.sh`: idempotent Ubuntu host bootstrap for Docker, GitHub CLI, cloudflared, and base tooling.
  - `infrastructure/README.md`: non-secret VM specification and operations notes.
  - `cmd/keelmesh-core/`: multi-role Go entrypoint for core, migration, topic initialization, load generation, worker supervision, and consumer children.
  - `internal/platform/`: M3 Kafka/PostgreSQL pipeline, deterministic producer, workers, platform aggregation, replay, retrieval, and evidence APIs.
  - `M3_IMPLEMENTATION_PLAN.md`: M3 topology, drill, acceptance, and production-shape boundary.
  - `deploy/kubernetes/`: validated Kustomize base and AWS endpoint overlay; these are production-shape artifacts only.
  - `Dockerfile` and `compose.yaml`: one-image multi-role appliance with core, loadgen, three workers, Kafka, PostgreSQL/pgvector, and one-shot initializers.

## Architecture Notes

- The LLM may retrieve history, recommend options, rank suggestions, ask clarifying questions, and produce typed plan requests. It must not directly control simulated vessels or authorize its own proposal.
- The main UI is a map with a compact command dock: up to three contextual suggestions, hold-to-talk, typed command entry, and an expandable chat drawer.
- Suggestions are grounded in current fleet state, approved runbooks, and similar historical missions. Each suggestion maps to a typed action template and shows its reason and provenance.
- Selecting a suggestion opens a planner preview. The planner displays routes, assignments, sequence, timing, resource projections, constraints, and icons/text on the map before execution.
- A deterministic policy gateway classifies plans as within the active mission lease, approval-required, or prohibited. Approval is bound to the exact plan version/hash.
- The simulated mission and safety controller must continue if cloud inference, local inference, voice, or connectivity fails.
- Every operator and vessel instance is modeled as an authenticated peer node with local policy, signed mission-package cache, monotonic lease timer, local event log, store-and-forward routing, resilient PNT, and optional local inference.
- Starlink is the preferred internet/cloud path, but it is not the control spine. Every authorized node may advertise healthy Starlink egress to peers. Simulated Wi-Fi HaLow is the local radio/MAC underlay; routing, store-and-forward delivery, security, and peer egress are implemented above it rather than assuming every HaLow product provides native mobile mesh.
- Network reachability is separate from authority. Relays may forward signed/encrypted bundles but cannot mint leases, rewrite plans, or bypass destination-side policy.
- The communication overlay is BPv7-inspired: transport-independent signed bundles, local queues, store-and-forward delivery, priorities, lifetimes, hop limits, bounded replication, deduplication, and eventual event reconciliation.
- Each vessel carries a 60-second rolling mission tape, initially six immutable ten-second trajectory-envelope segments. Segments contain corridors, speed envelopes, preconditions, expiry, failure behavior, hash, and signature; they are not raw rudder/throttle instructions.
- Mission-tape traffic, acknowledgements, clock/lease data, and safety events outrank telemetry, voice, and bulk data. Critical segments may be sent over direct Starlink and HaLow-to-peer-egress paths simultaneously and deduplicated at the destination.
- Segment lifecycle is `received -> validated -> armed -> started -> completed|skipped|expired|rejected|preempted`. Plan revisions activate at a future boundary only for the signed asset set that acknowledged `armed` before commit.
- Reconnection never replays late segments or accelerates a vessel through missed work. The vessel reports actual state plus its execution high-watermark, expires missed segments, and follows a policy-validated bridge to a future synchronization point. Autonomous bridging is allowed only within the active mission lease.
- The tape uses a signed mission activation reference plus local monotonic time; GNSS or wall-clock time is not authoritative for segment execution.
- Every vessel carries a signed communications-recovery behavior graph separate from ordinary mission segments. It starts passive discovery before tape exhaustion and may seek contact only through pre-authorized rendezvous corridors with hard time, distance, reserve, PNT, traffic, and geofence budgets.
- Communications recovery is subordinate to collision/grounding avoidance, geofence compliance, PNT integrity, and reserve limits. It never uses naive RSSI hill climbing. If recovery becomes unsafe or exhausts its budget, the vessel enters the lease-defined safe termination state while keeping passive discovery active.
- Deterministic rendezvous assignments and peer relay roles prevent every isolated vessel from chasing the same moving peer. Contact must be authenticated before state exchange, tape refill, and bridge-to-future rejoin.
- Quiet Fleet is a low-radio-duty coordination mode for small signed cells, not radio silence or a stealth claim. It uses scheduled compact HaLow windows, a bounded byte/round budget, local bulk-data suppression, and an urgent-safety exception.
- Node-local LLMs are advisory and may be correlated. They produce typed group-adaptation candidates; deterministic local planners validate assignments. Coupled changes require a majority of the original eligible membership plus `armed` from every affected vessel and a signed future commit.
- Local collision/grounding responses need no vote. Individual changes inside a current envelope may execute locally. Mission-scope, operating-area, membership, quorum, or authority changes always require a new operator-signed Group Mission Contract.
- A partitioned subgroup cannot lower quorum or redefine membership. Without quorum, it follows only the prior compatible group tape and explicitly leased partition behavior, then falls back to individual tape, communications recovery, or safe termination.
- P0 may simulate per-vessel edge agents as node-isolated contexts multiplexed through one local inference server, clearly labeled `simulated edge inference`; physically separate vessel GPUs are not required for the interview demo.
- The live demo is packaged as one mission appliance, not distributed across AWS or Proxmox hosts. Required runtime containers are `keelmesh-core`, `keelmesh-ai`, one official Apache Kafka KRaft broker, and PostgreSQL/pgvector; an optional local-model profile may add a fifth container.
- `keelmesh-core` is a modular Go monolith containing the simulator, logical peer nodes, planner/policy, mission tapes, Quiet Fleet, workers, metrics, API/WebSocket, and a React production build embedded and served from the same binary. Logical vessels are actors with isolated state, not one container each.
- Only the core application port is exposed. The AI service, Kafka, and PostgreSQL remain on the private Compose network. Metrics and the live cutaway are rendered inside the product; Prometheus server, Grafana, MinIO, MLflow, Dagster, Kubernetes, AWS, and homelab services are not live-demo dependencies.
- The mission-appliance bundle includes Compose, pinned images/offline archive option, one config file, local maps/runbooks/fixtures, deterministic provider fallback, one data root, and launcher commands for start/stop/status/verify/reset/export. It can run on the laptop or one Proxmox VM with no dependency on the rest of the cluster.
- Each node's deterministic PNT arbiter outputs fused pose plus uncertainty, integrity state, contributing/excluded sources, and reason codes. It cross-checks GNSS against dissimilar evidence such as INS, radar/shoreline, speed, depth/bathymetry, vision, and authenticated relative observations.
- PNT states are `trusted`, `suspect`, `denied`, and `unsafe`. Action scope and speed shrink as uncertainty grows; exceeding the lease threshold enters a scenario-specific pre-authorized contingency. Dead reckoning is never presented as indefinite and anchoring is not assumed universally safe.
- Multi-constellation GNSS is useful cross-checking but not a fully independent backup because the RF systems share common failure modes. Peer-reported positions are corroborating evidence, not truth, especially under correlated spoofing.
- Local inference is the availability baseline; cloud inference is an optional quality accelerator. Provider retries and failover must not duplicate actions.
- The product should have separate Operator and Platform views. The Operator view stays focused on a small active mission; the Platform view exposes aggregate fleet load, service topology, queue lag, storage throughput, model-provider health, and failure recovery.
- Scale is demonstrated through a configurable synthetic fleet/load generator using the same ingestion path as the visible mission. Raw telemetry is not sent directly to an LLM; streaming/aggregation workers produce incidents and summaries that agents inspect on demand.
- Planned scalable data path: edge buffering -> durable event bus -> horizontally scalable ingestion/normalization -> hot operational store plus immutable object storage -> asynchronous aggregation/indexing -> agent tools/RAG.
- Scale claims must be supported by reproducible measured results: event throughput, end-to-end latency percentiles, consumer lag, dropped/duplicate/quarantined records, recovery time, and provider latency. Do not pre-claim unmeasured capacity.
- Verified stack alignment (last verified 2026-09-01): recruiter transcript explicitly names Linux, Go, AWS, heavy containerization/multi-zone deployment, React, TypeScript, WebGL geospatial mapping, Semantic Kernel-like usage, vector databases/RAG, Kubernetes, and Docker.
- The supplied job posting additionally names Python, C++, MCP, LangGraph/LangChain/LlamaIndex/Semantic Kernel, OpenAI/Anthropic/local models, Ray/Airflow/Dagster, MLflow/W&B, Kafka, Postgres, and S3-compatible storage as relevant or preferred examples; do not claim all are Havoc production dependencies.
- Chosen demo boundary: Go for simulator/control/data/high-throughput services; Python for agent orchestration, MCP client, RAG, evals, and ML workflows; React/TypeScript/WebGL for the browser UI. Rust/Tauri is intentionally excluded from the core demo.
- Chosen live-demo infrastructure: one Go core with embedded UI and in-product Prometheus-compatible metrics, one Python AI container, Kafka, PostgreSQL/pgvector, and Docker Compose. MinIO, standalone observability backends, and Kubernetes manifests are optional P1/Scale Lab or production-shape material, not required runtime dependencies. These are project choices sourced from the posting, not confirmed Havoc deployment details.
- Deployment profiles: Local Demo runs the complete stack in Docker Compose; Scale Lab uses the same images/contracts with additional workers and synthetic load; Production Shape maps the services to AWS/Kubernetes without claiming access to Havoc's private topology.
- Havoc's public architecture (last verified 2026-09-01) is described as four product layers: Havoc C2 for one-to-many operator control, Havoc Insights for real-time stream analytics, Havoc Connect for resilient peer-to-peer DDIL data exchange, and Havoc OS for edge autonomy. The recruiter transcript adds a Linux/Go core, AWS multi-zone containers, and React/TypeScript/WebGL UI.
- Working inference, not confirmed fact: their operational autonomy platform is more mature than their internal agent/ML infrastructure. The open role likely exists to standardize connectors, MCP, RAG, data curation, evaluations, observability, and AI-specific security around existing telemetry, simulation, data lake, code, and documentation systems.
- Do not claim Havoc uses Kafka, Postgres, MinIO, MLflow, or Dagster today. These are posting-aligned implementation choices or preferred examples.
- Superseded target-topology note: the earlier edge-to-AWS split was too linear. Current decision is an offline-first peer-node fabric in which edge nodes own execution, safety, PNT, local state, and delay-tolerant communication; AWS/Kubernetes is an optional capacity, coordination, analytics, and archival domain.

## Verification Ledger

- 2026-09-02: Numbered/color-coded waypoint and water-context behavior passed the full Go suite, strict TypeScript, seven Vitest assertions, production frontend build, and all eleven Playwright workflows on VM 214. VM 214 and all twelve vessel nodes run core binary SHA-256 `81a7ed4750243d84730a70d27d6078b5ae9b313ce390c7a5f375eb3a195147d7`. No snapshot or GitHub-hosted workflow ran.
- 2026-09-02: Provisioned and booted twelve M7 vessel VMs as 2 vCPU/2 GiB/12 GiB linked clones: A01-A05 on `fourtyfour` (220-224), A06 plus B01-B03 on `mini42` (225,229,231,232), and B04-B06 on `mini43` (233,234,236). Every VM has management `192.168.50.<VMID>` on `eth0`, radio `10.77.0.<VMID>` on `eth1`, active QEMU guest agent, faction-pinned node service, and separately installed root-only OpenRouter secret. All twelve management health checks and cross-host radio connectivity passed. No snapshots were created.
- 2026-09-02: M7 verification passed on VM 214: twelve-node referee topology, disjoint six-node player projections, protected-plane denial, deterministic B coordinator failover `node-b-01`/VM 229 to `node-b-02`/VM 231 at epoch 2, exact-hash engagement authority, and semantic workspace actions. The Player B ingress followed that failover while preserving one URL; both LAN and public Quick Tunnel Playwright Arena workflows passed without page errors.
- 2026-09-02: A clean VM-only `scripts/keelmesh verify` completed after the M7 integration: all Go packages, strict TypeScript, seven Vitest tests, frontend build, Python lint/type/security/tests, Compose rebuild, M1-M7 verifiers, and all six Playwright workflows passed. Its M3 run attempted 123,820 events at 2,033 events/s, dropped zero, peaked at 649 lag, and recovered Worker 2 in 16.73 seconds. No GitHub-hosted workflow ran.
- 2026-09-02: M7 implementation commit `ff76bdd` was pushed to `main` from VM 214. GitHub Actions remained disabled and no hosted workflow files existed. The Player B ingress port is bound only to VM 214 loopback; Cloudflare is the external entry. Repository and deployed services were clean/healthy after the push.
- 2026-09-02: The physical radio helper rejected an `eth0` target, successfully partitioned only B01 `eth1`, kept `http://192.168.50.229:8080/healthz` available, and restored cross-host `10.77.0.220 -> 10.77.0.229` reachability. The helper schedules a 60-second rollback before impairment. Actual post-start host free RAM was approximately 63.85 GiB (`fourtyfour`), 42.12 GiB (`mini42`), and 9.65 GiB (`mini43`); thin pools were approximately 77.57%, 76.25%, and 63.09% used respectively.
- 2026-09-02: M6 commit `da99565` was pushed to `main` without GitHub Actions. It is deployed as `keelmesh/core:m6` on VM 214 and passed all Go tests/vet, strict TypeScript, seven Vitest tests, production build, Compose validation, M1-M6 API verifiers, tracked-file secret scan, and five Playwright workflows. Browser coverage includes the 48-vessel map, search/group/filter selection, reachability inspector, group manager, geometry-to-plan-to-execution flow, movable/dockable windows, legacy M2/M4/M5 panels, and 1280×720, 1366×768, and 1440×900 viewports.
- 2026-09-02: Post-M6 M3 measured run attempted/accounted 121,598 events with zero drops, 2,227 events/s, 3.38 ms foreground API p95, 202 peak lag, real Worker 2 PID `15 -> 29`, 19.10 s recovery, and exact 1,000/1,000 live/shadow replay checksum parity. M4 collected eight scoped MCP receipts and four citations, visibly failed over from an OpenRouter timeout to an accepted provider response, matched deterministic replay, and passed the mandatory 11-assertion mock evaluation.
- 2026-09-02: M6 map extract contains 1,171 local NOAA NCDS PNG tiles (13,792,933 bytes) for bounds `[-71.62,41.08,-71.08,41.62]`, zooms 8-14; manifest SHA-256 is `c23c6954bceb4662ffc6086b392ccca2c50c2aa6230d236332f7c297fc9c0683`. The superseded tiny.en smoke result is retained only as history. Current VM-local STT is `Systran/faster-distil-whisper-small.en` pinned to revision `ef77d90526ccd62cde3808ee70626a01e5cf83e4`, 321 MiB staged, manifest content SHA-256 `6044df99861f162f9f82b123819538d85cbba59007954f97db9a8180893cb141`, and model.bin SHA-256 `1187de3982cdcf962a2fb8f797e429fb4651b875b18fe9ce50b58b52fc9072b7`.
- 2026-09-02: Three transparent north-up vessel sprites were generated with built-in ImageGen and stored at `web/public/assets/vessels/{kestrel,mariner,atlas}.png`. Group identity is rendered separately through color/pattern overlays. No Proxmox snapshot was created.
- 2026-09-02: M5 release rehearsals passed twice consecutively in both six-minute and full modes. Two post-commit evidence exports passed all M1–M5 verifiers; each bundle is 56 KiB and validates against its `SHA256SUMS` manifest. Final bundle: `/srv/keelmesh/evidence/release/20260902T045512Z`; manifest SHA-256 `febca6e13ec501d0af8e99314542fc9fb9e059cfa78e49e6e225ce1e71ba9b5c`.
- 2026-09-02: Release commit `07e1048` and annotated tag `v0.5.0-interview` were pushed. Repository Actions remained disabled, no workflow files existed, and the tracked-file secret scan was clean. No hosted GitHub build or workflow was run.
- 2026-09-01: M5 VM verification passed. Go tests/vet, strict TypeScript, seven Vitest tests, production build, Ruff, strict mypy, five pytest tests, Bandit, Compose build/validation, M1–M5 API verifiers, and seven Playwright workflows passed. Browser coverage includes Quiet Fleet and 1280×720, 1366×768, and 1440×900.
- 2026-09-01: M5 measured Quiet Fleet proof: proposal 1 had 3/4 armed and 3/3 quorum but commit failed `AFFECTED_NODE_NOT_ARMED`; proposal 2 armed 4/4 and activated at T+60. Three coordination rounds used 4,616 bytes total under the per-window 4,096-byte budget.
- 2026-09-01: Offline verification passed from cached images with `--pull never`. AI/Kafka/PostgreSQL/workers ran only on an internal backend network, core retained a separate LAN bridge, provider mode reported `offline`, cloud/local were disabled, and M1–M5 passed. Restart verification passed for AI during mission authority, Worker 2 under load, Kafka/PostgreSQL recovery, and honest core in-memory reset.
- 2026-09-01: Repeated-run M3 verification initially found a live/shadow mismatch because per-run sequence numbers were compared globally in `vessel_latest`. The update guard is now run-aware and ordered by produced time; subsequent live/shadow replay matched 1,000/1,000 vessels, dropped zero events, and retained healthy mission API latency.
- 2026-09-01: M4 verification passed on VM 214. `scripts/verify_m4.py` observed eight scoped MCP receipts, four validated citations, a matching deterministic replay checksum with `live_state_changed=false`, tampered-hash rejection, exact-hash approval by `demo-engineer`, mandatory mock regression, explicit real OpenRouter pass/fail/skip accounting, nine trace spans, unauthorized MCP denial, healthy core, and available M3 platform state.
- 2026-09-01: The M4 Playwright Engineer workflow passed end to end in 35.4 seconds. Go tests/vet passed across all packages; frontend strict typecheck, six Vitest tests, and production build passed; Python Ruff, strict mypy, five pytest tests, and Bandit passed; Compose validation and image builds passed. Stopping the Python AI container left M1 bootstrap, M2 resilience, M3 platform, and core health operational; AI reported degraded and recovered after restart.
- 2026-09-01: OpenRouter investigation routing visibly failed over across the ranked free pool. Provider regression was changed to make a separate schema-validated model call; a model may pass or fail individual assertions, or be explicitly skipped when no free model returns a complete result. No canned cloud success is displayed.
- 2026-09-01: No M4 Proxmox snapshot was created. The durable thin-pool inspection and explicit-authorization rule remains in force.
- 2026-09-01: The user prohibited GitHub-hosted builds/tests because of possible cost. The workflow file was removed and GitHub Actions was disabled at the `fourtytwo42/keelmesh` repository level; future release gates run on VM 214. The already-started M4 run had completed before cancellation reached GitHub.

- 2026-09-01: Pushed M3 implementation `8416ccb`, Cutaway reconnect fix `975584a`, and CI topology correction `b3c8f1f`. GitHub Actions run `33581609728` passed in 3m53s: the degraded core job passed M1/M2 tests and three browser flows without Kafka/PostgreSQL, while the full scale job passed Compose/Kustomize validation, the real M3 fault/recovery/replay verifier, and the Cutaway browser workflow against all scale services.
- 2026-09-01: M3 release evidence on VM 214 passed with seed `424242`: 1,000 vessels at 2 Hz, 123,416 attempted and exactly accounted events, 2,216 events/s measured baseline, 13.4 ms ingest p95, 228.6 ms p99, 4.32 ms foreground API p95, peak Kafka lag 202, Worker 2 PID `15 -> 29`, two-worker degraded operation, 20.6 s recovery, 978 duplicates suppressed, 242 out-of-order records, 485 quarantined deliveries, zero drops, successful repair/redrive, and 1,000/1,000 live/shadow projection parity with matching SHA-256 checksum. Evidence files: `evidence/m3.json` and `evidence/m3.md`.
- 2026-09-01: Latest Go tests and vet passed; frontend strict typecheck, Vitest, and production build passed; Playwright passed four workflows including the live Cutaway; M1 and M2 verifiers remained green; base and AWS Kustomize overlays each rendered four deployments. Prometheus `/metrics`, eight PostgreSQL hash partitions, pgvector nearest-fixture retrieval, Compose health, and private-only Kafka/PostgreSQL networking were inspected directly.
- 2026-09-01: No M3 Proxmox snapshot was created. The durable thin-pool safety rule remains in force.

- 2026-09-01: Completed and deployed M2 on VM 214. `scripts/verify_m2.py` passed with tape depths `60,60,30,0,60`, one deduplicated delivery, 52 m maximum uncertainty, safe contingency at mission tick 60, three discarded stale segments, bridge target 9, and final `rejoined` state.
- 2026-09-01: All Go package tests passed for monotonic clock, tape hashes/signatures/lifecycle/expiry, bundle immutability/hop limits/deduplication, mesh partition routing, PNT spoof exclusion, engine stale-state/idempotency behavior, and M1 regressions. Frontend typecheck, Vitest, and production build passed.
- 2026-09-01: Playwright passed three workflows against VM 214: suggested-area M1, hand-drawn-area M1, and full M2 relay → partition → spoof/safe-hold → bridge/rejoin. A null-vs-empty Go slice crash discovered during browser QA was fixed by normalizing wire arrays and adding defensive client checks.
- 2026-09-01: Pushed M2 commit `79a9cee`; GitHub Actions run `33576109969` passed in 3m04s, including frontend audit/typecheck/unit/build, Go test/vet, Compose/container build, M1+M2 API verifiers, and all three Playwright workflows.

- 2026-09-01: Planned M2 as a deterministic Vessel 4 incident layered onto an authorized M1 mission: direct Starlink failure, Vessel 3 HaLow relay, full partition, six-segment tape depletion, GNSS source exclusion, uncertainty-triggered safe hold, stale-segment rejection, and policy-checked bridge-to-future rejoin.

- 2026-09-01: Completed M1 on VM 214. Go tests/vet cover contract decoding, polygon validation, deterministic plan/hash output, segment-level exclusion avoidance, policy validity, preview immutability, stale-state rejection, tamper rejection, idempotent start, and authorized movement.
- 2026-09-01: Frontend dependency audit reported zero vulnerabilities; Vitest, strict TypeScript, and production Vite build passed. The production MapLibre 6 worker is bundled with Vite `?worker&url`, preventing the silent missing-worker failure seen during visual QA.
- 2026-09-01: `scripts/verify_m1.py` passed against the clean Compose deployment with two plans, 1,165 one-second samples, required audit events, preview stability, exact-hash rejection/authorization, idempotency, and vessel movement.
- 2026-09-01: Playwright passed both suggested-area and hand-drawn-area workflows against VM 214. Visual QA confirmed local search/exclusion/holding geometry, computed route overlays, six vessel markers in DOM, plan comparison, and reconnect bootstrap refresh.
- 2026-09-01: GitHub Actions run `33573287885` passed for M1 commit `e235500` in 2m31s, including npm audit/typecheck/unit/build, Go tests/vet, Compose validation, clean Docker build, API smoke, and two Playwright workflows.

- 2026-09-01: Queried Vurra's read-only local catalog and transcript for asset `8d9083fc944e`; verified the recruiter stack and scale statements at approximately 00:07:40-00:13:50 and 00:18:24.
- 2026-09-01: Read the supplied AI Infrastructure Engineer job posting and incorporated agent, MCP, RAG, ML workflow, eval, observability, security, documentation, and infrastructure requirements into `PRD.md`.
- 2026-09-01: Reviewed `PRD.md` for stale NATS/JetStream references after choosing Kafka; none remain.
- 2026-09-01: Verified Wi-Fi HaLow's sub-1-GHz PHY/MAC role with IEEE/Wireless Broadband Alliance material and delay-tolerant store-carry-forward concepts with IETF RFC 9171. The demo treats multi-hop routing and peer egress as an overlay, not a capability guaranteed by every HaLow product.
- 2026-09-01: Rendered and visually checked `resilient-peer-fleet.html` at 736px, 360px, and 320px; verified the GNSS-spoof interaction updates metrics, `aria-pressed`, narrative, and rejected-source visualization.
- 2026-09-01: Rendered and visually checked `mission-tape-mesh.html` at 736px, 360px, and 320px in light/dark modes. Verified the reconnect interaction shows two expired segments, one bridge segment, a future-plan executor, and no late replay.
- 2026-09-01: Verified cluster-wide VMID 214 was available and `192.168.50.214` did not answer repeated LAN ARP discovery before allocation. Provisioned Ubuntu Server 24.04.4 LTS from the official Noble cloud image and matched its SHA-256 checksum to Ubuntu's published checksum.
- 2026-09-01: Verified VM cloud-init completion, static address, 92 GiB expanded root filesystem, key-only SSH, disabled SSH password authentication, active Docker and QEMU guest-agent services, and `keelmesh` membership in the Docker group.
- 2026-09-01: Built and launched `keelmesh/core:bootstrap`; verified Docker health, LAN `/healthz`, and an outbound Cloudflare Quick Tunnel `/healthz` response.
- 2026-09-01: Completed GitHub CLI device authorization on VM 214 and verified active API and HTTPS Git access for account `fourtytwo42` before repository creation.
- 2026-09-01: Created private repository `fourtytwo42/keelmesh` from `/srv/keelmesh`, pushed root commit `d91201b` to `main`, verified remote visibility/default branch, and created Proxmox snapshot `m0-keelmesh-baseline`.
- 2026-09-01: GitHub Actions run `33569548528` passed Go test/vet, Compose validation, and container build for M0; action majors were then updated to the current official v7 releases and caching disabled until a `go.sum` exists.
- 2026-09-01: Clean GitHub Actions run `33569697081` passed all M0 checks with official `actions/checkout@v7` and `actions/setup-go@v7`. Verified key-only `keelmesh` login, passwordless sudo, Docker/GitHub access, then locked the superseded VM account, changed its shell to `nologin`, and moved its old authorized-key/sudo files to root-only rollback locations.
- 2026-07-12: Read `AGENTS.md` and `MEMORY.md`; identified stale project-specific memory and preserved the repository-local instruction logic.
- 2026-07-12: Checked the workspace root; confirmed `AGENTS.md` and `MEMORY.md` are present and no `.git` directory is present at this level.

## Decisions

### 2026-09-01 - OpenRouter Adaptive Free-Model Pool for M4

- Context: The user supplied an OpenRouter credential and requested a broad free-model failover set rather than a single cloud model.
- Decision: Use an explicitly ranked OpenRouter free-model pool with per-model health, rate-limit/error cooldown, sequential failover, and `openrouter/free` as the final cloud router. Preserve local and deterministic mock fallbacks after the cloud pool. Keep the key only in a root-readable VM runtime secret and never in Git, logs, evidence, or memory.
- Result: Live catalog inspection found multiple tool-capable free models. Probe calls confirmed MiniMax M3, Nemotron Nano Omni, and the free router answered, while some models were transiently rate-limited or harness-restricted; M4 therefore treats catalog membership and health as runtime state rather than a permanent availability claim.
- Follow-up: Rotate the pasted OpenRouter key after the interview/demo.

### 2026-09-01 - Name the Project KeelMesh

- Context: The initial working name was unrelated to Havoc but collided with existing software brands; the repository also should not use Havoc's company name.
- Decision: Use **KeelMesh** for the product and `keelmesh` for repository, module, service, image, VM, and application-path identifiers. “Keel” communicates stability and bounded safe behavior; “Mesh” communicates peer coordination and resilient infrastructure.
- Files: Repository-wide branding/module rename; VM 214 renamed `keelmesh-demo`; application root moved to `/srv/keelmesh`.
- Result: Exact-name web and GitHub searches found no obvious AI/autonomy software collision or existing `fourtytwo42/keelmesh` repository. The renamed Docker image built and `/healthz` returned `keelmesh-core`.
- Follow-up: Treat naming availability as a practical project check, not trademark clearance.

### 2026-09-01 - Add an Autonomy Engineering Data Flywheel and Real Scale Proof

- Context: Auditing the Friday plan against the recruiter transcript and posting showed that the C2/autonomy product path was strong, but the actual role centers on internal AI infrastructure, ETL, autonomy tooling, MCP, RAG, evaluation, curation, observability, security, and infrastructure at scale.
- Decision: Keep the Operator mission demo and add a short Autonomy Engineer workflow that investigates the Vessel 4 incident through scoped MCP tools, retrieves cited runbook/history evidence through pgvector, replays the deterministic seed, requires human approval for eval promotion, and runs the versioned regression. Promote real multi-process Kafka consumer rebalance and end-to-end OpenTelemetry tracing to P0.
- Files: `PRD.md`, `IMPLEMENTATION_PLAN.md`, `ROLE_ALIGNMENT_AUDIT.md`.
- Result: Every major transcript/posting requirement is now assigned to live proof, measured evidence, or an explicit production-shape artifact; overlapping named frameworks remain intentional non-choices rather than checklist dependencies.
- Follow-up: Implement the golden mission loop first, then preserve the same Vessel 4 incident as the artifact that flows through telemetry, retrieval, replay, curation, and eval. Do not claim throughput until the VM benchmark is measured.

### 2026-09-01 - Vertical-Slice Friday Implementation Plan

- Context: The PRD's P0 contains many coupled product and infrastructure goals with only three build days before the interview.
- Decision: Implement in five gated slices: golden mission loop, resilient edge path, durable platform/cutaway, bounded AI/provider failover, then Quiet Fleet/release hardening. Every slice must leave a visible working product; deterministic Go behavior and fixtures precede optional AI/voice integrations.
- Files: `IMPLEMENTATION_PLAN.md`.
- Result: The build now has concrete package boundaries, contracts, APIs, scenario states, acceptance gates, a Tuesday-through-Friday schedule, contingency cuts, and an automated evidence matrix.
- Follow-up: Resolve the independent project name, push the baseline, and begin M1. Do not start Kafka, voice, or model integration before the golden mission loop works end to end.

### 2026-09-01 - Offline-First Peer Fabric and Resilient PNT

- Context: The prior architecture still made internet/cloud ingestion look like a linear control spine with many serial failure points. The user proposed giving every vessel and operator device multiple radios, local GPU/compute, and peer relay capability.
- Decision: Model every operator and vessel as an authenticated offline-first node. Use satellite, Wi-Fi HaLow, and development LAN as communication underlays beneath a signed store-and-forward overlay. AWS is an optional peer domain. Add a deterministic PNT arbiter and uncertainty-driven degradation for GNSS spoofing/denial.
- Files: `PRD.md` Draft 0.2 and `C:\Users\hendo420\.codex\visualizations\2026\09\01\01a05e24-30af-7ee1-8641-a026835396e6\resilient-peer-fleet.html`.
- Result: The demo story now shows satellite-to-peer route failover, complete partition with cached-lease execution, post-contact reconciliation, spoofed-GNSS rejection, and safe degradation without cloud or LLM dependence.
- Follow-up: Implement the deterministic Go link/PNT digital twin first. Do not attempt physical radios, anti-jam hardware, certified navigation, or a complete BPv7 stack before the interview.

### 2026-09-01 - Starlink/HaLow with a Rolling Mission Tape

- Context: The user chose to remove LoRa from the target design and asked for a CD anti-skip-like mechanism that lets a vessel remain coordinated through short Starlink/HaLow interruptions.
- Decision: Use Starlink as preferred internet egress and HaLow as the peer underlay. Add a 60-second rolling mission tape of signed, expiring, policy-validated trajectory envelopes; direct and relayed paths may carry duplicate critical segments with destination deduplication. A disconnected vessel executes only validated onboard work, then enters its lease-defined contingency.
- Files: `PRD.md` Draft 0.3 and `C:\Users\hendo420\.codex\visualizations\2026\09\01\01a05e24-30af-7ee1-8641-a026835396e6\mission-tape-mesh.html`.
- Result: The demo sequence is now normal direct Starlink, local Starlink loss with HaLow peer egress, degraded HaLow with priority refill, a 30-second partition that drains the tape, a 70-second partition that enters contingency, and reconnect with expiration plus a deterministic future rejoin.
- Follow-up: Implement six ten-second segments first and make tape depth, segment lifecycle, active path, execution high-watermark, and bridge rejoin visible. Treat the older `resilient-peer-fleet.html` radio topology as superseded, while retaining its resilient-PNT concept.

### 2026-09-01 - Bounded Communications Recovery

- Context: The user proposed a preprogrammed tape-empty behavior in which an isolated vessel seeks a usable HaLow mesh path.
- Decision: Add an always-resident signed recovery graph with passive discovery, tape-critical preparation, bounded seek-contact behavior, deterministic peer-assisted rendezvous, authenticated refill, and safe termination. Signal acquisition never outranks collision avoidance, water/geofence constraints, PNT integrity, or reserve limits.
- Files: `PRD.md` Draft 0.4, section 9.5, FR-38, acceptance criterion 17, UX copy, and risk controls.
- Result: Tape exhaustion is no longer a behavioral cliff. The vessel can intentionally recover connectivity without replaying mission work, chasing noisy RSSI, exceeding authority, or wandering indefinitely.
- Follow-up: Add this state machine to the link/mission-tape digital twin and expose recovery corridor, target peer/rendezvous, remaining recovery budget, and terminal safe state in the map and cutaway.

### 2026-09-01 - Quiet Fleet Cooperative Adaptation

- Context: The user proposed placing a group into a low-radio-noise HaLow mode where each vessel follows a preset mission, uses an onboard LLM to adapt, and votes on individual or whole-group compensation.
- Decision: Add Quiet Fleet for cells of three to eight vessels. Use node-local LLMs only to create typed candidates; deterministic planners produce signed `armed|reject|abstain` decisions. Coupled changes require original-membership quorum, every affected vessel armed, and future commit. Local safety always preempts without voting.
- Files: `PRD.md` Draft 0.5, executive summary, demo narrative, section 9.6, section 15.8, observability, evals, FR-39/FR-40, P0 scope, acceptance criterion 18, UX copy, and risks.
- Result: The design demonstrates distributed edge AI and collective autonomy without relying on free-text model votes, continuous chatter, split-brain authority, or identical-model agreement as safety evidence.
- Follow-up: Implement only one four-vessel scripted scenario before Friday: Vessel 4 slows, one candidate assignment is rejected locally, a revised allocation reaches quorum/all-affected arm, and the map activates it at a future mission tick while showing measured coordination bytes.

### 2026-09-01 - One-Machine Mission Appliance

- Context: The user wants the complete demonstration packaged without a dozen services scattered across cloud providers or the Proxmox cluster.
- Decision: Ship one Docker Compose appliance with four required containers: a modular `keelmesh-core` Go binary with embedded React UI, one `keelmesh-ai` Python container, one official Apache Kafka broker in single-node KRaft combined mode, and PostgreSQL/pgvector. Add local-model, scale, and tools only as optional profiles.
- Files: `PRD.md` Draft 0.6, sections 13.3-13.6, FR-41, P0 scope, acceptance criterion 19, and deployment risks.
- Result: The live system starts with one command, exposes one URL/port, keeps dependencies private, needs no live cloud/Kubernetes/homelab service, and still preserves production-shaped module contracts and optional scale roles.
- Follow-up: Scaffold the four-container Compose release first; use one local model endpoint or deterministic fallback, bundle offline maps/fixtures, and implement a single verification command before adding optional infrastructure.

### 2026-09-01 - Use the Havoc-Aligned Web Stack

- Context: A Tauri desktop shell was technically coherent but introduced a framework Havoc did not mention and another architectural story to defend.
- Decision: Use React/TypeScript/WebGL as a browser UI, Go platform services, Python agent/ML services, and containerized infrastructure. Do not include Rust/Tauri in the P0 architecture.
- Files: `PRD.md` sections 13.4-13.6.
- Result: The demo now maximizes direct stack and role alignment and starts through a pinned Docker Compose profile.
- Follow-up: Scaffold the Go/Python/React monorepo and preserve portability through containers rather than a desktop wrapper.

### 2026-09-01 - Superseded: Tauri Desktop with Deployable Sidecars

- Context: The user specializes in portable Rust/Tauri applications and wants the interface, local AI, and platform clients packaged cohesively.
- Decision: Use Tauri 2 as the desktop/security/supervision boundary, React/TypeScript/WebGL in the webview, one Go platform sidecar, and one standalone Python agent sidecar. Keep platform contracts network-deployable so the same UI can connect to Scale Lab services.
- Files: `PRD.md` sections 13.4-13.6.
- Result: Rust has a justified product role while Go remains aligned with Havoc's platform and Python remains aligned with agent/ML infrastructure.
- Follow-up: Implement Windows x86-64 first; keep Tauri capabilities narrow, use ephemeral loopback credentials, and do not require system Go/Python runtimes.
- Status: Superseded later on 2026-09-01 by the Havoc-aligned web-stack decision above.

### 2026-09-01 - Havoc-Aligned Polyglot Stack

- Context: The interview transcript explicitly describes a Go platform and React/TypeScript/WebGL UI; the job posting emphasizes Python and a broad AI/ML infrastructure ecosystem.
- Decision: Use Go for resilient platform and fleet/data services, Python for agents/RAG/evaluations/ML workflows, and React/TypeScript/WebGL for the interface. Use MCP as the agent-to-platform boundary and Kafka/Postgres/S3-compatible storage to demonstrate posting-aligned distributed data patterns.
- Files: `PRD.md`.
- Result: The demo matches the role without using multiple languages arbitrarily or presenting preferred technologies as confirmed Havoc internals.
- Follow-up: Keep P0 implementation small enough to run reliably from Docker Compose; add MLflow, Dagster, and Kubernetes only after the core mission path is stable.

### 2026-09-01 - Live System Cutaway Presentation

- Context: The user wants infrastructure scale to be visually understandable without leaving the simulator/control interface.
- Decision: Add an Operator/Cutaway transition. The active map remains visible while the interface expands downward into animated AI assistance, planning, mission control, and telemetry/data-platform layers. A Cloud Outage state reroutes the visible flow to local inference while mission and telemetry layers remain healthy.
- Files: Interactive concept at `C:\Users\hendo420\.codex\visualizations\2026\09\01\01a05e24-30af-7ee1-8641-a026835396e6\live-system-cutaway.html`.
- Result: The demo can explain product behavior and infrastructure internals as one continuous story rather than switching to a separate generic dashboard.
- Follow-up: Reproduce this transition in the application and connect every displayed metric/state to real simulator and service telemetry.

### 2026-09-01 - Demonstrate Measured Horizontal Scale

- Context: The recruiter repeatedly emphasized infrastructure operating at large data and fleet scale.
- Decision: Use the same application for a simple visible mission and a configurable high-volume background fleet. Add a Platform view and scripted load/failure sequence showing backpressure, horizontal consumer scaling, durable recovery, deduplication, and replay.
- Result: The demo can prove infrastructure behavior on one development machine without pretending to store production-scale data.
- Follow-up: Select the event bus/storage stack, define load profiles, and establish reproducible benchmark targets after measuring the available hardware.

### 2026-09-01 - Visible Plan-Before-Action UX

- Context: The operator needs simple suggestions, voice, and chat without allowing opaque agent actions.
- Decision: Every agent recommendation becomes a selectable plan candidate. A deterministic planner renders and simulates the proposed action on the map before it can be authorized or executed.
- Result: The primary interaction becomes suggest/ask -> preview plan -> simulate -> explain -> authorize when required -> execute and audit.
- Follow-up: Define the plan schema, map visualization states, approval tiers, and Friday demo scenario.

### 2026-07-12 - Reset Stale Project Context

- Removed old project identity, history, setup details, external-system references, and follow-ups from this memory file.
- Preserved the memory-system rules, privacy guidance, source-of-truth ordering, freshness rules, and handoff structure.

## Known Issues

- Proxmox warned that thin-provisioned virtual sizes exceed the physical thin-pool capacity while creating the M0 snapshot. The snapshot succeeded, but host storage utilization/auto-extension must be monitored before creating many additional snapshots.

## Completed Work

### 2026-09-02 - Depth-aware coastal intent and higher-accuracy STT

- Context: The exact typed request `patrol the beach, stay within 1nm from the beach as long as ocean depth permits` failed `COMMAND_AMBIGUOUS`, and the deployed tiny.en microphone route was not accurate enough.
- Decision: Resolve beach/coast/shoreline/nearshore/littoral language against selected-vessel positions and deterministic routes sampled from the packaged NOAA ETOPO 5 m contour; parse nautical-mile limits into an explicit maximum-shore-distance constraint and generated operating corridor. Rank policy-valid plans before prohibited candidates. Upgrade STT to pinned faster-distil-whisper-small.en int8, three-beam decoding, tighter VAD, higher-quality browser capture, and startup model warming; do not use the harmful hotword prompt observed during benchmarking.
- Files: `internal/domain/fleetops.go`, `internal/fleetops/manager.go`, `internal/fleetops/manager_test.go`, `web/src/types.ts`, `web/src/FleetWorkspace.tsx`, `web/e2e/mission.spec.ts`, `speech/keelmesh_speech/service.py`, `compose.yaml`, `scripts/download_stt_model.py`, and `scripts/benchmark_stt_smoke.py`.
- Commands/tests: full Go suite; strict TypeScript, seven Vitest assertions, Vite build; Python compile; Compose validation; all nine Playwright workflows; three-command Morgan/STT smoke; central and twelve-node health/hash checks.
- Result: The reported beach command now creates one depth-aware operating corridor, 13 patrol waypoints, a 1,852 m maximum coastal offset, three candidates, and an enabled exact-plan approval path. The exact command transcribed at 0.000 word error and 0.199 RTF in the deterministic Morgan smoke. Proper-name samples remained imperfect (Gannet 0.231 WER; Watch Shoal 0.182 WER), so this is improved smoke evidence, not the outstanding 100-utterance human/noise accuracy benchmark. VM 214 and all twelve vessel nodes run core binary SHA-256 `e355ec10f2daecabeb36cad190e9cbc10a789ed0ed02cb0e50a411c38edc967e`. No snapshot or GitHub-hosted workflow.

### 2026-09-02 - Expandable selected-assets drawer and group reassignment

- Context: The compact selected-assets ribbon did not expose a manageable full selection, scoped inspection, or direct membership reassignment.
- Decision: Retain the compact ribbon, add an expandable limited-height drawer organized by operational group, add independent group/vessel inspection actions and aggregate Inspect All, and implement drag-to-group reassignment as one version-checked backend mutation. Reject membership changes while the vessel is bound to an active mission.
- Files: `internal/fleetops/manager.go`, `internal/fleetops/manager_test.go`, `internal/api/server.go`, `internal/api/fleetops.go`, `web/src/FleetWorkspace.tsx`, `web/src/app.css`, and `web/e2e/mission.spec.ts`.
- Commands/tests: local strict TypeScript, seven Vitest assertions, and Vite production build; VM full Go suite; nine Playwright workflows including drawer expansion, scoped inspection, persistent drag/drop, and deterministic fixture restoration; central plus twelve-node health/hash checks.
- Result: Selection remains compact by default but can expose and manage the complete selected scope. Group moves persist atomically, update source/destination revisions and vessel identity, and fail closed during active missions. VM 214 and all twelve vessel nodes run binary SHA-256 `5fd915f33752a71a30d6057c2fb7871a2a8deac55092dab5039e383b58d540a3`. No snapshot or GitHub-hosted workflow.

### 2026-09-02 - One-action selection clearing

- Context: The top selection ribbon required removing selected vessel chips one at a time.
- Decision: Add a visible Clear control beside the selected-vessel chips and the same Clear Selection action to the map context menu; preserve Escape as the keyboard equivalent and Pirate-mode nomenclature.
- Files: `web/src/FleetWorkspace.tsx`, `web/src/OperationsMap.tsx`, `web/src/app.css`, and `web/e2e/mission.spec.ts`.
- Commands/tests: TypeScript typecheck, seven Vitest assertions, Vite production build, targeted Playwright coverage for both clear paths, and all twelve node health checks.
- Result: Any selection scope can be cleared atomically from either the top ribbon or map context menu. VM 214 and all twelve vessel nodes run binary SHA-256 `324bc56bdea367bfb8c36a4dbac79ee5d4fbe19f5a286aa0141755f86440005d`. No snapshot or GitHub-hosted workflow.

### 2026-09-02 - Planning-aware multi-scope map selection

- Context: Double-click selected six grouped vessels on the map, but the bottom Generate Options dock remained bound to a previous mission; broader viewport/fleet selection also required separate controls.
- Decision: Bind exact operational-group selections to the planning dock without mutating mission state. Generate Options reuses a matching active mission or creates a new frozen mission for the selected scope. Add double/group, triple/viewport, quadruple/accessible-fleet gestures plus a discoverable right-click scope menu.
- Files: `web/src/FleetWorkspace.tsx`, `web/src/OperationsMap.tsx`, `web/src/app.css`, and `web/e2e/mission.spec.ts`.
- Commands/tests: TypeScript typecheck, seven Vitest assertions, Vite production build; nine Playwright workflows across the full suite, with targeted group planning, multi-click, and context-menu coverage; all twelve node health checks.
- Result: Group selection is visible in the bottom planning dock and Generate Options plans for that exact group. Right-click supports operational group, all visible vessels, and all accessible vessels. VM 214 and all twelve vessel nodes run binary SHA-256 `aaf92a1196b968f4f7ee439384c2c21924b3d2ad84671c354e904e2111f058b0`. No snapshot or GitHub-hosted workflow.

### 2026-09-02 - Intent-derived shoreline patrol geometry

- Context: `Patrol the shoreline and reserve 20% battery` was blocked in the browser because the UI required manually drawn geometry before calling the compiler.
- Decision: Let the deterministic Go intent compiler resolve shoreline/coast patrol terms to the nearest prevalidated local water sector, persist its operating polygon and five-point closed patrol loop, and parse reserve percentages conservatively. A 20% request is recorded but remains bounded by the standing 30% fleet minimum.
- Files: `internal/domain/fleetops.go`, `internal/fleetops/manager.go`, `internal/fleetops/manager_test.go`, `web/src/types.ts`, `web/src/FleetWorkspace.tsx`, `web/src/app.css`, and `web/e2e/mission.spec.ts`.
- Commands/tests: local TypeScript typecheck, seven Vitest assertions, Vite production build; VM full Go suite; eight Playwright workflows plus targeted shoreline preview/authorization readiness; central and twelve-node health checks.
- Result: A shoreline mission now generates three inspectable formation candidates without drawing an area, displays the inferred source and reserve-floor explanation, previews through the existing exact-hash authority path, and leaves genuinely ambiguous geography fail-closed. VM 214 and all twelve vessel nodes run binary SHA-256 `5c69a838361e5cac52cfea109c6590b635073bd3a09a2cbe0aa8b6d43b8540e8`. No snapshot or GitHub-hosted workflow.

### 2026-09-02 - Water context navigation and colored waypoints

- Context: Water, vessels, and waypoints needed distinct right-click behavior, with persistent numbered/color-coded waypoint sets usable by both operators and natural-language planning.
- Decision: Restrict the navigation menu to rendered water, keep vessel selection in its own menu, and make waypoint right-click an immediate delete. Store stable waypoint identity/color/sequence in mission geometry; renumber after deletion and let the compiler select a named color in sequence. Keep go-to commands on the existing preview/hash/authorization path.
- Files: `internal/domain/fleetops.go`, `internal/fleetops/manager.go`, `internal/fleetops/manager_test.go`, `web/src/types.ts`, `web/src/FleetWorkspace.tsx`, `web/src/OperationsMap.tsx`, `web/src/app.css`, and `web/e2e/mission.spec.ts`.
- Commands/tests: local strict TypeScript, seven Vitest assertions, production Vite build; VM 214 full Go suite and all eleven Playwright workflows; central deployment plus twelve-node binary/health verification.
- Result: Water right-click exposes go-to, numbered waypoint, six colors, plan-by-color, clear-color/all, map centering, and visible/all-fleet selection. Waypoints display their sequence and retain color through API persistence. VM 214 and all twelve vessel nodes run binary SHA-256 `81a7ed4750243d84730a70d27d6078b5ae9b313ce390c7a5f375eb3a195147d7`. No snapshot or GitHub-hosted workflow.

### 2026-09-02 - Water-safe fleet spawning

- Context: widening the six-vessel formations exposed legacy seed points on narrow Narragansett Bay land polygons.
- Decision: relocate the affected northern cell centers into water, validate every seeded vessel against the same packaged GeoJSON coastline with a `0.004°` minimum test margin, and migrate retained compact/readable legacy spawn neighborhoods while leaving operated vessels untouched.
- Files: `internal/fleetops/manager.go`, `internal/fleetops/manager_test.go`, and `web/e2e/mission.spec.ts`.
- Commands/tests: VM full Go suite, live API-versus-coastline validation before and after core restart (`48` vessels, `0` unsafe), seven Playwright workflows, and deployed visual inspection with zero console errors.
- Result: initial and retained legacy fleet spawns are water-safe and persist across restart; generated plans still use the exact preview/hash/authorization path. VM 214 and all twelve vessel nodes run binary SHA-256 `01f34ad75e3438266f5de980600a9bc471468d6c4b5c964f510c43d1762d335c`. No snapshot or GitHub-hosted workflow.

### 2026-09-02 - Six-vessel rendering and 2.5D hull sprites

- Context: MapLibre coupled each vessel icon to its colliding callsign label, so only one hull remained visible in dense six-vessel groups; the original sprites also read flat when the map was pitched.
- Decision: Set `text-optional` so labels can deconflict without hiding icons, widen each seeded two-row formation using real simulated coordinates, reduce the visual dominance of group rings, and use generated transparent depth-aware sprites for Kestrel, Mariner, and Atlas while retaining original source sprites.
- Files: `internal/fleetops/manager.go`, `internal/fleetops/manager_test.go`, `web/src/OperationsMap.tsx`, and `web/public/assets/vessels/*-2p5d.png`.
- Commands/tests: local TypeScript typecheck, seven Vitest assertions, Vite production build; VM 214 full Go suite; seven Playwright workflows; deployed browser visual inspection at overview and pitched zoom with zero console errors.
- Result: Six distinct hulls render per operational group, all 48 vessels remain selectable, and all twelve vessel VMs run healthy binary SHA-256 `6f840d47f9a706378a9571fa2d8844bae56ba409d72b67744282c9eecdecbb0b`. No snapshot or GitHub-hosted workflow.

### 2026-09-02 - Native environmental map layers

- Context: The operator needed changing currents, wind, and depth context without another rectangular raster map overlay.
- Decision: Generate 5/10/20/40/80 m contour vectors from NOAA NCEI ETOPO 2022, render them as native MapLibre lines, and animate current/wind arrow fields over water anchors using the existing deterministic M6 environment fixture. Provide compact independent layer toggles and live fixture values.
- Files: `scripts/generate_narragansett_map.py`, `web/public/assets/maps/narragansett.geojson`, `web/public/assets/maps/narragansett-manifest.json`, `web/src/OperationsMap.tsx`, `web/src/app.css`, `web/e2e/mission.spec.ts`.
- Commands/tests: TypeScript typecheck, seven Vitest assertions, Vite production build, six Playwright workflows plus targeted Pirate regression, deployed browser visual inspection, zero console errors.
- Result: Wind/current arrows animate every 500 ms while depth remains stable; all layers are local/offline and individually toggleable. The partial NOAA chart raster remains disabled. No snapshot or GitHub-hosted workflow.

### 2026-09-02 - Restored geographic map after overlay cleanup

- Context: Removing both the partial NOAA raster and coarse placeholder land polygons over-corrected the beige-overlay issue and left a nearly featureless blue-green map.
- Decision: Keep the partial NOAA raster disabled, replace the hand-drawn polygons with a pinned Natural Earth 1:10m land dataset clipped to `[-72.1, 40.75, -70.55, 42.05]`, and render it with restrained graphite/olive fills. Preserve local/offline operation.
- Files: `scripts/generate_narragansett_map.py`, `web/public/assets/maps/narragansett.geojson`, `web/public/assets/maps/narragansett-manifest.json`, `web/src/OperationsMap.tsx`, `web/e2e/mission.spec.ts`.
- Commands/tests: TypeScript typecheck, seven Vitest assertions, Vite production build, six Playwright workflows, browser visual inspection at the deployed viewport, and zero browser-console errors.
- Result: Recognizable regional land/water context is restored without rectangular raster seams; automated browser tests now reset M6 mission state after each test. Deployed binary SHA-256 `f1a39c312538529f94cc9209f693f73a5ba3e563fc49dfb49b841dc516bf3183` on VM 214 and all twelve vessel nodes. No snapshot or GitHub-hosted workflow.

### 2026-09-02 - Icon system, clean map, and Pirate/Navy watch

- Context: The M6/M7 workspace mixed text/Unicode controls, and coarse NOAA/placeholder polygons appeared as beige rectangular overlays over the map.
- Decision: Pin `lucide-react` 1.39.0, use semantic icons throughout the workspace, remove the rendered raster and coarse land-fill layers, and add a persistent Pirate/Navy presentation toggle. Keep all mission and engagement authority semantics identical between personas.
- Files: `web/package.json`, `web/package-lock.json`, `web/src/FleetWorkspace.tsx`, `web/src/ArenaView.tsx`, `web/src/OperationsMap.tsx`, `web/src/WindowManager.tsx`, `web/src/app.css`, `web/e2e/mission.spec.ts`, `internal/arena/manager.go`, `internal/arena/manager_test.go`.
- Commands/tests: local TypeScript typecheck, seven Vitest assertions, production Vite build; VM Go arena/API tests; six Playwright mission/theme/map workflows.
- Result: Deployed on VM 214 and all twelve vessel VMs with binary SHA-256 `a7795d72d50e417869638c84ed8b4a9dab5ebf1a6cf780d5b816921178a0d82a`; no Proxmox snapshot and no GitHub-hosted workflow.

- 2026-09-02: Completed the M7 deterministic Arena/API/UI vertical slice, coordinator-aware faction ingress, twelve physical dual-plane vessel VMs, VM node deployment, and protected radio-fault tooling. Remaining distributed-consensus and per-node AI runtime work is explicitly recorded rather than presented as complete.
- 2026-09-02: Released and pushed `v0.5.0-interview` with two validated evidence bundles, repeated rehearsals, offline proof, restart isolation, and disabled GitHub Actions. Snapshot creation remains intentionally separate and requires Proxmox storage inspection plus explicit user authorization.
- 2026-09-01: Completed M5 Quiet Fleet authority/contracts/API/map/UI, VM release command, dual-network offline mode, restart verification, release viewport coverage, and M1–M5 regression verification. Fixed cross-run M3 `vessel_latest` updates using run-aware produced-time ordering; repeated 1,000-vessel shadow replay matched 1,000/1,000 with zero dropped events. No Proxmox snapshot was created.
- 2026-09-01: Completed M4 contracts, private MCP security boundary, Python typed incident agent, adaptive OpenRouter/local/mock provider routing, bounded fixture RAG, isolated deterministic replay, human-gated incident-to-evaluation promotion, provider regression execution, AI evidence export, Engineer UI, M4 verifier/browser coverage, Compose/Kubernetes production shape, and expanded CI.
- 2026-09-01: Completed M3 contracts, one-image multi-role topology, pinned Kafka/PostgreSQL stack, 1,000+ vessel producer, bounded outbox, three supervised consumers, set-based ingestion, idempotent projections, deterministic duplicate/out-of-order/quarantine faults, actual lag and process metrics, signed worker kill/recovery, redrive, Kafka shadow replay, pgvector fixture retrieval, Prometheus endpoint, live Cutaway UI, verifier/evidence, Kubernetes production shape, and expanded CI.
- 2026-09-01: Completed M2 contracts, deterministic runtime packages, APIs/WebSocket events, map overlays, operator drill, shared fixtures, verifier, browser coverage, Compose deployment, and documentation. No Proxmox snapshot was created.
- 2026-09-01: Completed M0 naming, repository, VM/application identity migration, healthy renamed container, initial private GitHub push, baseline CI definition, and Proxmox snapshot.
- 2026-09-01: Created the initial comprehensive `PRD.md`, including source alignment, traceability, UX, planner, guardrails, cutaway, scale, resilience, stack, MCP contract, data contracts, evaluations, priorities, and acceptance criteria.
- 2026-07-12: Reset stale project-specific memory while retaining the durable memory-management logic.

## Open Follow-ups

- Replace M7's central deterministic coordinator/failover model with actual per-faction replicated consensus and durable follower forwarding. Add unique faction/node mTLS identities and prove committed-state convergence plus split-brain prevention across real radio-plane partitions before describing the fabric as a real Raft deployment.
- Package and benchmark the independent Python agent, Pocket TTS, and node-local STT roles on every vessel VM. The current nodes run the same complete Go/API/UI binary and have provider credentials, but they do not yet run twelve separate Python/speech services or physical GPUs.
- Reuse MetaCog's architectural patterns—not its complete application—for the next KeelMesh agent harness: typed tool schemas, lease-bound capability sets, durable turn checkpoints, idempotent requests, approval pauses, immutable receipts, and provider/tool evidence. The agent may operate UI presentation and draft map/mission state, but movement authority and fictional game effects still require deterministic Go validation and the configured human/ROE gate.
- The user explicitly selected one OpenRouter key installed independently on all twelve nodes. It is root-owned mode `0400` and excluded from Git, images, cloud-init, logs, APIs, and evidence. Rotate toward per-node revocable credentials later; disconnected safety and cached execution must never depend on cloud inference.
- Replace the ephemeral Cloudflare Quick Tunnel with a named tunnel and stable hostname after the Cloudflare account/domain choice is available.
- Run the full M6 speech benchmark: at least 100 command utterances plus quiet/fan/office/wind/marine variants on the interview browser and VM; compare browser WebGPU Whisper, browser WASM sherpa-onnx, node-local faster-whisper, and a modeled trusted-peer route. The current node-local smoke is promising but does not establish the latency/accuracy gates.
- Add browser WebGPU/WASM STT and authenticated trusted-peer transcription routing. The current implementation is VM/node-local faster-whisper with typed fallback; browser capture still needs HTTPS through the ephemeral Quick Tunnel.
- Complete POI editing, explicit heading/orbit/rendezvous direct-guidance controls, per-group/per-vessel persistent constraint editors, and measured 1,000-feature map frame-time evidence. Typed intent, waypoint/area/exclusion geometry, mission constraints, and deterministic formations are functional now.
- Add formal JSON Schema files for M2 peer-bundle, tape, and PNT contracts if needed beyond the shared Go/TypeScript fixture tests.
- Replace M4's fixture-backed MCP retrieval with the planned bundled ONNX embedding model and live pgvector hybrid indexing; the current corpus schema and pgvector infrastructure exist, but runtime M4 retrieval is deterministic fixture-backed.
- Replace the current state-backed trace projection with a real private OTLP export/receiver/storage path. Python and Go tracing contracts/timing are present, but the full cross-process OTLP pipeline remains a hardening follow-up.
- Configure and benchmark an optional local OpenAI-compatible provider. OpenRouter and deterministic mock are live; local routing is implemented but intentionally reports standby until endpoint/model values are supplied.

## Do Not Forget

- Read `AGENTS.md` and `MEMORY.md` before making future changes in this workspace.
- Update `MEMORY.md` after durable discoveries, decisions, fixes, verification results, blocked work, and handoffs.
- Never store secrets, tokens, passwords, private keys, or sensitive personal data in `MEMORY.md`.
- Proxmox storage is thin-provisioned and warned of aggregate virtual allocation beyond physical pool capacity. Before every new snapshot, inspect pool data/metadata utilization, free capacity, and VM 214's existing snapshots. Create no routine/redundant snapshots, and never delete or roll back one without explicit user approval.

## Archive

- 2026-09-01: Maritime LoRa range research was completed but the radio was removed from the target demo architecture. Findings remain historical only: rough-sea links are highly sensitive to antenna height, Fresnel clearance, motion/polarization, multipath, and wave blockage; marketing range is not a reliability guarantee.
