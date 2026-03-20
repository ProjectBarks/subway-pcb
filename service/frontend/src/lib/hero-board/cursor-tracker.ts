/** Damping factor — lower = smoother/laggier. */
const DAMPING = 0.04;

/** Max rotation in radians the cursor can induce. */
const MAX_TILT = 0.12;

export interface CursorTracker {
	onMouseMove(clientX: number, clientY: number): void;
	onMouseLeave(): void;
	/** Returns smoothed [rotX, rotY] offset each frame. */
	update(): [number, number];
	dispose(): void;
}

export function createCursorTracker(): CursorTracker {
	let targetX = 0;
	let targetY = 0;
	let currentX = 0;
	let currentY = 0;

	return {
		onMouseMove(clientX: number, clientY: number) {
			// Normalize to [-1, 1] across the viewport
			targetX = (clientY / window.innerHeight) * 2 - 1;
			targetY = (clientX / window.innerWidth) * 2 - 1;
		},

		onMouseLeave() {
			targetX = 0;
			targetY = 0;
		},

		update(): [number, number] {
			currentX += (targetX - currentX) * DAMPING;
			currentY += (targetY - currentY) * DAMPING;
			return [currentX * MAX_TILT, currentY * MAX_TILT];
		},

		dispose() {
			targetX = targetY = currentX = currentY = 0;
		},
	};
}
