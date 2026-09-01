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

See `infrastructure/README.md` for the reproducible VM and host setup. Never
commit passwords, private keys, provider tokens, GitHub credentials, Cloudflare
credentials, or model secrets.
