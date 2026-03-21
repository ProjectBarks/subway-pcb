import "../lib/types";
import "./board.css";
import {
	type BoardViewerHandle,
	initBoardViewer,
	type LedInfo,
} from "../lib/board-viewer";
import { LuaRunner } from "../lib/lua-runner";
import { BoardSerial, encodeCommand } from "../lib/serial";
import {
	collectConfigToPresetForm,
	collectRouteColorsToForm,
	updatePreviewColor,
} from "./form-helpers";

// Expose form helpers globally for inline event handlers in Go templates
window.updatePreviewColor = updatePreviewColor;
window.collectRouteColorsToForm = collectRouteColorsToForm;
window.collectConfigToPresetForm = collectConfigToPresetForm;

// Initialize WebSerial for board settings
const boardSerial = new BoardSerial();
window.boardSerial = boardSerial;
window.encodeCommand = encodeCommand;

const canvasWrap = document.getElementById("canvas-wrap");
const boardUrl =
	canvasWrap?.dataset.boardUrl ??
	"/static/dist/boards/nyc-subway/v1/board.json";
const tooltip = document.getElementById("tooltip")!;

let handle: BoardViewerHandle | null = null;
let luaRunner: LuaRunner | null = null;
let ledCount = 0;
let lastFetchOk = false;
let mouseX = 0;
let mouseY = 0;

// Default track.lua for board page preview
const DEFAULT_TRACK_LUA = `function render()
    for i = 0, led_count() - 1 do
        if has_status(i, STOPPED_AT) then
            local route = get_route(i)
            if route then
                local r, g, b = get_rgb_config(route)
                if r then
                    set_led(i, r, g, b)
                end
            end
        end
    end
end`;

function updateStatus(trains: number, _seq: number): void {
	const dot = document.getElementById("dot");
	const statusText = document.getElementById("status-text");
	const trainCount = document.getElementById("train-count");
	const frameSeq = document.getElementById("frame-seq");
	if (lastFetchOk) {
		if (dot)
			dot.innerHTML =
				'<span class="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>';
		if (statusText) statusText.textContent = "Connected";
	} else {
		if (dot)
			dot.innerHTML =
				'<span class="relative inline-flex rounded-full h-2 w-2 bg-red-500"></span>';
		if (statusText) statusText.textContent = "Disconnected";
	}
	if (trainCount) trainCount.textContent = String(trains);
	if (frameSeq) frameSeq.textContent = "0";
}

function onLedHover(info: LedInfo | null): void {
	if (info) {
		tooltip.innerHTML = `<span class="sid">${info.stationId || "--"}</span> rgb(${info.r},${info.g},${info.b}) | strip ${info.strip} px ${info.pixel}`;
		tooltip.style.display = "block";
		tooltip.style.left = `${mouseX + 14}px`;
		tooltip.style.top = `${mouseY - 10}px`;
	} else {
		tooltip.style.display = "none";
	}
}

function collectInitialConfig(): Record<string, string> {
	const config: Record<string, string> = {};
	document
		.querySelectorAll<HTMLInputElement>(
			"input[type='color'][name], input[type='number'][name], select[name]",
		)
		.forEach((el) => {
			if (el.name) config[el.name] = el.value;
		});
	return config;
}

async function fetchState(): Promise<void> {
	try {
		const resp = await fetch("/api/v1/state?format=json");
		if (resp.ok) {
			const data = await resp.json();
			if (data.stations && luaRunner) {
				luaRunner.setMtaState(data.stations);
			}
			lastFetchOk = true;
			const activeCount = data.stations?.length ?? 0;
			updateStatus(activeCount, 0);
		}
	} catch {
		lastFetchOk = false;
		updateStatus(0, 0);
	}
}

function startRenderLoop(): void {
	if (!luaRunner || !handle) return;

	const runner = luaRunner;
	const viewer = handle;

	const renderFrame = async () => {
		const pixels = await runner.render();
		viewer.setPixels(pixels);
		requestAnimationFrame(renderFrame);
	};
	renderFrame();
}

async function init(): Promise<void> {
	const viewerContainer = document.getElementById("board-viewer");
	if (!viewerContainer) return;

	// Track mouse for tooltip positioning
	viewerContainer.addEventListener("mousemove", (e) => {
		mouseX = e.clientX;
		mouseY = e.clientY;
	});
	viewerContainer.addEventListener("mouseleave", () => {
		tooltip.style.display = "none";
	});

	// Init LuaRunner
	luaRunner = new LuaRunner();
	await luaRunner.init();

	// Load board data for LED map and viewer
	try {
		const resp = await fetch(boardUrl);
		const board = await resp.json();
		ledCount = board.ledCount ?? 478;
		const ledMap = new Array<string>(ledCount).fill("");
		for (const pos of board.ledPositions) {
			if (pos.index >= 0 && pos.index < ledCount && pos.stationId) {
				ledMap[pos.index] = pos.stationId;
			}
		}
		luaRunner.setLedMap(ledMap);
		if (board.strips) luaRunner.setStripSizes(board.strips);
	} catch {
		ledCount = 478;
	}

	handle = await initBoardViewer(viewerContainer, {
		boardUrl,
		mode: "inspect",
		onLedHover,
	});

	// Load initial config from the page's color inputs
	const initialConfig = collectInitialConfig();
	luaRunner.setConfig(initialConfig);

	// Load the Lua script
	await luaRunner.loadScript(DEFAULT_TRACK_LUA);

	// Expose for theme editing
	window._luaRunner = luaRunner;

	// Fetch state and start rendering
	await fetchState();
	setInterval(fetchState, 5000);
	startRenderLoop();
}

init();
