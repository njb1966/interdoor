# Tier 2 HTTP Example

Tier 2 adds event and roster sync. This is still small enough for non-Go games: use local storage
for your game, then publish small network facts through HTTP JSON.

## Push Events

Events must use the authenticated node as `source_node`. The hub assigns `hub_seq`.

```bash
curl -sS https://hub.example/v1/events \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer stored-api-key' \
  -d '{
    "events": [
      {
        "event_id": "mydoor:1",
        "source_node": "mydoor",
        "seq": 1,
        "type": "player.created",
        "ts": 1782250000,
        "payload": {
          "global_id": "mydoor:p_1",
          "name": "Ada",
          "home_node": "mydoor",
          "created_at": 1782250000
        }
      }
    ]
  }'
```

Expected response:

```json
{
  "accepted": 1,
  "duplicates": 0,
  "last_hub_seq": 13
}
```

## Pull Events

Store the last seen `hub_seq` locally and use it as the next `after` cursor.

```bash
curl -sS 'https://hub.example/v1/events?after=12&limit=500&exclude_self=true' \
  -H 'Authorization: Bearer stored-api-key'
```

## Push Roster

Roster is a small display projection, not a character sheet. Do not include credentials,
inventory, private state, or game-specific blobs.

```bash
curl -sS https://hub.example/v1/roster \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer stored-api-key' \
  -d '{
    "entries": [
      {
        "global_id": "mydoor:p_1",
        "name": "Ada",
        "level": 3,
        "status": "active",
        "last_seen": 1782250300
      }
    ]
  }'
```

The hub sets `node_id` from the authenticated API key. Client-supplied `node_id` is ignored.

## Pull Roster

```bash
curl -sS 'https://hub.example/v1/roster?exclude_self=true' \
  -H 'Authorization: Bearer stored-api-key'
```

Clients should mark remote roster entries stale when `last_seen` is more than 15 minutes old.
