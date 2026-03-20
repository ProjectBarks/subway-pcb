import type { Group, PerspectiveCamera, Scene, WebGLRenderer } from "three";
import type { CursorTracker } from "./cursor-tracker";

/** Base tilt — Y-axis rotation (0 = facing forward). */
const BASE_TILT_Y = 0;

/** Slow idle drift speed (radians per second). */
const IDLE_SPEED = 0.06;

export interface AnimationLoop {
	start(): void;
	stop(): void;
}

export function createAnimationLoop(
	renderer: WebGLRenderer,
	scene: Scene,
	camera: PerspectiveCamera,
	model: Group,
	ledGroup: Group,
	tracker: CursorTracker,
): AnimationLoop {
	let frameId = 0;

	function tick() {
		frameId = requestAnimationFrame(tick);

		const [cursorX, cursorY] = tracker.update();
		const idle = Math.sin(Date.now() * 0.001 * IDLE_SPEED) * 0.015;

		const ry = BASE_TILT_Y + cursorY + idle;
		const rx = cursorX;

		// Keep model and LEDs in sync
		model.rotation.x = rx;
		model.rotation.y = ry;
		ledGroup.rotation.x = rx;
		ledGroup.rotation.y = ry;

		renderer.render(scene, camera);
	}

	return {
		start() {
			if (!frameId) tick();
		},
		stop() {
			if (frameId) {
				cancelAnimationFrame(frameId);
				frameId = 0;
			}
		},
	};
}
