PIO  := $(shell which pio 2>/dev/null || echo "$(HOME)/.platformio/penv/bin/pio")
PORT := $(shell ls /dev/cu.usbserial-* 2>/dev/null | head -1)
BOARDS_DIR := public/boards

.DEFAULT_GOAL := help

# ─── Frontend ────────────────────────────────────────────

.PHONY: frontend/install frontend/build frontend/dev frontend/lint frontend/typecheck

frontend/install:                      ## Install frontend dependencies
	cd service && npm ci

frontend/build:                        ## Build frontend (TypeScript + Vite)
	cd service && npm run build

frontend/dev:                          ## Watch and rebuild frontend on changes
	cd service && npm run dev

frontend/lint:                         ## Lint frontend with Biome
	cd service && npx biome check

frontend/typecheck:                    ## Typecheck frontend TypeScript
	cd service && npx tsc --noEmit

# ─── Backend ─────────────────────────────────────────────

.PHONY: backend/generate backend/build backend/start backend/stop backend/dev

backend/generate:                      ## Generate templ + Go code
	cd service && go tool templ generate && go generate ./ui/...

backend/build: backend/generate        ## Build the Go backend binary
	cd service && go build ./cmd/subway-server/

backend/start: backend/build           ## Build and start backend → http://localhost:8080
	@pkill -9 -f subway-server 2>/dev/null; sleep 1
	cd service && ./subway-server --port 8080 --boards-dir $(BOARDS_DIR) --data-dir data --static-dir static &
	@echo "→ http://localhost:8080/"

backend/dev: backend/generate          ## Start backend with auto-reload on file changes
	cd service && air

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

# ─── Proto ───────────────────────────────────────────────

.PHONY: proto/generate

proto/generate:                        ## Regenerate protobuf code (Go + nanopb)
	protoc -I proto --go_out=service/gen \
		--go_opt=module=github.com/ProjectBarks/subway-pcb/service \
		proto/subway.proto
	protoc -I proto --nanopb_out=firmware/proto \
		--nanopb_opt=-f proto/subway.options \
		proto/subway.proto

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

# ─── Unit Tests ─────────────────────────────────────────

ALLURE_RESULTS := .test-results

.PHONY: test test/go test/frontend test/firmware report

test: test/go test/frontend test/firmware  ## Run all unit tests

test/go:                                   ## Run Go unit tests
	@mkdir -p $(ALLURE_RESULTS)
	cd service && go run gotest.tools/gotestsum@latest --junitfile ../$(ALLURE_RESULTS)/go.xml -- ./...

test/frontend: frontend/install            ## Run frontend unit tests (Lua conformance)
	@mkdir -p $(ALLURE_RESULTS)
	cd service && npx vitest run

test/firmware:                             ## Run firmware Lua conformance + E2E + stress tests (host)
	cd firmware/test && make test ALLURE_RESULTS=../../$(ALLURE_RESULTS)

test/report: test                               ## Run all tests and open Allure report
	npx allure generate -o .test-report .test-results
	npx allure open .test-report

# ─── E2E Tests ──────────────────────────────────────────

.PHONY: e2e/install e2e/test e2e/test-headed

e2e/install:                              ## Install E2E test deps + Chromium browser
	cd service && npm ci
	cd service && npx playwright install chromium --with-deps

E2E_PORT ?= 9199

e2e/test: frontend/build backend/build   ## Run E2E tests (headless)
	@mkdir -p service/e2e/screenshots service/e2e/reports
	@E2E_DATA=$$(mktemp -d); \
	service/subway-server --port $(E2E_PORT) --boards-dir service/$(BOARDS_DIR) --data-dir "$$E2E_DATA" --static-dir service/static --dev >>"$$E2E_DATA/server.log" 2>&1 & SERVER_PID=$$!; \
	trap 'kill $$SERVER_PID 2>/dev/null; rm -rf "$$E2E_DATA"' EXIT; \
	cd service && BASE_URL=http://localhost:$(E2E_PORT) ./node_modules/.bin/cucumber-js --config e2e/cucumber.mjs

e2e/test-headed: frontend/build backend/build  ## Run E2E tests (visible browser)
	@mkdir -p service/e2e/screenshots service/e2e/reports
	@E2E_DATA=$$(mktemp -d); \
	service/subway-server --port $(E2E_PORT) --boards-dir service/$(BOARDS_DIR) --data-dir "$$E2E_DATA" --static-dir service/static --dev >>"$$E2E_DATA/server.log" 2>&1 & SERVER_PID=$$!; \
	trap 'kill $$SERVER_PID 2>/dev/null; rm -rf "$$E2E_DATA"' EXIT; \
	cd service && BASE_URL=http://localhost:$(E2E_PORT) HEADED=true ./node_modules/.bin/cucumber-js --config e2e/cucumber.mjs

# ─── Site ────────────────────────────────────────────────

.PHONY: site/build site/preview

site/build: frontend/build backend/generate  ## Build static landing page → _site/
	cd service && go run ./cmd/generate-site/ ../_site
	mkdir -p _site/static/dist
	cp -r service/static/dist/* _site/static/dist/

site/preview: site/build               ## Build and open landing page in browser
	open _site/index.html

# ─── Lint ────────────────────────────────────────────────

.PHONY: lint lint/firmware fmt/firmware

CLANG_FMT := uvx clang-format==19.1.7

lint:                                      ## Run all linters (Trunk + firmware build)
	trunk check --no-fix
	$(MAKE) lint/firmware

lint/firmware:                             ## Lint firmware (build + format + cppcheck)
	cd firmware && $(PIO) run
	cd tools/debug-firmware && $(PIO) run
	cd firmware && find src -name '*.cpp' -o -name '*.hpp' | xargs $(CLANG_FMT) --dry-run --Werror
	cd firmware && $(PIO) check --skip-packages --fail-on-defect=high \
		--src-filters="+<src/>" \
		--flags="cppcheck: --suppress=unusedFunction --suppress=*:*components/*"

fmt/firmware:                              ## Format firmware C++ code
	cd firmware && find src -name '*.cpp' -o -name '*.hpp' | xargs $(CLANG_FMT) -i

# ─── Hooks ───────────────────────────────────────────────

.PHONY: hooks

hooks:                                 ## Install git pre-commit hook (uses Trunk for speed)
	@printf '#!/bin/sh\ntrunk check --no-fix\n' > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Installed .git/hooks/pre-commit"

# ─── Dev ────────────────────────────────────────────────

.PHONY: dev

dev: frontend/build backend/generate   ## Start frontend watch + backend with auto-reload
	@trap 'kill 0' EXIT; \
	(cd service && npm run dev) & \
	(cd service && air) & \
	wait

# ─── Shortcuts ───────────────────────────────────────────

.PHONY: all clean help

all: backend/build firmware/build      ## Build backend + firmware

clean:                                 ## Remove all build artifacts
	rm -rf firmware/.pio/build tools/debug-firmware/.pio/build service/subway-server

help:                                  ## Show available commands
	@echo ""
	@grep -E '^[a-z/._-]+:.*##' $(MAKEFILE_LIST) | \
		awk -F ':.*## ' '{printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2}'
	@echo ""
