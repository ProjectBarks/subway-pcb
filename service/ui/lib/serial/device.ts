import type { SerialProtocol } from "./serial-protocol";
import type { DeviceInfo, DiagData } from "./types";

export async function ping(p: SerialProtocol) {
	return p.send<string>("PING");
}

export async function getInfo(p: SerialProtocol): Promise<DeviceInfo> {
	const resp = await p.send<DeviceInfo>("GET INFO");
	if (!resp.ok) throw new Error(resp.error || "Failed to get device info");
	// biome-ignore lint/style/noNonNullAssertion: response data is validated by ok check
	return resp.data!;
}

export async function getDiag(p: SerialProtocol): Promise<DiagData> {
	const resp = await p.send<DiagData>("GET DIAG");
	if (!resp.ok) throw new Error(resp.error || "Failed to get diagnostics");
	// biome-ignore lint/style/noNonNullAssertion: response data is validated by ok check
	return resp.data!;
}

export async function reboot(p: SerialProtocol): Promise<void> {
	const resp = await p.send("DO REBOOT");
	if (!resp.ok) throw new Error(resp.error || "Failed to reboot");
}

export async function ledTest(p: SerialProtocol): Promise<void> {
	const resp = await p.send("DO LED_TEST");
	if (!resp.ok) throw new Error(resp.error || "Failed to trigger LED test");
}

export async function factoryReset(p: SerialProtocol): Promise<void> {
	// Step 1: Request nonce
	const resp1 = await p.send<{ token: string }>("DO FACTORY_RESET");
	if (!resp1.ok)
		throw new Error(resp1.error || "Failed to initiate factory reset");

	// Step 2: Confirm with nonce
	// biome-ignore lint/style/noNonNullAssertion: response data is validated by ok check
	const resp2 = await p.send(`DO FACTORY_RESET ${resp1.data!.token}`);
	if (!resp2.ok)
		throw new Error(resp2.error || "Factory reset confirmation failed");
}
