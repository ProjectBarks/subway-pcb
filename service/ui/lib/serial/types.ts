export interface DeviceInfo {
	mac: string;
	fw: string;
	hw: string;
	heap: number;
	ssid: string;
	wifi: boolean;
	ip: string;
	rssi: number;
	plugin: string;
	leds: number;
	reset: number;
	uptime: number;
}

export interface WifiNetwork {
	ssid: string;
	rssi: number;
	auth: number;
}

export interface WifiStatus {
	ssid: string;
	connected: boolean;
	ip: string;
	rssi: number;
}

export interface DiagData {
	heap: number;
	heap_min: number;
	lua_errors: number;
	lua_mem: number;
	render_fps: number;
	nonzero_px: number;
	uptime: number;
}

export interface FlashProgress {
	phase: "connecting" | "erasing" | "writing" | "verifying" | "done" | "error";
	percent: number;
	message: string;
}

export interface SerialResponse<T = unknown> {
	ok: boolean;
	cmd: string;
	seq: number;
	data?: T;
	error?: string;
	code?: string;
	v?: number;
	fw?: string;
}

export interface FirmwareRelease {
	tag: string;
	name: string;
	date: string;
	body: string;
	firmwareUrl: string;
}
