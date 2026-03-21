import type { LuaRunner } from "./lua-runner";

/**
 * Fetch board JSON, build the LED map, and configure the runner.
 * Returns the total LED count from the board data.
 */
export async function loadBoardData(
	runner: LuaRunner,
	boardUrl: string,
): Promise<number> {
	const resp = await fetch(boardUrl);
	const board: {
		ledCount: number;
		strips?: number[];
		ledPositions: Array<{ index: number; stationId: string }>;
	} = await resp.json();

	const ledCount = board.ledCount ?? 478;
	const ledMap = new Array<string>(ledCount).fill("");
	for (const pos of board.ledPositions) {
		if (pos.index >= 0 && pos.index < ledCount && pos.stationId) {
			ledMap[pos.index] = pos.stationId;
		}
	}
	runner.setLedMap(ledMap);
	if (board.strips) runner.setStripSizes(board.strips);

	return ledCount;
}
