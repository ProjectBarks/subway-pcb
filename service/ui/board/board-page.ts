import "../lib/types";
import "./board.css";
import {
	type BoardViewerHandle,
	initBoardViewer,
	type LedInfo,
} from "../lib/board-viewer";
import { PreviewRenderer } from "../lib/preview";
import { decodePixelFrame } from "../lib/protobuf";
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
const mac = canvasWrap?.dataset.deviceMac ?? "";
const boardUrl =
	canvasWrap?.dataset.boardUrl ??
	"/static/dist/boards/nyc-subway/v1/board.json";
const tooltip = document.getElementById("tooltip")!;

let handle: BoardViewerHandle | null = null;
let ledCount = 0;
let lastFetchOk = false;
let mouseX = 0;
let mouseY = 0;

function updateStatus(trains: number, seq: number): void {
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
	if (frameSeq) frameSeq.textContent = String(seq);
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

async function fetchPixels(): Promise<void> {
	try {
		const resp = await fetch(
			`/api/v1/pixels?device=${encodeURIComponent(mac)}`,
		);
		if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
		const buf = await resp.arrayBuffer();
		if (buf.byteLength === 0) {
			handle?.setPixels(new Uint8Array(ledCount * 3));
			lastFetchOk = true;
			updateStatus(0, 0);
			return;
		}
		const frame = decodePixelFrame(buf);
		if (frame.pixels && frame.pixels.length >= ledCount * 3) {
			handle?.setPixels(frame.pixels);
		}
		let activeCount = 0;
		if (frame.pixels) {
			for (let i = 0; i < ledCount; i++) {
				if (
					frame.pixels[i * 3] ||
					frame.pixels[i * 3 + 1] ||
					frame.pixels[i * 3 + 2]
				)
					activeCount++;
			}
		}
		lastFetchOk = true;
		updateStatus(activeCount, frame.sequence);
	} catch {
		lastFetchOk = false;
		updateStatus(0, 0);
	}
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

	// Fetch board manifest to get ledCount
	try {
		const resp = await fetch(boardUrl);
		const board = await resp.json();
		ledCount = board.ledCount ?? 478;
	} catch {
		ledCount = 478;
	}

	handle = await initBoardViewer(viewerContainer, {
		boardUrl,
		mode: "inspect",
		onLedHover,
	});

	// Initialize preview renderer for theme editing
	window._previewRenderer = new PreviewRenderer(handle);
	await window._previewRenderer.init();

	await fetchPixels();
	setInterval(fetchPixels, 1000);
}

init();
