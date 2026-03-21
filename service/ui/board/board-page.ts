import "../lib/types";
import "./board.css";
import { loadBoardData } from "../lib/board-data";
import {
	type BoardViewerHandle,
	initBoardViewer,
	type LedInfo,
} from "../lib/board-viewer";
import { LuaRunner } from "../lib/lua-runner";
import { pollMtaState } from "../lib/mta-polling";
import { initPreviews, type PreviewCleanup } from "../lib/preview-controller";
import { BoardSerial, encodeCommand } from "../lib/serial";
import {
	collectConfig,
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

function onMtaSuccess(stations: { stop_id: string }[]): void {
	lastFetchOk = true;
	updateStatus(stations.length, 0);
}

function onMtaError(): void {
	lastFetchOk = false;
	updateStatus(0, 0);
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
		await loadBoardData(luaRunner, boardUrl);
	} catch {
		// Board data load failed; viewer will use default LED count
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
	pollMtaState(luaRunner, 5000, {
		onSuccess: onMtaSuccess,
		onError: onMtaError,
	});
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
