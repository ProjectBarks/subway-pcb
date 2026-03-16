PIO  := $(shell which pio 2>/dev/null || echo "$(HOME)/.platformio/penv/bin/pio")
PORT := $(shell ls /dev/cu.usbserial-* 2>/dev/null | head -1)

.DEFAULT_GOAL := help

# ─── Server ──────────────────────────────────────────────

.PHONY: server server-build server-stop

server-build:                          ## Build Go server
	cd server && go build ./cmd/subway-server/

server: server-build                   ## Start Go server on :8080
	@pkill -9 -f subway-server 2>/dev/null; sleep 1
	cd server && ./subway-server --port 8080 --led-map led_map.json &
	@echo "→ http://localhost:8080/"

server-stop:                           ## Stop Go server
	@pkill -9 -f subway-server 2>/dev/null && echo "Stopped" || echo "Not running"

# ─── Production Firmware ─────────────────────────────────

.PHONY: fw-build fw-flash fw-erase fw-clean

fw-build:                              ## Build production firmware
	cd firmware && $(PIO) run

fw-flash:                              ## Flash production firmware
	cd firmware && $(PIO) run -t upload --upload-port $(PORT)

fw-erase:                              ## Erase ESP32 flash
	cd firmware && $(PIO) run -t erase --upload-port $(PORT)

fw-clean:                              ## Clean firmware build
	rm -rf firmware/.pio/build

# ─── Debug Firmware ──────────────────────────────────────

.PHONY: dbg-build dbg-flash

dbg-build:                             ## Build debug firmware (serial LED control)
	cd tools/debug-firmware && $(PIO) run

dbg-flash:                             ## Flash debug firmware
	cd tools/debug-firmware && $(PIO) run -t upload --upload-port $(PORT)

# ─── Tools ───────────────────────────────────────────────

.PHONY: monitor debugger

monitor:                               ## Start serial logger (auto-detect port)
	cd tools/serial-logger && uv run main.py

debugger:                              ## Start LED debugger web UI on :8090
	cd tools/led-debugger-ui && uv run debugger.py --port $(PORT) --http 8090

# ─── Housekeeping ────────────────────────────────────────

.PHONY: all clean help

all: server-build fw-build             ## Build everything

clean:                                 ## Clean all build artifacts
	rm -rf firmware/.pio/build
	rm -rf tools/debug-firmware/.pio/build
	rm -f server/subway-server

help:                                  ## Show this help
	@grep -E '^[a-z_-]+:.*##' $(MAKEFILE_LIST) | \
		awk -F ':.*## ' '{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'
