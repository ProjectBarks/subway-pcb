import "../lib/types";
import "./board.css";
import {
	type BoardViewerHandle,
	initBoardViewer,
	type LedInfo,
} from "../lib/board-viewer";
import { LuaRunner } from "../lib/lua-runner";
import { initPreviews, type PreviewCleanup } from "../lib/preview-controller";
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
let previewCleanup: PreviewCleanup | null = null;

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

function collectConfig(): Record<string, string> {
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

/** Read the active Lua source from the controls panel data attribute */
function getActiveLuaSource(): string {
	const controls = document.getElementById("controls");
	return controls?.dataset.luaSource ?? "";
}

/** Load the current plugin's Lua source and config into the runner */
async function loadActivePlugin(): Promise<void> {
	if (!luaRunner) return;
	const source = getActiveLuaSource();
	if (source) {
		luaRunner.setConfig(collectConfig());
		await luaRunner.loadScript(source);
	}
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

/** Init browse plugin previews using the shared preview-controller */
async function initBrowsePreviews(): Promise<void> {
	previewCleanup?.destroy();
	previewCleanup = null;

	const browseTab = document.getElementById("tab-browse");
	const cards = browseTab?.querySelectorAll("[data-preview-card]");
	if (!cards || cards.length === 0) return;

	previewCleanup = await initPreviews("#tab-browse [data-preview-card]");
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

	// Load the active plugin's Lua source
	await loadActivePlugin();

	// Expose for theme editing
	window._luaRunner = luaRunner;

	// Fetch state and start rendering
	await fetchState();
	setInterval(fetchState, 5000);
	startRenderLoop();

	// Listen for HTMX swaps on the controls panel — reload Lua source when plugin changes
	document.body.addEventListener("htmx:afterSwap", async (evt) => {
		const target = (evt as CustomEvent).detail?.target;
		if (target?.id === "controls" || target?.closest?.("#controls")) {
			await loadActivePlugin();
			initBrowsePreviews();
		}
	});

	// Init browse previews when the browse tab becomes visible
	const browseTab = document.getElementById("tab-browse");
	if (browseTab) {
		const observer = new MutationObserver(() => {
			if (!browseTab.classList.contains("hidden")) {
				initBrowsePreviews();
				observer.disconnect();
			}
		});
		observer.observe(browseTab, {
			attributes: true,
			attributeFilter: ["class"],
		});
	}
}

init();
