#!/usr/bin/env bash
#
# federation-demo.sh — run a hub + two federated nodes and watch state propagate.
# Usage: bash scripts/federation-demo.sh
#
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p ./bin

echo "Building hub + node..."
go build -o ./bin/idoor-hub ./cmd/hub
go build -o ./bin/idoor-node ./cmd/node

work="$(mktemp -d)"
pids=()
cleanup() {
  [[ ${#pids[@]} -gt 0 ]] && kill "${pids[@]}" 2>/dev/null || true
  rm -rf "$work"
  echo
  echo "Stopped hub and nodes."
}
trap cleanup EXIT INT TERM

./bin/idoor-hub -addr 127.0.0.1:8089 -db "$work/hub.db" -reg-token demo \
  >"$work/hub.log" 2>&1 &
pids+=("$!")
sleep 1

./bin/idoor-node -addr 127.0.0.1:2340 -db "$work/n1.db" -hostkey "$work/n1.key" \
  -node node01 -hub http://127.0.0.1:8089 -hub-reg-token demo -sync-interval 2 \
  >"$work/n1.log" 2>&1 &
pids+=("$!")

./bin/idoor-node -addr 127.0.0.1:2341 -db "$work/n2.db" -hostkey "$work/n2.key" \
  -node node02 -hub http://127.0.0.1:8089 -hub-reg-token demo -sync-interval 2 \
  >"$work/n2.log" 2>&1 &
pids+=("$!")
sleep 2

cat <<EOF

Federation running:  hub :8089    node01 :2340    node02 :2341

  >> In ANOTHER terminal, create a character on node01:

        ssh -p 2340 you@localhost      (make a character, then quit with Q)

  Below, this script polls each node's event log. When your character
  creation on node01 propagates through the hub, node02's "received" count
  ticks up — two independent servers sharing state. Ctrl-C to stop.

  Logs: ${work}/*.log

EOF

while true; do
  n1=$(sqlite3 "$work/n1.db" "SELECT COUNT(*) FROM events" 2>/dev/null || echo "?")
  n2=$(sqlite3 "$work/n2.db" "SELECT COUNT(*) FROM events WHERE source_node='node01'" 2>/dev/null || echo "?")
  printf '  [%s]  node01 emitted: %-3s   node02 received from node01: %-3s\n' \
    "$(date +%T)" "$n1" "$n2"
  sleep 3
done
