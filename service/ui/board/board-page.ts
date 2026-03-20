import "../lib/types";
import "./board.css";
import { Board, TOTAL_LEDS } from "../lib/board";
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

const board = new Board("c", "tooltip");
board.brightnessBoost = 1.0;
board.boardSvgOpacity = 0.07;

let lastFetchOk = false;

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

async function fetchPixels(): Promise<void> {
	try {
		const resp = await fetch(
			`/api/v1/pixels?device=${encodeURIComponent(mac)}`,
		);
		if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
		const buf = await resp.arrayBuffer();
		if (buf.byteLength === 0) {
			board.clearPixels();
			lastFetchOk = true;
			updateStatus(0, 0);
			return;
		}
		const frame = decodePixelFrame(buf);
		if (frame.pixels && frame.pixels.length >= TOTAL_LEDS * 3) {
			board.setPixels(frame.pixels);
		}
		let activeCount = 0;
		for (let i = 0; i < TOTAL_LEDS; i++) {
			if (
				board.pixelColors[i * 3] ||
				board.pixelColors[i * 3 + 1] ||
				board.pixelColors[i * 3 + 2]
			)
				activeCount++;
		}
		lastFetchOk = true;
		updateStatus(activeCount, frame.sequence);
	} catch {
		lastFetchOk = false;
		updateStatus(0, 0);
	}
}

async function init(): Promise<void> {
	await board.init("/static/leds.json", "/static/dist/board.svg");
	board.startDrawLoop();

	// Initialize preview renderer for theme editing
	window._previewRenderer = new PreviewRenderer(board);
	await window._previewRenderer.init();

	await fetchPixels();
	setInterval(fetchPixels, 1000);
}

init();
