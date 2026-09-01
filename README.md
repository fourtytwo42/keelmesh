# KeelMesh Fleet Intent Control

KeelMesh is an interview demonstration of user-friendly mission programming,
offline-first peer coordination, bounded edge autonomy, and observable AI/ML
infrastructure for a simulated maritime fleet.

The core thesis is:

> Program the mission once. Coordinate locally. Adapt together. Degrade safely
> when the uplink disappears.

`PRD.md` is the product source of truth. `MEMORY.md` holds durable project and
handoff context for coding agents.

## Bootstrap service

The first container is a small Go status service that proves the mission
appliance can build and run on the dedicated VM before the product modules are
added.

```bash
docker compose up -d --build
curl --fail http://localhost:8080/healthz
```

The application is available on the LAN at `http://192.168.50.214:8080`.

See `infrastructure/README.md` for the reproducible VM and host setup. Never
commit passwords, private keys, provider tokens, GitHub credentials, Cloudflare
credentials, or model secrets.

