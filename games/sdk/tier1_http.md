# Tier 1 HTTP Example

Tier 1 is enough for a non-Go game node to appear in the InterDoor directory and portal. The game
keeps its own player sessions and local storage. It only needs an HTTP client for registration and
heartbeat.

## Register

Registration is an operator-approved first-run action. Store the returned API key locally and stop
using the registration token after the node is registered.

```bash
curl -sS https://hub.example/v1/register \
  -H 'Content-Type: application/json' \
  -d '{
    "node_id": "mydoor",
    "registration_token": "operator-issued-token",
    "game_id": "my_custom_game",
    "game_title": "My Custom Game",
    "game_version": "1.0.0",
    "protocol_version": "1",
    "advertise_addr": "ssh://mydoor.example:2323"
  }'
```

Use `game_id` for compatibility and storage; use `game_title` for display in the public directory
and SSH portal. `game_title` is optional, but nodes should send it when they have a clean
player-facing name.

Expected response:

```json
{
  "api_key": "store-this-locally",
  "hub_seq_head": 0
}
```

## Heartbeat

Send heartbeat periodically with the stored API key.

```bash
curl -sS https://hub.example/v1/heartbeat \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer stored-api-key' \
  -d '{
    "node_id": "mydoor",
    "player_count": 3,
    "uptime_s": 3600,
    "game_version": "1.0.0"
  }'
```

Expected response:

```json
{
  "hub_seq_head": 12,
  "pending": {
    "events": 0,
    "pvp": 0,
    "travel": 0
  }
}
```

Tier 1 nodes do not push roster entries, events, PvP, travel, or shared obligations.
