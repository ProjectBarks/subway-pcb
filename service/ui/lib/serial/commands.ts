// Serial command constants — unified GET/SET/DO grammar
// Keep in sync with proto/serial-commands.json

export const CMD_PING = "PING";
export const CMD_GET_INFO = "GET INFO";
export const CMD_GET_WIFI = "GET WIFI";
export const CMD_GET_DIAG = "GET DIAG";
export const CMD_SET_WIFI_SSID = "SET WIFI_SSID";
export const CMD_SET_WIFI_PASS = "SET WIFI_PASS";
export const CMD_SET_SERVER_URL = "SET SERVER_URL";
export const CMD_DO_WIFI_SCAN = "DO WIFI_SCAN";
export const CMD_DO_WIFI_APPLY = "DO WIFI_APPLY";
export const CMD_DO_LED_TEST = "DO LED_TEST";
export const CMD_DO_REBOOT = "DO REBOOT";
export const CMD_DO_FACTORY_RESET = "DO FACTORY_RESET";
export const CMD_DO_SCRIPT_BEGIN = "DO SCRIPT_BEGIN";
export const CMD_DO_SCRIPT_CLEAR = "DO SCRIPT_CLEAR";
export const CMD_SCRIPT_END = "SCRIPT_END";

// All commands as an array for parity testing
export const ALL_COMMANDS = [
	CMD_PING,
	CMD_GET_INFO,
	CMD_GET_WIFI,
	CMD_GET_DIAG,
	CMD_SET_WIFI_SSID,
	CMD_SET_WIFI_PASS,
	CMD_SET_SERVER_URL,
	CMD_DO_WIFI_SCAN,
	CMD_DO_WIFI_APPLY,
	CMD_DO_LED_TEST,
	CMD_DO_REBOOT,
	CMD_DO_FACTORY_RESET,
	CMD_DO_SCRIPT_BEGIN,
	CMD_DO_SCRIPT_CLEAR,
	CMD_SCRIPT_END,
] as const;

export const MIN_PROTOCOL_VERSION = 1;
export const MAX_PROTOCOL_VERSION = 1;
