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
		_previewRenderer?: import("./lib/preview").PreviewRenderer;
		updatePreviewColor: (input: HTMLInputElement) => void;
		collectRouteColorsToForm: (form: HTMLFormElement) => void;
		collectConfigToPresetForm: (form: HTMLFormElement) => void;
		Board: typeof import("./lib/board").Board;
		boardSerial: import("./lib/serial").BoardSerial;
		encodeCommand: typeof import("./lib/serial").encodeCommand;
	}
}
