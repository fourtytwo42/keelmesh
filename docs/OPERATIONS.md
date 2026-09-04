# Operations runbook

## Contents

- [Environment](#environment)
- [Secrets](#secrets)
- [Lifecycle](#lifecycle)
- [Health](#health)
- [Verification](#verification)
- [Offline and restart](#offline-and-restart)
- [Reset and evidence](#reset-and-evidence)
- [Memory lab](#memory-lab)
- [Vessel nodes](#vessel-nodes)
- [Troubleshooting](#troubleshooting)

## Environment

The reference appliance runs on VM 214 with Docker Compose. Only `8080` is published. Kubernetes files describe production shape and provision no cloud infrastructure.

```bash
git clone https://github.com/fourtytwo42/keelmesh.git
cd keelmesh
docker compose config --quiet
```

## Secrets

Provider credentials are runtime files outside Git:

```bash
sudo install -o root -g root -m 0400 /dev/null /etc/keelmesh/secrets/openai_api_key
sudo install -o root -g root -m 0400 /dev/null /etc/keelmesh/secrets/openrouter_api_key
```

Populate them through an approved delivery method. Never echo secrets to logs or commit secret-bearing `.env` files.

## Lifecycle

```bash
scripts/keelmesh start
scripts/keelmesh status
scripts/keelmesh stop
```

Direct Compose operation:

```bash
docker compose up -d --build
docker compose ps
docker compose down
```

Do not add `-v` unless destructive volume deletion is explicitly authorized.

## Health

```bash
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
curl --fail http://127.0.0.1:8080/metrics
scripts/keelmesh status
```

Status reports Git state, service health, digests, disk use, provider mode, and subsystem phases where available.

## Verification

```bash
scripts/keelmesh verify
python3 scripts/verify_m7.py --base-url http://127.0.0.1:8080
python3 scripts/verify_m11.py http://127.0.0.1:8080
```

Verification runs on VM 214 and vessel nodes, not GitHub-hosted runners.

## Offline and restart

```bash
scripts/keelmesh offline-verify
scripts/keelmesh restart-verify
```

Offline verification uses `--pull never` and provider mode `offline`. Restart verification exercises AI, Worker 2, Kafka/PostgreSQL, and core while reporting real persistence boundaries.

## Reset and evidence

```bash
scripts/keelmesh reset-demo
scripts/keelmesh export-evidence
scripts/keelmesh rehearse --six-minute
```

Reset uses APIs and preserves volumes. Evidence bundles contain bounded JSON, Markdown, and checksums; they exclude credentials, raw environment values, and raw voice audio.

## Memory lab

```bash
docker compose --profile memory-lab up -d
scripts/keelmesh memory-status
scripts/keelmesh memory-verify
scripts/keelmesh memory-export
```

MinIO, Dagster, and MLflow are private and optional. Missions do not depend on this profile.

## Vessel nodes

Nodes use the same Go/UI binary and separate management from simulated-radio networking. Fault tooling may target only the radio NIC and must arm an automatic rollback watchdog.

Never sever management/inference during radio failure simulation. Never deliberately disable a node’s provider route as part of a radio drill.

## Troubleshooting

### Blank map

1. Hard refresh to replace stale hashed assets.
2. Verify `/healthz` and current JS/CSS assets return `200`.
3. Inspect browser console errors.
4. Confirm local map and MapLibre worker assets share the deployed build.

### AI degraded

Check `/api/v1/ai`, AI health, key-file permissions, and provider mode. Use manual planning; never bypass policy.

### Kafka or PostgreSQL recovery

Inspect Cutaway, lag, worker status, and offset commits. Do not manually advance offsets. At-least-once retry plus idempotency is the recovery path.

### Snapshots

No snapshot is routine. Inspect thin-pool data/metadata, physical headroom, and existing snapshots, then obtain explicit authorization immediately before creation.

