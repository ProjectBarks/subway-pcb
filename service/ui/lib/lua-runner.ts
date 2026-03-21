import { type LuaEngine, LuaFactory, LuaMultiReturn } from "wasmoon";

interface Train {
	route: string;
	status: string;
}

export interface StationState {
	stop_id: string;
	trains?: Train[];
}

export class LuaRunner {
	private engine: LuaEngine | null = null;
	private pixelBuffer: Uint8Array;
	private stationTrains: Map<string, Train[]> = new Map();
	private ledMap: string[] = [];
	private stationLeds: Map<string, number[]> = new Map();
	private config: Record<string, string> = {};
	private stripSizes: number[] = [97, 102, 55, 81, 70, 21, 22, 19, 11];
	private _ledCount = 478;
	private startTime = performance.now();

	onLog: ((msg: string) => void) | null = null;

	constructor() {
		this.pixelBuffer = new Uint8Array(this._ledCount * 3);
	}

	async init(): Promise<void> {
		const factory = new LuaFactory();
		this.engine = await factory.createEngine();
		this.registerBuiltins();
	}

	private buildStationLeds(): void {
		this.stationLeds.clear();
		for (let i = 0; i < this.ledMap.length; i++) {
			const sid = this.ledMap[i];
			if (sid) {
				let arr = this.stationLeds.get(sid);
				if (!arr) {
					arr = [];
					this.stationLeds.set(sid, arr);
				}
				arr.push(i);
			}
		}
	}

	private registerBuiltins(): void {
		if (!this.engine) return;
		const eng = this.engine;

		// LED Control
		eng.global.set(
			"set_led",
			(index: number, r: number, g: number, b: number) => {
				if (index >= 0 && index < this._ledCount) {
					this.pixelBuffer[index * 3] = Math.max(0, Math.min(255, r));
					this.pixelBuffer[index * 3 + 1] = Math.max(0, Math.min(255, g));
					this.pixelBuffer[index * 3 + 2] = Math.max(0, Math.min(255, b));
				}
			},
		);

		eng.global.set("clear_leds", () => {
			this.pixelBuffer.fill(0);
		});

		eng.global.set("led_count", () => this._ledCount);

		// MTA State Queries
		eng.global.set("has_train", (ledIndex: number) => {
			const sid = this.ledMap[ledIndex];
			if (!sid) return false;
			const trains = this.stationTrains.get(sid);
			return trains !== undefined && trains.length > 0;
		});

		eng.global.set("has_status", (ledIndex: number, status: string) => {
			const sid = this.ledMap[ledIndex];
			if (!sid) return false;
			const trains = this.stationTrains.get(sid);
			if (!trains) return false;
			return trains.some((t) => t.status === status);
		});

		eng.global.set("get_route", (ledIndex: number) => {
			const sid = this.ledMap[ledIndex];
			if (!sid) return null;
			const trains = this.stationTrains.get(sid);
			if (!trains || trains.length === 0) return null;
			return trains[0].route;
		});

		eng.global.set("get_routes", (ledIndex: number) => {
			const sid = this.ledMap[ledIndex];
			if (!sid) return [];
			const trains = this.stationTrains.get(sid);
			if (!trains) return [];
			return trains.map((t) => t.route);
		});

		eng.global.set("get_station", (ledIndex: number) => {
			if (ledIndex < 0 || ledIndex >= this.ledMap.length) return null;
			return this.ledMap[ledIndex] || null;
		});

		eng.global.set("get_leds_for_station", (stationId: string) => {
			const leds = this.stationLeds.get(stationId);
			return leds || [];
		});

		// Config Queries
		eng.global.set("get_string_config", (key: string) => {
			return this.config[key] ?? null;
		});

		eng.global.set("get_int_config", (key: string) => {
			const v = this.config[key];
			if (v === undefined) return null;
			const n = parseInt(v, 10);
			return Number.isNaN(n) ? null : n;
		});

		eng.global.set("get_rgb_config", (key: string) => {
			const hex = this.config[key];
			if (!hex || hex.length < 7 || hex[0] !== "#") return null;
			const r = parseInt(hex.slice(1, 3), 16);
			const g = parseInt(hex.slice(3, 5), 16);
			const b = parseInt(hex.slice(5, 7), 16);
			return LuaMultiReturn.of(r, g, b);
		});

		// Timing
		eng.global.set("get_time", () => {
			return (performance.now() - this.startTime) / 1000;
		});

		// Color Utilities
		eng.global.set("hsv_to_rgb", (h: number, s: number, v: number) => {
			const i = Math.floor(h * 6);
			const f = h * 6 - i;
			const p = v * (1 - s);
			const q = v * (1 - f * s);
			const t = v * (1 - (1 - f) * s);
			let r: number;
			let g: number;
			let b: number;
			switch (i % 6) {
				case 0:
					r = v;
					g = t;
					b = p;
					break;
				case 1:
					r = q;
					g = v;
					b = p;
					break;
				case 2:
					r = p;
					g = v;
					b = t;
					break;
				case 3:
					r = p;
					g = q;
					b = v;
					break;
				case 4:
					r = t;
					g = p;
					b = v;
					break;
				default:
					r = v;
					g = p;
					b = q;
					break;
			}
			return LuaMultiReturn.of(
				Math.round(r * 255),
				Math.round(g * 255),
				Math.round(b * 255),
			);
		});

		eng.global.set("hex_to_rgb", (hex: string) => {
			if (!hex || hex.length < 7 || hex[0] !== "#") return null;
			const r = parseInt(hex.slice(1, 3), 16);
			const g = parseInt(hex.slice(3, 5), 16);
			const b = parseInt(hex.slice(5, 7), 16);
			return LuaMultiReturn.of(r, g, b);
		});

		// Board Info
		eng.global.set("get_strip_info", () => {
			return this.stripSizes;
		});

		eng.global.set("led_to_strip", (index: number) => {
			let offset = 0;
			for (let s = 0; s < this.stripSizes.length; s++) {
				if (index < offset + this.stripSizes[s]) {
					return LuaMultiReturn.of(s + 1, index - offset);
				}
				offset += this.stripSizes[s];
			}
			return LuaMultiReturn.of(null, null);
		});

		// Logging
		eng.global.set("log", (msg: string) => {
			if (this.onLog) this.onLog(String(msg));
		});

		// Status Constants
		eng.global.set("STOPPED_AT", "STOPPED_AT");
		eng.global.set("INCOMING_AT", "INCOMING_AT");
		eng.global.set("IN_TRANSIT_TO", "IN_TRANSIT_TO");
	}

	async loadScript(source: string): Promise<string | null> {
		if (!this.engine) return "Engine not initialized";
		try {
			await this.engine.doString(source);
			return null;
		} catch (e: unknown) {
			return e instanceof Error ? e.message : String(e);
		}
	}

	setMtaState(stations: StationState[]): void {
		this.stationTrains.clear();
		for (const station of stations) {
			if (station.trains && station.trains.length > 0) {
				this.stationTrains.set(station.stop_id, station.trains);
			}
		}
	}

	setConfig(config: Record<string, string>): void {
		this.config = config;
	}

	setLedMap(ledMap: string[]): void {
		this.ledMap = ledMap;
		this._ledCount = ledMap.length || 478;
		this.pixelBuffer = new Uint8Array(this._ledCount * 3);
		this.buildStationLeds();
	}

	setStripSizes(sizes: number[]): void {
		this.stripSizes = sizes;
	}

	async render(): Promise<Uint8Array> {
		if (!this.engine) return this.pixelBuffer;
		this.pixelBuffer.fill(0);
		try {
			await this.engine.doString("if render then render() end");
		} catch (e: unknown) {
			if (this.onLog) {
				this.onLog(`Error: ${e instanceof Error ? e.message : String(e)}`);
			}
		}
		return this.pixelBuffer;
	}

	dispose(): void {
		if (this.engine) {
			this.engine.global.close();
			this.engine = null;
		}
	}
}
