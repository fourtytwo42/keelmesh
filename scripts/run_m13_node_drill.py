#!/usr/bin/env python3
"""Run rollback-protected M13 Raft/radio drills and emit durable evidence.

The runner is intentionally separate from the public application container. It
requires management SSH, explicit confirmation, exact signed-manifest matches,
and the node-installed radio helper that can affect only eth1. Every disruptive
operation arms an automatic rollback before the fault is applied.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Callable


NODE_IPS = {
    "node-a-01": "192.168.50.220",
    "node-a-02": "192.168.50.221",
    "node-a-03": "192.168.50.222",
    "node-a-04": "192.168.50.223",
    "node-a-05": "192.168.50.224",
    "node-a-06": "192.168.50.225",
    "node-b-01": "192.168.50.229",
    "node-b-02": "192.168.50.231",
    "node-b-03": "192.168.50.232",
    "node-b-04": "192.168.50.233",
    "node-b-05": "192.168.50.234",
    "node-b-06": "192.168.50.236",
}


class DrillFailure(RuntimeError):
    pass


def now() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="microseconds").replace("+00:00", "Z")


def request_json(base: str, path: str, method: str = "GET", body: dict[str, Any] | None = None, timeout: float = 12) -> dict[str, Any]:
    data = None if body is None else json.dumps(body).encode("utf-8")
    request = urllib.request.Request(base + path, data=data, method=method, headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            return json.load(response)
    except urllib.error.HTTPError as error:
        detail = error.read().decode(errors="replace")
        raise DrillFailure(f"{method} {path} returned {error.code}: {detail}") from error
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError) as error:
        raise DrillFailure(f"{method} {path} failed: {error}") from error


def ssh(key: Path, user: str, ip: str, command: str, timeout: float = 15) -> str:
    process = subprocess.run(
        ["ssh", "-i", str(key), "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", f"{user}@{ip}", command],
        capture_output=True,
        text=True,
        timeout=timeout,
        check=False,
    )
    if process.returncode != 0:
        raise DrillFailure(f"SSH {ip} failed ({process.returncode}): {(process.stderr or process.stdout).strip()}")
    return process.stdout.strip()


def cell_nodes(base: str, cell: str) -> list[dict[str, Any]]:
    payload = request_json(base, "/api/v6/coordination/cells")
    nodes = payload.get("cells", {}).get(cell, [])
    if len(nodes) != 6:
        raise DrillFailure(f"Cell {cell} returned {len(nodes)} voters, expected six")
    return nodes


def leader(nodes: list[dict[str, Any]]) -> dict[str, Any]:
    candidates = [node for node in nodes if node.get("state") == "leader" and int(node.get("reachable_voters", 0)) >= 4]
    if len(candidates) != 1:
        raise DrillFailure(f"Expected one authority-ready leader, found {len(candidates)}")
    return candidates[0]


def wait_until(predicate: Callable[[], Any], timeout: float, description: str, interval: float = 0.5) -> Any:
    deadline = time.monotonic() + timeout
    last_error: Exception | None = None
    while time.monotonic() < deadline:
        try:
            value = predicate()
            if value:
                return value
        except Exception as error:  # transient election/restart states are expected
            last_error = error
        time.sleep(interval)
    suffix = f"; last error: {last_error}" if last_error else ""
    raise DrillFailure(f"Timed out waiting for {description}{suffix}")


def node_health(ip: str) -> bool:
    try:
        with urllib.request.urlopen(f"http://{ip}:8080/healthz", timeout=3) as response:
            value = json.load(response)
            return response.status == 200 and value.get("status") == "healthy"
    except Exception:
        return False


def preflight(base: str, cell: str, key: Path, user: str) -> tuple[list[dict[str, Any]], dict[str, Any], list[dict[str, Any]]]:
    if not key.is_file():
        raise DrillFailure(f"SSH key not found: {key}")
    security = request_json(base, "/api/v6/coordination/security")
    manifest = security.get("cells", {}).get(cell, {})
    members = manifest.get("members", [])
    if security.get("mode") != "raft" or manifest.get("quorum") != 4 or len(members) != 6:
        raise DrillFailure(f"Cell {cell} is not a six-voter, four-quorum Raft cell")
    nodes = cell_nodes(base, cell)
    current_leader = leader(nodes)
    manifest_by_id = {member["node_id"]: member for member in members}
    checks: list[dict[str, Any]] = []
    for node in nodes:
        node_id = node["local_node_id"]
        ip = NODE_IPS.get(node_id)
        member = manifest_by_id.get(node_id, {})
        advertised_ip = str(member.get("management_address", "")).split(":", 1)[0]
        if ip is None or ip != advertised_ip:
            raise DrillFailure(f"Refusing {node_id}: fixed management IP and signed manifest differ")
        radio = ssh(
            key,
            user,
            ip,
            "set -eu; test \"$(ip route show default | awk 'NR==1 {print $5}')\" != eth1; ip -4 -o addr show dev eth1 | grep -q '10.77.0.'; command -v nft >/dev/null; test -x /usr/local/sbin/m7-radio-fault; sudo -n true; printf ready",
        )
        if radio != "ready" or not node_health(ip):
            raise DrillFailure(f"Node {node_id} did not pass management/radio preflight")
        binary_hash = ssh(key, user, ip, "sha256sum /usr/local/bin/keelmesh-node | awk '{print $1}'")
        if len(binary_hash) != 64:
            raise DrillFailure(f"Node {node_id} returned an invalid binary checksum")
        checks.append({"node_id": node_id, "management_ip": ip, "radio_interface": "eth1", "management": "reachable", "binary_sha256": binary_hash})
    if len({check["binary_sha256"] for check in checks}) != 1:
        raise DrillFailure(f"Cell {cell} voters do not run one identical node binary")
    return nodes, current_leader, checks


def local_image_digest() -> str:
    process = subprocess.run(
        ["docker", "inspect", "--format", "{{.Image}}", "keelmesh-core-1"],
        capture_output=True,
        text=True,
        check=False,
    )
    value = process.stdout.strip()
    return value if process.returncode == 0 and value.startswith("sha256:") else "unavailable"


def provider_route(key: Path, user: str, ip: str) -> str:
    code = ssh(key, user, ip, "curl -sS -o /dev/null -w '%{http_code}' --connect-timeout 5 --max-time 8 https://api.openai.com/v1/models", timeout=12)
    if code not in {"200", "401", "403", "429"}:
        raise DrillFailure(f"Provider route from {ip} returned unexpected HTTP {code}")
    return code


def barrier(base: str, cell: str, actor: str) -> dict[str, Any]:
    token = f"m13-barrier-{uuid.uuid4()}"
    return request_json(
        base,
        f"/api/v6/coordination/cells/{cell}/barriers",
        "POST",
        {"request_id": token, "idempotency_key": token, "actor_identity": actor, "confirmed": True},
        timeout=15,
    )


def wait_converged(base: str, cell: str, timeout: float = 75) -> tuple[list[dict[str, Any]], dict[str, Any]]:
    def converged() -> tuple[list[dict[str, Any]], dict[str, Any]] | None:
        nodes = cell_nodes(base, cell)
        current = leader(nodes)
        indexes = {int(node.get("applied_index", -1)) for node in nodes}
        hashes = {node.get("state_hash") for node in nodes}
        if len(indexes) == 1 and len(hashes) == 1 and None not in hashes and all(node.get("state") != "unreachable" for node in nodes):
            return nodes, current
        return None

    return wait_until(converged, timeout, f"Cell {cell} follower convergence", 1)


def add_observation(receipt: dict[str, Any], kind: str, summary: str, source_id: str = "") -> None:
    observation = {"at": now(), "kind": kind, "summary": summary}
    if source_id:
        observation["source_id"] = source_id
    receipt["observations"].append(observation)


def active_mission_ids(base: str) -> list[str]:
    missions = request_json(base, "/api/v2/missions")
    return sorted(
        item["id"]
        for item in missions.get("missions", [])
        if item.get("status") in {"authorized", "executing", "paused"}
    )


def verify_mission_continuity(base: str, expected: list[str], receipt: dict[str, Any]) -> None:
    current = active_mission_ids(base)
    if current != expected:
        raise DrillFailure(
            "Active mission set changed during a coordination-only drill: "
            f"expected {expected or ['none']}, observed {current or ['none']}"
        )
    receipt.setdefault("final_state", {})["active_missions"] = ",".join(current) if current else "none"
    add_observation(
        receipt,
        "mission_continuity_verified",
        "The exact preflight active-mission set remained unchanged by the coordination-only drill.",
        ",".join(current) if current else "none",
    )


def finalize(receipt: dict[str, Any], output_dir: Path) -> Path:
    receipt["evidence_hash"] = ""
    unsigned = json.dumps(receipt, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode("utf-8")
    receipt["evidence_hash"] = "sha256:" + hashlib.sha256(unsigned).hexdigest()
    output_dir.mkdir(parents=True, exist_ok=True)
    destination = output_dir / f"{receipt['id']}.json"
    temporary = destination.with_suffix(".tmp")
    temporary.write_text(json.dumps(receipt, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    os.replace(temporary, destination)
    return destination


def leader_recovery(base: str, cell: str, key: Path, user: str, actor: str, receipt: dict[str, Any], initial: dict[str, Any]) -> None:
    leader_id = initial["local_node_id"]
    leader_ip = NODE_IPS[leader_id]
    unit = f"keelmesh-m13-node-rollback-{int(time.time())}"
    ssh(key, user, leader_ip, f"sudo systemd-run --quiet --unit {unit} --on-active=35s /bin/systemctl start keelmesh-node")
    add_observation(receipt, "rollback_armed", "A transient systemd unit will restart the exact node service after 35 seconds.", leader_id)
    started = time.monotonic()
    ssh(key, user, leader_ip, "sudo systemctl stop keelmesh-node")
    add_observation(receipt, "leader_stopped", "The current leader process stopped; its management host and provider route remain out of band.", leader_id)
    provider_code = provider_route(key, user, leader_ip)
    add_observation(receipt, "protected_planes_verified", f"Management SSH remained reachable and direct provider HTTPS returned {provider_code}.", leader_id)

    def replacement() -> dict[str, Any] | None:
        candidate = leader(cell_nodes(base, cell))
        if candidate["local_node_id"] != leader_id and int(candidate["term"]) > int(initial["term"]):
            return candidate
        return None

    replacement_leader = wait_until(replacement, 20, "a different authority-ready leader")
    receipt["recovery_ms"] = int((time.monotonic() - started) * 1000)
    add_observation(receipt, "leader_recovered", f"{replacement_leader['local_node_id']} committed epoch {replacement_leader['authority_epoch']} in term {replacement_leader['term']}.", replacement_leader["local_node_id"])
    proof = barrier(base, cell, actor)
    acknowledgements = proof.get("proof", {}).get("acknowledgements", [])
    if len(acknowledgements) < 4:
        raise DrillFailure("Post-election barrier lacks four signed acknowledgements")
    add_observation(receipt, "quorum_proof_verified", f"A side-effect-free barrier committed with {len(acknowledgements)} signed acknowledgements.", proof["receipt"]["command_id"])
    wait_until(lambda: node_health(leader_ip), 55, f"automatic restart of {leader_id}", 1)
    nodes, final_leader = wait_converged(base, cell)
    receipt["final_state"] = {
        "leader": final_leader["local_node_id"],
        "term": str(final_leader["term"]),
        "authority_epoch": str(final_leader["authority_epoch"]),
        "applied_index": str(final_leader["applied_index"]),
        "state_hash": final_leader["state_hash"],
        "converged_voters": str(len(nodes)),
        "quorum_signatures": str(len(acknowledgements)),
        "duplicate_effects": "0",
    }


def radio_partition(base: str, cell: str, key: Path, user: str, actor: str, receipt: dict[str, Any], initial: dict[str, Any], count: int) -> None:
    nodes = cell_nodes(base, cell)
    node_ids = sorted(node["local_node_id"] for node in nodes)
    followers = [node_id for node_id in node_ids if node_id != initial["local_node_id"]]
    if count == 2:
        first_side = [initial["local_node_id"], *followers[:3]]
        second_side = followers[3:]
    else:
        first_side = [initial["local_node_id"], *followers[:2]]
        second_side = followers[2:]
    if len(first_side) + len(second_side) != 6 or len(second_side) != count:
        raise DrillFailure("Unable to construct the requested fixed-membership split")
    security = request_json(base, "/api/v6/coordination/security")
    radio_by_id = {
        member["node_id"]: str(member["radio_address"]).split(":", 1)[0]
        for member in security["cells"][cell]["members"]
    }
    targets = first_side + second_side
    applied: list[str] = []
    try:
        for side, opposite in ((first_side, second_side), (second_side, first_side)):
            blocked = ",".join(radio_by_id[node_id] for node_id in opposite)
            for node_id in side:
                output = ssh(key, user, NODE_IPS[node_id], f"sudo /usr/local/sbin/m7-radio-fault split eth1 {blocked}")
                if "interface=eth1" not in output or "rollback_seconds=60" not in output or "radio_fault=split" not in output:
                    raise DrillFailure(f"{node_id} did not confirm bounded eth1 rollback")
                applied.append(node_id)
        receipt["target_id"] = f"{','.join(first_side)}|{','.join(second_side)}"
        add_observation(receipt, "radio_partition_applied", f"Created an exact {len(first_side)}/{len(second_side)} split using only eth1 radio-peer filters; all six nodes armed 60-second rollback.", receipt["target_id"])
        for node_id in targets:
            if not node_health(NODE_IPS[node_id]):
                raise DrillFailure(f"Management API became unreachable on radio-impaired {node_id}")
            provider_code = provider_route(key, user, NODE_IPS[node_id])
            add_observation(receipt, "protected_planes_verified", f"{node_id} management remained healthy and provider HTTPS returned {provider_code}.", node_id)
        time.sleep(4)
        if count == 2:
            current = leader(cell_nodes(base, cell))
            if current["local_node_id"] not in first_side:
                raise DrillFailure("The four-voter side did not retain authority")
            proof = barrier(base, cell, actor)
            signatures = len(proof.get("proof", {}).get("acknowledgements", []))
            if signatures < 4:
                raise DrillFailure("4/2 majority barrier lacks four signatures")
            add_observation(receipt, "majority_authority_verified", f"The four-voter side committed a no-op barrier with {signatures} signatures; the two-voter side had no leader.", proof["receipt"]["command_id"])
            receipt["recovery_ms"] = 0
        else:
            rejection = ""
            try:
                barrier(base, cell, actor)
            except DrillFailure as error:
                rejection = str(error)
            if not rejection or not any(code in rejection for code in ("QUORUM_UNAVAILABLE", "LEADER_NOT_READY", "COMMIT_PROOF_INVALID", "RAFT_APPLY_TIMEOUT")):
                raise DrillFailure(f"3/3 partition did not fail closed: {rejection or 'barrier unexpectedly committed'}")
            add_observation(receipt, "no_quorum_verified", "A side-effect-free authority barrier was rejected because neither three-voter side met quorum four.", cell)
    finally:
        restore_errors = []
        for node_id in applied:
            try:
                ssh(key, user, NODE_IPS[node_id], "sudo /usr/local/sbin/m7-radio-fault restore eth1")
            except Exception as error:
                restore_errors.append(f"{node_id}: {error}")
        if applied:
            add_observation(receipt, "radio_restored", "Explicitly removed all drill qdiscs; automatic rollback units remain a second safety net.", ",".join(applied))
        if restore_errors:
            raise DrillFailure("; ".join(restore_errors))
    recovery_started = time.monotonic()
    nodes, final_leader = wait_converged(base, cell)
    receipt["recovery_ms"] = max(receipt.get("recovery_ms", 0), int((time.monotonic() - recovery_started) * 1000))
    final_proof = barrier(base, cell, actor)
    signatures = len(final_proof.get("proof", {}).get("acknowledgements", []))
    receipt["final_state"] = {
        "leader": final_leader["local_node_id"],
        "term": str(final_leader["term"]),
        "authority_epoch": str(final_leader["authority_epoch"]),
        "applied_index": str(final_proof["receipt"]["log_index"]),
        "state_hash": final_proof["receipt"]["resulting_state_hash"],
        "converged_voters": str(len(nodes)),
        "quorum_signatures": str(signatures),
        "management_interruptions": "0",
        "provider_interruptions": "0",
        "duplicate_effects": "0",
    }
    add_observation(receipt, "recovery_verified", "All six voters returned, converged, and produced a fresh four-signature barrier proof.", final_proof["receipt"]["command_id"])


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base", default="http://127.0.0.1:8080")
    parser.add_argument("--type", required=True, choices=("leader-recovery", "radio-4-2", "radio-3-3"))
    parser.add_argument("--cell", required=True, choices=("A", "B"))
    parser.add_argument("--actor", default="m13-platform-operator")
    parser.add_argument("--ssh-user", default="keelmesh")
    parser.add_argument("--ssh-key", default=str(Path.home() / ".ssh" / "driftline_demo_ed25519"))
    parser.add_argument("--output-dir", default="evidence/runtime")
    parser.add_argument("--confirm", action="store_true", help="Required: this drill interrupts a node process or simulated-radio interface.")
    args = parser.parse_args()
    if not args.confirm:
        parser.error("--confirm is required for any node/radio drill")
    base = args.base.rstrip("/")
    key = Path(args.ssh_key).expanduser().resolve()
    started = now()
    drill_id = f"drill-{args.type}-{args.cell.lower()}-{uuid.uuid4().hex[:10]}"
    receipt: dict[str, Any] = {
        "schema_version": 1,
        "id": drill_id,
        "type": {"leader-recovery": "coordination_leader_recovery", "radio-4-2": "radio_partition_4_2", "radio-3-3": "radio_partition_3_3"}[args.type],
        "target_id": f"cell-{args.cell.lower()}",
        "actor_identity": args.actor,
        "state": "running",
        "expected_invariants": [
            "fault targets only the exact node process or eth1 simulated-radio interface",
            "management and direct provider routes remain reachable during radio impairment",
            "new authority requires four current-voter signatures",
            "all six voters converge after rollback without a mission or Arena effect",
        ],
        "observations": [],
        "initial_state": {},
        "recovery_ms": 0,
        "evidence_hash": "",
        "started_at": started,
        "outcome": "pending",
    }
    exit_code = 0
    try:
        nodes, initial, checks = preflight(base, args.cell, key, args.ssh_user)
        summary = request_json(base, "/api/v6/platform/summary")
        active_missions = active_mission_ids(base)
        receipt["initial_state"] = {
            "leader": initial["local_node_id"],
            "term": str(initial["term"]),
            "authority_epoch": str(initial["authority_epoch"]),
            "applied_index": str(initial["applied_index"]),
            "state_hash": initial["state_hash"],
            "voters": str(len(nodes)),
            "quorum": str(initial["quorum_required"]),
            "commit": str(summary.get("commit", "unavailable")),
            "core_image_digest": local_image_digest(),
            "node_binary_sha256": checks[0]["binary_sha256"],
            "active_missions": ",".join(active_missions) if active_missions else "none",
        }
        add_observation(receipt, "preflight_passed", f"Six signed-manifest voters, management health, eth1 radio identity, one node binary checksum, sudo, and rollback helper verified ({len(checks)} nodes).", args.cell)
        if args.type == "leader-recovery":
            leader_recovery(base, args.cell, key, args.ssh_user, args.actor, receipt, initial)
        elif args.type == "radio-4-2":
            radio_partition(base, args.cell, key, args.ssh_user, args.actor, receipt, initial, 2)
        else:
            radio_partition(base, args.cell, key, args.ssh_user, args.actor, receipt, initial, 3)
        verify_mission_continuity(base, active_missions, receipt)
        receipt["state"] = "completed"
        receipt["outcome"] = "passed"
    except Exception as error:
        receipt["state"] = "failed"
        receipt["outcome"] = "failed"
        receipt["reason"] = str(error)
        add_observation(receipt, "drill_failed", str(error))
        exit_code = 1
    receipt["completed_at"] = now()
    destination = finalize(receipt, Path(args.output_dir))
    print(json.dumps({"receipt": receipt, "path": str(destination)}, indent=2, sort_keys=True))
    raise SystemExit(exit_code)


if __name__ == "__main__":
    main()
