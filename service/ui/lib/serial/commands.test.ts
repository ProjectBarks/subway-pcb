import { describe, expect, it } from "vitest";
import spec from "../../../../proto/serial-commands.json";
import { ALL_COMMANDS } from "./commands";

describe("serial commands parity", () => {
	const specCommands = spec.commands.map((c: { cmd: string }) => c.cmd);

	it("every spec command has a matching TS constant", () => {
		for (const cmd of specCommands) {
			expect(ALL_COMMANDS).toContain(cmd);
		}
	});

	it("every TS constant exists in the spec", () => {
		for (const cmd of ALL_COMMANDS) {
			expect(specCommands).toContain(cmd);
		}
	});

	it("command counts match", () => {
		expect(ALL_COMMANDS.length).toBe(specCommands.length);
	});

	it("protocol version matches", () => {
		expect(spec.version).toBe(1);
	});
});
