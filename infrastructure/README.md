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
