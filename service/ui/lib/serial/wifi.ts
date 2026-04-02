import type { SerialProtocol } from "./serial-protocol";
import type { WifiNetwork, WifiStatus } from "./types";

export async function getWifi(p: SerialProtocol): Promise<WifiStatus> {
	const resp = await p.send<WifiStatus>("GET WIFI");
	if (!resp.ok) throw new Error(resp.error || "Failed to get WiFi status");
	// biome-ignore lint/style/noNonNullAssertion: response data is validated by ok check
	return resp.data!;
}

export async function scanWifi(p: SerialProtocol): Promise<WifiNetwork[]> {
	const resp = await p.send<WifiNetwork[]>("DO WIFI_SCAN", 15000); // scan takes time
	if (!resp.ok) throw new Error(resp.error || "WiFi scan failed");
	// biome-ignore lint/style/noNonNullAssertion: response data is validated by ok check
	return resp.data!;
}

export async function configureWifi(
	p: SerialProtocol,
	ssid: string,
	pass: string,
): Promise<void> {
	const r1 = await p.send(`SET WIFI_SSID ${ssid}`);
	if (!r1.ok) throw new Error(r1.error || "Failed to set SSID");

	const r2 = await p.send(`SET WIFI_PASS ${pass}`);
	if (!r2.ok) throw new Error(r2.error || "Failed to set password");

	const r3 = await p.send("DO WIFI_APPLY");
	if (!r3.ok) throw new Error(r3.error || "Failed to apply WiFi config");
}
