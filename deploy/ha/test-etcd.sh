#!/usr/bin/env bash
# Isolated, loopback-only CI cluster. Never uses the application's etcd data.
set -euo pipefail

: "${ETCD_BIN:?Set ETCD_BIN to the etcd executable}"
: "${ETCDCTL_BIN:?Set ETCDCTL_BIN to the etcdctl executable}"
repo_dir="$(cd "$(dirname "$0")/../.." && pwd)"
test_dir="$(mktemp -d "${TMPDIR:-/tmp}/sub2api-etcd-ci.XXXXXX")"
pids=()
cleanup() {
  result=$?
  trap - EXIT
  if (( result != 0 )); then
    for log in "$test_dir"/*.log; do
      if [[ -f "$log" ]]; then tail -n 40 "$log"; fi
    done
  fi
  for pid in "${pids[@]}"; do kill "$pid" 2>/dev/null || true; done
  for pid in "${pids[@]}"; do wait "$pid" 2>/dev/null || true; done
  rm -rf "$test_dir"
  exit "$result"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# Distinct from the full local HA lab's ports (23791-23803).
peers='node1=http://127.0.0.1:32380,node2=http://127.0.0.1:32382,node3=http://127.0.0.1:32384'
endpoints='http://127.0.0.1:32379,http://127.0.0.1:32381,http://127.0.0.1:32383'
for i in 1 2 3; do
  port=$((32377 + i * 2))
  "$ETCD_BIN" --name "node$i" --data-dir "$test_dir/node$i" \
    --listen-client-urls "http://127.0.0.1:$port" \
    --advertise-client-urls "http://127.0.0.1:$port" \
    --listen-peer-urls "http://127.0.0.1:$((port + 1))" \
    --initial-advertise-peer-urls "http://127.0.0.1:$((port + 1))" \
    --initial-cluster "$peers" --initial-cluster-state new \
    --initial-cluster-token "$(basename "$test_dir")" --log-level warn \
    > "$test_dir/node$i.log" 2>&1 &
  pids+=("$!")
done
ready=false
for ((attempt=0; attempt<30; attempt++)); do
  # A port collision must fail rather than accidentally testing another cluster.
  for pid in "${pids[@]}"; do kill -0 "$pid"; done
  if "$ETCDCTL_BIN" --endpoints "$endpoints" --dial-timeout=1s --command-timeout=2s endpoint health > "$test_dir/health.log" 2>&1; then
    ready=true
    break
  fi
  sleep 1
done
if [[ "$ready" != true ]]; then exit 1; fi
cd "$repo_dir/backend"
SUB2API_TEST_ETCD_ENDPOINTS="$endpoints" "${GO_BIN:-go}" test -race ./internal/clusterha -count=1 -v
