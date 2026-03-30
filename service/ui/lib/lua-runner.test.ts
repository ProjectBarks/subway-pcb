import { readdirSync, readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { LuaRunner, type StationState } from "./lua-runner";

const __dirname = dirname(fileURLToPath(import.meta.url));
const CONFORMANCE_DIR = resolve(__dirname, "../../../tests/lua-conformance");

/* Fixtures — must match firmware/test/test_lua_conformance.c setup_fixtures() */
const LED_MAP = ["A01", "A01", "B02", "", "C03"];
const STRIP_SIZES = [3, 2];
const STATIONS: StationState[] = [
	{ stop_id: "A01", trains: [{ route: "1", status: "STOPPED_AT" }] },
	{ stop_id: "B02", trains: [{ route: "A", status: "IN_TRANSIT_TO" }] },
];
const CONFIG: Record<string, string> = {
	brightness: "200",
	color: "#FF8800",
	empty: "",
	name: "test",
};

const testFiles = readdirSync(CONFORMANCE_DIR)
	.filter((f: string) => f.startsWith("test_") && f.endsWith(".lua"))
	.sort();

const helpersSource = readFileSync(
	resolve(CONFORMANCE_DIR, "helpers.lua"),
	"utf-8",
);

describe("Lua API conformance", () => {
	let runner: LuaRunner;

	beforeAll(async () => {
		runner = new LuaRunner();
		await runner.init();
		runner.setLedMap(LED_MAP);
		runner.setStripSizes(STRIP_SIZES);
		runner.setMtaState(STATIONS);
		runner.setConfig(CONFIG);
	});

	afterAll(() => {
		runner.dispose();
	});

	for (const file of testFiles) {
		it(file, async () => {
			const source = readFileSync(resolve(CONFORMANCE_DIR, file), "utf-8");

			/* Reset _results and load helpers + test */
			await runner.loadScript("_results = { pass = 0, fail = 0, errors = {} }");
			await runner.loadScript(helpersSource);
			await runner.loadScript(source);

			/* Read _results from Lua engine */
			const engine = (
				runner as unknown as {
					engine: { global: { get: (k: string) => unknown } };
				}
			).engine;
			const results = engine.global.get("_results") as {
				pass: number;
				fail: number;
				errors: Map<number, string>;
			};

			expect(results).toBeDefined();

			let errorMessages: string[] = [];
			if (results.errors instanceof Map) {
				errorMessages = Array.from(results.errors.values()).map(String);
			}

			expect(results.fail, `Failures:\n${errorMessages.join("\n")}`).toBe(0);
			expect(results.pass).toBeGreaterThan(0);
		});
	}
});
