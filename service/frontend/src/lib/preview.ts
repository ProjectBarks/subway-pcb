import type { LedPosition } from "../types";
import type { Board } from "./board";
import { STRIP_SIZES, TOTAL_LEDS } from "./board";
import { hexToRgb } from "./color";

interface StationTrain {
	route: string;
}

interface StationState {
	stop_id: string;
	trains?: StationTrain[];
}

interface StateData {
	stations?: StationState[];
}

export class PreviewRenderer {
	board: Board;
	stateData: StateData | null = null;
	ledMap: LedPosition[] | null = null;
	ledMapFlat: string[] | null = null;
	routeColors: Record<string, string> | null = null;
	private _refreshInterval: ReturnType<typeof setInterval> | null = null;

	constructor(board: Board) {
		this.board = board;
	}

	async init(): Promise<void> {
		// Load LED map (station assignments)
		try {
			const resp = await fetch("/static/leds.json");
			this.ledMap = await resp.json();
		} catch (e) {
			console.warn("preview: failed to load leds.json", e);
		}

		// Load flat LED map for station-to-LED mapping
		try {
			const resp = await fetch("/static/led_map.json");
			const raw: Record<string, string> = await resp.json();
			this.ledMapFlat = new Array<string>(TOTAL_LEDS).fill("");
			let offset = 0;
			for (let strip = 0; strip < 9; strip++) {
				for (let pixel = 0; pixel < STRIP_SIZES[strip]; pixel++) {
					const key = `${strip},${pixel}`;
					if (raw[key]) {
						this.ledMapFlat[offset] = raw[key];
					}
					offset++;
				}
			}
		} catch (e) {
			console.warn("preview: failed to load led_map.json", e);
		}

		// Fetch initial state
		await this.refreshState();

		// Auto-refresh state every 5 seconds
		this._refreshInterval = setInterval(() => this.refreshState(), 5000);
	}

	async refreshState(): Promise<void> {
		try {
			const resp = await fetch("/api/v1/state?format=json");
			if (resp.ok) {
				this.stateData = await resp.json();
			}
		} catch (e) {
			console.warn("preview: failed to fetch state", e);
		}
	}

	setThemeColors(colors: Record<string, string>): void {
		this.routeColors = colors;
		this.render();
	}

	render(): void {
		if (!this.stateData || !this.ledMapFlat || !this.routeColors) return;

		const pixels = new Uint8Array(TOTAL_LEDS * 3);

		// Build station -> best route map from state
		const stationRoutes: Record<string, string> = {};
		if (this.stateData.stations) {
			for (const station of this.stateData.stations) {
				if (station.trains && station.trains.length > 0) {
					const routeEnum = station.trains[0].route;
					stationRoutes[station.stop_id] = routeEnum;
				}
			}
		}

		// Map station routes to LED pixels using custom colors
		for (let i = 0; i < TOTAL_LEDS; i++) {
			const sid = this.ledMapFlat[i];
			if (!sid) continue;

			const routeEnum = stationRoutes[sid];
			if (!routeEnum) continue;

			const hexColor = this.routeColors[routeEnum];
			if (!hexColor) continue;

			const rgb = hexToRgb(hexColor);
			if (rgb) {
				pixels[i * 3] = rgb.r;
				pixels[i * 3 + 1] = rgb.g;
				pixels[i * 3 + 2] = rgb.b;
			}
		}

		this.board.setPixels(pixels);
	}

	destroy(): void {
		if (this._refreshInterval) {
			clearInterval(this._refreshInterval);
			this._refreshInterval = null;
		}
	}
}
