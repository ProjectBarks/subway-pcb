PIO  := $(shell which pio 2>/dev/null || echo "$(HOME)/.platformio/penv/bin/pio")
PORT := $(shell ls /dev/cu.usbserial-* 2>/dev/null | head -1)

.DEFAULT_GOAL := help

# ─── Server ──────────────────────────────────────────────

.PHONY: server/build server/start server/stop server/dev

server/build:                          ## Build the Go server binary
	cd server && go build ./cmd/subway-server/

server/start: server/build             ## Build and start server → http://localhost:8080
	@pkill -9 -f subway-server 2>/dev/null; sleep 1
	cd server && ./subway-server --port 8080 --led-map led_map.json --data-dir data --template-dir templates &
	@echo "→ http://localhost:8080/"

server/dev:                            ## Start server with auto-reload on file changes
	cd server && air

server/stop:                           ## Stop the running server
	@pkill -9 -f subway-server 2>/dev/null && echo "Stopped" || echo "Not running"

# ─── Firmware (production) ───────────────────────────────

.PHONY: firmware/build firmware/flash firmware/erase firmware/clean

firmware/build:                        ## Compile production firmware
	cd firmware && $(PIO) run

firmware/flash:                        ## Compile and flash to ESP32
	cd firmware && $(PIO) run -t upload --upload-port $(PORT)

firmware/erase:                        ## Erase entire ESP32 flash
	cd firmware && $(PIO) run -t erase --upload-port $(PORT)

firmware/clean:                        ## Delete build artifacts
	rm -rf firmware/.pio/build

# ─── Firmware (debug) ────────────────────────────────────

.PHONY: firmware/debug-build firmware/debug-flash

firmware/debug-build:                  ## Compile debug firmware (serial LED control)
	cd tools/debug-firmware && $(PIO) run

firmware/debug-flash:                  ## Compile and flash debug firmware
	cd tools/debug-firmware && $(PIO) run -t upload --upload-port $(PORT)

# ─── Tools ───────────────────────────────────────────────

.PHONY: tools/monitor tools/debugger

tools/monitor:                         ## Stream ESP32 serial output to console + serial.log
	cd tools/serial-logger && uv run main.py

tools/debugger:                        ## Start click-to-light web debugger → http://localhost:8090
	cd tools/led-debugger-ui && uv run debugger.py --port $(PORT) --http 8090

# ─── Shortcuts ───────────────────────────────────────────

.PHONY: all clean help

all: server/build firmware/build       ## Build server + firmware

clean:                                 ## Remove all build artifacts
	rm -rf firmware/.pio/build tools/debug-firmware/.pio/build server/subway-server

help:                                  ## Show available commands
	@echo ""
	@grep -E '^[a-z/._-]+:.*##' $(MAKEFILE_LIST) | \
		awk -F ':.*## ' '{printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2}'
	@echo ""
