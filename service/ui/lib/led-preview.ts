import "./led-preview.css";

export interface BoardData {
	ledPositions: Array<{
		x: number;
		y: number;
		index: number;
		stationId: string;
	}>;
	ledCount: number;
	strips: number[];
	ledMap: string[]; // index → stationId
	bounds: { minX: number; maxX: number; minY: number; maxY: number };
}

const boardCache = new Map<string, BoardData>();

export async function loadBoardData(boardUrl: string): Promise<BoardData> {
	const cached = boardCache.get(boardUrl);
	if (cached) return cached;

	const resp = await fetch(boardUrl);
	const json = await resp.json();

	const positions: BoardData["ledPositions"] = json.ledPositions ?? [];
	const ledCount: number = json.ledCount ?? 478;
	const strips: number[] = json.strips ?? [];

	// Build ledMap (index → stationId)
	const ledMap = new Array<string>(ledCount).fill("");
	for (const pos of positions) {
		if (pos.index >= 0 && pos.index < ledCount && pos.stationId) {
			ledMap[pos.index] = pos.stationId;
		}
	}

	// Compute coordinate bounds from actual positions
	let minX = Infinity;
	let maxX = -Infinity;
	let minY = Infinity;
	let maxY = -Infinity;
	for (const pos of positions) {
		if (pos.x < minX) minX = pos.x;
		if (pos.x > maxX) maxX = pos.x;
		if (pos.y < minY) minY = pos.y;
		if (pos.y > maxY) maxY = pos.y;
	}

	const board: BoardData = {
		ledPositions: positions,
		ledCount,
		strips,
		ledMap,
		bounds: { minX, maxX, minY, maxY },
	};

	boardCache.set(boardUrl, board);
	return board;
}

export function setupCanvas(canvas: HTMLCanvasElement): {
	ctx: CanvasRenderingContext2D;
	w: number;
	h: number;
} {
	const dpr = window.devicePixelRatio || 1;
	const rect = canvas.getBoundingClientRect();
	canvas.width = rect.width * dpr;
	canvas.height = rect.height * dpr;
	const ctx = canvas.getContext("2d")!;
	ctx.scale(dpr, dpr);
	return { ctx, w: rect.width, h: rect.height };
}

export function drawLeds(
	ctx: CanvasRenderingContext2D,
	pixels: Uint8Array,
	board: BoardData,
	w: number,
	h: number,
): void {
	ctx.clearRect(0, 0, w, h);

	const { minX, maxX, minY, maxY } = board.bounds;
	const rangeX = maxX - minX;
	const rangeY = maxY - minY;
	if (rangeX === 0 || rangeY === 0) return;

	// Uniform scale with 10% padding
	const pad = 0.1;
	const scaleX = (w * (1 - 2 * pad)) / rangeX;
	const scaleY = (h * (1 - 2 * pad)) / rangeY;
	const scale = Math.min(scaleX, scaleY);

	const offsetX = (w - rangeX * scale) / 2;
	const offsetY = (h - rangeY * scale) / 2;

	const radius = Math.max(1.2, Math.min(2.5, scale * 1.5));

	// Draw off LEDs first (batch)
	ctx.fillStyle = "rgba(255, 255, 255, 0.06)";
	ctx.beginPath();
	for (const pos of board.ledPositions) {
		const i3 = pos.index * 3;
		if (i3 + 2 >= pixels.length) continue;
		const r = pixels[i3];
		const g = pixels[i3 + 1];
		const b = pixels[i3 + 2];
		if (r !== 0 || g !== 0 || b !== 0) continue;

		const px = (pos.x - minX) * scale + offsetX;
		const py = (maxY - pos.y) * scale + offsetY; // flip Y
		ctx.moveTo(px + radius, py);
		ctx.arc(px, py, radius, 0, Math.PI * 2);
	}
	ctx.fill();

	// Draw lit LEDs grouped by color
	const colorGroups = new Map<string, Array<{ px: number; py: number }>>();
	for (const pos of board.ledPositions) {
		const i3 = pos.index * 3;
		if (i3 + 2 >= pixels.length) continue;
		const r = pixels[i3];
		const g = pixels[i3 + 1];
		const b = pixels[i3 + 2];
		if (r === 0 && g === 0 && b === 0) continue;

		const px = (pos.x - minX) * scale + offsetX;
		const py = (maxY - pos.y) * scale + offsetY;
		const key = `${r},${g},${b}`;

		let group = colorGroups.get(key);
		if (!group) {
			group = [];
			colorGroups.set(key, group);
		}
		group.push({ px, py });
	}

	for (const [color, points] of colorGroups) {
		ctx.fillStyle = `rgb(${color})`;
		ctx.beginPath();
		for (const { px, py } of points) {
			ctx.moveTo(px + radius, py);
			ctx.arc(px, py, radius, 0, Math.PI * 2);
		}
		ctx.fill();
	}
}

/**
 * Draw all LEDs in the dim "off" state — used as fallback for missing/broken scripts.
 */
export function drawOffState(
	canvas: HTMLCanvasElement,
	board: BoardData,
): void {
	const { ctx, w, h } = setupCanvas(canvas);
	const emptyPixels = new Uint8Array(board.ledCount * 3);
	drawLeds(ctx, emptyPixels, board, w, h);
}
