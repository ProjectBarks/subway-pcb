PIO := $(shell which pio 2>/dev/null || echo "$(HOME)/.platformio/penv/bin/pio")
PORT := $(shell ls /dev/cu.usbserial-* 2>/dev/null | head -1)

# ── Server ───────────────────────────────────────────────
.PHONY: server server-build server-run server-stop

server-build:
	cd server && go build ./cmd/subway-server/

server-run: server-build
	@pkill -f subway-server 2>/dev/null; sleep 1
	cd server && ./subway-server --port 8080 --led-map led_map.json &
	@echo "Server at http://localhost:8080/"

server-stop:
	@pkill -f subway-server 2>/dev/null && echo "Stopped" || echo "Not running"

server: server-run

# ── Firmware ─────────────────────────────────────────────
.PHONY: firmware firmware-build firmware-flash firmware-monitor firmware-erase firmware-clean

firmware-build:
	cd firmware && $(PIO) run

firmware-flash:
	cd firmware && $(PIO) run -t upload --upload-port $(PORT)

firmware-monitor:
	cd tools && python3 serial_log.py

firmware-erase:
	cd firmware && $(PIO) run -t erase --upload-port $(PORT)

firmware-clean:
	cd firmware && rm -rf .pio/build

# ── Debug Firmware ───────────────────────────────────────
.PHONY: debug-build debug-flash

debug-build:
	cd tools/debug-firmware && $(PIO) run

debug-flash:
	cd tools/debug-firmware && $(PIO) run -t upload --upload-port $(PORT)

# ── LED Debugger UI ──────────────────────────────────────
.PHONY: debugger

debugger:
	cd tools/led-debugger-ui && python3 debugger.py --port $(PORT) --http 8090

# ── All ──────────────────────────────────────────────────
.PHONY: all clean

all: server-build firmware-build

clean:
	cd firmware && rm -rf .pio/build
	cd tools/debug-firmware && rm -rf .pio/build
	cd server && rm -f subway-server

# ── Help ─────────────────────────────────────────────────
.PHONY: help

help:
	@echo "Server:"
	@echo "  make server          Build and start Go server (port 8080)"
	@echo "  make server-stop     Stop server"
	@echo ""
	@echo "Firmware:"
	@echo "  make firmware-build  Build production firmware"
	@echo "  make firmware-flash  Build and flash to ESP32"
	@echo "  make firmware-monitor Start serial logger"
	@echo "  make firmware-erase  Erase ESP32 flash"
	@echo ""
	@echo "Debug:"
	@echo "  make debug-flash     Flash debug firmware (serial LED control)"
	@echo "  make debugger        Start LED debugger web UI (port 8090)"
	@echo ""
	@echo "Options:"
	@echo "  PORT=/dev/cu.usbserial-210  Serial port (default)"
