# NYC Subway PCB - Real-Time Train Display System

## Context

The subway-pcb project is a physical NYC subway map with 478 WS2812B LEDs, each representing a station. The hardware (ESP32-WROOM-32 PCB with 9 LED strips) and LED-to-station mapping (`mapper/ui/led_map.json`) already exist. The goal is to bring it to life: a **Go server** polls MTA's free GTFS real-time feeds, aggregates train positions, and serves the current subway state as a protobuf message. The ESP32 firmware fetches this state, maps it to its own LED layout, and renders train positions with route-appropriate colors.

WiFi provisioning (captive portal) and OTA updates (GitHub releases) are already handled by existing libraries in `firmware/lib/`.

---

## Architecture Overview

```
MTA GTFS-RT Feeds (9 feeds, protobuf, free, no auth)
         |  polled every 15s
         v
   ┌──────────────┐
   │   Go Server   │  Decodes GTFS, aggregates into SubwayState protobuf
   │  (single bin)  │  Serves via GET /api/v1/state (binary protobuf)
   └──────┬────────┘
          │  HTTP GET every 10-15s (~2-3KB protobuf)
          v
   ┌──────────────┐
   │   ESP32 PCB   │  Decodes SubwayState (nanopb), maps stop_ids to
   │  (firmware)   │  LEDs, picks colors from route, drives 9 strips
   └──────────────┘
```

**Key design decisions:**
- **Protocol models trains, not LEDs.** The server sends semantic subway data (stations + trains + routes). The firmware decides how to render it.
- **Protobuf with nanopb** for schema evolution. Old firmware ignores new fields. New firmware handles missing fields with defaults. Backwards-compatible as long as we follow protobuf rules (no removing/renumbering fields).
- **Board-agnostic.** Ship a new PCB layout? Update the firmware's station-to-LED mapping. Protocol stays the same.
- **Route colors live in firmware.** The server doesn't know or care about LED colors — it sends route identifiers, the board decides what color that is.

---

## Project Structure

```
subway-pcb/
├── proto/
│   └── subway.proto                 # Shared schema (source of truth)
├── server/                          # Go server (NEW)
│   ├── cmd/subway-server/main.go    # Entry point
│   ├── internal/
│   │   ├── mta/
│   │   │   ├── feeds.go             # Feed URLs + poller goroutines
│   │   │   ├── gtfsrt.go            # GTFS-RT protobuf decoding
│   │   │   └── aggregator.go        # Merge feeds -> SubwayState
│   │   └── api/
│   │       └── server.go            # HTTP routes: /state, /health
│   ├── gen/                         # Generated Go protobuf code
│   │   └── subway.pb.go
│   ├── go.mod
│   └── Dockerfile
├── firmware/                        # ESP32 firmware (EXISTING, extend)
│   ├── src/
│   │   ├── main.c                   # App entry: WiFi -> OTA -> poller -> LEDs
│   │   ├── led_driver.c/.h          # 9-strip WS2812B control (8 RMT + 1 SPI)
│   │   ├── subway_client.c/.h       # HTTP GET poller, protobuf decode (nanopb)
│   │   ├── renderer.c/.h            # SubwayState -> LED colors (route->color, persistence)
│   │   ├── station_map.c/.h         # Auto-generated: stop_id -> (strip, pixel) lookup
│   │   └── config.h                 # Pin assignments, server URL, timing constants
│   ├── proto/
│   │   ├── subway.pb.c/.h           # nanopb-generated from subway.proto
│   │   └── subway.options           # nanopb field size constraints
│   ├── lib/
│   │   ├── esp_ghota/               # EXISTING - GitHub OTA updates
│   │   ├── esp32-wifi-manager/      # EXISTING - Captive portal WiFi setup
│   │   └── nanopb/                  # nanopb protobuf library
│   ├── platformio.ini               # EXISTING - add led_strip dependency
│   ├── partitions.csv               # EXISTING - 1MB OTA slots, 960KB SPIFFS
│   └── sdkconfig.defaults           # EXISTING - WiFi/OTA config
├── mapper/                          # EXISTING - LED mapping tool
├── pcb/                             # EXISTING - KiCad hardware design
└── mcp/                             # EXISTING - Camera capture for Claude
```

---

## Part 1: Shared Protobuf Schema (`proto/subway.proto`)

The schema models the subway system, not the board. Both Go and ESP32 compile from this single source.

```protobuf
syntax = "proto3";
package subway;

enum Route {
  ROUTE_UNKNOWN = 0;
  ROUTE_1 = 1;   ROUTE_2 = 2;   ROUTE_3 = 3;
  ROUTE_4 = 4;   ROUTE_5 = 5;   ROUTE_6 = 6;
  ROUTE_7 = 7;
  ROUTE_A = 8;   ROUTE_B = 9;   ROUTE_C = 10;
  ROUTE_D = 11;  ROUTE_E = 12;  ROUTE_F = 13;
  ROUTE_G = 14;
  ROUTE_J = 15;  ROUTE_L = 16;  ROUTE_M = 17;
  ROUTE_N = 18;  ROUTE_Q = 19;  ROUTE_R = 20;
  ROUTE_W = 21;  ROUTE_Z = 22;
  ROUTE_S = 23;       // 42nd St Shuttle
  ROUTE_FS = 24;      // Franklin Ave Shuttle
  ROUTE_GS = 25;      // Grand Central Shuttle
  ROUTE_SI = 26;      // Staten Island Railway
}

enum TrainStatus {
  STATUS_NONE = 0;
  STOPPED_AT = 1;
  INCOMING_AT = 2;
  IN_TRANSIT_TO = 3;
}

message Train {
  Route route = 1;
  TrainStatus status = 2;
}

message Station {
  string stop_id = 1;         // MTA parent stop ID: "101", "A02", etc.
  repeated Train trains = 2;  // active trains at this station
}

message SubwayState {
  uint64 timestamp = 1;        // Unix timestamp
  uint32 sequence = 2;         // incrementing frame counter
  repeated Station stations = 3; // only stations with active trains (sparse)
}
```

**nanopb options** (`firmware/proto/subway.options`):
```
subway.Station.stop_id    max_size:5
subway.Station.trains     max_count:4
subway.SubwayState.stations max_count:500
```

**Why this works:**
- Sparse: only stations with trains are included (~200-300 at peak). Estimated ~2-3KB per message.
- Semantic: a new board revision just updates its stop_id-to-LED mapping. The proto doesn't change.
- Evolvable: add `Direction direction = 3` to Train later — old firmware ignores it. Add `repeated Alert alerts = 4` to SubwayState — old firmware skips it.

---

## Part 2: Go Server (`server/`)

### 2.1 MTA Feed Polling (`internal/mta/`)

**Feeds to poll** (no auth required, HTTP GET returns protobuf):

| Feed | Lines | URL suffix |
|------|-------|-----------|
| Main | 1,2,3,4,5,6,S | `gtfs` |
| ACE | A,C,E | `gtfs-ace` |
| NQRW | N,Q,R,W | `gtfs-nqrw` |
| BDFM | B,D,F,M | `gtfs-bdfm` |
| L | L | `gtfs-l` |
| G | G | `gtfs-g` |
| JZ | J,Z | `gtfs-jz` |
| 7 | 7,GS | `gtfs-7` |
| SI | SIR | `gtfs-si` |

Base URL: `https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct/`

- One goroutine per feed, polling every 15s (staggered)
- Decode GTFS-RT protobuf using `github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs`
- Extract `TripUpdate` and `VehiclePosition` entities
- Normalize stop_id: `"101N"` -> `"101"`, `"A02S"` -> `"A02"`

### 2.2 Aggregator (`internal/mta/aggregator.go`)

- Maintains `map[string]*StationState` keyed by parent stop_id
- Each station tracks its active trains (route + status)
- **10-second persistence**: once a train appears at a station, it persists for at least 10s even if it disappears from the next feed update (prevents flicker)
- On each feed update, builds a `SubwayState` protobuf message
- Serializes to binary protobuf, caches the latest bytes

### 2.3 HTTP API (`internal/api/`)

| Endpoint | Purpose |
|----------|---------|
| `GET /api/v1/state` | Latest SubwayState as binary protobuf (`application/x-protobuf`) |
| `GET /api/v1/state?format=json` | Same data as JSON (debugging/web dashboard) |
| `GET /health` | Server status, last update time |

ESP32 polls `GET /api/v1/state` every 10-15 seconds. Response is ~2-3KB of protobuf. Simple, stateless, cacheable.

### 2.4 Server Deployment (Cloud VPS)

- Single static binary: `go build -o subway-server ./cmd/subway-server/`
- Config via flags/env: `PORT`, `POLL_INTERVAL`
- Dockerfile: multi-stage build -> `FROM scratch` + binary
- Deploy to cloud VPS (DigitalOcean, Fly.io, etc.)
- TLS termination via reverse proxy (Caddy/nginx) or platform-native

---

## Part 3: ESP32 Firmware (`firmware/`)

### 3.1 LED Driver (`src/led_driver.c`)

**Problem:** ESP32 has 8 RMT channels but 9 LED strips.
**Solution:** 8 strips on RMT backend, 1 strip (pin 26, 11 LEDs) on SPI backend.

- Add `espressif/led_strip: "^2.5"` to `src/idf_component.yml`
- Uses ESP-IDF `led_strip` component (same API for RMT and SPI backends)
- Pin assignments: GPIO 16-23, 25, 26 (matching bridge.ino and PCB schematic)
- Default brightness: ~6% (15/255) to manage power draw
- All strips refreshed after full frame buffer update

**Strip config:**
| Strip | GPIO | LEDs | Backend |
|-------|------|------|---------|
| 0 | 16 | 97 | RMT ch0 |
| 1 | 17 | 102 | RMT ch1 |
| 2 | 18 | 54 | RMT ch2 |
| 3 | 19 | 80 | RMT ch3 |
| 4 | 21 | 70 | RMT ch4 |
| 5 | 22 | 21 | RMT ch5 |
| 6 | 23 | 22 | RMT ch6 |
| 7 | 25 | 19 | RMT ch7 |
| 8 | 26 | 11 | SPI |

### 3.2 Subway Client (`src/subway_client.c`)

- Uses `esp_http_client` to poll `GET /api/v1/state` every 10-15 seconds
- Receives binary protobuf response (~2-3KB)
- Decodes with **nanopb** into `SubwayState` struct (zero dynamic allocation)
- Passes decoded state to renderer
- Server URL stored in NVS (configurable), with compiled-in default

### 3.3 Renderer (`src/renderer.c`)

This is where subway data becomes LED output. The renderer owns all display logic:

- **Route-to-color table** (firmware-side, not protocol-side):
  ```c
  static const uint8_t route_colors[][3] = {
      [ROUTE_1] = {238,53,46},   // Red
      [ROUTE_2] = {238,53,46},   // Red
      [ROUTE_3] = {238,53,46},   // Red
      [ROUTE_4] = {0,147,60},    // Green
      // ...etc for all 26 routes
  };
  ```
- **Status-to-brightness**: `STOPPED_AT` = full, `INCOMING_AT` = pulse, `IN_TRANSIT_TO` = dimmer
- **10-second persistence**: tracks `last_change_time` per LED, won't switch color until 10s elapsed
- **Multi-train resolution**: at interchange stations, picks train with highest-priority status

### 3.4 Station Map (`src/station_map.c`)

Auto-generated at build time from `mapper/ui/led_map.json`. A script converts the JSON into a C lookup table:

```c
// Generated from led_map.json — do not edit
typedef struct { uint8_t strip; uint16_t pixel; } led_pos_t;

// Hash map or sorted array: stop_id -> led_pos_t
// "101" -> {strip=2, pixel=47}
// "A02" -> {strip=1, pixel=80}
```

~476 entries * ~6 bytes each = ~2.9KB. Fits comfortably in flash.

### 3.5 Main Application Flow (`src/main.c`)

```
app_main()
  1. nvs_flash_init()
  2. led_driver_init() + boot animation
  3. wifi_manager_start() with callback
  4. ghota_init() + ghota_start_update_timer() (check every 60 min)

on_wifi_connected():
  5. xTaskCreate(subway_client_task)
     -> polls /api/v1/state every 10s
     -> nanopb decode -> renderer -> led_driver
```

**FreeRTOS tasks:**
| Task | Stack | Priority | Purpose |
|------|-------|----------|---------|
| wifi_manager | 4KB | 5 | WiFi provisioning/reconnect |
| subway_client | 8KB | 4 | HTTP polling + protobuf decode + render |
| ghota_timer | 4KB | 2 | OTA update checks |

### 3.6 Existing Libraries (reused as-is)

- **esp32-wifi-manager** (`lib/esp32-wifi-manager/`): Captive portal on SSID "nyc-subway-pcb", auto-reconnects, auto-shuts-down AP after connection. **No modifications needed.**
- **esp_ghota** (`lib/esp_ghota/`): OTA from `ProjectBarks/subway-pcb` GitHub releases, A/B slot switching with rollback. **No modifications needed.**

### 3.7 nanopb Integration

- Add nanopb as a git submodule in `lib/nanopb/` or via PlatformIO library
- `.proto` compiled to `.pb.c` / `.pb.h` using `nanopb_generator`
- `.options` file constrains field sizes for static allocation
- Decode buffer: ~4KB static buffer for incoming protobuf message

---

## Part 4: Data Flow (end-to-end)

```
MTA feed "101N" STOPPED_AT on route "1"
  ↓ server strips N/S suffix
Server aggregates: station "101" has Train{route=ROUTE_1, status=STOPPED_AT}
  ↓ serialized to protobuf SubwayState
ESP32 fetches GET /api/v1/state
  ↓ nanopb decode
Renderer looks up "101" in station_map → strip=2, pixel=47
  ↓ looks up ROUTE_1 in route_colors → (238,53,46) Red
  ↓ applies brightness for STOPPED_AT (full)
led_driver sets strip 2, pixel 47 to (238,53,46)
```

**Separation of concerns:**
- Server knows about MTA feeds and aggregation. Does NOT know about LEDs or colors.
- Firmware knows about LEDs, colors, and rendering. Does NOT know about MTA feeds.
- Protobuf schema is the contract between them — models trains, not hardware.

---

## Implementation Order

### Phase 1: Proto Schema + Code Generation
1. Write `proto/subway.proto`
2. Generate Go code: `protoc --go_out=server/gen proto/subway.proto`
3. Add nanopb submodule to `firmware/lib/nanopb/`
4. Write `firmware/proto/subway.options` (field size constraints)
5. Generate nanopb C code: `nanopb_generator proto/subway.proto`
6. **Test:** both Go and C compile with generated code

### Phase 2: Go Server
1. `go mod init`, add GTFS bindings + protobuf dependencies
2. Implement `internal/mta/feeds.go` - poll one feed (1/2/3/4/5/6)
3. Implement `internal/mta/gtfsrt.go` - GTFS-RT decode
4. Implement `internal/mta/aggregator.go` - merge feeds into SubwayState, 10s persistence
5. Add all 9 feed pollers
6. Implement `internal/api/` with `GET /api/v1/state`
7. **Test:** `curl localhost:8080/api/v1/state?format=json | jq '.stations | length'`

### Phase 3: Firmware LED Driver
1. Add `espressif/led_strip` to `idf_component.yml`
2. Create `src/config.h` with pin/strip definitions
3. Implement `src/led_driver.c` - 8 RMT + 1 SPI strip init
4. Test with static colors (hardcoded palette walk)
5. **Test:** `pio run -t upload`, verify LEDs light up

### Phase 4: Firmware Subway Client + Renderer
1. Generate `src/station_map.c` from `led_map.json` (build script)
2. Implement `src/renderer.c` - route->color table, status->brightness, persistence
3. Implement `src/subway_client.c` - HTTP GET + nanopb decode
4. Wire up `src/main.c` - WiFi callback starts subway client
5. **Test:** ESP32 connects to local server, LEDs display live subway data

### Phase 5: Integration & Polish
1. End-to-end test with live MTA data
2. OTA update test via GitHub release
3. WiFi reconnect / server disconnect handling
4. Error recovery (bad responses, network loss)
5. Brightness tuning based on power measurement
6. Dockerfile for cloud VPS deployment

---

## Verification Plan

### Server Tests
```bash
# Unit tests
go test ./internal/...

# Verify MTA feed access
curl -s https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct/gtfs | wc -c

# Verify SubwayState as JSON
curl 'localhost:8080/api/v1/state?format=json' | jq '.stations | length'

# Verify SubwayState as protobuf (binary)
curl -s localhost:8080/api/v1/state | protoc --decode=subway.SubwayState proto/subway.proto

# Health check
curl localhost:8080/health
```

### Firmware Tests
```bash
# Build (must fit in 1MB OTA slot)
pio run

# Upload
pio run -t upload --upload-port /dev/cu.usbserial-210

# Monitor serial output
pio device monitor -p /dev/cu.usbserial-210
```

### End-to-End Verification
1. Flash firmware -> boot animation plays on all 9 strips
2. Connect phone to "nyc-subway-pcb" WiFi AP -> configure home WiFi
3. ESP32 polls server -> serial log shows decoded station count
4. LEDs light up with route colors matching live subway positions
5. Kill server -> ESP32 retries, reconnects when server returns
6. Tag a GitHub release -> ESP32 auto-updates firmware via OTA

### Hardware Confirmed
- **ESP32 detected:** `/dev/cu.usbserial-210` (CH340C, VID:PID=1A86:7523)
- **PlatformIO:** `/Users/alexgaribaldi/.platformio/penv/bin/pio`
- **Build verified:** firmware compiles (RAM: 3.2% = 10KB/320KB, Flash: 16.6% = 174KB/1MB)

---

## Critical Files

| File | Action | Purpose |
|------|--------|---------|
| `proto/subway.proto` | Create | Shared schema (trains, stations, routes) |
| `firmware/proto/subway.options` | Create | nanopb field size constraints |
| `firmware/proto/subway.pb.c/.h` | Generate | nanopb C code from proto |
| `firmware/src/main.c` | Rewrite | Full app_main implementation |
| `firmware/src/led_driver.c/.h` | Create | 9-strip LED control (8 RMT + 1 SPI) |
| `firmware/src/subway_client.c/.h` | Create | HTTP GET poller + nanopb decode |
| `firmware/src/renderer.c/.h` | Create | Route->color, status->brightness, persistence |
| `firmware/src/station_map.c/.h` | Generate | stop_id->(strip,pixel) from led_map.json |
| `firmware/src/config.h` | Create | Pin assignments, server URL, timing |
| `firmware/src/idf_component.yml` | Edit | Add `espressif/led_strip` dep |
| `firmware/platformio.ini` | Edit | Add component registry source |
| `server/cmd/subway-server/main.go` | Create | Server entry point |
| `server/internal/mta/feeds.go` | Create | MTA feed polling |
| `server/internal/mta/gtfsrt.go` | Create | GTFS-RT decoding |
| `server/internal/mta/aggregator.go` | Create | Feed aggregation + persistence |
| `server/internal/api/server.go` | Create | HTTP routes |
| `server/gen/subway.pb.go` | Generate | Go protobuf code |
| `server/go.mod` | Create | Go module |
| `server/Dockerfile` | Create | Container build |

---

## Resolved Decisions

- **Wire format:** Protobuf (`subway.proto`) — models trains and stations, not LEDs or colors. Backwards-compatible schema evolution.
- **ESP32 protobuf:** nanopb — zero-allocation C structs, ~5KB code footprint.
- **Transport:** HTTP GET polling (simple, stateless). ESP32 polls every 10-15s.
- **Rendering:** Firmware-side. Route->color mapping, status->brightness, 10s persistence all live in `renderer.c`.
- **Station mapping:** Firmware-side. Auto-generated `station_map.c` from `led_map.json`. New board = new map, same proto.
- **Server hosting:** Cloud VPS (Docker deployment).
- **WiFi provisioning:** Existing `esp32-wifi-manager` (captive portal, no changes).
- **OTA updates:** Existing `esp_ghota` (GitHub releases, no changes).
- **Multi-line stations:** Highest-priority status wins (`STOPPED_AT` > `INCOMING_AT` > `IN_TRANSIT_TO`), then most recent, with 10s persistence minimum.
