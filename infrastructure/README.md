# KeelMesh Mission Appliance Infrastructure

The interview environment is one Ubuntu Server 24.04 LTS VM on Proxmox:

- Proxmox VMID: `214`
- VM name: `keelmesh-demo`
- Static address: `192.168.50.214/24`
- Gateway and DNS: `192.168.50.1`
- Compute: 8 vCPU, 16 GiB RAM with 8 GiB balloon floor
- Disk: 96 GiB on `local-lvm`
- Application root: `/srv/keelmesh`
- Application URL after deployment: `http://192.168.50.214:8080`

The VM uses key-only SSH for the `keelmesh` user. Do not commit private keys,
passwords, GitHub tokens, Cloudflare credentials, tunnel tokens, or model-provider
secrets.

Run the idempotent bootstrap from the repository root:

```bash
./infrastructure/bootstrap-vm.sh
```

The script installs Docker Engine and Compose, GitHub CLI, `cloudflared`, the
QEMU guest agent, and base diagnostic/build tools. It creates `/srv/keelmesh`
and adds the current user to the `docker` group.

The intended live deployment is a single Compose application with only the
core HTTP port published. Cloudflare Tunnel connects outbound to that origin;
no router port-forward is required.

## M7 vessel-node fabric

The M7 lab uses twelve linked clones across the three active Proxmox hosts. The
authoritative allocation is committed in `infrastructure/m7-node-topology.json`.
Each node has 2 vCPU, 2 GiB RAM, a 12 GiB sparse disk, QEMU guest agent, and two
planes:

- `eth0`: management, browser ingress, metrics, and provider HTTPS on
  `192.168.50.0/24`; this is the default route and is never faulted.
- `eth1`: simulated mission radio on `10.77.0.0/24`, with no default route.

`infrastructure/m7-radio-fault` may be installed as
`/usr/local/sbin/m7-radio-fault`. It accepts `degrade`, `partition`, or
`restore`, rejects a requested interface other than `eth1`, refuses to run if
`eth1` is the default route, and schedules rollback before applying a fault.

Player B traffic enters VM 214 on the private `player-b-ingress` Compose service.
The ingress pins all `/api/v3` requests to faction B and follows the currently
advertised B coordinator. The Cloudflare systemd unit publishes that ingress;
its generated URL is temporary and must not be treated as a release hostname.

No M7 VM snapshots were created. Thin-pool allocation remains over-provisioned;
all future snapshots still require a fresh storage inspection and explicit user
authorization.

## M12 consensus rollout

`infrastructure/m12-target-topology.json` records the active 2/2/2 placement
for each six-voter cell. The five migrations completed on 2026-09-04 after fresh
Proxmox CPU, RAM, thin-pool data/metadata, physical-headroom, VM, snapshot, and
target-storage checks passed. M12 never faults the management/default-route NIC.
VM 233 moved off mini43 before VMs 223 and 224 arrived, preventing the temporary
two-VM RAM spike that an ID-ordered rollout would create. Each guest was stopped,
moved, restarted, and verified before the next migration. No snapshot was made.

Generate production PKI only on VM 214 into a new mode-0700 directory. The
initializer refuses to overwrite a non-empty authority. Rotation is staged into
a second directory with old/new trust overlap and preserved application-signing
identities; deploy the combined trust/manifests to all peers before switching
leaf certificates, then invoke `systemctl reload keelmesh-node`.

VM 214 keeps the offline authority root private and exposes only a runtime copy
of the referee leaf identity and signed cell manifests to the unprivileged core
container. Start the gateway with `compose.m12.yaml`; the override defaults to
`shadow` and can be switched explicitly to `raft` after shadow receipts match.

Run the software-only two-cell proof before deployment:

```bash
scripts/keelmesh m12-local-verify
```

It starts twelve isolated loopback node processes, proves two four-vote cells,
leader replacement, a four-voter whole-host-loss shape, 3/3 write rejection,
cross-cell future activation, and convergence. It does not contact or modify the
vessel VMs.
