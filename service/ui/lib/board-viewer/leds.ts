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
	type Raycaster,
	Sprite,
	SpriteMaterial,
	type Vector3,
} from "three";
import { decodePixelFrame } from "../protobuf";

const LED_SIZE = 2.5;
const POLL_INTERVAL = 2000;

export interface LedEntry {
	x: number;
	y: number;
	z: number;
	ref: string;
	index: number;
	stationId: string;
}

export interface LedInfo {
	ledIndex: number;
	ref: string;
	stationId: string;
	strip: number;
	pixel: number;
	r: number;
	g: number;
	b: number;
}

export interface LedSystem {
	group: Group;
	meshes: Mesh[];
	ledData: LedEntry[];
	reposition(boardOffset: Vector3): void;
	setPixels(pixels: Uint8Array): void;
	startPolling(apiUrl: string, deviceId?: string): void;
	raycast(raycaster: Raycaster): LedInfo | null;
	dispose(): void;
}

function createGlowTexture(): CanvasTexture {
	const c = document.createElement("canvas");
	c.width = c.height = 64;
	// biome-ignore lint/style/noNonNullAssertion: 2d context is always available on a freshly created canvas
	const ctx = c.getContext("2d")!;
	const g = ctx.createRadialGradient(32, 32, 0, 32, 32, 32);
	g.addColorStop(0, "rgba(255,255,255,1)");
	g.addColorStop(0.3, "rgba(255,255,255,0.4)");
	g.addColorStop(1, "rgba(255,255,255,0)");
	ctx.fillStyle = g;
	ctx.fillRect(0, 0, 64, 64);
	return new CanvasTexture(c);
}

function getStripPixel(idx: number, strips: number[]): [number, number] {
	let off = 0;
	for (let s = 0; s < strips.length; s++) {
		if (idx < off + strips[s]) return [s, idx - off];
		off += strips[s];
	}
	return [-1, -1];
}

export function createLedSystem(
	ledData: LedEntry[],
	totalLeds: number,
	strips: number[],
): LedSystem {
	const group = new Group();
	const geo = new BoxGeometry(LED_SIZE, LED_SIZE, 0.5);
	const glowTex = createGlowTexture();
	const meshes: Mesh[] = [];
	const pixelColors = new Uint8Array(totalLeds * 3);
	let pollTimer = 0;

	// Create LED meshes
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

	function applyPixels() {
		for (let i = 0; i < totalLeds && i < meshes.length; i++) {
			const r = pixelColors[i * 3];
			const g = pixelColors[i * 3 + 1];
			const b = pixelColors[i * 3 + 2];
			if (r === 0 && g === 0 && b === 0) setOff(i);
			else setLit(i, r, g, b);
		}
	}

	async function poll(apiUrl: string, deviceId?: string) {
		try {
			const headers: Record<string, string> = {};
			let url = apiUrl;
			if (deviceId) {
				url = `${apiUrl}?device=${encodeURIComponent(deviceId)}`;
			} else {
				headers["X-Device-ID"] = "hero-landing";
			}
			const r = await fetch(url, { headers });
			if (!r.ok) return;
			const buf = await r.arrayBuffer();
			if (buf.byteLength === 0) return;
			const frame = decodePixelFrame(buf);
			if (!frame.pixels || frame.pixels.length < totalLeds * 3) return;
			pixelColors.set(frame.pixels.subarray(0, totalLeds * 3));
			applyPixels();
		} catch {
			// silent
		}
	}

	return {
		group,
		meshes,
		ledData,

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

		setPixels(pixels: Uint8Array) {
			if (pixels.length >= totalLeds * 3) {
				pixelColors.set(pixels.subarray(0, totalLeds * 3));
			}
			applyPixels();
		},

		startPolling(apiUrl: string, deviceId?: string) {
			poll(apiUrl, deviceId);
			pollTimer = window.setInterval(
				() => poll(apiUrl, deviceId),
				POLL_INTERVAL,
			);
		},

		raycast(raycaster: Raycaster): LedInfo | null {
			const intersects = raycaster.intersectObjects(meshes);
			if (intersects.length === 0) return null;

			const mesh = intersects[0].object as Mesh;
			const idx = meshes.indexOf(mesh);
			if (idx < 0) return null;

			const led = ledData[idx];
			const [strip, pixel] = getStripPixel(idx, strips);

			return {
				ledIndex: idx,
				ref: led.ref,
				stationId: led.stationId,
				strip,
				pixel,
				r: pixelColors[idx * 3],
				g: pixelColors[idx * 3 + 1],
				b: pixelColors[idx * 3 + 2],
			};
		},

		dispose() {
			if (pollTimer) clearInterval(pollTimer);
			geo.dispose();
			glowTex.dispose();
			for (const m of meshes) (m.material as Material).dispose();
		},
	};
}
