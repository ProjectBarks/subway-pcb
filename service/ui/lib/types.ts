export type LedPosition = [number, number, string];

export interface PixelFrame {
	timestamp: number;
	sequence: number;
	ledCount: number;
	pixels: Uint8Array | null;
}

export interface RGB {
	r: number;
	g: number;
	b: number;
}

declare global {
	interface Window {
		_luaRunner?: import("./lua-runner").LuaRunner;
		updatePreviewColor: (input: HTMLInputElement) => void;
		collectRouteColorsToForm: (form: HTMLFormElement) => void;
		collectConfigToPresetForm: (form: HTMLFormElement) => void;
		boardSerial: import("./serial").BoardSerial;
		encodeCommand: typeof import("./serial").encodeCommand;
	}
}
