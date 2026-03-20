# Instructions

Never include "Co-Authored-By" or "Generated with Claude Code" in commit
messages.

## Make Commands

| Command | Description |
|---|---|
| `make frontend/install` | Install frontend npm dependencies |
| `make frontend/build` | Build frontend (TypeScript + Vite) |
| `make frontend/dev` | Watch and rebuild frontend on changes |
| `make frontend/lint` | Lint frontend TypeScript |
| `make backend/dev` | Start backend with auto-reload on file changes (requires `air`) |
| `make backend/build` | Build the Go backend binary |
| `make backend/start` | Build and start backend at http://localhost:8080 |
| `make backend/stop` | Stop the running backend |
| `make firmware/build` | Compile production firmware |
| `make firmware/flash` | Compile and flash to ESP32 |
| `make firmware/erase` | Erase entire ESP32 flash |
| `make firmware/clean` | Delete firmware build artifacts |
| `make firmware/debug-build` | Compile debug firmware (serial LED control) |
| `make firmware/debug-flash` | Compile and flash debug firmware |
| `make tools/monitor` | Stream ESP32 serial output to console |
| `make tools/debugger` | Start click-to-light web debugger at http://localhost:8090 |
| `make all` | Build backend + firmware |
| `make clean` | Remove all build artifacts |
| `make help` | Show available commands |

## Dev Setup

Install `air` for live reload: `go install github.com/air-verse/air@latest`

## Service

Unified Go + TypeScript service (`service/`):

- Go module: `github.com/ProjectBarks/subway-pcb/service`
- Build Go: `cd service && go build ./cmd/subway-server/`
- Build frontend: `cd service && npm run build`
- Deployed on Railway with MySQL (`MYSQL_DSN` env var)
- Local dev uses bbolt (embedded, zero config)

## Project Structure

```
service/
  cmd/
    subway-server/         — entry point
    generate-site/         — static site generator
    gen-ui-exports/        — barrel export generator
  internal/
    api/                   — HTTP handlers, pixel rendering
    plugin/                — mode interface + registry
      track/               — live subway map mode
      snake/               — animated snake mode
    model/                 — domain types
    store/                 — persistence interface
      bolt/                — bbolt backend
      mysql/               — GORM MySQL backend
    middleware/             — auth + authz
    mta/                   — MTA feed aggregator
  ui/                      — re-exports all page + component templates
    components/            — re-exports shared pieces (icon, nav, toast, layout)
      icon/                — SVG icon helpers
      nav/                 — navigation bar + JS toggle
      toast/               — toast notifications + CSS
      layout/              — Base HTML wrapper + global CSS + app-shell JS
    board/                 — board detail page (templ + TS + CSS)
    dashboard/             — device dashboard page
    community/             — community plugin browser
    editor/                — Lua plugin editor (templ + Preact)
    landing/               — public landing page (templ + Three.js + CSS)
    login/                 — OAuth login page
    lib/                   — pure TS (board renderer, serial, protobuf, hero-board)
  gen/subwaypb/            — protobuf generated code
  public/                  — static assets (leds.json, board.svg, etc.)
  static/dist/             — Vite build output (gitignored)
firmware/                  — ESP32 firmware (PlatformIO)
tools/                     — serial monitor, LED debugger, PCB GLB generator
pcb/                       — PCB gerber files, 3D models, viewers
```
