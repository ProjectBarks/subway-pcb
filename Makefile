PIO  := $(shell which pio 2>/dev/null || echo "$(HOME)/.platformio/penv/bin/pio")
PORT := $(shell ls /dev/cu.usbserial-* 2>/dev/null | head -1)

.DEFAULT_GOAL := help

# ─── Frontend ────────────────────────────────────────────

.PHONY: frontend/install frontend/build frontend/dev frontend/lint

frontend/install:                      ## Install frontend dependencies
	cd service/frontend && npm install

frontend/build:                        ## Build frontend (TypeScript + Vite)
	cd service/frontend && npm run build

frontend/dev:                          ## Watch and rebuild frontend on changes
	cd service/frontend && npm run dev

frontend/lint:                         ## Lint frontend TypeScript
	cd service/frontend && npm run lint

# ─── Backend ─────────────────────────────────────────────

.PHONY: backend/build backend/start backend/stop backend/dev

backend/build:                         ## Build the Go backend binary
	cd service/backend && go build ./cmd/subway-server/

backend/start: backend/build           ## Build and start backend → http://localhost:8080
	@pkill -9 -f subway-server 2>/dev/null; sleep 1
	cd service/backend && ./subway-server --port 8080 --led-map led_map.json --data-dir data --static-dir ../static &
	@echo "→ http://localhost:8080/"

backend/dev:                           ## Start backend with auto-reload on file changes
	cd service/backend && air

backend/stop:                          ## Stop the running backend
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

.PHONY: tools/monitor tools/debugger tools/viewer

tools/monitor:                         ## Stream ESP32 serial output to console + serial.log
	cd tools/serial-logger && uv run main.py

tools/debugger:                        ## Start click-to-light web debugger → http://localhost:8090
	cd tools/led-debugger-ui && uv run debugger.py --port $(PORT) --http 8090

tools/viewer:                          ## Start standalone board viewer → http://localhost:8888
	cd tools/board-viewer && uv run serve.py

# ─── Site ────────────────────────────────────────────────

.PHONY: site/build site/preview

site/build:                            ## Build static landing page → _site/
	cd service/backend && templ generate ./internal/ui/ && go run ./cmd/generate-site/ ../../_site
	cp -r service/frontend/public/* _site/ 2>/dev/null || true

site/preview: site/build               ## Build and open landing page in browser
	open _site/index.html

# ─── Shortcuts ───────────────────────────────────────────

.PHONY: all clean help

all: backend/build firmware/build      ## Build backend + firmware

clean:                                 ## Remove all build artifacts
	rm -rf firmware/.pio/build tools/debug-firmware/.pio/build service/backend/subway-server

help:                                  ## Show available commands
	@echo ""
	@grep -E '^[a-z/._-]+:.*##' $(MAKEFILE_LIST) | \
		awk -F ':.*## ' '{printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2}'
	@echo ""
