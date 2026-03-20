import { createAnimationLoop } from "./animation";
import { createHeroMode } from "./interaction/hero";
import { createInspectMode } from "./interaction/inspect";
import { createPreviewMode } from "./interaction/preview";
import type { InteractionMode } from "./interaction/types";
import type { LedInfo } from "./leds";
import { createLedSystem } from "./leds";
import { createScene } from "./scene";

export interface BoardManifest {
	id: string;
	name: string;
	version: number;
	ledCount: number;
	strips: number[];
	model: string;
	camera: { fov: number; distance: number };
	defaultPlugin: string;
	defaultPreset: string;
	features: string[];
	ledPositions: Array<{
		ref: string;
		x: number;
		y: number;
		z: number;
		angle: number;
		index: number;
		stationId: string;
	}>;
}

export interface BoardViewerConfig {
	boardUrl: string;
	mode: "hero" | "inspect" | "preview";
	pixelsUrl?: string;
	deviceId?: string;
	camera?: { fov?: number; distance?: number };
	onLedHover?: (info: LedInfo | null) => void;
	onLedClick?: (info: LedInfo) => void;
}

export interface BoardViewerHandle {
	setPixels(data: Uint8Array): void;
	dispose(): void;
}

export type { LedInfo };

/**
 * Mount a 3D board viewer into a container element.
 * Returns a handle for setting pixels and cleanup.
 */
export async function initBoardViewer(
	container: HTMLElement,
	config: BoardViewerConfig,
): Promise<BoardViewerHandle> {
	const resp = await fetch(config.boardUrl);
	const board: BoardManifest = await resp.json();

	const boardScene = createScene({
		fov: config.camera?.fov ?? board.camera.fov,
		distance: config.camera?.distance ?? board.camera.distance,
	});
	boardScene.mount(container);

	// Create LED system and load model in parallel
	const ledSystem = createLedSystem(
		board.ledPositions,
		board.ledCount,
		board.strips,
	);
	const glbUrl = config.boardUrl.replace("board.json", board.model);
	const model = await boardScene.loadModel(glbUrl);

	boardScene.scene.add(ledSystem.group);
	ledSystem.reposition(boardScene.boardOffset);

	// Create interaction mode
	let mode: InteractionMode;
	switch (config.mode) {
		case "hero":
			mode = createHeroMode(model, ledSystem.group);
			break;
		case "inspect":
			mode = createInspectMode(
				boardScene.camera,
				boardScene.renderer,
				ledSystem,
				{ onHover: config.onLedHover, onClick: config.onLedClick },
			);
			break;
		case "preview":
			mode = createPreviewMode();
			break;
	}

	mode.init();

	const loop = createAnimationLoop(
		boardScene.renderer,
		boardScene.scene,
		boardScene.camera,
		mode,
	);
	loop.start();

	// Hero mode polls internally; inspect/preview get pixels from caller
	if (config.mode === "hero" && config.pixelsUrl) {
		ledSystem.startPolling(config.pixelsUrl);
	}

	const onResize = () => boardScene.resize();
	window.addEventListener("resize", onResize);

	return {
		setPixels(data: Uint8Array) {
			ledSystem.setPixels(data);
		},
		dispose() {
			loop.stop();
			mode.dispose();
			ledSystem.dispose();
			window.removeEventListener("resize", onResize);
			boardScene.dispose();
		},
	};
}
