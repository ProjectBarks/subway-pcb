import {
	type PerspectiveCamera,
	Raycaster,
	Vector2,
	type WebGLRenderer,
} from "three";
import { OrbitControls } from "three/addons/controls/OrbitControls.js";
import type { LedInfo, LedSystem } from "../leds";
import type { InteractionMode } from "./types";

export function createInspectMode(
	camera: PerspectiveCamera,
	renderer: WebGLRenderer,
	ledSystem: LedSystem,
	callbacks: {
		onHover?: (info: LedInfo | null) => void;
		onClick?: (info: LedInfo) => void;
	},
): InteractionMode {
	let controls: OrbitControls | null = null;
	const raycaster = new Raycaster();
	const mouse = new Vector2();

	const onMouseMove = (e: MouseEvent) => {
		const rect = renderer.domElement.getBoundingClientRect();
		mouse.x = ((e.clientX - rect.left) / rect.width) * 2 - 1;
		mouse.y = -((e.clientY - rect.top) / rect.height) * 2 + 1;

		raycaster.setFromCamera(mouse, camera);
		const info = ledSystem.raycast(raycaster);
		callbacks.onHover?.(info);
	};

	const onClick = (e: MouseEvent) => {
		const rect = renderer.domElement.getBoundingClientRect();
		mouse.x = ((e.clientX - rect.left) / rect.width) * 2 - 1;
		mouse.y = -((e.clientY - rect.top) / rect.height) * 2 + 1;

		raycaster.setFromCamera(mouse, camera);
		const info = ledSystem.raycast(raycaster);
		if (info) callbacks.onClick?.(info);
	};

	return {
		init() {
			controls = new OrbitControls(camera, renderer.domElement);
			controls.enableDamping = true;
			controls.dampingFactor = 0.05;
			controls.enablePan = false;
			controls.minDistance = 100;
			controls.maxDistance = 800;

			renderer.domElement.addEventListener("mousemove", onMouseMove);
			renderer.domElement.addEventListener("click", onClick);
		},

		update(_dt: number) {
			controls?.update();
		},

		dispose() {
			controls?.dispose();
			renderer.domElement.removeEventListener("mousemove", onMouseMove);
			renderer.domElement.removeEventListener("click", onClick);
		},
	};
}
