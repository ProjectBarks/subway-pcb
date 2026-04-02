import "./board.css";
import Alpine from "alpinejs";
import { loadBoardData } from "../lib/board-data";
import {
	type BoardViewerHandle,
	type LedInfo,
	initBoardViewer,
} from "../lib/board-viewer";
import { LuaRunner } from "../lib/lua-runner";
import { pollMtaState } from "../lib/mta-polling";
import { type PreviewCleanup, initPreviews } from "../lib/preview-controller";
import {
	factoryReset,
	getDiag,
	getInfo,
	ledTest,
	ping,
	reboot,
} from "../lib/serial/device";
import { flashFirmware } from "../lib/serial/flash";
import { downloadFirmware, fetchReleases } from "../lib/serial/github-releases";
import { clearScript } from "../lib/serial/script";
import { SerialProtocol } from "../lib/serial/serial-protocol";
import type {
	DeviceInfo,
	DiagData,
	FirmwareRelease,
	FlashProgress,
	WifiNetwork,
	WifiStatus,
} from "../lib/serial/types";
import { configureWifi, getWifi, scanWifi } from "../lib/serial/wifi";
import { collectConfig, collectConfigToPresetForm } from "./form-helpers";

// ── Store type helpers ──────────────────────────────────────────────
interface SerialStore {
	connected: boolean;
	protocol: SerialProtocol | null;
	info: DeviceInfo | null;
	logs: string[];
}

interface ToastStore {
	show(message: string, type: string): void;
}

function serialStore(): SerialStore {
	return Alpine.store("serial") as unknown as SerialStore;
}

function toastStore(): ToastStore {
	return Alpine.store("toast") as unknown as ToastStore;
}

function showError(e: unknown): void {
	toastStore().show(e instanceof Error ? e.message : String(e), "error");
}

// ── Alpine serial store ──────────────────────────────────────────────
Alpine.store("serial", {
	connected: false,
	protocol: null as SerialProtocol | null,
	info: null as DeviceInfo | null,
	logs: [] as string[],
});

// ── Alpine.data: wifiPanel ───────────────────────────────────────────
Alpine.data("wifiPanel", () => ({
	open: true,
	ssid: "",
	password: "",
	scanning: false,
	networks: [] as WifiNetwork[],
	show_password: false,
	wifi: {
		ssid: "",
		connected: false,
		ip: "",
		rssi: 0,
	} as WifiStatus,

	async init() {
		const store = serialStore();
		if (store.protocol) {
			try {
				this.wifi = await getWifi(store.protocol);
			} catch {
				// ignore
			}
		}
	},

	async scan() {
		const store = serialStore();
		if (!store.protocol) return;
		this.scanning = true;
		try {
			this.networks = await scanWifi(store.protocol);
		} catch (e: unknown) {
			showError(e);
		} finally {
			this.scanning = false;
		}
	},

	async save() {
		const store = serialStore();
		if (!store.protocol || !this.ssid) return;
		try {
			await configureWifi(store.protocol, this.ssid, this.password);
			toastStore().show("WiFi credentials saved. Reconnecting...", "success");
		} catch (e: unknown) {
			showError(e);
		}
	},
}));

// ── Alpine.data: firmwarePanel ───────────────────────────────────────
Alpine.data("firmwarePanel", () => ({
	open: true,
	releases: [] as FirmwareRelease[],
	selected: "",
	flashProgress: null as FlashProgress | null,
	loading: false,
	updateAvailable: false,
	protocolMismatch: false,

	async init() {
		this.loading = true;
		try {
			this.releases = await fetchReleases();
			const store = serialStore();
			if (store.info && this.releases.length > 0) {
				this.updateAvailable = this.releases[0].tag !== `v${store.info.fw}`;
			}
		} catch {
			// ignore
		} finally {
			this.loading = false;
		}
	},

	phaseIndex(phase: string) {
		return ["connecting", "erasing", "writing", "verifying", "done"].indexOf(
			phase,
		);
	},

	async flash() {
		const store = serialStore();
		if (!store.protocol || !this.selected) return;

		const release = this.releases.find(
			(r: FirmwareRelease) => r.tag === this.selected,
		);
		if (!release) return;

		try {
			this.flashProgress = {
				phase: "connecting",
				percent: 0,
				message: "Downloading firmware...",
			};
			const bin = await downloadFirmware(release);

			const port = store.protocol.port;
			if (!port) throw new Error("No serial port available");
			await store.protocol.disconnect();
			store.connected = false;

			await flashFirmware(port, bin, (p: FlashProgress) => {
				this.flashProgress = p;
			});

			// Reconnect after flash
			await new Promise((r) => setTimeout(r, 3000));
			const protocol = new SerialProtocol();
			await protocol.connect();

			store.protocol = protocol;
			store.connected = true;

			await ping(protocol);
			store.info = await getInfo(protocol);

			this.flashProgress = {
				phase: "done",
				percent: 100,
				message: "Flash complete!",
			};
			toastStore().show(`Firmware updated to ${store.info?.fw}`, "success");
		} catch (e: unknown) {
			const msg = e instanceof Error ? e.message : String(e);
			this.flashProgress = {
				phase: "error",
				percent: 0,
				message: msg,
			};
			toastStore().show(`Flash failed: ${msg}`, "error");
		}
	},
}));

// ── Alpine.data: advancedPanel ──────────────────────────────────────
Alpine.data("advancedPanel", () => ({
	open: false,
	data: null as DiagData | null,

	async refresh() {
		const store = serialStore();
		if (!store.protocol) return;
		try {
			this.data = await getDiag(store.protocol);
		} catch (e: unknown) {
			showError(e);
		}
	},

	clearLogs() {
		serialStore().logs = [];
	},

	formatBytes(bytes: number | undefined) {
		if (bytes == null) return "\u2014";
		if (bytes < 1024) return `${bytes} B`;
		return `${(bytes / 1024).toFixed(1)} KB`;
	},

	async testLeds() {
		const store = serialStore();
		if (!store.protocol) return;
		try {
			await ledTest(store.protocol);
			toastStore().show("LED test triggered", "success");
		} catch (e: unknown) {
			showError(e);
		}
	},

	async reboot() {
		if (!confirm("Reboot the device?")) return;
		const store = serialStore();
		if (!store.protocol) return;
		try {
			await reboot(store.protocol);
			toastStore().show("Rebooting device...", "success");
		} catch (e: unknown) {
			showError(e);
		}
	},

	async factoryReset() {
		if (!confirm("This will erase all settings. Are you sure?")) return;
		const store = serialStore();
		if (!store.protocol) return;
		try {
			await factoryReset(store.protocol);
			toastStore().show(
				"Factory reset complete. Device rebooting...",
				"success",
			);
		} catch (e: unknown) {
			showError(e);
		}
	},
}));

// ── Alpine.data: scriptPanel ─────────────────────────────────────────
Alpine.data("scriptPanel", () => ({
	open: true,
	get info() {
		return serialStore().info;
	},

	async clearScript() {
		const store = serialStore();
		if (!store.protocol) return;
		try {
			await clearScript(store.protocol);
			store.info = await getInfo(store.protocol);
			toastStore().show("Local script cleared", "success");
		} catch (e: unknown) {
			showError(e);
		}
	},
}));

// ── Serial connect handler ───────────────────────────────────────────
async function handleSerialConnect() {
	const store = serialStore();
	const protocol = new SerialProtocol();

	protocol.addEventListener("log", ((e: CustomEvent) => {
		store.logs.push(e.detail);
		if (store.logs.length > 200) store.logs.shift();
	}) as EventListener);

	protocol.addEventListener("disconnect", () => {
		store.connected = false;
		store.protocol = null;
		store.info = null;
	});

	try {
		await protocol.connect();

		// Board may reset on port open (DTR/RTS auto-reset circuit).
		// Retry PING with backoff to wait for boot to complete.
		let pingOk = false;
		for (let attempt = 0; attempt < 8; attempt++) {
			try {
				await ping(protocol);
				pingOk = true;
				break;
			} catch {
				// Wait with increasing delay: 1s, 1.5s, 2s, 2s, 2s, ...
				await new Promise((r) =>
					setTimeout(r, Math.min(1000 + attempt * 500, 2000)),
				);
			}
		}
		if (!pingOk) throw new Error("Board did not respond after boot");

		const info = await getInfo(protocol);

		store.protocol = protocol;
		store.connected = true;
		store.info = info;
	} catch (e: unknown) {
		toastStore().show(
			`Connection failed: ${e instanceof Error ? e.message : String(e)}`,
			"error",
		);
	}
}

// ── Existing event delegation ────────────────────────────────────────

// Event delegation for preset form submission
document.addEventListener("submit", (e) => {
	const form = (e.target as HTMLElement).closest(
		"[data-preset-form]",
	) as HTMLFormElement | null;
	if (form) collectConfigToPresetForm(form);
});

// Event delegation for USB serial connect (updated to use SerialProtocol)
document.addEventListener("click", (e) => {
	if ((e.target as HTMLElement).closest("[data-serial-connect]")) {
		handleSerialConnect();
	}
});

// ── Existing board viewer, Lua runner, MTA polling ───────────────────

const canvasWrap = document.getElementById("canvas-wrap");
const boardUrl =
	canvasWrap?.dataset.boardUrl ??
	"/static/dist/boards/nyc-subway/v1/board.json";
const tooltip =
	document.getElementById("tooltip") ?? document.createElement("div");

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

// Update Lua runner config when any color input changes
document.addEventListener("input", (e) => {
	const el = e.target;
	if (
		el instanceof HTMLInputElement &&
		el.type === "color" &&
		el.name &&
		luaRunner
	) {
		luaRunner.setConfig(collectConfig());
	}
});

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
