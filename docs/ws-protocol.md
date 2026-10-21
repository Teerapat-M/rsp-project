# WebSocket protocol (phase 3)

Live room and match traffic. The server is authoritative. JSON text frames, one object per message, `type` is the discriminator.

Connect:

```
GET /ws?room=<room uuid>&token=<access JWT>
```

The token is a short-lived **access JWT** (same value as `Authorization: Bearer`). Origin checks are open for local dev. In `APP_ENV=dev` only, `X-Player-ID` / a non-JWT `?token=` still work for curl.

You must already be in the room (create or join via REST). Unknown room or a stranger gets HTTP 403/404 **before** the upgrade.

On connect the server sends `room.updated` (and `presence` if anyone else is watching). If a match is already in progress it also sends `match.started` (or `match.finished`) for this player — the same snapshot a socket that was connected at `POST /rooms/:id/start` would have received. A late or reconnecting socket must not wait for a start event that already fired.

## Server → client

| `type` | When | Body |
| --- | --- | --- |
| `room.updated` | Join, leave, ready, presence, snapshot on connect | `{ "room": Room }` |
| `player.joined` | A new seat is filled | `{ "player": RoomPlayer }` |
| `player.left` | Seat emptied (REST leave, WS `leave`, or waiting-room grace expiry) | `{ "playerId": "…" }` |
| `presence` | Socket up/down | `{ "playerId", "connected", "reconnectSeconds" }` |
| `player.disconnected` | Last socket for a player closed; grace started | `{ "playerId", "reconnectSeconds" }` |
| `player.reconnected` | They came back before the grace timer | `{ "playerId" }` |
| `lobby.ready` | Waiting-room ready flag | `{ "playerId", "playerReady", "ready", "need" }` |
| `match.started` | Host started, or both players chose Play Again | `{ "view": PlayerView }` — **per player** (hidden opponent move) |
| `tick` | Once a second during a turn | `{ "remainingSeconds": n }` |
| `view` | After submit / one-sided ready / reconnect | `{ "view": PlayerView }` |
| `round.resolved` | Both Ready, or the turn timer fired | `{ "result": RoundResultView, "view": PlayerView }` — **per player** |
| `round.started` | Result pause ended; next turn clock is live | `{ "view": PlayerView }` — **per player**. Clients leave the round-result overlay. |
| `match.finished` | Someone reached `winsToFinish`, or a forfeit | `{ "view": PlayerView }` after the result pause |
| `opponent.disconnected` | Opponent's last socket dropped **during a match** | `{ "view": PlayerView }` (`opponentDisconnected`, `reconnectSeconds`) |
| `opponent.reconnected` | They returned before grace | `{ "view": PlayerView }` |
| `play_again.waiting` | After a finished match | `{ "ready", "need" }` |
| `pong` | Reply to `ping` | `{}` |
| `error` | Bad client message | `{ "code", "message" }` |

`Room` / `RoomPlayer` match `rsp-client/src/types.ts` (camelCase) plus `ready` and `connected` on each player. Each `RoomPlayer` includes that user's `users.avatar` / `users.color` (the profile picture from guest picker / `user_profile`), not the throw they played.

`PlayerView` / `RoundResultView` match the same file. `yourMove` is this player's card (or `""`). The opponent's current card is omitted until `round.resolved`. History on the view reveals both moves for **completed** rounds only. `view.you.avatar` and `opponentAvatar` are the same profile pictures as the room snapshot.

`connection.lost` / `connection.restored` are client-local (the browser's socket). The server never sends them.

Disconnect grace (default 20s, `DISCONNECT_GRACE`):

- **Waiting room:** the player stays seated with `connected: false`. Reconnect cancels the timer. If it fires they are removed (`player.left`). Empty rooms close so the passcode can be reused.
- **During a match:** the clock **pauses**. The opponent gets `opponent.disconnected`. Reconnect resumes the remaining seconds. If grace fires, the disconnected player **forfeits** (opponent wins, match is persisted). They keep their seat so a rematch is possible.

`POST /rooms/:id/start` (host, two players) creates the in-memory match, persists a `matches` row, and broadcasts `match.started`.

## Client → server

| `type` | Body | Effect |
| --- | --- | --- |
| `lobby.ready` / `ready` | `{}` | Waiting: lobby ready flag. Playing: lock this round's move (requires a prior `submit`). |
| `lobby.unready` | `{}` | Waiting-room only |
| `leave` | `{}` | Same as REST leave. Mid-match this forfeits, then removes the player. |
| `ping` | `{}` | `pong` |
| `submit` | `{ "move": "rock"\|"paper"\|"scissors" }` | Store a hidden playable move. Allowed until Ready. Changing the card replaces the previous one. |
| `claim_win` | `{}` | If the opponent is disconnected mid-match, forfeit them immediately (same as grace expiry). |
| `play_again` | `{}` | After `match.finished`. When both have sent it, a new match starts in the same room. |

Ready without a card → `error` `VALIDATION` (`ready requires a submitted move`). After Ready the move is locked.

## Round clock

The server owns the clock. Each round lasts `turnSeconds` (the room setting). `tick` broadcasts remaining whole seconds. When it hits 0 the hub calls `game.Match.Timeout(roundNo)`.

- A submitted-but-not-ready move **counts**.
- No move → the server picks rock, paper, or scissors at random (`yourAutoPicked` / `opponentAutoPicked` on `round.resolved`, `autoPicked` on history). The round then resolves with the usual cycle.
- The timeout closure captures `roundNo`. A late timer from round N cannot resolve round N+1 (`ErrStaleRound`).

After `round.resolved` there is a 3s pause (`nextRoundIn`) so the UI can show the result. Then the hub either broadcasts `round.started` (fresh `remainingSeconds`, turn clock running) or `match.finished`. Both players advance on that event so they stay in sync. Clients may also count down locally and dismiss the overlay if the event is late.

## Persistence

At round end the server writes `rounds` + `round_moves` (`auto_picked` is true when the timer chose the card). At match end it sets `matches.winner_id` / `ended_at` and increments `user_stats` (matches played, wins/losses, throw counts — auto-picks count as throws).

## Identity

REST and WS both trust the access JWT (`Authorization: Bearer` or `?token=`). Refresh tokens are rotated via `POST /auth/refresh`. Logout revokes the refresh token.

`POST /auth/guest`, `/auth/register`, `/auth/login`, `/auth/google` mint `{ accessToken, refreshToken, user }`. Guests can `POST /auth/upgrade` (same `users.id`).

`POST /dev/guest` is the same as `/auth/guest` and exists **only when `APP_ENV=dev`**. Production rejects it (404) and ignores `X-Player-ID`.
