#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work_dir="$(mktemp -d /tmp/keelmesh-m12-local.XXXXXX)"
bin_dir="${work_dir}/bin"
mkdir -p "${bin_dir}"
pids=()
declare -A node_pids
declare -A node_configs

cleanup() {
  for pid in "${pids[@]:-}"; do
    kill "${pid}" 2>/dev/null || true
  done
  wait 2>/dev/null || true
  if [[ "${KEELMESH_M12_KEEP:-0}" != "1" ]]; then
    rm -rf -- "${work_dir}"
  else
    printf 'M12 local artifacts retained at %s\n' "${work_dir}"
  fi
}
trap cleanup EXIT INT TERM

printf 'Building M12 local verification binaries...\n'
cd "${repo_dir}"
go build -o "${bin_dir}/keelmesh-pki" ./cmd/keelmesh-pki
go build -o "${bin_dir}/keelmesh-coordination-node" ./cmd/keelmesh-coordination-node
go build -o "${bin_dir}/keelmesh-coordination-check" ./cmd/keelmesh-coordination-check

topology="${work_dir}/topology.json"
printf '%s\n' '[
  {"NodeID":"node-a-01","CellID":"A","VMID":220,"Host":"local-1","ManagementIP":"127.88.0.11","RadioIP":"127.77.0.11"},
  {"NodeID":"node-a-02","CellID":"A","VMID":221,"Host":"local-2","ManagementIP":"127.88.0.12","RadioIP":"127.77.0.12"},
  {"NodeID":"node-a-03","CellID":"A","VMID":222,"Host":"local-3","ManagementIP":"127.88.0.13","RadioIP":"127.77.0.13"},
  {"NodeID":"node-a-04","CellID":"A","VMID":223,"Host":"local-1","ManagementIP":"127.88.0.14","RadioIP":"127.77.0.14"},
  {"NodeID":"node-a-05","CellID":"A","VMID":224,"Host":"local-2","ManagementIP":"127.88.0.15","RadioIP":"127.77.0.15"},
  {"NodeID":"node-a-06","CellID":"A","VMID":225,"Host":"local-3","ManagementIP":"127.88.0.16","RadioIP":"127.77.0.16"},
  {"NodeID":"node-b-01","CellID":"B","VMID":229,"Host":"local-1","ManagementIP":"127.88.0.21","RadioIP":"127.77.0.21"},
  {"NodeID":"node-b-02","CellID":"B","VMID":231,"Host":"local-2","ManagementIP":"127.88.0.22","RadioIP":"127.77.0.22"},
  {"NodeID":"node-b-03","CellID":"B","VMID":232,"Host":"local-3","ManagementIP":"127.88.0.23","RadioIP":"127.77.0.23"},
  {"NodeID":"node-b-04","CellID":"B","VMID":233,"Host":"local-1","ManagementIP":"127.88.0.24","RadioIP":"127.77.0.24"},
  {"NodeID":"node-b-05","CellID":"B","VMID":234,"Host":"local-2","ManagementIP":"127.88.0.25","RadioIP":"127.77.0.25"},
  {"NodeID":"node-b-06","CellID":"B","VMID":236,"Host":"local-3","ManagementIP":"127.88.0.26","RadioIP":"127.77.0.26"}
]' > "${topology}"

pki_dir="${work_dir}/pki"
"${bin_dir}/keelmesh-pki" --output "${pki_dir}" --valid-days 2 --topology "${topology}"

start_node() {
  local node_id="$1" cell_id="$2" vmid="$3" management_ip="$4" radio_ip="$5"
  local node_dir="${work_dir}/nodes/${node_id}"
  mkdir -p "${node_dir}/state"
  local config="${node_dir}/config.json"
  printf '{"node_id":"%s","cell_id":"%s","vm_id":%s,"management_ip":"%s","radio_ip":"%s","data_dir":"%s","pki_dir":"%s","bootstrap":true}\n' \
    "${node_id}" "${cell_id}" "${vmid}" "${management_ip}" "${radio_ip}" "${node_dir}/state" "${pki_dir}/nodes/${node_id}" > "${config}"
  "${bin_dir}/keelmesh-coordination-node" --config "${config}" > "${node_dir}/node.log" 2>&1 &
  local pid="$!"
  pids+=("${pid}")
  node_pids["${node_id}"]="${pid}"
  node_configs["${node_id}"]="${config}"
}

stop_node() {
  local node_id="$1" pid="${node_pids[$1]:-}"
  if [[ -n "${pid}" ]] && kill -0 "${pid}" 2>/dev/null; then
    kill "${pid}"
    wait "${pid}" 2>/dev/null || true
  fi
  node_pids["${node_id}"]=""
}

restart_node() {
  local node_id="$1" config="${node_configs[$1]}" node_dir
  node_dir="$(dirname "${config}")"
  "${bin_dir}/keelmesh-coordination-node" --config "${config}" >> "${node_dir}/node.log" 2>&1 &
  local pid="$!"
  pids+=("${pid}")
  node_pids["${node_id}"]="${pid}"
}

run_check() {
  local run_id="$1" output="$2"
  "${bin_dir}/keelmesh-coordination-check" --pki "${pki_dir}" --state "${work_dir}/gateway/state.json" --run-id "${run_id}" --timeout 12s > "${output}" 2> "${work_dir}/check.err"
}

wait_for_check() {
	local run_id="$1" output="$2" attempts="${3:-20}"
	for _ in $(seq 1 "${attempts}"); do
		if run_check "${run_id}" "${output}"; then
			return 0
		fi
		sleep 0.5
	done
	return 1
}

start_node node-a-01 A 220 127.88.0.11 127.77.0.11
start_node node-a-02 A 221 127.88.0.12 127.77.0.12
start_node node-a-03 A 222 127.88.0.13 127.77.0.13
start_node node-a-04 A 223 127.88.0.14 127.77.0.14
start_node node-a-05 A 224 127.88.0.15 127.77.0.15
start_node node-a-06 A 225 127.88.0.16 127.77.0.16
start_node node-b-01 B 229 127.88.0.21 127.77.0.21
start_node node-b-02 B 231 127.88.0.22 127.77.0.22
start_node node-b-03 B 232 127.88.0.23 127.77.0.23
start_node node-b-04 B 233 127.88.0.24 127.77.0.24
start_node node-b-05 B 234 127.88.0.25 127.77.0.25
start_node node-b-06 B 236 127.88.0.26 127.77.0.26

printf 'Waiting for two authority-ready leaders...\n'
result="${work_dir}/result.json"
for attempt in $(seq 1 40); do
	if run_check baseline "${result}"; then
		break
	fi
	sleep 0.5
done

if [[ ! -s "${result}" ]]; then
	printf 'M12 local verification failed:\n' >&2
	cat "${work_dir}/check.err" >&2
	exit 1
fi

cat "${result}"
leader_a="$(sed -n 's/.*"A":{[^}]*"leader":"\([^"]*\)".*/\1/p' "${result}")"
if [[ -z "${leader_a}" ]]; then
	printf 'Unable to resolve Cell A leader from verification output.\n' >&2
	exit 1
fi
printf 'Isolating elected Cell A leader %s...\n' "${leader_a}"
stop_node "${leader_a}"
sleep 2
wait_for_check leader-failover "${work_dir}/leader-failover.json" 20
cat "${work_dir}/leader-failover.json"
restart_node "${leader_a}"
sleep 2

printf 'Simulating one whole local host loss (two voters per cell)...\n'
for node_id in node-a-01 node-a-04 node-b-01 node-b-04; do stop_node "${node_id}"; done
sleep 2
wait_for_check host-loss "${work_dir}/host-loss.json" 20
cat "${work_dir}/host-loss.json"
for node_id in node-a-01 node-a-04 node-b-01 node-b-04; do restart_node "${node_id}"; done
sleep 3

printf 'Proving three remaining voters cannot commit...\n'
for node_id in node-a-01 node-a-02 node-a-03 node-b-01 node-b-02 node-b-03; do stop_node "${node_id}"; done
sleep 2
if run_check no-quorum "${work_dir}/no-quorum.json"; then
	printf 'Three-voter partition unexpectedly committed.\n' >&2
	exit 1
fi
for node_id in node-a-01 node-a-02 node-a-03 node-b-01 node-b-02 node-b-03; do restart_node "${node_id}"; done
sleep 3
wait_for_check recovered "${work_dir}/recovered.json" 20
cat "${work_dir}/recovered.json"
printf 'M12 local two-cell process verification passed (12 nodes, leader failover, 4/2 host loss, 3/3 rejection, recovery).\n'
exit 0

printf 'M12 local verification failed:\n' >&2
cat "${work_dir}/check.err" >&2
for log in "${work_dir}"/nodes/*/node.log; do
  printf '\n--- %s ---\n' "${log}" >&2
  tail -n 20 "${log}" >&2
done
exit 1
