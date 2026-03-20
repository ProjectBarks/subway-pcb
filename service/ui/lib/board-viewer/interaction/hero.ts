import type { Group } from "three";
import { createCursorTracker } from "../cursor-tracker";
import type { InteractionMode } from "./types";

/** Base tilt — Y-axis rotation (0 = facing forward). */
const BASE_TILT_Y = 0;

/** Slow idle drift speed (radians per second). */
const IDLE_SPEED = 0.06;

export function createHeroMode(model: Group, ledGroup: Group): InteractionMode {
	const tracker = createCursorTracker();

	const onMove = (e: MouseEvent) => tracker.onMouseMove(e.clientX, e.clientY);
	const onLeave = () => tracker.onMouseLeave();

	return {
		init() {
			window.addEventListener("mousemove", onMove);
			window.addEventListener("mouseleave", onLeave);
		},

		update(_dt: number) {
			const [cursorX, cursorY] = tracker.update();
			const idle = Math.sin(Date.now() * 0.001 * IDLE_SPEED) * 0.015;

			const ry = BASE_TILT_Y + cursorY + idle;
			const rx = cursorX;

			model.rotation.x = rx;
			model.rotation.y = ry;
			ledGroup.rotation.x = rx;
			ledGroup.rotation.y = ry;
		},

		dispose() {
			window.removeEventListener("mousemove", onMove);
			window.removeEventListener("mouseleave", onLeave);
			tracker.dispose();
		},
	};
}
