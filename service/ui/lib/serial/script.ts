import type { SerialProtocol } from "./serial-protocol";

export async function uploadScript(
	p: SerialProtocol,
	lua: string,
): Promise<{ size: number; saved: boolean }> {
	// Enter transfer mode
	const begin = await p.send<string>("DO SCRIPT_BEGIN");
	if (!begin.ok)
		throw new Error(begin.error || "Failed to begin script transfer");

	// Send raw Lua lines
	const lines = lua.split("\n");
	for (const line of lines) {
		await p.sendRaw(line);
	}

	// End transfer
	const end = await p.send<{ size: number; saved: boolean }>(
		"SCRIPT_END",
		10000,
	);
	if (!end.ok) throw new Error(end.error || "Failed to save script");
	// biome-ignore lint/style/noNonNullAssertion: response data is validated by ok check
	return end.data!;
}

export async function clearScript(p: SerialProtocol): Promise<void> {
	const resp = await p.send("DO SCRIPT_CLEAR");
	if (!resp.ok) throw new Error(resp.error || "Failed to clear script");
}
