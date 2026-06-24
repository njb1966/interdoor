#!/usr/bin/env bash
#
# play.sh — build and run a local InterDOOR node so you can SSH in and play.
# Usage: bash scripts/play.sh [port]   (default port 2323)
#
set -euo pipefail
cd "$(dirname "$0")/.."

port="${1:-2323}"
mkdir -p ./bin

echo "Building the node..."
go build -o ./bin/idoor-node ./cmd/node

cat <<EOF

InterDOOR node starting on port ${port}.

  >> In ANOTHER terminal, connect and play:

        ssh -p ${port} you@localhost

  - Any username works; you create your character in-game.
  - First connection: ssh asks to trust the host key — answer "yes".
  - No ssh password is needed; the game handles login.
  - Your character persists in ./bin/play.db — quit with Q, reconnect,
    and you'll still be there.

  Press Ctrl-C here to stop the node.

EOF

exec ./bin/idoor-node \
  -addr ":${port}" \
  -db ./bin/play.db \
  -hostkey ./bin/play.hostkey \
  -node local
