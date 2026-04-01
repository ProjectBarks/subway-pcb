export type LedPosition = [number, number, string];

export interface PixelFrame {
	timestamp: number;
	sequence: number;
	ledCount: number;
	pixels: Uint8Array | null;
}
