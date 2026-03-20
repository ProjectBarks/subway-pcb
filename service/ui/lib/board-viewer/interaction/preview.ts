import type { InteractionMode } from "./types";

export function createPreviewMode(): InteractionMode {
	return {
		init() {},
		update(_dt: number) {},
		dispose() {},
	};
}
