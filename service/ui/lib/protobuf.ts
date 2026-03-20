import type { PixelFrame } from "./types";

export function decodePixelFrame(buf: ArrayBuffer): PixelFrame {
	const bytes = new Uint8Array(buf);
	let pos = 0;
	const result: PixelFrame = {
		timestamp: 0,
		sequence: 0,
		ledCount: 0,
		pixels: null,
	};

	function readVarint(): number {
		let val = 0,
			shift = 0;
		while (pos < bytes.length) {
			const b = bytes[pos++];
			val |= (b & 0x7f) << shift;
			shift += 7;
			if ((b & 0x80) === 0) break;
		}
		return val >>> 0;
	}

	while (pos < bytes.length) {
		const tag = readVarint();
		const fieldNum = tag >>> 3,
			wireType = tag & 0x7;
		if (wireType === 0) {
			const val = readVarint();
			if (fieldNum === 1) result.timestamp = val;
			else if (fieldNum === 2) result.sequence = val;
			else if (fieldNum === 3) result.ledCount = val;
		} else if (wireType === 2) {
			const len = readVarint();
			if (fieldNum === 4) result.pixels = bytes.slice(pos, pos + len);
			pos += len;
		} else if (wireType === 5) {
			pos += 4;
		} else if (wireType === 1) {
			pos += 8;
		}
	}
	return result;
}
