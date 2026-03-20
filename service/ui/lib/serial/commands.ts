// Serial command format: "CMD:KEY=VALUE\n"
// Commands:
//   SET:WIFI_SSID=MyNetwork
//   SET:WIFI_PASS=password123
//   SET:BRIGHTNESS=128
//   SET:PLUGIN=track
//   GET:VERSION
//   GET:STATUS
//   REBOOT

export function encodeCommand(
	cmd: string,
	key?: string,
	value?: string,
): string {
	if (key && value) return `${cmd}:${key}=${value}\n`;
	if (key) return `${cmd}:${key}\n`;
	return `${cmd}\n`;
}
