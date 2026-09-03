# KeelMesh Fleet Intent Control

KeelMesh is an interview demonstration of user-friendly mission programming,
offline-first peer coordination, bounded edge autonomy, and observable AI/ML
infrastructure for a simulated maritime fleet.

The core thesis is:

> Program the mission once. Coordinate locally. Adapt together. Degrade safely
> when the uplink disappears.

`PRD.md` is the product source of truth. `MEMORY.md` holds durable project and
handoff context for coding agents.

## M1 golden mission loop

The single offline appliance now serves the React/TypeScript/MapLibre operator
interface and the Go mission authority. Select or draw a polygon, compile typed
intent, compare two computed plans, preview without moving the fleet, authorize
the exact SHA-256 plan hash, and watch six simulated vessels execute it.

```bash
docker compose up -d --build
curl --fail http://localhost:8080/healthz
python3 scripts/verify_m1.py
```

M1 needs no map tile service, Python runtime, model provider, Kafka, PostgreSQL,
or internet connection. The multi-stage image pins frontend and Go dependencies
and embeds the production UI in `keelmesh-core`.

The application is available on the LAN at `http://192.168.50.214:8080`.

## M2 resilient autonomy

After starting an authorized M1 mission, the Resilience Drill runs a deterministic
Vessel 4 incident: direct Starlink failure, Vessel 3 HaLow relay, full partition,
60-second signed mission-tape depletion, GNSS spoof exclusion, safe hold, and a
policy-checked bridge to future work. Manual controls and auto-run use the same
versioned fault API and state machine.

```bash
python3 scripts/verify_m2.py http://localhost:8080
```

The drill is simulation-only. Radio links and PNT evidence are modeled locally;
it makes no physical range, anti-jam, or certified-navigation claim.

## M3 infrastructure at scale

Switch the live interface from **Operator** to **Cutaway** to inspect the measured
pipeline: 1,000 deterministic background vessels, Kafka KRaft, three real
consumer child processes, PostgreSQL projections, pgvector fixture retrieval,
quarantine/redrive, cooperative rebalance, and Kafka-to-shadow replay.

```bash
docker compose up -d --build
python3 scripts/verify_m3.py http://localhost:8080 \
  --json evidence/m3.json --markdown evidence/m3.md
```

Only port 8080 is published. Kafka, PostgreSQL, workers, and control traffic stay
on the private Compose network. M1/M2 continue in degraded mode if M3 services
are unavailable. Measurements apply only to the recorded VM and workload.

## M4 AI infrastructure and autonomy tooling

Switch to **Engineer** to turn the deterministic Vessel 4 incident into a
human-approved evaluation. A separate non-root Python service collects bounded
evidence through the official MCP Streamable HTTP boundary, retrieves cited
runbooks and historical fixtures, verifies an isolated replay, and drafts an
immutable candidate. Only `demo-engineer` can approve the exact candidate hash.

Cloud inference uses an adaptive OpenRouter free-model pool with per-model
cooldowns, followed by an optional local OpenAI-compatible endpoint and the
mandatory deterministic mock. Provider failure never affects mission authority.

```bash
sudo install -m 600 /dev/null /etc/keelmesh/secrets/openrouter_api_key
# Put the runtime-only OpenRouter key in that file, then:
docker compose up -d --build
python3 scripts/verify_m4.py http://localhost:8080 \
  --json evidence/m4.json --markdown evidence/m4.md
```

## M5 interview release

M5 adds the Quiet Fleet authority demonstration: a four-vessel cell rejects an unsafe redistribution even though it has quorum, arms a safe revision on every affected node, then atomically activates the exact signed commit at a future mission-tape boundary.

VM 214 is the only release-verification environment. GitHub Actions are intentionally disabled. From `/srv/keelmesh`:

```bash
scripts/keelmesh start
scripts/keelmesh verify
scripts/keelmesh offline-verify
scripts/keelmesh restart-verify
scripts/keelmesh export-evidence
scripts/keelmesh rehearse --six-minute
```

`start` and offline verification use cached images with `--pull never`. The connected OpenRouter exercise is supplementary; deterministic mock/offline AI is the release gate. No Proxmox snapshot is created by these commands.

Only port 8080 remains published. The Python API, MCP server, generated
capability tokens, provider credential, Kafka, and PostgreSQL stay private.
CI uses an empty provider secret and proves the same workflow through mock.

## M6 fleet operations workspace

M6 makes the default interface a compact map-first operating workspace for 48
persistent named vessels in Narragansett Bay and Rhode Island Sound. Operators
can select individuals or complete groups, inspect mesh reachability separately
from authority, retain overlapping collections, manage operational-group
identity, and run independent mission tabs. Mission authoring is deliberately
contained inside each floating Mission Planner: assigned assets, operating and
exclusion areas, numbered routes, hold/orbit points, formation, constraints,
and authorization stay scoped to the active draft. Fleet/Groups can snap left,
Mission Planner can snap right, and both remain movable and floating by default.

The structured builder has separate **Generate routes · no AI** and **Ask AI**
paths. Manual planning never contacts a provider; AI-assisted planning records
the accepted provider attempt. Both compile into the same deterministic Go
planner and share preview → exact-hash authorization → execution. Empty-water
and land right-click are read-only inspections, controlled-vessel right-click
contains awareness plus bounded group hold, and contact planning opens an
uncommitted planner seed before any mission is created.

Idle operational groups also have a bounded map-navigation controller. Hold
points and numbered waypoints are draggable; group-colored routes can run once,
switch to a loop, or pause after completing the current leg. Clearing a route
creates a deterministic hold around the group's lowest vessel ID, and changing
the group hold point translates the full formation at the operator-selected
simulation rate. The lower status bar provides pause, 1×, 5×, 20×, 100×, and
500× controls; every rate advances the same bounded 200 ms simulation steps so
fast-forward does not bypass trajectory, energy, collision, or waypoint logic.
Right-click vessel controls use the same group route API.

The chart is a packaged NOAA NCDS extract and all vessel/environment state is
clearly labeled simulation data. Pocket TTS and faster-whisper run privately on
VM 214; the user-authorized Jarvis clone is the Navy-mode default, Captain
Barbossa is the Pirate-mode default, and both custom voice artifacts remain
VM-local runtime artifacts outside Git. Browser capture requires HTTPS, so the
temporary Cloudflare Quick Tunnel is used for microphone demos while the LAN URL
continues to provide the rest of the workspace.

The operating picture includes twelve fictional, non-commandable surface contacts on deterministic looped routes. Six original transparent traffic sprites cover container ship, tanker, ferry, fishing trawler, patrol ship, and sailing yacht classes and remain visually unchanged in Pirate mode. Names, simulated boat IDs, colors, motion, and predicted tracks are available to the bounded mission advisor for policy-checked intercept and follow plans. Right-clicking a contact can inspect it or seed Mission Planner; using that seed remains an explicit current/new mission choice and never creates authority by itself.

```bash
scripts/keelmesh status
python3 scripts/verify_m6.py http://localhost:8080
```

The deterministic VM-local STT smoke measurement is recorded in the M6 plan.
Browser WebGPU/WASM and trusted-peer speech routes remain measured follow-up
work and are not represented as completed physical-node redundancy.

## M7 symmetric fleet arena

M7 adds a knowledge-limited two-faction Arena and a real twelve-VM node fabric.
Each faction has six symmetric vessel nodes, a quorum-backed coordinator view,
protected management/inference and simulated-radio planes, semantic workspace
actions, deterministic energy progression, and exact-hash authorization for
fictional engagement effects. VM 214 remains the neutral authority and Player B
ingress router.

The current physical allocation is five nodes on `fourtyfour`, four on `mini42`,
and three on `mini43`. Each linked clone has 2 vCPU, 2 GiB RAM, a 12 GiB disk,
`eth0` management at `192.168.50.<VMID>`, and `eth1` simulated radio at
`10.77.0.<VMID>`. The tracked radio-fault helper rejects every interface except
`eth1` and arms a 60-second automatic rollback before applying impairment.

```bash
python3 scripts/verify_m7.py --base-url http://localhost:8080
# Player B's Quick Tunnel opens directly into Arena:
# https://<ephemeral-host>.trycloudflare.com/?arena=1
```

The Quick Tunnel hostname is deliberately ephemeral. The current release proves
the public faction boundary, coordinator-aware ingress, deterministic Arena
authority, physical dual-plane VMs, and radio-only failure safety. It does not
claim that the current central deterministic coordinator model is production
Raft, that physical HaLow radios are attached, or that twelve independent GPU
inference services are running.

See `infrastructure/README.md` for the reproducible VM and host setup. Never
commit passwords, private keys, provider tokens, GitHub credentials, Cloudflare
credentials, or model secrets.
