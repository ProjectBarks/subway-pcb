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

The service is split into frontend and backend:

- **Backend**: Go API server (`service/backend/`)
  - Go module: `github.com/ProjectBarks/subway-pcb/server`
  - Build: `cd service/backend && go build ./cmd/subway-server/`
  - Deployed on Railway with MySQL (`MYSQL_DSN` env var)
  - Local dev uses bbolt (embedded, zero config)

- **Frontend**: TypeScript + Vite (`service/frontend/`)
  - Deployed statically (not served by Go backend)
  - Build: `cd service/frontend && npm run build`

## Project Structure

```
service/
  backend/
    cmd/subway-server/     — entry point
    internal/
      api/                 — HTTP handlers, pixel rendering
      mode/                — mode interface + registry
        track/             — live subway map mode
        snake/             — animated snake mode
      model/               — domain types
      store/               — persistence interface
        bolt/              — bbolt backend
        mysql/             — GORM MySQL backend
      middleware/           — auth + authz
      mta/                 — MTA feed aggregator
      ui/                  — templ components (type-safe HTML rendering)
  frontend/
    src/                   — TypeScript source
      entries/             — Vite entry points
      lib/                 — shared modules (board, protobuf, preview)
      global/              — global UI (nav, toast, forms)
    public/                — static assets (leds.json, board.svg, etc.)
firmware/                  — ESP32 firmware (PlatformIO)
tools/                     — serial monitor, LED debugger, PCB GLB generator
pcb/                       — PCB gerber files, 3D models, viewers
```
