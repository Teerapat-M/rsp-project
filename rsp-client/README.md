# RSP web client

React + Vite + TypeScript UI for the online Rock-Paper-Scissors game. Drag-and-drop uses `@dnd-kit`; routing is `react-router`. Layouts follow the Penpot boards (desktop 1440×810 and mobile 390×844).

The client talks to the Go server. Vite proxies `/api` and `/ws` to `http://localhost:8080`. Start `rsp-server` first (`make run`).

## Run

```bash
cd rsp-client
cp .env.example .env
npm install
npm run dev
```

Open http://localhost:5173. Leave `VITE_WS_URL` blank so the browser uses the same host. Set `VITE_GOOGLE_CLIENT_ID` (same value as `GOOGLE_CLIENT_ID` on the server) to show Continue with Gmail.

Hosted (Vercel): set Root Directory to `rsp-client`. `vercel.json` rewrites SPA routes to `index.html`. Build-time env:

```
VITE_API_BASE=https://rsp-api.onrender.com
VITE_WS_URL=wss://rsp-api.onrender.com/ws
VITE_GOOGLE_CLIENT_ID=
```

No trailing slash on `VITE_API_BASE`. The Go server is not under `/api` when it is reached directly (Render). Local Vite still uses `/api` and strips it in the proxy.

| Script | What |
| --- | --- |
| `npm run dev` | Vite dev server (port 5173), proxies `/api` and `/ws` to `:8080` |
| `npm run build` | `tsc -b` + production bundle |
| `npm run lint` | oxlint |
| `npm run preview` | serve the production build |

```
VITE_API_BASE=/api
VITE_WS_URL=
VITE_GOOGLE_CLIENT_ID=
```

## Layout

```
src/api          typed REST + WS client (HTTP to the Go server)
src/components   Card, Button, Input, Timer, Modal, Logo, Avatar
src/pages        one route per Penpot board
src/context      auth + live match
```
