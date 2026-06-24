# SYSOP GUIDE
## Running an InterDOOR Node

*Target: zero to running node in under an hour on a fresh Debian/Ubuntu server.*

---

## What you are setting up

An **InterDOOR node** is an SSH game server. Players connect via SSH, create accounts, and play
*Ledger of the Low* — a daily-turn survival RPG. Your node is independent: it runs its own SQLite
database and serves players entirely on its own. Federation (connecting to the hub and interacting
with players on other nodes) is optional but recommended.

You will end up with:
- A game server accessible at `ssh -p 2323 your.server.example.com`
- Optionally: federation with the InterDOOR network (cross-node wanderers, PvP, travel, shared debts)

Once federated, your node appears in the hub's public SSH portal — an ANSI terminal directory
players can browse to discover games on the network:
```bash
ssh -p 2300 hub.interdoor.net
```

---

## Prerequisites

- A Linux server (Debian 12+ or Ubuntu 22.04+), VPS or bare metal
- A non-root user with `sudo` access
- Port 2323 accessible from the internet (or whatever port you choose)
- A domain name (optional but makes the node address nicer to share)

---

## 1. Install the binary

**Option A: download a release binary**

```bash
# Replace VERSION and ARCH (amd64 or arm64) as appropriate.
VERSION=latest
ARCH=amd64

wget https://github.com/interdoor-net/interdoor/releases/latest/download/interdoor-node-linux-${ARCH}
sudo mv interdoor-node-linux-${ARCH} /usr/local/bin/interdoor-node
sudo chmod +x /usr/local/bin/interdoor-node
```

**Option B: build from source** (requires Go 1.24+)

```bash
git clone https://github.com/interdoor-net/interdoor
cd interdoor
make release
sudo cp dist/interdoor-node-linux-amd64 /usr/local/bin/interdoor-node
```

Verify:
```bash
interdoor-node -help
```

---

## 2. Create directories

```bash
sudo mkdir -p /etc/interdoor /var/lib/interdoor
sudo chown $USER /etc/interdoor /var/lib/interdoor
```

---

## 3. Write the config file

Create `/etc/interdoor/node.json`:

```json
{
  "addr": ":2323",
  "db": "/var/lib/interdoor/node.db",
  "hostkey": "/etc/interdoor/hostkey",
  "node": "mynode",
  "max_sessions": 32,
  "idle_timeout_sec": 600,
  "hub_url": "https://hub.interdoor.net",
  "hub_reg_token": "REPLACE_WITH_TOKEN_FROM_HUB_OPERATOR",
  "advertise_addr": "ssh://mynode.example.com:2323",
  "sync_interval_sec": 20
}
```

**Required fields:**

| Field | What to set |
|---|---|
| `node` | A short lowercase identifier unique on the network (e.g. `tavern`, `ironport`). Only alphanumeric + hyphens. |
| `advertise_addr` | How players and other nodes reach you — the public SSH address shown in the node directory. Format: `ssh://hostname:port`. |
| `hub_url` | URL of the InterDOOR hub. Get this from the network operator. |
| `hub_reg_token` | A registration token issued by the hub operator. Used only on first start by the node; current hub code treats it as an operator-set shared token. |

**Standalone mode** (no federation): omit `hub_url` and `hub_reg_token`. The node runs fully
independently — no cross-node players or shared events.

---

## 4. First run

The host key (SSH server identity) is generated automatically on first start if it doesn't exist.

```bash
interdoor-node -config /etc/interdoor/node.json
```

Expected output:
```
registered node "mynode" with hub https://hub.interdoor.net
federation sync active -> https://hub.interdoor.net (every 20s)
InterDOOR node "mynode" on :2323 — game "Ledger of the Low: The Old Bargain" (max 32 sessions, idle 600s)
```

After the first successful run:
- The host key is saved to `/etc/interdoor/hostkey` (keep this file; losing it disconnects all players)
- The API key issued by the hub is stored in the node database; `hub_reg_token` is no longer needed
  (you can remove it from the config file, but leaving it in is harmless)

Test by connecting from another machine:
```bash
ssh -p 2323 mynode.example.com
```

You should see the Ledger of the Low title screen. Create an account and verify it works.

---

## 5. Run as a systemd service

Create `/etc/systemd/system/interdoor-node.service`:

```ini
[Unit]
Description=InterDOOR node — Ledger of the Low
After=network.target

[Service]
Type=simple
User=interdoor
Group=interdoor
ExecStart=/usr/local/bin/interdoor-node -config /etc/interdoor/node.json
Restart=on-failure
RestartSec=10

# Harden the service (drop most capabilities)
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/var/lib/interdoor /etc/interdoor

[Install]
WantedBy=multi-user.target
```

Create a dedicated user and enable the service:

```bash
sudo useradd -r -s /usr/sbin/nologin -d /var/lib/interdoor interdoor
sudo chown -R interdoor:interdoor /etc/interdoor /var/lib/interdoor
sudo chmod 600 /etc/interdoor/hostkey  # if already generated
sudo systemctl daemon-reload
sudo systemctl enable --now interdoor-node
sudo systemctl status interdoor-node
```

View logs:
```bash
sudo journalctl -u interdoor-node -f
```

---

## 6. Open the firewall

Allow SSH game traffic in:

```bash
# ufw
sudo ufw allow 2323/tcp comment "InterDOOR SSH"
sudo ufw status

# OR iptables
sudo iptables -A INPUT -p tcp --dport 2323 -j ACCEPT
```

Your server's regular SSH (port 22) is unaffected — InterDOOR uses its own port and its own
host key. Players never get shell access.

---

## 7. Verify federation

After the node is running and federated, connect and look at the **Wanderers** screen from the
main menu. Within one sync interval (default 20 s), you should see players from other nodes
appearing alongside your own. If you see only local players, check the logs for sync errors:

```bash
sudo journalctl -u interdoor-node | grep -E "sync:|register|hub"
```

You can also confirm your node is visible in the hub SSH portal:
```bash
ssh -p 2300 hub.interdoor.net
```
Your node should appear in the numbered list within one sync interval of its first heartbeat.
Press the number next to your node to see its detail screen and verify the address and game ID.

The public node directory is also available as JSON:
```
GET https://hub.interdoor.net/v1/directory
```

Your node should appear there within a minute of first heartbeat.

---

## 8. Configuration reference

All fields are optional unless noted. Precedence: defaults < config file < command flags.

| JSON key | Flag | Default | Notes |
|---|---|---|---|
| `addr` | `-addr` | `:2323` | SSH listen address |
| `db` | `-db` | `interdoor.db` | SQLite database path |
| `hostkey` | `-hostkey` | `hostkey` | SSH host key path (generated on first run) |
| `node` | `-node` | `node01` | **Required** — unique node identifier |
| `max_sessions` | `-max-sessions` | `64` | Concurrent player cap (0 = unlimited) |
| `idle_timeout_sec` | `-idle-timeout` | `600` | Seconds before idle connections are closed |
| `hub_url` | `-hub` | *(empty)* | Hub base URL; empty = standalone mode |
| `hub_reg_token` | `-hub-reg-token` | *(empty)* | Operator-issued registration token used when no API key is stored yet |
| `advertise_addr` | `-advertise` | *(empty)* | Public SSH address shown in node directory |
| `sync_interval_sec` | `-sync-interval` | `20` | Federation sync cadence in seconds |

---

## 9. Updates

Replace the binary and restart:

```bash
sudo systemctl stop interdoor-node
sudo mv interdoor-node-linux-amd64 /usr/local/bin/interdoor-node
sudo chmod +x /usr/local/bin/interdoor-node
sudo systemctl start interdoor-node
sudo systemctl status interdoor-node
```

The database schema migrates automatically on startup. **Back up your database before updating:**

```bash
sudo -u interdoor sqlite3 /var/lib/interdoor/node.db ".backup /var/lib/interdoor/node.db.bak"
```

---

## 10. Troubleshooting

**"bind: permission denied" on :2323**
Port 2323 is unprivileged (>1024) and should not require root. If you see this, check that no
other process owns the port:
```bash
ss -ltn | grep 2323
```

**"no stored API key and no hub-reg-token"**
The node has lost its database or the `hub_reg_token` was removed before registration completed.
Contact the hub operator for a new token, then add it back to the config.

**Players report their character is "traveling" and can't log in**
The player is in transit between nodes. They need to connect to the destination node, complete
their visit, and return home. If travel is stuck, the hub operator must inspect the travel queue;
Phase 1 does not automatically expire pending travel.

**Federation sync errors in the logs**
Check that `hub_url` is reachable from your server:
```bash
curl -s https://hub.interdoor.net/v1/status
```
This should return a JSON object with `"hub": "ok"`. If it fails, check DNS and firewall on
your end, and confirm the hub is up.

**Host key changed warning on reconnect**
This happens if the hostkey file was regenerated or replaced. Players will see an SSH warning
about the host identity changing. They can clear it with:
```bash
ssh-keygen -R "[mynode.example.com]:2323"
```
To avoid this: keep the `hostkey` file safe and back it up alongside the database.

---

## Hub operators

To run your own hub (for private networks or the public InterDOOR network), install the hub binary:

```bash
sudo cp dist/interdoor-hub-linux-amd64 /usr/local/bin/interdoor-hub
sudo cp dist/interdoor-hub-admin-linux-amd64 /usr/local/bin/interdoor-hub-admin
```

The hub requires a **Caddy reverse proxy** for TLS termination (nodes connect over HTTPS).

`/etc/caddy/Caddyfile`:
```
hub.yourdomain.net {
    reverse_proxy localhost:8080
}
```

**Generate a host key for the SSH portal** (one-time):
```bash
sudo mkdir -p /etc/interdoor-hub
sudo ssh-keygen -t ed25519 -f /etc/interdoor-hub/portal-hostkey -N '' -C 'interdoor-hub-portal'
sudo chown interdoor-hub:interdoor-hub /etc/interdoor-hub/portal-hostkey*
sudo chmod 600 /etc/interdoor-hub/portal-hostkey
```

**Open the portal port** in your firewall:
```bash
# ufw
sudo ufw allow 2300/tcp comment "InterDOOR hub SSH portal"

# OR iptables
sudo iptables -A INPUT -p tcp --dport 2300 -j ACCEPT
```

Start the hub with the portal enabled:
```bash
interdoor-hub \
    -addr :8080 \
    -db /var/lib/interdoor-hub/hub.db \
    -reg-token-file /etc/interdoor-hub/reg-token.txt \
    -ssh-addr :2300 \
    -ssh-hostkey /etc/interdoor-hub/portal-hostkey
```

The `-ssh-addr` and `-ssh-hostkey` flags are both required to enable the portal. Omit them to
run the hub in HTTP-only mode (the portal will simply not start).

As a systemd service at `/etc/systemd/system/interdoor-hub.service`:
```ini
[Unit]
Description=InterDOOR hub
After=network.target

[Service]
Type=simple
User=interdoor-hub
Group=interdoor-hub
ExecStart=/usr/local/bin/interdoor-hub \
    -addr :8080 \
    -db /var/lib/interdoor-hub/hub.db \
    -reg-token-file /etc/interdoor-hub/reg-token.txt \
    -ssh-addr :2300 \
    -ssh-hostkey /etc/interdoor-hub/portal-hostkey
Restart=on-failure
RestartSec=10
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/var/lib/interdoor-hub

[Install]
WantedBy=multi-user.target
```

Verify the portal is running:
```bash
ssh -p 2300 your.hub.domain
```
You should see the ANSI INTERDOOR logo and the live node directory.

Issue registration tokens to node operators out-of-band (email, Signal, etc.). Each token can
be used once; the hub rejects duplicate `node_id` registrations.

**Hub endpoints:**

| Endpoint | Auth | Description |
|---|---|---|
| `GET /v1/directory` | none | JSON list of active registered nodes (online/offline, player counts) |
| `GET /v1/status` | none | Hub health: active node count, online count, event count |
| `SSH :2300` | none | ANSI terminal portal — live node directory, numbered navigation |

Back up the hub database regularly — it is the authoritative record for cross-node obligations,
the event feed, and travel state:
```bash
sudo -u interdoor-hub interdoor-hub-admin \
    -db /var/lib/interdoor-hub/hub.db \
    backup /var/lib/interdoor-hub/hub.db.$(date +%Y%m%d-%H%M%S).bak
```

For a live SQLite hub, use `interdoor-hub-admin backup` or SQLite's `.backup` mechanism. Do not
copy only `hub.db` while a `hub.db-wal` file is active and assume that copy is complete.

### Hub admin CLI

Phase 1 hub administration is local-only. There is no public admin API or web console. Use
`interdoor-hub-admin` on the hub host.

Dry-run commands inspect before changing anything:

```bash
sudo -u interdoor-hub interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db nodes
sudo -u interdoor-hub interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db queues
sudo -u interdoor-hub interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db node-suspend NODE_ID
sudo -u interdoor-hub interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db node-remove NODE_ID
sudo -u interdoor-hub interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db queue-retry pvp REQUEST_ID
sudo -u interdoor-hub interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db queue-retry travel TRAVEL_ID
```

Mutating commands require `--execute`:

```bash
sudo -u interdoor-hub interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db node-suspend --execute NODE_ID
sudo -u interdoor-hub interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db node-activate --execute NODE_ID
sudo -u interdoor-hub interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db rotate-key --execute NODE_ID
sudo -u interdoor-hub interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db queue-retry --execute pvp REQUEST_ID
sudo -u interdoor-hub interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db queue-retry --execute travel TRAVEL_ID
```

Issue or revoke the one active registration token:

```bash
sudo -u interdoor-hub interdoor-hub-admin \
    token-issue --file /etc/interdoor-hub/reg-token.txt --execute

sudo -u interdoor-hub interdoor-hub-admin \
    token-revoke --file /etc/interdoor-hub/reg-token.txt --execute
```

`rotate-key --execute` prints a new one-time API key. The old key stops authenticating
immediately, so coordinate the node update as a maintenance action.

`queue-retry --execute` resets a `blocked` PvP or travel row back to `pending` and clears its
error text. Use it only after the malformed payload or affected node state has been repaired.
Transient resolver/import failures should remain `pending`; `blocked` is reserved for malformed
non-retryable queue payloads.

Successful database-backed mutations, such as node suspension, activation, removal, backup, and
API-key rotation, and blocked queue retry write a row to the hub SQLite `hub_admin_audit` table.
Dry-runs do not write audit rows. Registration-token file changes are outside the hub database and
should be protected with normal host file permissions and system logging.

### Removing a node from the hub

Use this only when a registered node must be removed from the network, such as a faulty deployment,
retired host, compromised API key, or operator request.

Suspending is the safer first action for most incidents. A suspended node is hidden from
`/v1/directory`, excluded from `/v1/status` node counts, and its existing API key is rejected by
protected hub endpoints.

Safe removal sequence:

1. Stop and disable the node service on the node host first. This prevents a heartbeat or roster
   push from racing the cleanup.
2. Preserve the node binary, config, database, host key, and service unit in a dated backup or
   quarantine path.
3. Suspend the node on the hub.
4. Back up the hub database.
5. Count affected hub rows before deleting anything.
6. If the node has pending PvP or travel rows, pause and decide how players should recover. Do not
   blindly delete pending queues for a valid active node.
7. Delete only rows tied to the removed node ID.
8. Verify `/v1/directory`, `/v1/status`, hub logs, and remaining nodes.
9. Update the infrastructure record with the backup paths and hub rows changed.

Example for a SQLite-backed Phase 1 hub, replacing `NODE_ID`:

```bash
NODE_ID="badnode"
DB="/var/lib/interdoor-hub/hub.db"
STAMP="$(date +%Y%m%d-%H%M%S)"

sudo -u interdoor-hub interdoor-hub-admin -db "$DB" node-suspend --execute "$NODE_ID"
sudo -u interdoor-hub interdoor-hub-admin -db "$DB" queues --node "$NODE_ID"
sudo -u interdoor-hub interdoor-hub-admin -db "$DB" node-remove "$NODE_ID"
sudo -u interdoor-hub interdoor-hub-admin -db "$DB" backup "${DB}.pre-remove-${NODE_ID}-${STAMP}.bak"
```

If the counts are expected and no player recovery is pending:

```bash
sudo -u interdoor-hub interdoor-hub-admin -db "$DB" node-remove --execute "$NODE_ID"

curl -s https://hub.interdoor.net/v1/directory
curl -s https://hub.interdoor.net/v1/status
```

This is intentionally conservative. Keep the SQL runbook from older documentation only as a manual
fallback when the admin binary is unavailable.

---

*InterDOOR protocol v1. For issues or questions, see the project repository.*
