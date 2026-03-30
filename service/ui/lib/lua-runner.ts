import { type LuaEngine, LuaFactory, LuaMultiReturn } from "wasmoon";

interface Train {
	route: string;
	status: string;
}

export interface StationState {
	stop_id: string;
	trains?: Train[];
}

/* Runtime type guards — mirrors C-side luaL_checkinteger / luaL_checkstring.
 * Usage: eng.global.set("fn", typed([int, str], (i, s) => { ... })); */
type Check<T> = (v: unknown, arg: number) => T;

/* luaL_checkinteger: integers, integer-valued floats, numeric strings.
 * Errors on non-integer floats ("number has no integer representation"). */
const int: Check<number> = (v, a) => {
	if (typeof v === "string") {
		const n = Number(v);
		if (!Number.isNaN(n) && Number.isInteger(n)) return n;
	}
	if (typeof v === "number") {
		if (Number.isInteger(v)) return v;
		throw new Error(
			`bad argument #${a} (number has no integer representation)`,
		);
	}
	throw new Error(`bad argument #${a} (number expected, got ${typeof v})`);
};

/* luaL_checknumber: all numbers, numeric strings. */
const num: Check<number> = (v, a) => {
	if (typeof v === "string") {
		const n = Number(v);
		if (!Number.isNaN(n)) return n;
	}
	if (typeof v === "number") return v;
	throw new Error(`bad argument #${a} (number expected, got ${typeof v})`);
};

/* luaL_checkstring: strings, numbers coerced to string. */
const str: Check<string> = (v, a) => {
	if (typeof v === "number") return String(v);
	if (typeof v !== "string")
		throw new Error(`bad argument #${a} (string expected, got ${typeof v})`);
	return v;
};

// biome-ignore lint/suspicious/noExplicitAny: generic type inference requires any
type Infer<C extends Check<any>[]> = {
	[K in keyof C]: C[K] extends Check<infer T> ? T : never;
};

// biome-ignore lint/suspicious/noExplicitAny: generic type inference requires any
function typed<C extends Check<any>[]>(
	checks: [...C],
	fn: (...args: Infer<C>) => unknown,
): (...args: unknown[]) => unknown {
	return (...args: unknown[]) => {
		const checked = checks.map((c, i) => c(args[i], i + 1));
		return (fn as (...a: unknown[]) => unknown)(...checked);
	};
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
			typed([int, int, int, int], (index, r, g, b) => {
				if (index >= 0 && index < this._ledCount) {
					this.pixelBuffer[index * 3] = Math.max(0, Math.min(255, r));
					this.pixelBuffer[index * 3 + 1] = Math.max(0, Math.min(255, g));
					this.pixelBuffer[index * 3 + 2] = Math.max(0, Math.min(255, b));
				}
			}),
		);

		eng.global.set("clear_leds", () => {
			this.pixelBuffer.fill(0);
		});

		eng.global.set("led_count", () => this._ledCount);

		// MTA State Queries
		eng.global.set(
			"has_train",
			typed([int], (ledIndex) => {
				const sid = this.ledMap[ledIndex];
				if (!sid) return false;
				const trains = this.stationTrains.get(sid);
				return trains !== undefined && trains.length > 0;
			}),
		);

		eng.global.set(
			"has_status",
			typed([int, str], (ledIndex, status) => {
				const sid = this.ledMap[ledIndex];
				if (!sid) return false;
				const trains = this.stationTrains.get(sid);
				if (!trains) return false;
				return trains.some((t) => t.status === status);
			}),
		);

		eng.global.set(
			"get_route",
			typed([int], (ledIndex) => {
				const sid = this.ledMap[ledIndex];
				if (!sid) return undefined;
				const trains = this.stationTrains.get(sid);
				if (!trains || trains.length === 0) return undefined;
				return trains[0].route;
			}),
		);

		eng.global.set(
			"get_routes",
			typed([int], (ledIndex) => {
				const sid = this.ledMap[ledIndex];
				if (!sid) return [];
				const trains = this.stationTrains.get(sid);
				if (!trains) return [];
				return trains.map((t) => t.route);
			}),
		);

		eng.global.set(
			"get_station",
			typed([int], (ledIndex) => {
				if (ledIndex < 0 || ledIndex >= this.ledMap.length) return undefined;
				return this.ledMap[ledIndex] || undefined;
			}),
		);

		eng.global.set(
			"get_leds_for_station",
			typed([str], (stationId) => {
				const leds = this.stationLeds.get(stationId);
				return leds || [];
			}),
		);

		// Config Queries
		eng.global.set(
			"get_string_config",
			typed([str], (key) => {
				return this.config[key] ?? undefined;
			}),
		);

		eng.global.set(
			"get_int_config",
			typed([str], (key) => {
				const v = this.config[key];
				if (v === undefined) return undefined;
				const n = parseInt(v, 10);
				return Number.isNaN(n) ? 0 : n;
			}),
		);

		eng.global.set(
			"get_rgb_config",
			typed([str], (key) => {
				const hex = this.config[key];
				if (!hex || hex.length < 7 || hex[0] !== "#") return undefined;
				const r = parseInt(hex.slice(1, 3), 16);
				const g = parseInt(hex.slice(3, 5), 16);
				const b = parseInt(hex.slice(5, 7), 16);
				return LuaMultiReturn.of(r, g, b);
			}),
		);

		// Timing
		eng.global.set("get_time", () => {
			return (performance.now() - this.startTime) / 1000;
		});

		// Color Utilities
		eng.global.set(
			"hsv_to_rgb",
			typed([num, num, num], (h, s, v) => {
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
			}),
		);

		eng.global.set(
			"hex_to_rgb",
			typed([str], (hex) => {
				if (hex.length < 7 || hex[0] !== "#") return undefined;
				const r = parseInt(hex.slice(1, 3), 16);
				const g = parseInt(hex.slice(3, 5), 16);
				const b = parseInt(hex.slice(5, 7), 16);
				return LuaMultiReturn.of(r, g, b);
			}),
		);

		// Board Info
		eng.global.set("get_strip_info", () => {
			return this.stripSizes;
		});

		eng.global.set(
			"led_to_strip",
			typed([int], (index) => {
				let offset = 0;
				for (let s = 0; s < this.stripSizes.length; s++) {
					if (index < offset + this.stripSizes[s]) {
						return LuaMultiReturn.of(s + 1, index - offset);
					}
					offset += this.stripSizes[s];
				}
				return LuaMultiReturn.of(undefined, undefined);
			}),
		);

		// Logging
		eng.global.set(
			"log",
			typed([str], (msg) => {
				if (this.onLog) this.onLog(msg);
			}),
		);

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

	getLedMap(): string[] {
		return this.ledMap;
	}

	getStripSizes(): number[] {
		return this.stripSizes;
	}

	getMtaState(): StationState[] {
		const stations: StationState[] = [];
		for (const [stopId, trains] of this.stationTrains) {
			stations.push({ stop_id: stopId, trains });
		}
		return stations;
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
