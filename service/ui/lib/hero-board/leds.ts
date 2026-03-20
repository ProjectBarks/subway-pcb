import {
	AdditiveBlending,
	BoxGeometry,
	CanvasTexture,
	Color,
	Group,
	type Material,
	Mesh,
	MeshBasicMaterial,
	MeshStandardMaterial,
	Sprite,
	SpriteMaterial,
	type Vector3,
} from "three";
import { TOTAL_LEDS } from "../board";
import { decodePixelFrame } from "../protobuf";

const LED_SIZE = 2.5;
const POLL_INTERVAL = 2000;

interface LedEntry {
	x: number;
	y: number;
	z: number;
	ref: string;
	index: number;
}

function createGlowTexture(): CanvasTexture {
	const c = document.createElement("canvas");
	c.width = c.height = 64;
	const ctx = c.getContext("2d")!;
	const g = ctx.createRadialGradient(32, 32, 0, 32, 32, 32);
	g.addColorStop(0, "rgba(255,255,255,1)");
	g.addColorStop(0.3, "rgba(255,255,255,0.4)");
	g.addColorStop(1, "rgba(255,255,255,0)");
	ctx.fillStyle = g;
	ctx.fillRect(0, 0, 64, 64);
	return new CanvasTexture(c);
}

export interface LedSystem {
	group: Group;
	/** Call after model is loaded and centered to position LEDs. */
	reposition(boardOffset: Vector3): void;
	startPolling(apiUrl: string): void;
	dispose(): void;
}

export async function createLedSystem(ledsUrl: string): Promise<LedSystem> {
	const resp = await fetch(ledsUrl);
	const ledData: LedEntry[] = await resp.json();

	const group = new Group();
	const geo = new BoxGeometry(LED_SIZE, LED_SIZE, 0.5);
	const glowTex = createGlowTexture();
	const meshes: Mesh[] = [];
	let pollTimer = 0;

	// Create LED meshes (positioned later by reposition())
	for (const led of ledData) {
		const mat = new MeshStandardMaterial({
			color: 0xdddddd,
			emissive: 0x080808,
			emissiveIntensity: 0.1,
			transparent: true,
			opacity: 0.65,
			roughness: 0.4,
			metalness: 0.0,
		});
		const mesh = new Mesh(geo, mat);
		mesh.position.set(led.x, led.y, led.z + 0.15);
		group.add(mesh);
		meshes.push(mesh);
	}

	function setLit(idx: number, r: number, g: number, b: number) {
		const mesh = meshes[idx];
		(mesh.material as Material).dispose();

		const max = Math.max(r, g, b) || 1;
		const color = new Color(r / max, g / max, b / max);
		const hsl = { h: 0, s: 0, l: 0 };
		color.getHSL(hsl);
		color.setHSL(hsl.h, 1.0, 0.45);

		mesh.material = new MeshBasicMaterial({ color });

		if (!mesh.userData.glow) {
			const sprite = new Sprite(
				new SpriteMaterial({
					map: glowTex,
					color,
					transparent: true,
					opacity: 0.5,
					blending: AdditiveBlending,
					depthWrite: false,
				}),
			);
			sprite.scale.set(3, 3, 1);
			mesh.add(sprite);
			mesh.userData.glow = sprite;
		} else {
			const s = mesh.userData.glow as Sprite;
			(s.material as SpriteMaterial).color.copy(color);
			s.visible = true;
		}
	}

	function setOff(idx: number) {
		const mesh = meshes[idx];
		(mesh.material as Material).dispose();
		mesh.material = new MeshStandardMaterial({
			color: 0xdddddd,
			emissive: 0x080808,
			emissiveIntensity: 0.1,
			transparent: true,
			opacity: 0.65,
			roughness: 0.4,
			metalness: 0.0,
		});
		if (mesh.userData.glow) (mesh.userData.glow as Sprite).visible = false;
	}

	async function poll(apiUrl: string) {
		try {
			const r = await fetch(apiUrl, {
				headers: { "X-Device-ID": "hero-landing" },
			});
			if (!r.ok) return;
			const buf = await r.arrayBuffer();
			if (buf.byteLength === 0) return;
			const frame = decodePixelFrame(buf);
			if (!frame.pixels || frame.pixels.length < TOTAL_LEDS * 3) return;

			for (let i = 0; i < TOTAL_LEDS && i < meshes.length; i++) {
				const ri = frame.pixels[i * 3];
				const gi = frame.pixels[i * 3 + 1];
				const bi = frame.pixels[i * 3 + 2];
				if (ri === 0 && gi === 0 && bi === 0) setOff(i);
				else setLit(i, ri, gi, bi);
			}
		} catch {
			// silent on landing page
		}
	}

	return {
		group,

		reposition(boardOffset: Vector3) {
			meshes.forEach((mesh, i) => {
				const led = ledData[i];
				mesh.position.set(
					led.x - boardOffset.x,
					led.y - boardOffset.y,
					led.z + 0.15 - boardOffset.z,
				);
			});
		},

		startPolling(apiUrl: string) {
			poll(apiUrl);
			pollTimer = window.setInterval(() => poll(apiUrl), POLL_INTERVAL);
		},

		dispose() {
			if (pollTimer) clearInterval(pollTimer);
			geo.dispose();
			glowTex.dispose();
			for (const m of meshes) (m.material as Material).dispose();
		},
	};
}
