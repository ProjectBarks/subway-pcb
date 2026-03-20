import type { PerspectiveCamera, Scene, WebGLRenderer } from "three";
import type { InteractionMode } from "./interaction/types";

export interface AnimationLoop {
	start(): void;
	stop(): void;
}

export function createAnimationLoop(
	renderer: WebGLRenderer,
	scene: Scene,
	camera: PerspectiveCamera,
	mode: InteractionMode,
): AnimationLoop {
	let frameId = 0;
	let lastTime = 0;

	function tick(time: number) {
		frameId = requestAnimationFrame(tick);
		const dt = lastTime ? (time - lastTime) / 1000 : 0;
		lastTime = time;
		mode.update(dt);
		renderer.render(scene, camera);
	}

	return {
		start() {
			if (!frameId) frameId = requestAnimationFrame(tick);
		},
		stop() {
			if (frameId) {
				cancelAnimationFrame(frameId);
				frameId = 0;
			}
		},
	};
}
