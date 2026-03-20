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

interface BoardJson {
	ledCount: number;
	strips: number[];
	ledPositions: Array<{
		index: number;
		stationId: string;
	}>;
}

export class PreviewRenderer {
	private setPixelsFn: (data: Uint8Array) => void;
	stateData: StateData | null = null;
	ledMapFlat: string[] | null = null;
	routeColors: Record<string, string> | null = null;
	private totalLEDs = 0;
	private _refreshInterval: ReturnType<typeof setInterval> | null = null;

	constructor(handle: { setPixels(data: Uint8Array): void }) {
		this.setPixelsFn = (data) => handle.setPixels(data);
	}

	async init(): Promise<void> {
		// Load LED map from board.json, deriving flat station IDs from ledPositions
		try {
			const resp = await fetch("/static/dist/boards/nyc-subway/v1/board.json");
			const board: BoardJson = await resp.json();
			this.totalLEDs = board.ledCount;
			this.ledMapFlat = new Array<string>(this.totalLEDs).fill("");
			for (const pos of board.ledPositions) {
				if (pos.index >= 0 && pos.index < this.totalLEDs && pos.stationId) {
					this.ledMapFlat[pos.index] = pos.stationId;
				}
			}
		} catch (e) {
			console.warn("preview: failed to load board.json", e);
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

		const pixels = new Uint8Array(this.totalLEDs * 3);

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
		for (let i = 0; i < this.totalLEDs; i++) {
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

		this.setPixelsFn(pixels);
	}

	destroy(): void {
		if (this._refreshInterval) {
			clearInterval(this._refreshInterval);
			this._refreshInterval = null;
		}
	}
}
