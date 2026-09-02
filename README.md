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
identity, and run independent mission tabs. Area, exclusion, and waypoint
geometry plus typed or spoken intent produce deterministic formation options;
every option still follows preview → exact-hash authorization → execution.

The chart is a packaged NOAA NCDS extract and all vessel/environment state is
clearly labeled simulation data. Pocket TTS and faster-whisper run privately on
VM 214; Morgan is the default voice. Browser capture requires HTTPS, so the
temporary Cloudflare Quick Tunnel is used for microphone demos while the LAN URL
continues to provide the rest of the workspace.

```bash
scripts/keelmesh status
python3 scripts/verify_m6.py http://localhost:8080
```

The deterministic VM-local STT smoke measurement is recorded in the M6 plan.
Browser WebGPU/WASM and trusted-peer speech routes remain measured follow-up
work and are not represented as completed physical-node redundancy.

See `infrastructure/README.md` for the reproducible VM and host setup. Never
commit passwords, private keys, provider tokens, GitHub credentials, Cloudflare
credentials, or model secrets.
