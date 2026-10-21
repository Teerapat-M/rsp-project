# WebSocket protocol (phase 2)

Live room traffic. The server is authoritative. JSON text frames, one object per message, `type` is the discriminator.

Connect:

```
GET /ws?room=<room uuid>&token=<dev guest token>
```

The token is the **temporary** player identity (same value as `Authorization: Bearer` / `X-Player-ID`). Phase 4 swaps this for a JWT. Origin checks are open for local dev.

You must already be in the room (create or join via REST). Unknown room or a stranger gets HTTP 403/404 **before** the upgrade.

On connect the server sends `room.updated` (and `presence` if anyone else is watching).

## Server → client

| `type` | When | Body |
| --- | --- | --- |
| `room.updated` | Join, leave, ready, presence, snapshot on connect | `{ "room": Room }` |
| `player.joined` | A new seat is filled | `{ "player": RoomPlayer }` |
| `player.left` | Seat emptied (REST leave, WS `leave`, or grace expiry) | `{ "playerId": "…" }` |
| `presence` | Socket up/down | `{ "playerId", "connected", "reconnectSeconds" }` |
| `player.disconnected` | Last socket for a player closed; grace started | `{ "playerId", "reconnectSeconds" }` |
| `player.reconnected` | They came back before the grace timer | `{ "playerId" }` |
| `lobby.ready` | Waiting-room ready flag | `{ "playerId", "playerReady", "ready", "need" }` |
| `pong` | Reply to `ping` | `{}` |
| `error` | Bad client message | `{ "code", "message" }` |

`Room` / `RoomPlayer` match `rsp-client/src/types.ts` (camelCase) plus `ready` and `connected` on each player. The React mock already consumes `room.updated`.

Disconnect grace (default 20s, `DISCONNECT_GRACE`): the player **stays in the room** with `connected: false`. Reconnect with the same identity cancels the timer. If it fires, they are removed (`player.left`). Empty rooms are closed so the passcode can be reused.

### Phase 3 (not sent yet)

These names are reserved so the frontend contract in `rsp-client/src/types.ts` does not have to change:

`match.started`, `tick`, `view`, `round.resolved`, `match.finished`, `opponent.disconnected`, `opponent.reconnected`, `play_again.waiting`, `connection.lost`, `connection.restored`.

`POST /rooms/:id/start` exists and returns `501 NOT_IMPLEMENTED` until that phase wires `hub.Config.OnStart`.

## Client → server

| `type` | Phase 2 | Body |
| --- | --- | --- |
| `lobby.ready` / `ready` | Yes, while `status=waiting` | `{}` — sets this player ready in the lobby |
| `lobby.unready` | Yes | `{}` |
| `leave` | Yes | `{}` — same as REST leave |
| `ping` | Yes | `{}` |
| `submit` | Hook | `{ "move": "rock"\|"paper"\|"scissors" }` → `error` `NOT_IMPLEMENTED` |
| `ready` while `playing` | Hook | same |
| `claim_win` / `play_again` / `simulate` | Hook | `error` `NOT_IMPLEMENTED` |

`hub.Config.OnPlay` is the phase 3 seam. Until it is set, play messages are rejected with a typed error rather than crashing.

## Identity (temporary)

Until phase 4:

1. `X-Player-ID` (any string; UUIDs kept, anything else hashed to a stable UUID)
2. `Authorization: Bearer <token>`
3. `?token=` on the WebSocket URL

`POST /dev/guest` mints a guest user and returns `{ accessToken, user }` in the shape the React client already stores. The access token **is** the user UUID — not a JWT.
