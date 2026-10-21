# RSP web client

React + Vite + TypeScript UI for the online Rock-Paper-Scissors game. Drag-and-drop uses `@dnd-kit`; routing is `react-router`. Layouts follow the Penpot boards (desktop 1440×810 and mobile 390×844).

Phase 3 of the Go server implements create/join/leave rooms, waiting-room WebSocket, and live match play (countdown, hidden moves, resolve). The client still ships a **typed API + WebSocket contract** (`src/api`) and a **mock implementation** (on by default) so every screen is clickable end to end. Point `VITE_USE_MOCK=false` at the Go process for live rooms and matches; auth is still temporary until phase 4 (`/dev/guest` or `X-Player-ID`).

## Run

```bash
cd rsp-client
npm install
npm run dev
```

Open http://localhost:5173. The mock is on by default (`VITE_USE_MOCK` is not `false`).

| Script | What |
| --- | --- |
| `npm run dev` | Vite dev server (port 5173) |
| `npm run build` | `tsc -b` + production bundle |
| `npm run lint` | oxlint |
| `npm run preview` | serve the production build |

Copy `.env.example` to `.env` if you want to change the toggle:

```
VITE_USE_MOCK=true
VITE_API_BASE=/api
VITE_WS_URL=ws://localhost:8080/ws
```

Set `VITE_USE_MOCK=false` later to hit the Go server. Vite proxies `/api` to `http://localhost:8080`.

## Mock shortcuts

Sign in with any password as **Player123**, or **Continue as Guest**, or **Continue with Gmail**.

| Passcode | Result |
| --- | --- |
| `482910` or `000000` | Wrong passcode modal |
| `111111` or `482915` | Room is full modal |
| any other 6 digits | Join Bell’s room; match auto-starts |

Create a room to host: Bell joins after ~1s, then **Start Game**. Drag a card onto the drop slot and press **Ready**. The bot picks on a delay; the timer resolves the round if nobody locks in.

While playing (mock only), the footer links simulate **opponent disconnected** and **connection lost**. The player chip menu has **Screen gallery** (`/demo`) with one-click jumps to every board.

## Layout

```
src/api          typed REST + WS client, mock, HTTP stub
src/game/rules   move resolution used by the mock
src/components   Card, Button, Input, Timer, Modal, Logo, Avatar
src/pages        one route per Penpot board
src/context      auth + live match
```
