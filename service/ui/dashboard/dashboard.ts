import { initPreviews, type PreviewCleanup } from "../lib/preview-controller";

let cleanup: PreviewCleanup | null = null;

async function init() {
	cleanup = await initPreviews();
}
init();

// Re-init after htmx replaces the board grid (5s polling)
document.addEventListener("htmx:afterSwap", (e: Event) => {
	const detail = (e as CustomEvent).detail;
	if (detail?.target?.id === "board-grid") {
		cleanup?.destroy();
		initPreviews().then((c) => {
			cleanup = c;
		});
	}
});
