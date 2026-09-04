# Project Memory

Durable project context lives here. Update this file whenever information should survive chat context compression or future agent handoff.

## Active Handoff

- Current task: unsaved mission-draft reuse is complete.
- Last meaningful change: Mission and `+` now reopen the active or newest unsaved draft instead of creating duplicates; an explicit Save draft action releases the creator, and a started mission also no longer blocks new creation.
- Next step: none for this change; continue normal product iteration.
- Blockers: none. No snapshot is authorized.

## Current State

- Project concept: an offline-first one-operator/many-vessel peer-autonomy demonstration for a Havoc AI infrastructure interview.
- Core scenario: the operator defines a simulated search mission through the map, suggestions, voice, or chat; the system previews and simulates a constrained plan before authorization.
- Core demo thesis: “Program the mission once. Coordinate locally. Adapt together. Degrade safely when the uplink disappears.” The mesh transports signed state and proposals; authenticated edge nodes make bounded decisions under the Group Mission Contract.
- `PRD.md` is the product source of truth. The M1 React/TypeScript/MapLibre interface and deterministic Go mission authority are embedded in one offline Compose appliance running on VM 214.
- M2 is live in the same appliance. It adds a mission-relative clock, six signed 10-second tape segments per vessel, authenticated/deduplicated peer bundles, deterministic Starlink/HaLow faults, PNT evidence arbitration, bounded safe hold, and stale-safe bridge rejoin.
- M3 is live in the same appliance as a degraded-optional scale plane: Apache Kafka KRaft, PostgreSQL/pgvector with eight telemetry hash partitions, deterministic load generation, three supervised real consumer children, bounded bbolt producer outbox, quarantine/redrive, actual Kafka lag, Prometheus metrics, earliest-offset shadow replay, pgvector fixture retrieval, and an Operator/Cutaway UI. Only port 8080 is exposed.
- M4 is live as a degraded-optional AI plane. A private Go Streamable HTTP MCP server exposes ten allow-listed evidence/replay/draft tools to a non-root Python agent; the Engineer UI shows actual tool receipts, citations, provider attempts, isolated replay, immutable candidate hash, human approval, provider regression results, and trace timing. GPT-5.6 Luna is the primary mission advisor, with OpenRouter, optional local, and deterministic target-aware fallbacks. Mission authority and M1–M3 remain independent of AI health.
- M5 is live. Quiet Fleet coordinates Vessels 2–5 under a signed fixed-membership contract: the unsafe first proposal reaches quorum but cannot commit after Vessel 2 rejects it; the safe revision arms all affected nodes and activates atomically at a future tape boundary. VM-local release commands cover startup, status, reset, verification, offline proof, restart recovery, evidence, rehearsal, and stop.
- M6 is live on VM 214 with twelve persistent operating vessels mapped one-to-one to the twelve provisioned VM nodes. The reset baseline has zero operational groups/collections/missions and disperses every vessel across shoreline-validated open water; operators may create groups explicitly. The graphite MapLibre workspace retains fleet/search/map selection, vessel/environment/reachability inspection, movable/dockable windows, PostgreSQL-backed mission workspaces, deterministic formation candidates, exact-plan authorization, Jarvis/Barbossa TTS, and VM-local faster-whisper STT.
- M7 is live as a deterministic Arena vertical slice plus a physical node fabric. VM 214 exposes neutral/referee state and coordinator-aware Player B ingress; twelve Ubuntu vessel VMs run the same Go/UI node binary with faction-pinned APIs across separate management (`192.168.50.0/24`) and simulated-radio (`10.77.0.0/24`) interfaces. The UI demonstrates knowledge filtering, protected planes, 20× time, coordinator failover, semantic workspace actions, and human-approved fictional engagement effects. Real distributed Raft replication, node mTLS enforcement, and twelve independent Python/TTS/STT runtimes remain follow-ups and must not be claimed complete.
- M8 is live on VM 214 and all twelve vessel nodes. Missions compile into complete signed ten-second trajectory programs with no fixed six-segment ceiling; nodes expose a rolling 60-second hot tape, execution cursor, immutable revisions, future activation, and restart-persistent program state. Routine close-contact and reserve pressure produce deterministic bounded speed/heading/lateral adjustments. Groups persist formation, spacing, and assembly points and station-keep using deterministic environment, propulsion, and day/night solar accounting. The workspace now has unassigned vessels, safe group deletion, content-sized windows, map-click selection without unsolicited inspectors, and a Markdown mission chat with live provider receipts, route visuals, compact cards, and viewport-safe two-pane expansion.
- M9 work in progress: `/mcp/control` is a separate private Streamable HTTP MCP boundary with its own runtime token, strict schemas/body/deadline bounds, operator-visible reads, mission draft/compile/plan/preview tools, and presentation-only workspace actions. It intentionally omits direct authorize/start/effect tools and returns a hash-bound human approval pause for effects. Design and follow-ups are in `M9_GROUP_AUTONOMY_AND_EXTERNAL_MCP.md`.
- M10 is live on VM 214 and all twelve vessel nodes. The assistant produces strict scene intents that trusted Go code resolves into ordered A2UI v1.0 surfaces using pinned `@a2ui/react` 0.11.0, React 19.2.8, and Zod 3.25.76. One unpinned scene is active per browser session; up to four pinned scenes survive reload, scene values refresh from current fleet state without a model call, scene annotations are independent from durable mission overlays, and separate sessions cannot replace/read/mutate one another. The private MCP exposes session-scoped scene list/get/compose tools but no authorization or effect execution. Conversation detail lives in the separate global Assistant/History surface; Mission is a manual editor with optional AI refinement. Critical PNT, tape, and reserve transitions are detected deterministically and composed without granting model authority.
- M11 is live, last verified 2026-09-03: Go is the scoped memory authority; PostgreSQL/pgvector stores central turns, scenes, semantic items/revisions, entities/edges, candidates, receipts, contexts, tombstones, sync/replay evidence, and monotonic state versions. Context assembly supplies the latest 12 exact turns, up to six semantic memories, four procedures, and three episodes under an 8,000-token cap. Retrieval uses the implemented 55/25/10/10 vector/FTS/freshness/trust score and degrades to keyword or current-turn-only operation. Four memory Kafka topics and a real at-least-once memory worker are live. The private official MCP boundary exposes five read/draft-only tools and binds the investigator token to its server-side operator scope.
- M11 node memory, last verified 2026-09-03: all twelve vessel VMs run binary SHA-256 `bab309b98a4a204f2f8f955322135d12a342d168b1a7892511af85b086a7230f` and report `node-local`. Each has a mode-0700 memory directory, SQLite WAL store, and mode-0600 bbolt journal. A real A-01 test retained one explicit preference, two turns, and one trusted A2UI scene across service restart. Signed `MemoryBundleV1` contracts exist, but real radio-plane bundle exchange remains blocked on the broader M7 physical replication transport and must not be claimed live.
- M11 memory lab, last verified 2026-09-03: private MinIO, Dagster, and MLflow services are running as the optional `memory-lab` profile; a real Dagster memory asset materialization succeeded and MLflow uses PostgreSQL plus private MinIO. Normal mission startup remains independent of this profile. `scripts/keelmesh memory-status`, `memory-verify`, and `memory-export` pass; latest evidence directory is `/srv/keelmesh/evidence/memory/20260903T192557Z`.
- Browser startup compatibility, last verified 2026-09-03: VM 214's LAN HTTP origin uses a `crypto.getRandomValues` UUID fallback because `crypto.randomUUID` is secure-context-only. Fingerprinted assets are immutable while SPA HTML is `no-store`; Vite targets ES2020 and the app loads behind a module-error recovery boundary/watchdog. Lazy MapLibre CSS cannot override the operations map's absolute sizing. Live 1280×720 visual QA showed the regional chart, vessels, depth contours, and environment overlays with zero console/page errors; the map-first Playwright workflow and all three phone/tablet/desktop responsive workflows passed against `http://192.168.50.214:8080`.
- Universal assistant chat, last verified 2026-09-03: the old boxed Assistant History trigger is replaced by a compact unboxed chat icon aligned immediately left of the primary microphone, with a warm-white bubble fill, gold outline, three centered gold dots, and a layered whole-icon shadow. It toggles a movable/closable KeelMesh Assistant window with the actor/session-scoped transcript shared by voice and typed turns, Markdown text replies, activity state, recent command artifacts, and a keyboard/touch composer. The chat has Snap Right and workspace expand/restore controls but deliberately has no minimize control or minimized-shelf entry. Typed requests use the same memory, workspace actions, mission drafting, deterministic planning, and approval boundaries as voice but intentionally invoke neither STT nor TTS. Live browser verification confirmed open/close toggle, right docking, expansion, zero shelf entry, three rendered dots, layered shadow, and zero console errors; a prior typed operational check received the correct `20×` response while keeping the voice orb `idle`.
- Session-scoped chat clearing, last verified 2026-09-03: the Assistant chat header has a compact trash action with an explicit confirmation that long-term memory remains. `POST /api/v4/assistant/history:clear` removes only exact turns matching the supplied operator and browser session from in-memory state, PostgreSQL, and node-local SQLite; it preserves committed pgvector/semantic items, preferences, candidates, entities, scenes, retrieval receipts, and other sessions. Durable-store failure fails closed rather than reporting a misleading clear. Full Go tests/vet, strict TypeScript, 13 Vitest assertions, production Docker build, and a focused deployed Playwright workflow pass. VM 214 and all twelve nodes are healthy on binary SHA-256 `ef4b35158822ac2b744202af5ad885a53a8dcea681b85b07171ae2510ee2b770`; no GitHub-hosted workflow or Proxmox snapshot ran.
- Mission draft reuse, last verified 2026-09-03: both the top Mission control and mission-bar `+` reuse the active unsaved draft, or the newest unsaved draft if another mission is active. They restore its Fleet selection and Mission window without issuing another create request. Mission drafts expose an explicit persistent `draft_saved` lifecycle marker and Save draft action; once saved, or once execution starts, either entry point may create a fresh mission. Saved drafts remain editable and visible as `saved draft`. Full Go tests/vet, strict TypeScript, 13 Vitest assertions, production Docker build, and the focused three-workflow Mission Playwright suite pass. VM 214 and all twelve nodes are healthy on binary SHA-256 `cd68445954e1122d389c5991e1270c881ecb7f2c6861775e5c89d383117523be`; no GitHub-hosted workflow or Proxmox snapshot ran.
- Fleet hover summaries, last verified 2026-09-03: normal-mode primary window labels are shortened from `Fleet / Groups` and `Mission Canvas` to `Fleet` and `Mission`; Pirate equivalents are `Flotilla` and `Voyage`. Hovering a Fleet group now shows current availability, underway count, formation/spacing/heading, average reserve, route mode, and decision node. Hovering a vessel shows identity/class, motion, reserve projection, PNT uncertainty, health, and group. These use the single portal-rendered gold tooltip, preserve row selection/drag/drop, and are suppressed on coarse pointers. Strict TypeScript, 12 Vitest assertions, production Vite/Docker builds, and focused deployed Fleet/Mission Playwright workflows passed. VM 214 and all twelve vessel nodes are healthy on binary SHA-256 `c5169021daad7acf20bf16e277153635d4fb2932a1e1aa2e6f02dbd82e6bea3b`; no GitHub-hosted workflow or Proxmox snapshot ran.
- Conversational spatial grounding, last verified 2026-09-03: the reported failure was reproduced from persisted session `7013…`: exact voice turns were present, but the global assistant payload omitted vessel/contact coordinates and deterministic proximity facts. The payload now exposes the latest 12 voice/text turns as explicit `conversation_history`, preserves that history when a Mission becomes active, derives newest-first visible entity references for pronouns, includes live `[longitude, latitude]`, heading, and speed, and supplies core-computed nearest-controlled-vessel distances. Exact position/reference/nearest answers are state-backed by core even if the model or provider degrades. Full Go tests/vet and focused continuity/spatial regressions pass. A live OpenAI two-turn proof answered Atlantic Beacon's position, then resolved “that one” to report Whimbrel (KM-249) at 5.56 NM. VM 214 and all twelve nodes are healthy on binary SHA-256 `c22ba8600417765e8ef75e92849e3fbc55bbe0a8161ae4641083e1f13612f3dd`; no GitHub-hosted workflow or Proxmox snapshot ran.
- Independent detail windows, last verified 2026-09-03: every controlled vessel, neutral surface contact, and operational group opens under an entity-specific window ID, so multiple inspectors/managers remain open, minimized, restored, or closed independently. Each vessel inspector owns its reachability request. The bottom shelf contains only minimized context windows, restores on task click, closes permanently through its trash action, omits the redundant minus glyph, and horizontally scrolls through overflow by wheel/trackpad or touch pan with a thin themed scrollbar. A live 1366×768 browser check opened 12 vessel and four group windows concurrently, minimized all 16, scrolled the overflowing shelf, restored one, closed one without minimizing it, trashed another, and produced zero console errors.
- Vessel/group intelligence cards, last verified 2026-09-03: vessel detail now labels M8's `tape_depth_seconds` as the `HOT EXECUTION BUFFER` and explicitly explains that the full mission program may be arbitrarily long while only the next 60 seconds are validated and armed as a rolling resilient buffer. The vessel card uses an energy ring, projected-reserve/buffer/range bars, compact navigation/PNT/solar tiles, conditions, two-column peers, and an authority boundary. Group detail uses a live formation diagram, energy/formation/decision-node insights, bounded-autonomy state, and collapsed Identity/Station Policy, Assembly Point, and Members sections; its six expanded inputs share identical 29 px height and 9 px typography. Live 1440×900 checks found no vessel scroll, a compact 440 px collapsed group, correct expansion, no legacy `MISSION TAPE` label, and zero console errors.
- AI memory boundary, last verified 2026-09-03: global voice and A2UI turns use `store:false` provider calls but now receive a fresh authorized memory context containing exact recent turns plus bounded retrieved memory. Conversation turns and trusted scene history persist centrally and node-locally; scenes restore inertly after restart and pending approval authority is never revived. Mission-scoped turns use the same context assembly path when an active mission ID is supplied.
- Global voice control, last verified 2026-09-03: the fixed lower-right orb is green when ready/speaking, red while held, and blue with a circular stage gauge during transcription, model inference, response receipt, and TTS synthesis. Release auto-sends VM-local STT output to `/api/v3/assistant:command`. GPT-5.6 Luna receives a bounded current fleet/group/contact/mission/environment/workspace projection and returns validated conversation, presentation, or mission-draft actions. General answers remain audio-only; mission commands create/open Mission Planner and continue through the existing deterministic route, hash, and authorization boundary. Jarvis is Navy voice and Captain Barbossa is Pirate voice. The Go core and Python AI service use separate UID-owned `0400` OpenAI runtime copies so both remain non-root and least-privilege.
- Fictional surface traffic, last verified 2026-09-03: the M6 operating picture includes sixteen deterministic AIS-like contacts across container ship, tanker, ferry, trawler, fictional patrol vessel, and yacht classes: twelve follow bounded looped routes at 1.2–2.8 m/s and four remain at named anchorages with zero speed. The UI leads with m/s and shows knots secondarily to avoid mistaking a 5.4 kn label for 5.4 m/s. Each has a stable generated name, Boat ID, callsign, color identity, speed/course/dimensions/activity, and programmed route or anchorage. Controlled Atlas/Mariner/Kestrel classes top out at 3.0/3.2/3.4 m/s so every class has positive intercept headroom. Contact-follow plans bind the contact, 60-second refresh cadence, and 180-second prediction horizon into the approved hash; the core continually signs future trajectory revisions while live map lines update from each follower to the moving target. Completed/deleted missions clear route/geometry overlays and regroup released vessels around their current operating center rather than the launch point. Contacts remain simulation-only and cannot be directly commanded; their neutral generated art is unchanged by Pirate mode.
- Semantic contact planning, last verified 2026-09-03: AI-assisted mission compilation now calls a strict-schema semantic interpreter before deterministic planning. It may resolve only supplied contact IDs and emits guidance, contact behavior, dynamic-target binding, formation, stand-off, reserve, speed, hold intent, summary, and provider receipts. `approach and surround Safe Haven` resolves to `surface-16`, dynamic `surround`, ring geometry, three OpenAI strategies, three ring plans, and zero ambiguities. `go to Blue Finch` resolves to dynamic `approach` against `surface-12`; `go to Blue Finch and keep tracking it if it moves` resolves to dynamic follow. Contact plans retain live identity and rolling signed revision behavior instead of collapsing to a coordinate.
- Group map navigation, last verified 2026-09-03: group assembly translation uses the operator-selected monotonic simulation cadence shown by the UI. Mission waypoint metadata includes an owning group; exact whole-group selection fixes new waypoint color/ownership to that group. Draggable waypoints and hold points persist through existing mutation APIs. The group route state machine is `once -> loop -> pause_pending -> paused`, with route completion/loop advancement only after the whole formation reaches its current leg. Clear and vessel-anchor hold commands discard the route and move the formation to a deterministic hold. Idle groups maintain distinct rotated formation slots using persistent formation shape, spacing, and true heading; every stationary hull aligns to the common group heading. These idle controls reject conflicts with executing or authorized mission movement.
- Mission creation, last verified 2026-09-03: pressing `+` never requires a prior Fleet selection. With no selection it creates a uniquely named, zero-asset persistent draft and opens Mission Planner; with a selection it carries those assets into the draft. For AI-assisted chat/voice on an empty draft, the AI receives only bounded available group/vessel facts and selects an exact roster before geometry and strategy planning. Core independently validates and persists that roster; no movement occurs without the existing exact-plan approval. Manual generation still requires explicit Fleet selection.
- Mission route consumption, last verified 2026-09-03: active route overlays are a live remaining-work projection rather than a historical trail. Execution cursors provide a monotonic progress floor, current vessel positions provide smooth within-segment clipping, and group-owned or mission-wide waypoint markers disappear only after all relevant assigned vessels have passed them. Signed plans, hashes, journals, and replay evidence are not mutated.
- Mission layout, last verified 2026-09-03: Mission first opens docked right at 350 px, Fleet first opens docked left at 245 px, and attached windows retain user-resizable height. The compact editor exposes mission definition, assets mirrored from Fleet, map/route authoring, formation, constraints, route preview, and execution. Wide/maximized mode uses a balanced editor/workbench layout. Embedded mission chat, speech, history, and AI surfaces are intentionally absent.
- Mission refinement and approval, last verified 2026-09-03: deterministic manual generation produces one route without a provider. The optional collapsed AI refinement drawer accepts extra instruction and either modifies the route once or returns three A/B/C alternatives when explicitly requested. Card selection previews the route on the live map without executing it; `Review and start selected route` opens the exact-hash confirmation before preview/authorization/start. The server revalidates every lease, state version, policy result, and idempotency boundary.
- Mission command-center redesign, last verified 2026-09-03: `web/src/FleetWorkspace.tsx` and `web/src/app.css` replace the prior dense editor with compact Plan / Review & Run stages. Fleet is the sole asset selector and supports individual vessels, groups, and mixed multi-selection; the active Mission mirrors that selection without duplicating asset-management controls. Plan uses the reclaimed space for objective/type, formation, loop/end behavior, mission-scoped map authoring, geometry inventory, constraints, and provider-free route generation. Review contains active-program state, compact route alternatives, exact start, and optional AI refinement. Top-nav Mission and `+` create the same uniquely named persistent draft; a new draft excludes vessels already bound to another active mission instead of causing an authority conflict. Strict TypeScript, 13 Vitest assertions, production Vite/Docker builds, and focused desktop/phone Playwright workflows pass. Live in-app-browser QA confirmed Fleet-left/Mission-right layout and immediate Gannet/Osprey selection propagation. Commit `aa3ca51` is pushed to `main`; VM 214 and all twelve vessel nodes are healthy on binary SHA-256 `2df1e8586159d7542059419aba3b68b4f2e476360186fc9af1abe5dc9b6246ae`. No GitHub-hosted workflow or Proxmox snapshot ran.
- Responsive workspace, last verified 2026-09-03: desktop retains movable/resizable/dockable windows; tablets use adaptive dock widths and touch-sized controls; phones use viewport-clamped full-width windows between the mission/navigation bars and the status/minimized shelf. Touch/pen long press (560 ms with movement tolerance) mirrors vessel/contact/location right-click menus without blocking ordinary map/list scrolling. Mission rectangle drawing, operating/exclusion areas, and draggable waypoints/POIs/assembly points support touch. Coarse pointers use 44 px targets, phone inputs avoid browser zoom, contextual menus become bottom sheets, and map hover help yields to touch context interactions.
- Operator time control, last verified 2026-09-02: Fleet Operations exposes pause, 1×, 5×, 20×, 100×, and 500× in the persistent lower status bar. The API records the authoritative rate and simulation tick; accelerated modes repeat the same 200 ms simulation step so mission segments, group routes, energy, environment, contacts, and local guardrails advance together without stale skipping. Restart defaults to 20×.
- Vessel framing and game clock, last verified 2026-09-03: selecting a vessel's Fleet eye opens its independent inspector and sends a one-shot MapLibre camera request that centers the vessel at a useful minimum zoom without creating a persistent camera lock. The lower-right clock is no longer wall time; it displays `DAY n · HH:MM:SS` from the authoritative simulation tick aligned with the 08:00 solar-cycle start. It advances at 1×/5×/20×/100×/500× and freezes when paused. Strict TypeScript, 13 Vitest assertions, production Vite/Docker builds, focused desktop/phone Playwright, and live in-app-browser visual checks passed. VM 214 and all twelve vessel nodes are healthy on binary SHA-256 `935b6f4a04b2899da1195734772333b5a42acc47a595f7938fa8315d707af479`. No GitHub-hosted workflow or Proxmox snapshot ran.
- Visual system, last verified 2026-09-02: `lucide-react` 1.39.0 provides coherent UI icons. Navy mode shows a skull control to enter Pirate mode; Pirate mode shows an anchor control to return. The selected mode persists in local storage, changes workspace/Arena nomenclature and palette, selects the matching Jarvis/Barbossa speech persona, and sends a typed persona to the agent. Persona changes never grant additional tools or bypass exact-plan/effect approval. The default map uses the pinned Natural Earth 1:10m land dataset revision `ca96624a56bd078437bca8184e78163e5039ad19`, clipped locally by `scripts/generate_narragansett_map.py`; the incomplete NOAA raster extract is not rendered by default.
- Pirate vessel art, last verified 2026-09-02: Pirate mode replaces Kestrel, Mariner, and Atlas hulls in both MapLibre and inspectors with three generated fictional transparent tall-ship assets. The art is an original genre treatment and does not copy named film vessels.
- `IMPLEMENTATION_PLAN.md` is the Friday delivery plan. It sequences the work as visible vertical slices with acceptance gates, a three-day schedule, contingency cuts, API/contracts, verification evidence, and a six-minute rehearsal.
- `ROLE_ALIGNMENT_AUDIT.md` is the coverage contract against the recruiter transcript and job posting. It requires both an Operator workflow and a scoped Autonomy Engineer incident-to-eval workflow.

## User Preferences

- Keep `MEMORY.md` updated consistently with durable context that should survive chat compression.
- Do not run builds, tests, or workflows on GitHub-hosted Actions because they may cost money. Run release verification on VM 214 instead; repository pushes must not trigger GitHub workflows.
- Do not adopt OpenAI Realtime or paid cloud speech for the demo. Keep the existing VM-local faster-whisper STT and Jarvis/Pocket TTS path; cloud LLM use remains a separate bounded mission-advisor concern.
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
- External systems: VM-local STT/TTS are implemented. GPT-5.6 Luna is the primary cloud mission advisor through the Responses API; OpenRouter, optional local inference, and deterministic target-aware planning remain bounded fallbacks.
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
- The main UI is a map-first floating workspace. Mission creation is initiated from the mission `+` control or selected Fleet assets. Conversation is handled by the separate global Assistant voice/chat surface; Mission is a manual authoring and execution window with an optional bounded AI refinement drawer.
- Suggestions are grounded in current fleet state, approved runbooks, and similar historical missions. Each suggestion maps to a typed action template and shows its reason and provenance.
- Selecting a suggestion opens a planner preview. The planner displays routes, assignments, sequence, timing, resource projections, constraints, and icons/text on the map before execution.
- A deterministic policy gateway classifies plans as within the active mission lease, approval-required, or prohibited. Approval is bound to the exact plan version/hash.
- The simulated mission and safety controller must continue if cloud inference, local inference, voice, or connectivity fails.
- Every operator and vessel instance is modeled as an authenticated peer node with local policy, signed mission-package cache, monotonic lease timer, local event log, store-and-forward routing, resilient PNT, and optional local inference.
- Starlink is the preferred internet/cloud path, but it is not the control spine. Every authorized node may advertise healthy Starlink egress to peers. Simulated Wi-Fi HaLow is the local radio/MAC underlay; routing, store-and-forward delivery, security, and peer egress are implemented above it rather than assuming every HaLow product provides native mobile mesh.
- Network reachability is separate from authority. Relays may forward signed/encrypted bundles but cannot mint leases, rewrite plans, or bypass destination-side policy.
- The communication overlay is BPv7-inspired: transport-independent signed bundles, local queues, store-and-forward delivery, priorities, lifetimes, hop limits, bounded replication, deduplication, and eventual event reconciliation.
- Each vessel retains the complete signed mission program and materializes a 60-second rolling hot tape from it. The M2 incident fixture remains exactly six immutable ten-second segments, but ordinary missions may contain any finite number of ten-second trajectory-envelope segments. A hot window may cross an armed revision boundary so future work is available before atomic activation. Segments contain timestamped position/speed targets, corridors, reserve/separation/PNT envelopes, expiry, failure behavior, predecessor hash, and signature; they are not raw rudder/throttle instructions.
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

### 2026-09-03 - Unsaved mission-draft reuse

- Context: Mission and the mission-bar `+` always called mission creation, so reopening the creator could produce duplicate numbered drafts.
- Decision: persist an explicit `draft_saved` marker. Both creator entry points restore the active or newest unsaved draft and its Fleet selection; Save draft releases the creator, while starting the mission naturally removes it from draft consideration.
- Files: `internal/domain/fleetops.go`, `internal/fleetops/manager.go`, `internal/fleetops/manager_test.go`, `web/src/types.ts`, `web/src/FleetWorkspace.tsx`, and `web/e2e/mission-workspace.spec.ts`.
- Commands/tests: full Go test suite and vet in Go 1.27; strict TypeScript; 13 Vitest assertions; production Vite/Docker builds; three focused deployed Mission/phone/clock Playwright workflows; central and twelve-node binary/health verification.
- Result: VM 214 and all twelve vessel nodes are healthy on binary SHA-256 `cd68445954e1122d389c5991e1270c881ecb7f2c6861775e5c89d383117523be`. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - Session-scoped chat clearing

- Context: the universal Assistant window exposed the shared voice/text transcript but had no way to start a clean short-term conversation without resetting durable learned memory.
- Decision: add a confirmed chat-header clear action and an exact actor/session mutation. Delete only conversation turns from volatile state, PostgreSQL, and node-local SQLite; preserve vector/semantic memory, learned preferences, candidates, entities, scenes, receipts, and other sessions. Fail closed if either durable transcript store cannot complete the deletion.
- Files: `internal/memory/manager.go`, `internal/memory/node_store.go`, `internal/api/scenes.go`, `internal/api/server.go`, `internal/memory/manager_test.go`, `web/src/FleetWorkspace.tsx`, `web/src/app.css`, and `web/e2e/chat-clear.spec.ts`.
- Commands/tests: full Go test suite and vet in Go 1.27; strict TypeScript; 13 Vitest assertions; production Docker build; live API boundary probe; focused deployed Playwright clear/confirmation workflow; central and twelve-node binary/health verification.
- Result: VM 214 and all twelve vessel nodes are healthy on binary SHA-256 `ef4b35158822ac2b744202af5ad885a53a8dcea681b85b07171ae2510ee2b770`. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - Twelve VM-backed operating vessels

- Context: Fleet Operations still showed a separate 48-vessel/eight-group fixture even though twelve real vessel VMs already existed.
- Decision: use one shared manifest for operating-vessel, Arena-node, VMID, management-IP, host, faction, identity, class, and spawn mappings. Default/reset state is twelve unassigned stationary vessels, zero groups, and dispersed open-water positions; `legacy48` remains test-only compatibility.
- Files: `internal/domain/nodefleet.go`, `internal/domain/fleetops.go`, `internal/domain/arena.go`, `internal/fleetops/manager.go`, `internal/arena/manager.go`, `cmd/keelmesh-core/main.go`, UI/types/styles, focused tests, M6 verifier, README/operator/status documentation, and the live documentation screenshot.
- Commands/tests: full Go suite and vet; strict TypeScript; 13 Vitest assertions; production Docker build; shoreline/open-water and minimum-separation tests; focused deployed Playwright VM-fleet/radio/PNT tests; M6 verifier; visual 1440×900 inspection; all twelve real node `/healthz` endpoints; representative direct-node A/B fault checks; central topology/Fleet projection checks.
- Result: VM 214 reports 12 vessels, 12 unique node/VM bindings, zero groups, zero missions, 12 healthy management endpoints, 12 connected radio states, and 12 accepted nominal GNSS states. A simulated Starlink failure moves all six affected faction nodes to HaLow-only while management and inference stay connected; spoofed/jammed GNSS is rejected and the fused pose remains unchanged. All twelve nodes run binary SHA-256 `70ce289820989c5bd8b8f7ed1d6e3e4cc76c5dd9c6d6996b157ae0b588d2734f`. Radio and GNSS faults are simulation state, not physical hardware tests. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - System and AI Lab workspace separation

- Context: Cutaway mixed live infrastructure, AI evidence, and legacy demo controls while Engineer overlapped with it, making both views sparse and difficult to explain.
- Decision: rename and redesign Cutaway as `System`, dedicated to live operational observability, and Engineer as `AI Lab`, dedicated to AI investigation/evaluation evidence. Add direct cross-navigation while keeping both as independent movable primary windows.
- Files: `web/src/PlatformCutaway.tsx`, `web/src/EngineerView.tsx`, `web/src/FleetWorkspace.tsx`, `web/src/app.css`, `web/e2e/mission.spec.ts`, and `web/e2e/responsive.spec.ts`.
- Commands/tests: strict TypeScript, 13 Vitest assertions, production Vite and VM Docker builds, focused deployed Playwright window/viewport coverage, and live 1440×900, 1280×720, and 390×844 visual/geometry checks with zero browser console errors.
- Result: System polls real platform and 12-node topology state every second, distinguishes protected management/inference from simulated radio, and displays the real data pipeline and recovery evidence. AI Lab contains provider attempts, MCP receipts, retrieval sources, citations, replay, evaluation, approval, and trace evidence. VM 214 and all twelve nodes are healthy on binary SHA-256 `bab309b98a4a204f2f8f955322135d12a342d168b1a7892511af85b086a7230f`. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - Viewport-safe Cutaway

- Context: the M11 memory section increased Cutaway's natural content height while the desktop body still forced `overflow: hidden`, clipping its evidence/footer at common release viewports.
- Decision: let Cutaway use the largest safe first-open height, reduce the pipeline height without removing measured state, apply a short-window compact layout through the existing window container, and retain a thin graphite/amber internal-scroll fallback on constrained layouts.
- Files: `web/src/FleetWorkspace.tsx`, `web/src/app.css`, and `web/e2e/mission.spec.ts`.
- Commands/tests: strict TypeScript, 13 Vitest assertions, production Vite and VM Docker builds, focused deployed Playwright window coverage, live geometric checks at 1440×900, 1366×768, 1280×720, 820×1180, and 390×844, and zero console/page errors.
- Result: all Cutaway sections and the footer are visible without scrolling at all three release desktop sizes; tablet/phone windows remain inside the workspace and expose their stacked content through touch scrolling. VM 214 and all twelve vessel nodes run binary SHA-256 `3dbb0a80a15ee7440c6c4ff808e1d9c63af5ccd373c281309cd429a82f746a06`. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - Enterprise documentation portal

- Context: the root README stopped at M7, described superseded Mission chat controls, lacked a documentation index, and had no current product image or consolidated operating/security/reference material.
- Decision: replace README with a concise executive entry point and add a role-based `docs/` portal covering operator behavior, architecture, AI/memory, operations, security, verification, API/repository reference, and honest delivery status. Capture the live 1440×900 VM 214 operations workspace rather than using a mockup.
- Files: `README.md`, `docs/README.md`, `docs/OPERATOR_GUIDE.md`, `docs/ARCHITECTURE.md`, `docs/AI_AND_MEMORY.md`, `docs/OPERATIONS.md`, `docs/SECURITY.md`, `docs/VERIFICATION.md`, `docs/REFERENCE.md`, `docs/STATUS.md`, and `docs/assets/keelmesh-operations.png`.
- Commands/tests: live Playwright screenshot capture; `git diff --check`; local relative-link validation across all ten Markdown entry points; secret-pattern review; and image existence/dimension inspection.
- Result: documentation is GitHub/offline-clone navigable, separates implemented/simulated/planned claims, records the current test migration debt, and contains no credentials. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - Intentional phone map gestures

- Context: releasing a normal touch did not cancel the 560 ms long-press timer, so an ocean tap could unexpectedly open a delayed location card. Controlled-vessel taps also only selected the vessel instead of opening its status window.
- Decision: own touch taps in the pointer lifecycle, cancel the long-press timer on release, suppress synthetic mobile click/contextmenu events, leave open-water taps inert, open controlled-vessel/contact details on a short tap, and reserve contextual actions for deliberate long press. Mouse click and right-click behavior remains unchanged.
- Files: `web/src/OperationsMap.tsx` and `web/e2e/responsive.spec.ts`.
- Commands/tests: strict TypeScript, 13 Vitest assertions, production Vite/Docker build, and focused deployed 390×844 Playwright coverage passed. The browser regression waits beyond the old timer and proves a short ocean tap creates no context menu before proving long press still does.
- Result: VM 214 and all twelve vessel nodes are healthy on binary SHA-256 `d79b8721c229543c87da02dbcd350db07f8cafebfafd4ea7d77711ff1ed215d3`. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - Manual-first Mission editor and optional AI refinement

- Context: Mission still duplicated the global assistant with embedded chat, speech, history, and generated scene content, while the operator needed a complete non-AI authoring path plus optional AI help applied to an existing draft.
- Decision: make Mission a direct editor for mission definition, assets, map/route geometry, formation, constraints, route preview, and execution. `Build route` uses `planning_mode: manual` and bypasses providers. `Refine with AI` is collapsed by default, accepts an additional instruction, and either applies one recommendation or returns three alternatives when `Offer alternatives` is checked. Selecting an option changes the live map preview; only `Review and start selected route` opens the exact-hash confirmation.
- Files: `web/src/FleetWorkspace.tsx`, `web/src/app.css`, and `web/e2e/mission.spec.ts`.
- Commands/tests: strict TypeScript, 13 Vitest assertions, and production Vite/Docker builds passed. The focused deployed Playwright workflow passed manual authoring, explicit AI alternatives, preview-without-execution, and confirmation. The full legacy Playwright run found 13 passing tests and 11 failures, primarily obsolete assertions for the intentionally removed embedded mission chat; those tests require migration and must not drive the deleted UI back into Mission.
- Result: VM 214 and all twelve vessel nodes are healthy on binary SHA-256 `77eb2cc7353a9f9032af1b9c2f86d00a4808de548d28d3208e6043aebb9c3d67`. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - Underway battery depletion

- Context: Moving vessels could remain at 100% because the enlarged demo solar arrays exceeded cruise load and reserve was clamped to full each tick.
- Decision: Preserve fast stationary daylight charging, but cap effective solar offset to 65% of total load whenever speed exceeds 0.05 m/s. Propulsion remains class-calibrated and cubic with speed, so faster and farther travel consumes more stored energy while sunlight still extends range.
- Files: `internal/fleetops/manager.go`, `internal/fleetops/manager_test.go`, and `web/src/FleetWorkspace.tsx`.
- Commands/tests: full Go tests/vet passed in the pinned Go 1.27 container; strict TypeScript, 13 Vitest assertions, production Vite/Docker builds, focused deployed Fleet Playwright coverage, and a live 2.38 m/s four-sample check passed. Live reserve fell monotonically `99.839% -> 99.705% -> 99.572% -> 99.438%` while position advanced.
- Result: VM 214 and all twelve vessel nodes run binary SHA-256 `59d15500d582d45375d10a3caa05f801fc411909a19ed4fda4ab22ab83899621`. Reserve labels show two decimals near full charge so consumption is immediately visible. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - Stable live AI artifacts and releasable map focus

- Context: AI-opened A2UI windows visibly flashed on each live refresh, especially on phones, and their continuously recomputed camera recentered the map after the operator panned away or closed the window.
- Decision: retain the A2UI message processor for the lifetime of a surface, apply subsequent ordered messages without replaying `createSurface`, and separate live entity bindings from one-shot scene camera requests. Closing, minimizing, or dismissing an artifact clears its active scene focus and annotations; explicit Frame/Open actions may request a new camera move.
- Files: `web/src/A2UISurface.tsx`, `web/src/A2UISurface.test.tsx`, `web/src/FleetWorkspace.tsx`, `web/src/OperationsMap.tsx`, and `web/e2e/scene.spec.ts`.
- Commands/tests: strict TypeScript; 13 Vitest assertions; production Vite/Docker builds; deployed 390×844 Playwright scene regression proving the A2UI DOM remains mounted across more than two live refresh intervals and camera/annotation focus clears on close; central and twelve-node health/hash verification.
- Result: VM 214 and all twelve vessel nodes run binary SHA-256 `e64862ef69c8b975fe4c3292e08dfe48a7ba6aaea0d37d38e02f2f12b12f0e38`. No GitHub-hosted workflow or Proxmox snapshot ran.
- Follow-up: none.

### 2026-09-03 - Vessel and group intelligence-card redesign

- Context: the vessel card still presented the M2-era `MISSION TAPE 60s` label without explaining M8's arbitrarily long program plus rolling hot buffer, while group settings had inconsistent typography and little visual hierarchy.
- Files: `web/src/FleetWorkspace.tsx` and `web/src/app.css`.
- Commands/tests: strict TypeScript, all 12 Vitest assertions, production Vite build, VM 214 Docker build, and live Playwright checks at 1440×900, 1280×720, and 390×844 passed. The final desktop check verified no vessel overflow, collapsed group height 440 px, matching input/select dimensions, correct buffer/program labels, and zero console errors.
- Result: vessel and group windows are compact operational dashboards with progressive-disclosure editing and clearer autonomy, energy, formation, navigation, environment, connectivity, and authority insight.
- Follow-up: none.

### 2026-09-03 - Independent vessel/group windows and scrolling shelf

- Context: vessel and group inspectors reused one global window/state slot, and minimized task entries contained a redundant minus icon without reliable desktop horizontal-wheel overflow behavior.
- Files: `web/src/FleetWorkspace.tsx`, `web/src/WindowManager.tsx`, and `web/src/app.css`.
- Commands/tests: strict TypeScript passed; all 12 Vitest assertions passed; the production Vite and VM 214 Docker builds passed. Live Playwright opened 12 vessel plus four group windows, minimized all 16, verified horizontal overflow and a 420 px wheel scroll, restored and closed independently, deleted one shelf entry, found no minus glyph, and captured zero console errors.
- Result: controlled vessels, neutral contacts, and groups now use independent detail windows; the minimized shelf is a touch-capable horizontal task strip with restore and trash semantics.
- Follow-up: none.

### 2026-09-03 - M11 distributed memory deployed

- Context: Implemented the M11 central, retrieval, learning-candidate, node-local, MCP, optional MLOps, UI observability, verification, and evidence paths.
- Decision: PostgreSQL/pgvector remains the sole central vector store; SQLite WAL plus bbolt is the bounded node store; no Redis or second vector database was added. Restored scenes are inert and cannot revive pending approval authority.
- Files: `internal/memory/`, `internal/platform/schema.sql`, `internal/api/memory.go`, `internal/api/scenes.go`, `internal/domain/memory.go`, `ai/keelmesh_ai/embeddings.py`, `compose.yaml`, `memory_lab/`, `web/src/FleetWorkspace.tsx`, `scripts/verify_m11.py`, and `scripts/keelmesh`.
- Commands/tests: all Go tests and vet passed; strict TypeScript, 12 Vitest assertions, and production Vite build passed; Python Ruff formatting/lint, strict mypy, 14 pytest assertions, and Bandit passed after documenting the tokenizer `[PAD]` false positive. Compose base/memory-lab validation and `verify_m11.py` passed. Central hybrid retrieval returned four cited hits, replay matched six live items, and state version `19` survived restart. A real scene and exact turns survived central restart. All twelve nodes passed health/hash/node-local checks; A-01 retained one preference, two turns, and one scene across restart.
- Result: VM 214 runs healthy core/AI/Kafka/PostgreSQL/memory-worker plus optional private MinIO/Dagster/MLflow. M11 evidence exported to `/srv/keelmesh/evidence/memory/20260903T192557Z`. No GitHub-hosted workflow or Proxmox snapshot ran.
- Follow-up: implement physical signed memory-bundle exchange only when the outstanding M7 radio replication transport is built; contracts and local journals are ready, but transport is not claimed live.

### 2026-09-03 - Active overlays independent of planner focus

- Context: a live contact route and ETA disappeared when the operator clicked the mission tab because the map overlay was coupled to the planner-selected mission and its local plan list.
- Files: `web/src/FleetWorkspace.tsx`, `web/src/OperationsMap.tsx`, and `web/e2e/mission.spec.ts`.
- Commands/tests: production TypeScript/Vite Docker build, non-mutating live VM 214 browser check, and isolated Playwright contact-rendezvous regression on node 220.
- Result: every authorized/executing/paused mission contributes its own contact overlay. The live ETA remained visible and continued changing with Mission Planner hidden; the isolated regression passed through mission-tab hide/show. VM 214 and all twelve nodes are healthy on binary SHA-256 `d17a96c0599b9ede2b5d01e30585e09a282df30811dcd3d0f1810c193fcd76af`.
- Follow-up: none for this regression.

### 2026-09-03 - Live rendezvous ETA and restart-safe plan recovery

- Context: Yellow/NG was executing a rendezvous with anchored contact `MT Safe Haven`, but the map showed neither the active relationship nor an ETA and retained a misleading idle hold state.
- Files: `web/src/OperationsMap.tsx`, `web/src/FleetWorkspace.tsx`, `web/e2e/mission.spec.ts`, `internal/fleetops/manager.go`, and `internal/fleetops/manager_test.go`.
- Commands/tests: strict TypeScript check, seven Vitest tests, production Vite build, full Go tests, full Go vet, Docker rebuild, non-mutating live Playwright inspection, `/healthz`, and twelve-node service/hash checks.
- Result: the live map displayed a yellow rendezvous line and changing `ETA ... · 3.9 NM` label while the count of visible idle hold groups dropped from eight to seven. Restarted core returned the exact authorized plan for the executing mission. VM 214 and all twelve nodes are healthy on binary SHA-256 `9bc6fefb17931aa35de244694c7bfc182603c6128ecd7b2b4bdadf267bf790b1`.
- Follow-up: the dedicated destructive E2E workflow was added but not run against the current live mission because its setup resets operator mission state; the equivalent behavior was verified non-mutating against the active mission.

### 2026-09-03 - Controlled-vessel intercept speed calibration

- Context: fictional NPC traffic reached 8.0 m/s while controlled vessels were capped at 1.9–3.2 m/s and ordinary mission policy defaulted to 1.8 m/s, making contact follow/intercept impossible.
- Decision: Normalize NPC traffic to a deterministic 1.2–2.8 m/s simulation envelope; set Atlas, Mariner, and Kestrel limits to 3.0, 3.2, and 3.4 m/s; and make contact-follow drafts automatically request the slowest selected vessel's exact-approval speed ceiling. Follow strategies consume different portions of the available headroom but all maintain positive closure speed.
- Files: `internal/fleetops/manager.go` and `internal/fleetops/manager_test.go`.
- Commands/tests: full Go tests and vet passed in Go 1.27; focused follow planning proves every class can overtake the fastest contact, every generated assignment is faster than its target, and hardware limits are respected. The deployed surface-traffic Playwright workflow passed; live API inspection confirmed the exact class/contact speed envelopes.
- Result: VM 214 and all twelve vessel nodes are healthy on core binary SHA-256 `e809342456d1318839ab2ac18072ed0395b14ae9d14db9933b5990c179de2993`. The 20/30/45 nm nominal ranges remain calibrated at 1.8 m/s cruise; higher interception speed intentionally consumes more energy. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - Responsive touch-ready operations workspace

- Context: The workspace needed to retain its graphite/amber visual system while working coherently with mouse, keyboard, touch, and pen across desktop, tablet, and phone viewports.
- Decision: Add one reusable long-press/context-key interaction path; make map authoring touch-capable; give coarse pointers 44 px controls; adapt windows, navigation, menus, status controls, technical views, and the minimized shelf by viewport while preserving desktop docking and resizing.
- Files: `web/src/useLongPressContext.ts`, `web/src/FleetWorkspace.tsx`, `web/src/OperationsMap.tsx`, `web/src/WindowManager.tsx`, `web/src/app.css`, `web/e2e/responsive.spec.ts`, and `web/e2e/mission.spec.ts`.
- Commands/tests: strict TypeScript, seven Vitest assertions, and the production Vite build passed. The full Playwright suite passed 20/20; the dedicated phone/tablet/desktop suite passed three consecutive repeats (9/9). Live visual inspection covered 390x844 and 820x1180. VM 214 plus all twelve vessel nodes passed `/healthz` after rollout.
- Result: VM 214 and all twelve vessel nodes run core binary SHA-256 `e0eb84bd98afe423dc48cbd1713918aa0793f5bda586e7ceb5bed1998418625d`. Removing only regenerable Docker BuildKit cache reduced VM 214 root use from 98% to 27% (68 GiB free); tagged images, volumes, databases, evidence, and snapshots were retained. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - Workspace density, hover help, and window toggles

- Context: Mission Planner could not condense like Fleet/Groups, several technical windows retained oversized legacy styling, the header carried low-value simulation counters and drill links, and controls/map entities lacked consistent hover explanations.
- Decision: Add per-window minimum sizing and primary-window toggle semantics; make the minimum Planner chat-only; keep its default content free of scroll through progressive disclosure; remove Simulation/stat/Fleet Arena/Resilience/Quiet Fleet header items without deleting their implementations; add shared delayed tooltips and state-backed map hover cards; and restyle Engineer/Cutaway as compact graphite/amber consoles.
- Files: `web/src/WindowManager.tsx`, `web/src/HoverHelp.tsx`, `web/src/FleetWorkspace.tsx`, `web/src/OperationsMap.tsx`, `web/src/app.css`, and `web/e2e/mission.spec.ts`.
- Commands/tests: strict TypeScript, seven Vitest assertions, production Vite/Docker builds, full Go tests/vet in the pinned Go container, and all seventeen serial Playwright workflows passed. Live browser measurements show zero overflow in the normal Planner and in Engineer/Cutaway.
- Result: VM 214 and all twelve vessel nodes are healthy on core binary SHA-256 `42305da64d5f4bbafc05c1b532daf26cc41abf313e89225ad4cedbf41c31618f`. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - Mission Planner density redesign

- Context: Assigned Assets consumed disproportionate vertical space, neighboring sections used inconsistent sizing, and maximizing the Planner stretched sparse content instead of using the wider workspace professionally.
- Decision: Consolidate assignment and formation into one collapsible section, remove the redundant zero-asset pending card, shorten asset actions and guidance, and add separate density rules for floating, snapped-right, and maximized modes.
- Files: `web/src/FleetWorkspace.tsx`, `web/src/app.css`, and `web/e2e/mission.spec.ts`.
- Commands/tests: strict TypeScript, seven Vitest assertions, production Vite/Docker build, and all 17 Playwright workflows passed. Live browser inspection covered floating, snapped-right, and maximized layouts.
- Result: Implementation commit `ac28d23` is pushed to `main`. VM 214 and all twelve vessel nodes are healthy on core binary SHA-256 `ee255f6d752d87cf1e516e4cebb26129646d59b2888d6560fda038763356edc4`. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-02 - Compact route schematics and explicit docking icons

- Context: Straight multi-vessel routes rendered as oversized blue/green boxes with an unexplained white line, strategy details consumed too much expanded-planner space, and the plain panel docking glyph did not clearly communicate Mission Planner's right-side destination.
- Decision: Render compact aspect-preserving graphite route schematics with colored vessel tracks plus start/end markers; keep candidate details collapsed until explicitly expanded in every window mode; fit three summary cards across wide planners; and use directional panel-arrow icons for Snap left/right and Return to floating.
- Files: `web/src/FleetWorkspace.tsx`, `web/src/WindowManager.tsx`, `web/src/app.css`, and `web/e2e/mission.spec.ts`.
- Commands/tests: strict TypeScript, seven Vitest assertions, and production Vite/Docker builds passed; focused Mission Planner/manual and AI planner Playwright workflows passed; deployed browser inspection measured one visible 462×73 route summary, zero visible collapsed details, and no browser warnings/errors.
- Result: Implementation commit `bd3c5c4` is pushed to `main`. VM 214 and all twelve vessel nodes are healthy on core binary SHA-256 `b996432edfb084c6fe0afee9408ec4626c0427c8901ce0ffe98be51df21ec929`. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-02 - Captain Barbossa Pirate voice

- Context: The user cloned a Captain Barbossa voice in PPTtoVoice and requested it as the default spoken AI persona in Pirate mode while retaining Jarvis for Navy mode.
- Decision: Register `barbossa` as a fourteenth Pocket TTS voice, keep the safetensors artifact VM-local and outside Git/images, select Barbossa when Pirate mode is active, restore Jarvis when returning to Navy mode, and update Pirate Arena nomenclature without changing agent capabilities or approval boundaries.
- Files: `speech/keelmesh_speech/service.py`, `internal/fleetops/manager.go`, `internal/arena/manager.go`, `web/src/FleetWorkspace.tsx`, `web/src/ArenaView.tsx`, `scripts/verify_m6.py`, `web/e2e/mission.spec.ts`, and `README.md`.
- Commands/tests: Go tests and vet passed; strict TypeScript, seven Vitest assertions, and production Vite build passed; M6 verifier passed with fourteen voices; direct Barbossa synthesis returned HTTP 200 and a valid 169,004-byte RIFF WAV; focused Pirate/Navy Playwright switching passed.
- Result: Implementation commit `52266eb` is pushed to `main`. VM 214 and all twelve vessel nodes are healthy on core binary SHA-256 `44c738025902998fc35528d7a9bdcbdbcf4ae4a0c1ba6a9705403267feeca978`. Captain Barbossa and Jarvis model artifacts remain VM-local outside Git and container images. No GitHub-hosted workflow or Proxmox snapshot ran.
- Follow-up: Independent speech containers are still not deployed on every vessel VM; node-local speech redundancy remains a documented follow-up.

### 2026-09-02 - Draggable Group Navigation

- Context: Correct heading-only-looking hold behavior and make map waypoints executable by operational groups.
- Decision: Keep idle/group navigation distinct from exact-hash mission execution, but make it stateful, bounded, persistent, and conflict-aware. Route play cycles once, loop, then pause-after-current-leg.
- Files: `internal/domain/fleetops.go`, `internal/fleetops/manager.go`, `internal/api/fleetops.go`, `internal/api/server.go`, `web/src/OperationsMap.tsx`, `web/src/FleetWorkspace.tsx`, `web/src/types.ts`, and `web/src/app.css`.
- Commands/tests: full Go test/vet passed on VM 214; TypeScript typecheck, seven Vitest assertions, and production build passed; live read-only sample showed 20 vessels translating in two seconds; reversible API drill returned `once, loop, pause_pending, moving_to_hold` and restored the original BG assembly point.
- Result: VM 214 and all twelve vessel nodes are healthy on binary SHA-256 `a169ce68d821a772d7bc8493c284e71b5d1ac6406c07845ea10b9a23271ee67a`, with draggable waypoint/hold callbacks, group-colored route controls, right-click hold/clear commands, mission geometry save semantics, and mission group reassignment.
- Follow-up: deploy the new core binary to all vessel nodes and add a browser drag/route regression once the shared E2E fixture isolates persisted mission state cleanly.

### 2026-09-02 - Fictional Surface Traffic And Follow Planning

- Context: Add moving non-fleet water traffic, inspection, original art, and natural-language follow commands without confusing contacts with friendly controllable vessels.
- Files: `internal/domain/fleetops.go`, `internal/fleetops/manager.go`, `internal/api/fleetops.go`, `internal/agent/manager.go`, `ai/keelmesh_ai/service.py`, `web/src/OperationsMap.tsx`, `web/src/FleetWorkspace.tsx`, `web/public/assets/traffic/`, and M6 verification/tests.
- Commands/tests: full Go test/vet passed; Python Ruff/mypy/11 pytest passed; TypeScript typecheck, 7 Vitest assertions, and production build passed; `scripts/verify_m6.py` passed with 12 contacts and exact follow-target assertions; focused Playwright surface-traffic test passed without browser console errors.
- Result: VM 214 and all twelve vessel nodes serve the same binary SHA-256 `2da92ebe2a7cd7a8016b63796e9efb1a1e7c1056d68a4c7e7615ba65976fd956`; every node passed `/healthz`. Visual capture confirmed readable routes and sprites. A full legacy Playwright run reached 14/16; the pre-existing dragged-geometry and retained-chat-state cases remain flaky because the shared live server is not fully reset between cases and are not attributed to the contact path.
- Follow-up: consider receding-horizon replanning for indefinite pursuit; the current implementation safely previews a twelve-minute predicted target track and can be revised as the contact moves.

### 2026-09-02 - Spoken group colors and Jarvis default voice

- Context: Waypoint colors were generic rather than tied to operational groups, spoken commands lacked authoritative group-color aliases, and the new PPTtoVoice Jarvis clone was not available in KeelMesh.
- Decision: Assign eight stable, visually distinct, plain-language group colors; carry group name, code, and color name as equivalent target aliases into the model context; construct the water-context waypoint palette from live group metadata; and load Jarvis from a VM-local safetensors runtime artifact while retaining all built-in voices.
- Files: `internal/domain/fleetops.go`, `internal/fleetops/manager.go`, `internal/agent/manager.go`, `internal/api/fleetops.go`, `ai/keelmesh_ai/service.py`, `speech/keelmesh_speech/service.py`, `web/src/types.ts`, `web/src/FleetWorkspace.tsx`, `web/src/OperationsMap.tsx`, `web/src/app.css`, verifiers/tests, and `README.md`.
- Commands/tests: full Go test suite and `go vet` passed before final frontend-only polish; Python Ruff, strict mypy, and eleven pytest assertions passed; strict TypeScript, seven Vitest assertions, and production Vite build passed; all fifteen Playwright workflows passed; M6 verifier reports 48 vessels, eight groups, three plans, thirteen voices, and Jarvis default. Direct Jarvis synthesis returned HTTP 200 and a valid 295,724-byte RIFF WAV in 2.336 seconds.
- Result: Implementation commit `9634379` is pushed to `main`. VM 214 and all twelve vessel nodes are healthy on core binary SHA-256 `f69d90d431a90b02b1ff9e6bca8d0470ad5887d0cb3d666421625d0a09f5724c`. The custom voice remains outside Git and container images. No GitHub-hosted workflow or Proxmox snapshot ran.
- Follow-up: Independent speech containers are still not deployed on every vessel VM; node-local speech redundancy remains a documented follow-up.

### 2026-09-02 - Spoken multi-leg routes, audible replies, and compact expanded chat

- Context: A six-vessel command to travel two nautical miles south, then two nautical miles west, then hold received a valid model summary but failed deterministic route generation with `COMMAND_AMBIGUOUS`; auto-read was off in a fresh browser and expanded chat messages stretched into large empty cards.
- Decision: Parse bounded ordered numeric or spoken-number cardinal legs into sequential geometry before strategy planning; default the versioned Morgan auto-read preference on unless explicitly disabled; provide replay on every AI reply; and make chat rows/content cards intrinsically sized in both compact and expanded layouts.
- Files: `internal/fleetops/manager.go`, `internal/fleetops/manager_test.go`, `web/src/FleetWorkspace.tsx`, `web/src/app.css`, `web/e2e/mission.spec.ts`.
- Commands/tests: full Go test suite and `go vet`; strict TypeScript; seven Vitest assertions; production Docker build; direct Morgan synthesis returned HTTP 200 `audio/wav` with 280,364 bytes; all fifteen Playwright workflows passed, including the exact reported command, two generated waypoints, multiple strategy cards, Morgan synthesis request, and compact maximized cards.
- Result: VM 214 and all twelve vessel nodes are healthy on binary SHA-256 `25de728c6163557c26f3a125a4a561c65e56220d725d522b279c010259ee6aa4`. No GitHub-hosted workflow or Proxmox snapshot ran.
- Follow-up: Browser playback can still be blocked by browser/device autoplay policy; the visible per-message speaker button provides an explicit retry path.

### 2026-09-02 - Streamlined mission voice loop

- Context: The mission chat had lost its microphone control, exposed speech as a one-off button, and started with a canned command.
- Decision: Restore tap-to-start/tap-to-stop STT inside the chat composer, persist an opt-in auto-read checkbox, synthesize each newly accepted AI reply through the existing selected Pocket TTS voice, and keep the composer blank/clear after successful submission.
- Files: `web/src/FleetWorkspace.tsx`, `web/src/app.css`, `web/e2e/mission.spec.ts`.
- Commands/tests: local TypeScript typecheck and seven Vitest assertions passed; VM 214 production Docker build passed; all 13 Playwright mission tests passed against the deployed app; direct Morgan synthesis returned a valid 84,524-byte 24 kHz mono PCM WAV; all twelve vessel nodes restarted healthy on binary SHA-256 `5b0553be537c6d5d188dccea82da6c1deb9455daeede5e4a9c5de2769d9a4002`.
- Result: The low-cost local speech loop is live on VM 214 and every vessel node. No GitHub-hosted workflow or Proxmox snapshot was used.
- Follow-up: Browser microphone accuracy remains dependent on the current VM-local faster-whisper model and should continue to be measured separately from this UI repair.

### 2026-09-02 - Push-to-talk auto-send

- Context: The restored microphone required a second tap to stop and left the operator to submit the transcript manually.
- Decision: Make the mission-chat microphone press-and-hold. Releasing records the final chunk, invokes VM-local transcription, and automatically submits a non-empty transcript through the same command compile/planner path as typed Send. Keyboard Space/Enter uses the same press/release semantics.
- Files: `web/src/FleetWorkspace.tsx`, `web/e2e/mission.spec.ts`.
- Commands/tests: local TypeScript typecheck and seven Vitest assertions passed; VM 214 production Docker build passed; deployed Playwright voice-control test used a deterministic MediaRecorder/STT fixture and proved release produced the operator chat message and cleared the composer; all twelve vessel nodes restarted healthy on SHA-256 `7d5397a83138e90d5e3dac6413556f35a6f059bb9cfc31183d3d84cc2cea1d45`.
- Result: Hold, speak, and release now sends without an extra click. Empty/failed transcription still sends nothing and leaves typed input available. No GitHub-hosted workflow or Proxmox snapshot was used.
- Follow-up: Continue improving STT model/noise accuracy independently of this interaction path.

- 2026-09-02: M8 passed the complete Go test suite and `go vet`, strict TypeScript, seven Vitest assertions, production Vite build, Python Ruff/strict mypy/eleven pytest assertions, and all thirteen serial Playwright Operations/Arena workflows. A live 830-second route produced 83 signed segments with a six-segment/60-second rolling hot tape; GPT-5.6 Luna returned a mission-specific Markdown recommendation and three target-aware options. VM 214 and all twelve vessel nodes are healthy on binary SHA-256 `5c7167060c6ea5b0d1e9b6eacfe0169e5cee75ebba4e18ff428e4e106025eef7`. No snapshot or GitHub-hosted workflow ran.

- 2026-09-02: AI geometry selection passed the full Go suite/vet, strict TypeScript, seven Vitest assertions, Vite build, Ruff, and ten pytest assertions. Live GPT-5.6 Luna selected distinct exact sectors for Sakonnet North (`coastal-corridor-03`) and Block Island West (`coastal-corridor-08`), with different boundaries/13-point routes; a location-unspecified search also received a provider-selected nearby sector instead of `COMMAND_AMBIGUOUS`. VM 214 and all twelve nodes are healthy on binary SHA-256 `ed0d7b787dd340d519930b16b2086ec4c6db402e2cef87262800cadaede1c55b`. No snapshot or hosted workflow ran.

- 2026-09-02: Entity naming passed all Go tests/vet, strict TypeScript, seven Vitest assertions, Vite production build, Python Ruff/nine pytest assertions, Compose validation, live API create/AI-name/rename/restore/delete cleanup, and browser inspection of the inline mission editor. GPT-5.6 Luna changed a temporary AI draft from `Operation Pending` to `Operation Steady Shoreline Patrol`; an operator draft received `Harbor Lantern`. VM 214 and all twelve nodes are healthy on binary SHA-256 `e885764330c80aeb3954087ab42ebf2d17e0d4b4febdf2edd32e353111c8cc12`. No snapshot or hosted workflow ran.

- 2026-09-02: Corrected mission deletion passed strict TypeScript, seven Vitest assertions, production Vite build, Playwright test discovery, all Go tests/vet, live browser confirmation-sheet inspection, exact zero-pixel icon-center deltas, and an immediate create/delete race proof followed by core restart (`0` matching API missions and `0` PostgreSQL rows). VM 214 and all twelve nodes run binary SHA-256 `882858d8fce265d1ca79699247e82f176cf10f4985261b52fb1afc515ab8de60`. No snapshot or GitHub-hosted workflow ran.
- 2026-09-02: Mission lifecycle changes passed all Go tests/vet, strict TypeScript, seven Vitest assertions, production Vite build, Playwright test discovery, live browser control/state inspection, and a durable API deletion followed by core restart. Existing persisted duplicates were deterministically repaired to `Mission 1` and `Mission 2`; all twelve vessel nodes are healthy on binary SHA-256 `5c8c74aac153c7e0ab4f8f4203665eb4ea45b0f5864501ed7aa73d6fa0356b82`. The full Playwright suite was not run because its reset hook would delete the user's current missions. No snapshot or GitHub-hosted workflow ran.
- 2026-09-02: New Mission plus alignment passed strict TypeScript, seven Vitest assertions, production Vite build, and a Playwright geometry assertion requiring the icon and tab centers to differ by less than two pixels on both axes. VM 214 and all twelve nodes run binary SHA-256 `ec8c2dd0363f02b0f59d34c88efa2799d3d5f6338c376f3cefcb59cdd31bf77f`. No snapshot or GitHub-hosted workflow ran.
- 2026-09-02: Workspace semantics passed all Go tests/vet, strict TypeScript, seven Vitest assertions, production Vite build, ten unchanged Playwright workflows, and the corrected dedicated window workflow on VM 214. Coverage proves top-nav primary windows never enter the lower taskbar or expose Close, context/detail windows minimize/restore/delete through the taskbar, mission pause/resume/delete is state-backed, and Cutaway retains legacy controls. VM 214 and all twelve nodes run binary SHA-256 `306a0cb2fa84937b628c4fa53111f615cc357052f7cb637e7636b84e2c6144a6`. No snapshot or GitHub-hosted workflow ran.
- 2026-09-02: Fleet selection/status polish passed strict TypeScript, seven Vitest assertions, production Vite build, and all eleven Playwright workflows on VM 214. Tests cover exact selection highlighting state, absent collection-chip UI, vessel status, group status, group reassignment, all mission flows, Arena failover, and release viewports. VM 214 and all twelve vessel nodes run binary SHA-256 `ec74e473578ef84b67c41689b48a875f2fa5e2b6ff74953b98df9c880ffc62be`. No snapshot or GitHub-hosted workflow ran.
- 2026-09-02: Selection consolidation passed strict TypeScript, seven Vitest assertions, production Vite build, targeted three-workflow Playwright coverage, and the full eleven-workflow browser suite on VM 214. Coverage proves the duplicate selection DOM is absent, Fleet / Groups reflects exact counts, minimized Fleet / Groups automatically restores on map selection, and drag/context group reassignment remains functional. VM 214 and all twelve vessel nodes run binary SHA-256 `1b8f7bca0c724b00bb606f825edc28189af2fe2314204b148b1d3360c78dde04`. No snapshot or GitHub-hosted workflow ran.
- 2026-09-02: Inline group drag/drop and vessel context assignment passed strict TypeScript, seven Vitest assertions, production frontend build, and all eleven Playwright workflows on VM 214. Coverage includes the exact `Jaeger → BL Bay Lantern` move in both lists, context-menu reassignment, create-group form/API, fixture cleanup, selection retention, and inspection. VM 214 and all twelve vessel nodes run binary SHA-256 `320885ef2e8c82029213ddddfa6489fac3150d84d0a3189aadb15026c5eab5fc`. No snapshot or GitHub-hosted workflow ran.
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

### 2026-09-02 - Real mission advisor and symmetric node provider path

- Context: A single selected vessel received canned fleet formations, and live tracing proved the M6 planner had not contacted an LLM.
- Decision: Route bounded `MissionPlanningContextV2` through GPT-5.6 Luna using strict Responses API JSON Schema, then let deterministic Go code compute routes, metrics, policy, recommendation, hash, and approval. Reject fleet-only single-vessel semantics and retain target-aware deterministic fallback. Never grant the model authority.
- Files: `ai/keelmesh_ai/service.py`, `internal/agent/manager.go`, `internal/api/fleetops.go`, `internal/domain/fleetops.go`, `internal/fleetops/manager.go`, `web/src/FleetWorkspace.tsx`, `compose.yaml`, and `infrastructure/systemd/keelmesh-node-openai.conf`.
- Result: Implementation commit `4198725` was pushed to `main`. Central live proof accepted GPT-5.6 Luna in 4.4 seconds and produced three independent shoreline strategies. Deterministic planning marked one exact option recommended/approval-required and two slower alternatives prohibited. VM 220 independently proved direct node-to-OpenAI fallback in 5.3 seconds. All twelve nodes are healthy, advertise `cloud-openai / gpt-5.6-luna`, can read their separately installed `0400` credential, and match VM 214 binary SHA-256 `bd7af374ae65f85c91e93fb95ebe9033c00e676a6e5c436ea0915b9244fd1296`. Full Go test/vet, Python Ruff/mypy/eight pytest tests, strict TypeScript/seven Vitest tests/build, Compose validation, tracked secret scan, and the targeted Playwright workflow passed. LAN and main Quick Tunnel health returned 200. No GitHub-hosted workflow or Proxmox snapshot ran.
- Follow-up: Rotate the API credential that was pasted into chat after the interview/demo. Do not store it in Git, images, logs, evidence, or memory.

## Decisions

### 2026-09-02 - GPT-5.6 Luna primary cloud advisor

- Context: Free OpenRouter models were contacted but intermittently truncated structured output, ignored single-vessel schema constraints, or returned 429 capacity errors.
- Decision: Use OpenAI `gpt-5.6-luna` as the primary paid provider because the official model catalog identifies it as the cost-sensitive GPT-5.6 API model. Preserve OpenRouter/local/mock fallback and fail closed on malformed or semantically invalid output.
- Result: Provider provenance and latency are visible in the planner; model output remains advisory and is never trusted for geometry, kinematics, policy, signatures, leases, or execution.

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

### 2026-09-02 - Guarded group decision node, pirate hulls, and external control MCP foundation

- Context: Groups need predictable AI-assisted adaptation inside mission guardrails, isolated-vessel contingencies, custom Pirate-mode hulls, and a robust semantic boundary for external agents.
- Decision: Elect the lowest reachable `decision_capable` vessel ID as advisory group decision node; preserve deterministic per-vessel validation and stop/request instruction outside PNT or reserve limits. Keep signal-seek/return-home executable only when pre-authorized. Give the external MCP a separate token and typed canonical-manager tools, while withholding authorize/start/effect, secrets, shell, filesystem, and arbitrary-network capabilities.
- Files: `internal/domain/fleetops.go`, `internal/domain/trajectory.go`, `internal/fleetops/manager.go`, `internal/agent/mcp_control.go`, matching tests/types/UI, `compose.yaml`, `web/public/assets/vessels/pirate-*.png`, and `M9_GROUP_AUTONOMY_AND_EXTERNAL_MCP.md`.
- Commands/tests: strict TypeScript typecheck; full Go test/vet in a pinned Go container; seven Vitest assertions; production Docker build; authenticated MCP initialize plus unauthorized rejection; Pirate Playwright workflow; water-context regression; central and twelve-node health/hash verification.
- Result: VM 214 and all twelve nodes run the same healthy binary SHA-256 `e8ba0a40c7e40f4a6c27205f283f9f8bdf6f9b82e7ed3c9080748be70b6e22d5`. Live group snapshots expose leader policy/node/epoch/fallback. The external MCP vertical slice is private on management port 8081; signed group replans and executable signal-seek/home corridors remain staged follow-ups. No snapshot or GitHub-hosted workflow ran.

### 2026-09-02 - Adaptive mission programs and conversational planning

- Context: one-minute tapes were being treated as the mission itself, idle groups had no configurable station geometry, and the planner was a short intent form rather than a durable conversation.
- Decision: retain complete immutable trajectory programs separately from a bounded rolling hot tape; represent mid-mission changes as future signed revisions; keep routine avoidance and reserve compensation deterministic; persist group formation/spacing/assembly state; and replace the global command input with a per-mission Markdown AI workspace.
- Files: `M8_ADAPTIVE_MISSION_EXECUTION_PLAN.md`, `internal/domain/trajectory.go`, `internal/trajectory/`, `internal/fleetops/manager.go`, `internal/platform/schema.sql`, `internal/agent/manager.go`, `ai/keelmesh_ai/service.py`, `web/src/FleetWorkspace.tsx`, `web/src/OperationsMap.tsx`, `web/src/WindowManager.tsx`, `web/src/app.css`, mirrored contracts, and tests.
- Commands/tests: full Go suite/vet; strict TypeScript, Vitest, and Vite production build; Ruff, strict mypy, and pytest; live OpenAI mission probe; long-program API probe; thirteen Playwright workflows; central and twelve-node health/hash verification.
- Result: long missions, rolling preload, future revision activation, group station keeping, unassigned membership, AI chat, route visuals, and compact/expanded strategy comparison are deployed. Provider loss cannot block tape execution, local correction, or safe hold. Automatic out-of-envelope node-agent escalation remains an explicit follow-up rather than an overstated capability.

### 2026-09-02 - Bounded AI map-geometry selection

- Context: Natural-language mission creation repeatedly showed the same deterministic coastal square because the model could recommend strategies but was explicitly forbidden from choosing geometry.
- Decision: Expose seven distinct, depth-aware local corridor candidates with stable IDs, names, centers, exact boundaries, ordered waypoints, and target distance. Require the model to select one allow-listed ID; explicit place names outrank proximity, while an unspecified patrol/search uses vessel position and environment. Deterministic Go copies and validates the candidate and rejects arbitrary coordinates.
- Files: `internal/domain/fleetops.go`, `internal/fleetops/manager.go`, `internal/agent/manager.go`, `ai/keelmesh_ai/service.py`, `web/src/types.ts`, `web/src/FleetWorkspace.tsx`, `web/src/app.css`, and tests.
- Commands/tests: full Go suite/vet; TypeScript/Vitest/Vite; Ruff/pytest; two named-region live OpenAI proofs; generic search proof; temporary mission cleanup; central and twelve-node deployment.
- Result: AI-created and natural-language patrol/search missions can place their own reviewed boundary and waypoints without inventing coordinates or bypassing policy. The planner identifies the provider-selected sector in its evidence card. Implementation commit `a801bbe` is pushed; VM 214 and all twelve nodes run binary SHA-256 `ed0d7b787dd340d519930b16b2086ec4c6db402e2cef87262800cadaede1c55b`. No snapshot or GitHub-hosted workflow.

### 2026-09-02 - Persistent semantic entity naming

- Context: New missions were still labeled `Mission 1`/`Mission 2`, and missions and vessel callsigns lacked direct rename controls while groups exposed only a less-obvious identity form.
- Decision: Assign operator-created unnamed missions from a collision-safe maritime operation-name sequence; create AI drafts as `Operation Pending` and rename them from the accepted provider's first bounded strategy; treat manual renames as operator ownership so later advisor runs cannot overwrite them. Keep vessel designations immutable while allowing unique callsign changes, and validate unique mission/group/vessel names server-side.
- Files: `internal/domain/fleetops.go`, `internal/fleetops/manager.go`, `internal/api/fleetops.go`, `internal/api/server.go`, `internal/agent/manager.go`, `ai/keelmesh_ai/service.py`, `web/src/types.ts`, `web/src/FleetWorkspace.tsx`, `web/src/app.css`, and associated tests.
- Commands/tests: full Go suite and vet on VM 214; local strict TypeScript, seven Vitest assertions, Vite production build, Ruff, and nine pytest assertions; Compose validation; live OpenAI and CRUD smoke; browser DOM inspection; thirteen central/node health checks.
- Result: Newly created entities now have readable, persistent identities and all three entity types can be renamed through stale-safe APIs. Existing user names are preserved. Implementation commit `ca1342b` is pushed; VM 214 and all twelve nodes run binary SHA-256 `e885764330c80aeb3954087ab42ebf2d17e0d4b4febdf2edd32e353111c8cc12`. No snapshot or GitHub-hosted workflow.

### 2026-09-02 - Race-safe mission deletion and centered controls

- Context: The first durable-delete implementation removed PostgreSQL rows synchronously but did not serialize older `persistAsync` snapshots. A snapshot captured during mission creation could finish after deletion and reinsert the row. The native confirmation dialog was also easy to suppress or miss, and inherited two-row button CSS placed action icons in the upper half of each tab.
- Decision: Invalidate stale persistence generations, serialize asynchronous snapshots and deletes behind a dedicated mutex, fail the API without changing memory if the durable transaction cannot commit, replace native confirmation with a visible application dialog, and reset action-button grid rows explicitly.
- Files: `internal/fleetops/manager.go`, `web/src/FleetWorkspace.tsx`, `web/src/app.css`, and `web/e2e/mission.spec.ts`.
- Commands/tests: strict TypeScript, seven Vitest assertions, production frontend build, Playwright discovery, full Go tests/vet, live browser DOM/geometry inspection, and a create-immediate-delete test followed by core restart and direct API/PostgreSQL checks.
- Result: Delete confirmation is always visible in the workspace, failure cannot be reported as success before durable commit, stale snapshots cannot resurrect deleted workspaces, and every pause/trash icon is centered with zero measured x/y offset. Implementation commit `22943bc` is pushed; VM 214 and all twelve nodes run binary SHA-256 `882858d8fce265d1ca79699247e82f176cf10f4985261b52fb1afc515ab8de60`. No snapshot or hosted workflow.
- Follow-up: None.

### 2026-09-02 - Mission lifecycle and durable deletion

- Context: Active and draft workspaces could both be named `Mission 1`; deleting a draft removed only in-memory state, so its PostgreSQL row resurrected it after core restart. Mission controls were also split between tab clicks and a right-click menu.
- Decision: Allocate the next unused numbered name, repair retained duplicate names deterministically, delete mission-owned drafts/plans/workspace rows transactionally, and replace the mission context menu with direct pause/resume and confirmation-gated trash controls in both the mission row and planner window. Clicking a mission row always opens or restores its planner; minimizing returns it to the mission row.
- Files: `internal/fleetops/manager.go`, `internal/fleetops/manager_test.go`, `web/src/FleetWorkspace.tsx`, `web/src/app.css`, and `web/e2e/mission.spec.ts`.
- Commands/tests: all Go tests/vet on VM 214; local strict TypeScript, seven Vitest assertions, Vite production build, Playwright discovery; deployed browser inspection; temporary mission API delete followed by core restart; twelve-node hash and health verification.
- Result: Existing missions are uniquely displayed as `Mission 1` and `Mission 2`, drafts can be permanently deleted, and mission control follows one direct interaction model. Pause remains available only for executing/paused missions, matching backend state transitions. Implementation commit `ecdf8cb` is pushed; VM 214 and all twelve vessel nodes run binary SHA-256 `5c8c74aac153c7e0ab4f8f4203665eb4ea45b0f5864501ed7aa73d6fa0356b82`. No snapshot or hosted workflow.
- Follow-up: Full Playwright execution remains deferred until the user's current missions may be safely reset; targeted behavior was verified without mutating them.

### 2026-09-02 - Relative seaward route resolution

- Context: GPT-5.6 Luna returned three valid advisory strategies for `Travel 15nm from current location heading out to sea, then return`, but deterministic planning rejected the draft with `COMMAND_AMBIGUOUS` because no area or waypoint existed.
- Decision: Resolve bounded relative-distance commands in deterministic Go using selected-vessel positions, explicit/cardinal or fixture-backed seaward bearing, exact unit conversion, optional return-to-start, and a review corridor. Preserve the requested distance for policy evaluation; never let model prose invent coordinates or silently shorten an unsafe request.
- Files: `internal/fleetops/manager.go` and `internal/fleetops/manager_test.go`.
- Commands/tests: containerized Go tests for fleet operations, agent, and API; production Docker build; live API test using the exact sentence and GPT-5.6 Luna.
- Result: The live workflow returned geometry source `intent:relative-seaward:15.0nm`, zero ambiguities, three accepted OpenAI strategies, and three deterministic plan cards. All three were honestly prohibited because the 55.56 km round trip exceeds the retained 25 km route/energy envelope. The temporary verification mission was deleted. Commit `7691de8` is pushed; VM 214 and all twelve vessel nodes are healthy and run binary SHA-256 `873149509eaedc2056f6e8293d3d0c331ccbe097ed0b032748f35f42a097f67e`. No snapshot or GitHub-hosted workflow.

### 2026-09-02 - Centered New Mission control

- Context: The add icon inherited the normal mission tab's seven-pixel status grid column, so it appeared at the left of an otherwise empty tab.
- Decision: Collapse the add tab to one full-width/full-height grid cell and center its icon.
- Files: `web/src/app.css`, `web/e2e/mission.spec.ts`.
- Commands/tests: strict TypeScript, seven Vitest assertions, production build, and browser bounding-box verification.
- Result: The icon is centered within two pixels on each axis and deployed to VM 214 plus all twelve nodes.
- Follow-up: None.

### 2026-09-02 - Logical workspace window and mission lifecycle semantics

- Context: Primary navigation tools and transient detail cards previously shared ambiguous minimize/close behavior and competed for the lower workspace edge.
- Decision: Classify Fleet, Mission, Arena, Resilience, Quiet Fleet, Engineer, and Cutaway as top-nav primary windows; only minimized vessel/group/detail windows enter the bottom taskbar. Add mission-tab pause/resume/delete and restyle Cutaway in the shared graphite/amber system.
- Files: `web/src/WindowManager.tsx`, `web/src/FleetWorkspace.tsx`, `web/src/app.css`, `web/e2e/mission.spec.ts`, `internal/fleetops/manager.go`, `internal/api/fleetops.go`, `internal/api/server.go`, `internal/fleetops/manager_test.go`.
- Commands/tests: strict TypeScript, seven Vitest assertions, production frontend build, all Go tests/vet in Go 1.27, Compose build/health, ten existing Playwright workflows plus the corrected dedicated window workflow.
- Result: Pause freezes executing mission movement; resume preserves plan authority; delete releases vessels and removes mission plans/leases. VM 214 and all twelve vessel nodes run binary SHA-256 `306a0cb2fa84937b628c4fa53111f615cc357052f7cb637e7636b84e2c6144a6` and passed health checks. No snapshot or GitHub-hosted workflow ran.
- Follow-up: Rehearse primary and context-window behavior through the current Quick Tunnel.

### 2026-09-02 - Selection visibility and fleet status polish

- Context: Selected vessels were not visually prominent enough; Fleet / Groups lacked direct status buttons, old saved-collection chips consumed space, and native scrollbars clashed with the workspace.
- Decision: Render a 21 px amber selection ring around selected map vessels, strengthen selected rail rows, add eye-button vessel/group status, include real group aggregate metrics, remove collection chips/save action, and theme all workspace scrollbars.
- Files: `web/src/FleetWorkspace.tsx`, `web/src/OperationsMap.tsx`, `web/src/app.css`, and `web/e2e/mission.spec.ts`.
- Commands/tests: strict TypeScript, seven Vitest assertions, production Vite build, full eleven-test Playwright suite, central plus twelve-node binary/health verification.
- Result: Map and rail selection is obvious, live status is one click away at both scopes, Fleet / Groups is less cluttered, and scrolling matches Navy/Pirate palettes. Binary SHA-256 is `ec74e473578ef84b67c41689b48a875f2fa5e2b6ff74953b98df9c880ffc62be`. No snapshot or GitHub-hosted workflow.

### 2026-09-02 - Unified fleet selection surface

- Context: Fleet / Groups and the top selected-assets ribbon/drawer represented the same selection twice and consumed map space.
- Decision: Keep all selection state and group manipulation in Fleet / Groups. Automatically restore and focus that window whenever a non-empty map, group, filtered, collection, triple-click, or four-click selection occurs.
- Files: `web/src/FleetWorkspace.tsx`, `web/src/WindowManager.tsx`, and `web/e2e/mission.spec.ts`.
- Commands/tests: strict TypeScript, seven Vitest assertions, production Vite build, targeted three-test browser run, full eleven-test Playwright run, central and twelve-node health/hash verification.
- Result: The ribbon/drawer is no longer rendered; selected rows and group counts remain highlighted in Fleet / Groups, clear and mission creation stay in the same window, and minimized/closed Fleet / Groups reopens on selection. Binary SHA-256 is `1b8f7bca0c724b00bb606f825edc28189af2fe2314204b148b1d3360c78dde04`. No snapshot or GitHub-hosted workflow.

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

### 2026-09-02 - Inline group reassignment in vessel lists

- Context: The selected-assets drawer required dragging into a detached color strip, and Fleet/Groups rows offered no direct drag or context-menu reassignment.
- Decision: Render every operational group as an inline drawer destination, make Fleet/Groups sections drop targets, and expose one shared portal-based vessel context menu in both lists. The menu lists all groups, marks the current group, and can create a named group containing the selected vessel. Preserve the existing version-checked exclusive-membership API and active-authority fail-closed behavior.
- Files: `web/src/FleetWorkspace.tsx`, `web/src/OperationsMap.tsx`, `web/src/app.css`, and `web/e2e/mission.spec.ts`.
- Commands/tests: strict TypeScript, seven Vitest assertions, production Vite build; targeted and full eleven-workflow Playwright runs on VM 214; central plus twelve-node binary and health verification.
- Result: Jaeger can be dragged directly from Block Guard into Bay Lantern in either list, reassigned from a right-click group picker, or used to create a new named group. Drag completion no longer toggles/clears selection; Escape closes the assignment menu without triggering the global clear-selection shortcut. VM 214 and all twelve vessel nodes run binary SHA-256 `320885ef2e8c82029213ddddfa6489fac3150d84d0a3189aadb15026c5eab5fc`. No snapshot or GitHub-hosted workflow.

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

### 2026-09-02 - Contact follow command and simulation time control

- Context: Neutral surface contacts needed a group-scoped right-click follow action, and the fixed demo clock needed a compact SimCity-style fast-forward control.
- Decision: Keep contacts non-commandable; resolve follow by stable Boat ID through the existing AI advisor and deterministic plan/preview/authorization boundary. Expose only pause, 1×, 5×, 20×, 100×, and 500× and implement acceleration by repeating the same 200 ms authoritative step.
- Files: `internal/domain/fleetops.go`, `internal/fleetops/manager.go`, `internal/fleetops/manager_test.go`, `internal/api/fleetops.go`, `internal/api/server.go`, `web/src/types.ts`, `web/src/OperationsMap.tsx`, `web/src/FleetWorkspace.tsx`, `web/src/app.css`, and `README.md`.
- Commands/tests: strict TypeScript, seven Vitest assertions, production Vite build, full Go tests/vet, live pause/5×/500× API timing, browser control state/interaction, central health and Quick Tunnel health, and twelve-node binary/health verification.
- Result: A complete selected group can right-click neutral traffic and create a bounded follow proposal. Fleet Operations can pause or accelerate controlled vessels and neutral surface traffic on one clock without skipping route, energy, or guardrail evaluation. Implementation commit `35a1171` is pushed. VM 214 and all twelve nodes run binary SHA-256 `b4d25948f17da32b4811888bc5ef99c2c9aaca3c9ffbe7a47cb5cd1eee05d332`. LAN and Quick Tunnel health both return 200. No snapshot or hosted workflow.
- Follow-up: Arena retains its separate deterministic 20× clock; unify it with the operator control only if cross-workspace time coupling is desired.

### 2026-09-02 - Mission-centered map interaction refactor

- Context: Permanent map-authoring controls and overloaded right-click menus blurred awareness, mission drafting, and authority; manual planning also needed a provable no-LLM path.
- Decision: Put rectangle assignment and all mission geometry inside the active Mission Planner, keep Fleet/Groups and Mission Planner floating with optional left/right snap, make water/land context inspection-only, reduce controlled-vessel context to awareness plus bounded group hold, and open neutral contacts as uncommitted planner seeds. Split manual deterministic generation from AI-assisted interpretation while preserving one preview/hash/authorization/start boundary.
- Files: `internal/api/fleetops.go`, `internal/domain/fleetops.go`, `internal/fleetops/manager.go`, `internal/fleetops/manager_test.go`, `web/src/FleetWorkspace.tsx`, `web/src/OperationsMap.tsx`, `web/src/WindowManager.tsx`, `web/src/types.ts`, `web/src/app.css`, `web/e2e/mission.spec.ts`, and `README.md`.
- Commands/tests: full Go tests and vet; strict TypeScript; seven Vitest assertions; production Vite and Docker builds; a live API smoke proving three manual strategies with deterministic provider state; all fifteen Playwright Operations workflows; LAN and Quick Tunnel health; central plus twelve-node health/hash verification.
- Result: Implementation commit `70630db` is pushed to `main`. VM 214 and all twelve vessel nodes are healthy on binary SHA-256 `a56fceca232fe5d5c03ebf550b729e0bcc009342bd1c7dc39b7db01a6695b73b`. No GitHub-hosted workflow or Proxmox snapshot ran.
- Follow-up: Polygon vertex editing remains a future enhancement; current area editing supports focus, replacement by redraw, delete, clear by type, and undo.

### 2026-09-03 - Selection-optional mission planning

- Context: The mission-bar `+` refused to open Mission Planner unless vessels were selected first.
- Decision: Treat selection as an optional convenience, not a prerequisite. An empty click creates a persistent zero-asset draft; an existing selection is still carried into the new draft for compatibility. Asset assignment lives inside Planner and can use a complete group or any mixed Fleet selection.
- Files: `web/src/FleetWorkspace.tsx`, `web/src/app.css`, `web/e2e/mission.spec.ts`, and `internal/fleetops/manager_test.go`.
- Commands/tests: focused Go manager test; strict TypeScript; seven Vitest assertions; production Vite/Docker builds; all sixteen Playwright Operations workflows; LAN/Quick Tunnel health; central plus twelve-node health/hash verification.
- Result: Implementation commit `bffe313` is pushed to `main`. VM 214 and all twelve nodes are healthy; vessel nodes run binary SHA-256 `cb6d72a454b9975f6b6b5abb9c8bec952aaf6b8229f75682102896caaeda8f5e`. No GitHub-hosted workflow or Proxmox snapshot ran.
- Follow-up: The current AI strategy call still requires assigned targets before route generation; external-agent-driven target selection remains part of the broader typed workspace-tool work rather than implicit model authority.

### 2026-09-03 - Oriented idle formation holding

- Context: Group members occupied separate slots, but each hull retained its last bearing toward its slot, making the group look like every boat was pointing into one common point.
- Decision: Add versioned `formation_heading_deg` group state; rotate canonical formation offsets in local metre space around the assembly point; align every idle member to that heading while preserving independent slot, spacing, route, authority, and energy state.
- Files: `internal/domain/fleetops.go`, `internal/fleetops/manager.go`, `internal/fleetops/manager_test.go`, `web/src/types.ts`, `web/src/FleetWorkspace.tsx`, and `web/e2e/mission.spec.ts`.
- Commands/tests: full Go tests/vet; strict TypeScript; seven Vitest assertions; production Vite/Docker builds; focused group-manager Playwright workflow; live eight-group heading inspection; LAN browser visual inspection; twelve-node deployment/health verification.
- Result: Implementation commit `cfa1faf` is pushed to `main`. VM 214 is live and all twelve vessel nodes run binary SHA-256 `4c9c6530a7dcc01a4073afaeebd6be650a9c85b3b77146d1b93ebea9dc0a8f07`. No GitHub-hosted workflow or Proxmox snapshot ran.
- Follow-up: If desired, route legs can later derive and commit a new formation heading automatically from each leg bearing; current group heading is explicitly operator-controlled and remains stable through station keeping.

### 2026-09-03 - Mission roster, docking, voice, and loop behavior

- Context: The operator wanted only Fleet/Groups and Mission Planner attachable, deterministic first-open placement, less Planner chrome, automatic persona voice, mission membership tied to Fleet selection, relocated environment/time controls, and explicit loop-versus-hold completion.
- Decision: Default Fleet to a 245 px left attachment and Planner to a 350 px right attachment while preserving floating and vertical resize. Remove docking from every other window. Make current Fleet selection the active mission roster and treat mid-execution membership or loop changes as a fail-closed replan boundary. Use Jarvis automatically in Navy mode and Captain Barbossa automatically in Pirate mode. Keep full signed trajectories and restart loops through a fresh revision from actual final pose.
- Files: `internal/domain/fleetops.go`, `internal/fleetops/manager.go`, `internal/fleetops/manager_test.go`, `web/src/FleetWorkspace.tsx`, `web/src/OperationsMap.tsx`, `web/src/WindowManager.tsx`, `web/src/types.ts`, `web/src/app.css`, and `web/e2e/mission.spec.ts`.
- Commands/tests: strict TypeScript; seven Vitest assertions; Vite production build; full Go tests and vet in the pinned Go 1.27 container; production Docker build; live in-app-browser geometry/state inspection; all seventeen serial Playwright Operations/Arena workflows; central and twelve-node binary/hash/health verification; LAN and Quick Tunnel health.
- Result: VM 214 and all twelve vessel nodes run binary SHA-256 `44e786331f75f6f67071a7cd18c342e1e8f6f2e92726a0295471f1e04f3bdfae`. The Player A/main Quick Tunnel returns HTTP 200. No GitHub-hosted workflow or Proxmox snapshot ran.
- Follow-up: A future product pass can expose a compact roster-change diff before invalidating an executing mission; the current implementation performs the safer immediate station-keep and exact reauthorization path.

### 2026-09-03 - AI-selected mission rosters

- Context: An empty mission draft rejected `Move a group 1 nm east and hold position` with `TARGETS_REQUIRED` before the AI was contacted.
- Decision: Add a bounded target-selection phase ahead of compilation. The Python AI service receives available vessel/group IDs, calls GPT-5.6 Luna with a strict schema, and returns exact target IDs plus an explanation. Fleetops revalidates IDs, movement-authority conflicts, complete-group integrity, state version, geometry, policy, and authorization. Provider failure falls back visibly to deterministic name/code/color resolution; manual planning retains the explicit-target requirement.
- Files: `ai/keelmesh_ai/service.py`, `ai/tests/test_service.py`, `internal/agent/manager.go`, `internal/agent/manager_test.go`, `internal/api/fleetops.go`, `internal/domain/fleetops.go`, `internal/fleetops/manager.go`, `internal/fleetops/manager_test.go`, `web/src/FleetWorkspace.tsx`, and `web/src/types.ts`.
- Commands/tests: twelve Python tests plus Ruff/mypy; full Go tests/vet in Go 1.27; strict TypeScript, seven Vitest assertions, Vite production build; all seventeen Playwright workflows; live empty-roster API proof using the exact eastbound hold request.
- Result: Live GPT-5.6 Luna selected all six members of `WS · Watch Shoal`, returned an accepted provider receipt, and the system persisted the roster, generated one exact 1 nm east hold point, reported zero ambiguities, and produced three OpenAI strategies. VM 220 independently passed the same empty-roster workflow through its direct node OpenAI route. The temporary verification missions were cleared. Implementation commit `0b347d1` is pushed; VM 214 and all twelve vessel nodes are healthy on binary SHA-256 `a9328c7d6d1ecdd3179cb99b410fc3dc782fe2b1bf070d28210a59bb62640322`. VM 214 Docker build cache reached 100% filesystem use during repeated builds; only regenerable builder cache was pruned, reclaiming 9.929 GiB. No volumes, evidence, application data, snapshots, or GitHub-hosted workflows were touched.
- Follow-up: Keep VM 214 build-cache growth monitored; prune only regenerable cache when needed and preserve the snapshot/storage safety rules.

### 2026-09-03 - Unified hover help

- Context: Controls with HTML `title` attributes displayed both the browser-native tooltip and KeelMesh's styled hover card.
- Decision: Normalize static, dynamically mounted, and late-updated titles into `data-help`, remove the native title trigger, and keep the shared amber-border tooltip as the sole hover-help presentation.
- Files: `web/src/HoverHelp.tsx`, `web/src/app.css`, and `web/e2e/mission.spec.ts`.
- Commands/tests: strict TypeScript; seven Vitest assertions; Vite production build; deployed focused Playwright hover regression; central and twelve-node health/hash verification.
- Result: VM 214 and all twelve vessel nodes run the single-tooltip build with binary SHA-256 `b6d4146cbbc7a8a86dd10e4a566b9560d82a3dd2c828865a12e717666ca840e4`. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - Mission-only route playback

- Context: Fleet/Groups still exposed a legacy group play/loop button even though waypoint routes are authored and controlled through missions.
- Decision: Remove the route button and its client-only cycling path from Fleet/Groups while retaining the bounded group-hold command and backend compatibility interfaces.
- Files: `web/src/FleetWorkspace.tsx`, `web/src/app.css`, and `web/e2e/mission.spec.ts`.
- Commands/tests: strict TypeScript; seven Vitest assertions; Vite production build; focused deployed Playwright Fleet/Groups workflow; central and twelve-node health/hash verification.
- Result: VM 214 and all twelve vessel nodes run binary SHA-256 `7ef37d97730c84f5698995cebb56a59889ec0c2563cd39a7b3391a18e348ba7c`. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - Nominal range and daylight solar calibration (superseded underway behavior)

- Context: The old planner capped routes at 25 km (13.5 nm), used a separate reserve approximation, and disagreed with the runtime propulsion model; solar recharge was too weak to read clearly in the accelerated demonstration.
- Decision: Use one class-aware cubic propulsion model for planning and execution, calibrated at 1.8 m/s to 20 nm Kestrel, 30 nm Mariner, and 45 nm Atlas battery-only nominal range. Begin the demo solar day at 08:00, use 4/9/20 kW peak arrays, and expose nominal remaining range plus peak solar output in Vessel Inspector. Increase default mission bounds to 85 km and 12 hours; battery and reserve policy still reject infeasible routes per vessel.
- Files: `internal/domain/fleetops.go`, `internal/fleetops/manager.go`, `internal/fleetops/manager_test.go`, `web/src/types.ts`, `web/src/FleetWorkspace.tsx`, and `web/e2e/mission.spec.ts`.
- Commands/tests: full Go tests/vet in Go 1.27; strict TypeScript; seven Vitest assertions; Vite production build; live class-contract inspection; focused deployed Playwright vessel-inspector workflow; central and twelve-node health/hash verification.
- Result: At full sun, a stationary Kestrel gains about 21% charge per simulated hour. The former solar-positive underway behavior was superseded on 2026-09-03 because it pinned moving vessels at full charge; daylight now offsets at most 65% of underway load. The historical deployment used binary SHA-256 `9fa3c69fa39f4ff73241799c3c3b4c131a7f3b5bf97d2fcfad914c1bfbce984a`. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - Continuous contact tracking and anchored traffic

- Context: Contact inspectors led with knots, making the 2.8 m/s NPC cap appear to exceed 4 m/s; follow missions ended after one finite predicted route; mission completion/deletion could leave stale overlays or pull released groups toward an old assembly point.
- Decision: Lead with m/s and show knots secondarily. Add four zero-speed anchored contacts beside the twelve bounded moving contacts. Bind contact identity, a 60-second refresh cadence, and a 180-second horizon into follow-plan hashes, then generate signed future trajectory revisions from current vessel/contact state. On completion/deletion, replace the group's assembly point with the released vessels' current centroid. Hide completed/ended mission geometry and render live follower-to-contact lines for continuous tracking.
- Files: `internal/domain/fleetops.go`, `internal/fleetops/manager.go`, `internal/fleetops/manager_test.go`, `web/src/types.ts`, `web/src/OperationsMap.tsx`, `web/src/FleetWorkspace.tsx`, and `web/e2e/mission.spec.ts`.
- Commands/tests: full Go tests/vet in Go 1.27; strict TypeScript; seven Vitest assertions; Vite production build; focused deployed Playwright surface-traffic workflow; live API and browser inspection; LAN/Quick Tunnel health; central and twelve-node binary/hash/health verification.
- Result: Commit `8743a50` is pushed. VM 214 and all twelve vessel nodes are healthy on binary SHA-256 `531a1068fa8f9e5ba2de6434dcdaef69d3d6234da82b670e376ddb24eeb098e4`. Live state reports 16 contacts: 12 underway, four anchored, and 2.8 m/s maximum. VM 214 root storage is 30% used. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - Anchored-contact AI contract repair

- Context: Sending `approach and surround Safe Haven` returned `Target-aware fallback used: AI service unavailable` and left the planner without the expected AI options.
- Cause: the new anchored contacts expose one fixed anchorage point, while the Python `SurfaceContact` boundary still required every contact route to contain at least two points. FastAPI rejected the otherwise valid planning context with HTTP 422 even though the AI container itself was healthy.
- Decision: accept one-point routes as valid anchored-contact context and mount the existing read-only OpenAI runtime secret into core as an independent direct-provider fallback.
- Files: `ai/keelmesh_ai/service.py`, `ai/tests/test_service.py`, and `compose.yaml`.
- Commands/tests: 13 Python tests; Ruff; mypy; Compose validation; live AI/core health; temporary end-to-end anchored Safe Haven mission compile/plan/delete.
- Result: the live service returned three accepted GPT-5.6 Luna strategies and three deterministic plans for anchored Safe Haven. The temporary probe mission was deleted. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - Model-first dynamic contact objectives

- Context: `approach and surround Safe Haven` reached GPT-5.6 Luna, but deterministic compilation ran first and rejected the request for missing area/waypoint geometry. Contact language also needed to distinguish follow, go-to/approach, observe, intercept, and surround without freezing a moving vessel into one coordinate.
- Decision: Run a strict-schema semantic command interpreter before route compilation. Bind accepted output to supplied contact IDs and carry behavior, dynamic-target status, stand-off, formation, reserve/speed limits, hold intent, provider, and attempts into the immutable draft and plan. Deterministic Go remains authoritative for contact-relative geometry, safety, hashes, leases, and rolling trajectory revisions. A multi-vessel surround always materializes as a ring even if an advisory strategy suggests another approach formation.
- Files: `ai/keelmesh_ai/service.py`, `ai/tests/test_service.py`, `internal/agent/manager.go`, `internal/agent/manager_test.go`, `internal/api/fleetops.go`, `internal/domain/fleetops.go`, `internal/fleetops/manager.go`, `internal/fleetops/manager_test.go`, and `web/src/types.ts`.
- Commands/tests: full Go tests/vet passed in Go 1.27; Python Ruff, strict mypy, and 14 pytest tests passed; strict TypeScript, seven Vitest assertions, and production Vite/Docker builds passed. Live API probes proved exact Safe Haven surround and moving Blue Finch go-to/follow language, then removed all temporary missions.
- Result: VM 214 and all twelve vessel nodes are healthy on binary SHA-256 `a0a28875a89d51e11e5082ef018b939a52cc0b62f149d6a6884a06a665c1172c`. The exact Safe Haven request produced three OpenAI strategies and three ring plans, all dynamic and bound to `surface-16`, with zero ambiguity. VM 220 independently resolved `go to Blue Finch` through its direct OpenAI route and then deleted the temporary mission successfully. Standalone node startup now disables the unavailable central PostgreSQL mission-workspace path, avoiding DNS waits while retaining node-local signed trajectory/journal execution. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - Global voice command orb

- Context: The operator needed one always-available push-to-talk control that could answer questions verbally, arrange the workspace, or begin mission authoring without exposing ordinary conversation text or opening Mission Planner unnecessarily.
- Decision: Add a fixed bottom-right state orb and a strict global assistant contract. The model classifies each utterance as conversation, workspace, or mission and may only return allow-listed presentation/selection/inspection/theme/simulation-rate actions or a mission-draft request. Authorization, execution, deletion, and simulated effects remain on existing deterministic approval paths.
- Files: `internal/domain/workspace_assistant.go`, `internal/agent/manager.go`, `internal/agent/manager_test.go`, `internal/api/arena.go`, `internal/api/server.go`, `compose.yaml`, `web/src/FleetWorkspace.tsx`, `web/src/types.ts`, `web/src/app.css`, and `web/e2e/mission.spec.ts`.
- Commands/tests: full Go tests/vet; strict TypeScript and seven Vitest assertions; production Vite/Docker build; Compose validation; live GPT-5.6 Luna conversation/workspace/mission classification; focused Playwright placement/no-overflow regression; LAN and Quick Tunnel health; direct node-220 GPT-5.6 Luna workspace action; twelve-node binary/health verification.
- Result: Commit `ab8e2ad` is pushed to `main`. VM 214 and all twelve nodes run the global voice UI and bounded agent endpoint. The node binary SHA-256 is `a5c9c898366f6f7db49769ab9afc3ccdc1e49c85c24c3c4b698733765eb76408`. A deployment defect that made the OpenAI runtime copy unreadable to the non-root Go UID was fixed with separate Go/Python UID-owned `0400` volumes. No GitHub-hosted workflow or Proxmox snapshot ran.
- Follow-up: Complete the recorded multi-noise speech benchmark; browser audio autoplay behavior should be rehearsed through the HTTPS Quick Tunnel on the interview device.

### 2026-09-03 - Conversational three-option mission execution

- Context: Mission Planner exposed redundant route schematics, manual generation buttons, voice-status chrome, and a multi-step preview/authorize/start flow after the AI responded.
- Decision: Keep chat primary, require exactly three model strategies, and place compact A/B/C cards immediately below it. Collapse manual Objective and Map Authoring controls at the bottom. A card uses one exact-hash confirmation; a conversational A/B/C selection is the equivalent approval and executes through the same server-side preview, lease, policy, and idempotency checks.
- Files: `ai/keelmesh_ai/service.py`, `ai/tests/test_service.py`, `internal/agent/manager.go`, `internal/agent/manager_test.go`, `internal/api/fleetops.go`, `internal/api/server.go`, `internal/domain/workspace_assistant.go`, `internal/fleetops/manager.go`, `web/src/FleetWorkspace.tsx`, `web/src/app.css`, `web/src/types.ts`, and `web/e2e/mission.spec.ts`.
- Commands/tests: full Go tests/vet; Python Ruff/mypy/14 pytest tests; strict TypeScript, seven Vitest assertions, Vite production build; all twenty serial Playwright Arena/mission/responsive workflows; central and twelve-node health/hash verification.
- Result: VM 214 and all twelve vessel nodes run binary SHA-256 `ae69779a47cd48ef446d7710ba66095a1680f92355ecd9e07b7a48e65d83662e`. The end-to-end browser test cancels a card-click confirmation, says `Go with option A`, and verifies the mission reaches executing state without a second prompt. No GitHub-hosted workflow or Proxmox snapshot ran.
- Follow-up: Rehearse browser audio autoplay and microphone permissions through the ephemeral HTTPS Quick Tunnel on the interview device.

### 2026-09-03 - M10 trusted A2UI command scenes

- Context: the chat-centered planner obscured operational context and made the assistant feel like a text box rather than a first-class workspace operator.
- Decision: accept typed/voice objectives through `/api/v4`, let the model produce only a strict scene intent, and let trusted Go code resolve visible entities, state, actions, A2UI structure, map annotations, and authority. Scope active/pinned scenes by browser session; retain deterministic/manual mission authority and exact-hash approvals.
- Files: `M10_AI_COMMAND_SURFACE.md`, `internal/domain/scenes.go`, `internal/agent/scenes.go`, `internal/agent/mcp_control.go`, `internal/api/scenes.go`, `web/src/A2UISurface.tsx`, `web/src/FleetWorkspace.tsx`, `web/src/OperationsMap.tsx`, `web/src/types.ts`, `web/src/app.css`, and tests.
- Commands/tests: full Go test/vet; strict TypeScript, eight Vitest assertions, and Vite production build; Ruff formatting/lint, mypy, Bandit, and 14 Python tests; M1-M7 live verifiers; full serial Playwright suite plus a three-run map multi-click regression; central and twelve-node health/hash checks.
- Result: trusted ordered A2UI create/update/delete, live bindings, session isolation, scene lifecycle/history, map callouts/camera, Mission Canvas integration, deterministic critical scenes, and scene-aware MCP are deployed. The final node binary SHA-256 is `5b16da3607a26dab9c5446185d191018075a37681f964b0702b4a16c5443d392`. M3 attempted/accounted 121,800 events with zero drops; M4 replay checksums matched; M6 verifies 48 vessels and 16 contacts; M7 verifies all 12 physical nodes. No GitHub-hosted workflow or Proxmox snapshot ran.
- Follow-up: persist pinned/history scenes in PostgreSQL and node-local bbolt across service restarts; current persistence is browser reload/session persistence, not core-process durability. Expand the trusted catalog from the initial operational primitives into richer native plan/metric/evidence components and complete pending Approval Card orchestration for every consequential semantic tool.

### 2026-09-03 - Consumable mission route overlays

- Context: executing missions left their full authorized polylines and passed waypoint markers on the map, visually creating a historical trail.
- Decision: preserve immutable authority/replay data while deriving a live remaining-route projection from trajectory cursor progress and current vessel pose. Remove a waypoint only after every relevant assigned vessel has passed it.
- Files: `web/src/OperationsMap.tsx`, `web/src/routeProgress.ts`, `web/src/routeProgress.test.ts`, and `web/e2e/mission.spec.ts`.
- Commands/tests: strict TypeScript; 11 Vitest assertions; Vite/Docker production build; focused deployed Playwright route-consumption workflow; central and twelve-node health/hash verification.
- Result: commit `384eb34` is pushed. VM 214 and all twelve vessel nodes run the change; node binary SHA-256 is `93288a39cab7d8d40f4ff31189e7d2c820bb40a95688a6144c9011405cbe9daa`. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - Quieter explicit hover help

- Context: the global tooltip fallback treated structural accessibility labels as hover help, causing low-value messages such as `Map` whenever the pointer crossed the operating canvas.
- Decision: make styled hover help explicit through `data-help` or authored `title` copy only. Preserve accessibility labels independently and retain rich map-object telemetry for vessels, contacts, waypoints, environmental vectors, and hold points.
- Files: `web/src/HoverHelp.tsx` and `web/e2e/mission.spec.ts`.
- Commands/tests: strict TypeScript; 11 Vitest assertions; Docker production build; focused deployed Playwright map-hover regression; central and twelve-node health/hash verification.
- Result: commit `196590c` is pushed. VM 214 and all twelve vessel nodes run binary SHA-256 `685321d1cdb4f93477188fda7f1fb1a823ce5c91b6f360a9fc5e34f941657b37`. No GitHub-hosted workflow or Proxmox snapshot ran.

### 2026-09-03 - AI memory architecture audit

- Context: the operator observed that follow-up requests did not behave as if the assistant remembered prior conversation.
- Finding: the observation is accurate. Global assistant calls include rich current world state but no previous user/assistant turns. OpenAI storage is disabled, command scenes are not fed back to the model, and core restart clears scene/turn state. Only the latest 12 messages inside one mission are provided to the mission-strategy advisor and persisted with that mission on VM 214.
- Follow-up: add a structured, inspectable operator-session memory ledger with bounded recent turns, entity-reference resolution, summarized tool/scene receipts, mission linkage, durable PostgreSQL/node checkpoints, and selective retrieval. Never store secrets or treat remembered text as authority; every effect remains state/version/hash validated.

### 2026-09-03 - Intentional mission options and predictive intercept

- Context: routine mission language always exposed three strategies and Mission Canvas, while assistant mission/group management was incomplete and contact-follow routes aimed independently at the contact instead of a shared lead point.
- Decision: request one strategy by default and three only for explicit options/alternative/comparison language. Keep routine scenes hidden, accept a natural exact-plan confirmation, and allow the model to return a clarifying conversation with no mutation. Add typed open/pause/resume/delete mission and create/delete/reassign group actions. Compute one bounded shared contact intercept from all assigned vessels and continuously refresh it; deterministic route/policy/lease authority remains unchanged.
- Files: `internal/agent/manager.go`, `internal/agent/manager_test.go`, `internal/domain/fleetops.go`, `internal/domain/workspace_assistant.go`, `internal/fleetops/manager.go`, `internal/fleetops/manager_test.go`, `web/src/FleetWorkspace.tsx`, `web/src/OperationsMap.tsx`, `web/src/types.ts`, and `web/e2e/mission.spec.ts`.
- Commands/tests: full Go tests and vet; strict TypeScript; 13 Vitest assertions; production Vite/Docker builds; live one-plan hidden-canvas confirmation; explicit three-option rendezvous; live ETA/hold suppression; Fleet/group management; three consecutive group create/move/delete workflows; VM 214 plus twelve-node health/hash checks.
- Result: VM 214 and all twelve vessel nodes are healthy on binary SHA-256 `2bc7b150e3030e37e51023ebd23a561e12bc391304f843064a8225568179ca72`. Scenario reset restores 20x pacing and asynchronous persistence coalesces stale bursts so durable group deletion is responsive. No GitHub-hosted workflow or Proxmox snapshot ran.

## Open Follow-ups

- Complete M8D automatic node-agent interruption escalation. The deployed controller already performs bounded deterministic collision/energy correction and an operator can request a live AI replan that becomes a future exact revision, but a complex out-of-envelope encounter does not yet autonomously invoke the node OpenAI route. Add a typed asynchronous proposal queue, bounded context, deterministic corridor/path validation, provider timeout/fallback, and approval only when authority expands.
- Extend M9 from the deployed election/MCP vertical slice to signed group proposals, all-affected arming, synchronized future activation, simulated decision-node loss/re-election, and executable pre-authorized signal-seek/return-home corridors. Add expiring observer/operator-assistant/diagnostic MCP identities, rotation, per-tool budgets, and security fuzzing before exposing `/mcp/control` through an authenticated management ingress.
- When all generated strategies are prohibited, keep the cards visible but add a concise constraint explanation and a bounded revision action. The verified 15 nm outbound-plus-return request is 55.56 km, so the current 25 km route limit and energy model correctly prohibit all three strategies rather than silently shortening the operator's order.
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
