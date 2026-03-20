import {
	AmbientLight,
	Box3,
	DirectionalLight,
	type Group,
	PerspectiveCamera,
	Scene,
	Vector3,
	WebGLRenderer,
} from "three";
import { DRACOLoader } from "three/addons/loaders/DRACOLoader.js";
import { GLTFLoader } from "three/addons/loaders/GLTFLoader.js";

const DRACO_CDN = "https://www.gstatic.com/draco/versioned/decoders/1.5.7/";

export interface HeroScene {
	renderer: WebGLRenderer;
	camera: PerspectiveCamera;
	scene: Scene;
	model: Group | null;
	/** Bounding-box center used to align LEDs — same as boardOffset in view_live. */
	boardOffset: Vector3;
	mount(container: HTMLElement): void;
	loadModel(url: string): Promise<Group>;
	resize(): void;
	dispose(): void;
}

export function createScene(): HeroScene {
	const scene = new Scene();

	const camera = new PerspectiveCamera(40, 1, 0.1, 2000);
	camera.position.set(0, 0, 420);

	const renderer = new WebGLRenderer({
		antialias: true,
		alpha: true,
	});
	renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
	renderer.setClearColor(0x000000, 0);

	// Lighting
	scene.add(new AmbientLight(0xffffff, 1.2));

	const key = new DirectionalLight(0xffffff, 0.8);
	key.position.set(100, -50, 400);
	scene.add(key);

	const fill = new DirectionalLight(0xffffff, 0.3);
	fill.position.set(-100, 50, 200);
	scene.add(fill);

	// Draco-enabled loader
	const draco = new DRACOLoader();
	draco.setDecoderPath(DRACO_CDN);
	const loader = new GLTFLoader();
	loader.setDRACOLoader(draco);

	let model: Group | null = null;
	const boardOffset = new Vector3();

	return {
		renderer,
		camera,
		scene,
		boardOffset,
		get model() {
			return model;
		},

		mount(container: HTMLElement) {
			container.appendChild(renderer.domElement);
			renderer.domElement.style.display = "block";
			this.resize();
		},

		async loadModel(url: string): Promise<Group> {
			const gltf = await loader.loadAsync(url);
			const root = gltf.scene;

			const box = new Box3().setFromObject(root);
			const center = box.getCenter(new Vector3());
			boardOffset.copy(center);
			root.position.sub(center);

			scene.add(root);
			model = root;
			return root;
		},

		resize() {
			const el = renderer.domElement.parentElement;
			if (!el) return;
			const w = el.clientWidth;
			const h = el.clientHeight;
			camera.aspect = w / h;
			camera.updateProjectionMatrix();
			renderer.setSize(w, h);
		},

		dispose() {
			renderer.dispose();
			draco.dispose();
			renderer.domElement.remove();
		},
	};
}
