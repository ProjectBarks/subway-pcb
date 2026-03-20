import { createAnimationLoop } from "./animation";
import { createCursorTracker } from "./cursor-tracker";
import { createLedSystem } from "./leds";
import { createScene } from "./scene";

export interface HeroBoardConfig {
	glbUrl: string;
	ledsUrl: string;
	pixelsUrl: string;
}

/**
 * Mount an interactive 3D PCB board into a container element.
 * Streams live LED data from the pixel API.
 * Returns a cleanup function.
 */
export async function initHeroBoard(
	container: HTMLElement,
	config: HeroBoardConfig,
): Promise<() => void> {
	const heroScene = createScene();
	heroScene.mount(container);

	// Start loading LEDs and model in parallel
	const [leds, model] = await Promise.all([
		createLedSystem(config.ledsUrl),
		heroScene.loadModel(config.glbUrl),
	]);

	// Add LED group to scene (not model) — same as view_live.html
	heroScene.scene.add(leds.group);

	// Reposition LEDs relative to board center — same as view_live.html
	leds.reposition(heroScene.boardOffset);

	// Start streaming pixel data
	leds.startPolling(config.pixelsUrl);

	const tracker = createCursorTracker();
	const loop = createAnimationLoop(
		heroScene.renderer,
		heroScene.scene,
		heroScene.camera,
		model,
		leds.group,
		tracker,
	);

	const onMove = (e: MouseEvent) => tracker.onMouseMove(e.clientX, e.clientY);
	const onLeave = () => tracker.onMouseLeave();
	const onResize = () => heroScene.resize();

	window.addEventListener("mousemove", onMove);
	window.addEventListener("mouseleave", onLeave);
	window.addEventListener("resize", onResize);

	loop.start();

	return () => {
		loop.stop();
		leds.dispose();
		window.removeEventListener("mousemove", onMove);
		window.removeEventListener("mouseleave", onLeave);
		window.removeEventListener("resize", onResize);
		tracker.dispose();
		heroScene.dispose();
	};
}
