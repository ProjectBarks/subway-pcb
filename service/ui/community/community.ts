import Prism from "prismjs";
import "prismjs/components/prism-lua";
import "../editor/prism-lua.css";
import { initPreviews, type PreviewCleanup } from "../lib/preview-controller";

export function highlightLua(code: string): string {
	return Prism.highlight(code, Prism.languages.lua, "lua");
}

(window as unknown as Record<string, unknown>).highlightLua = highlightLua;

let cleanup: PreviewCleanup | null = null;

async function startPreviews() {
	cleanup?.destroy();
	cleanup = await initPreviews();
}

startPreviews();

document.body.addEventListener("htmx:afterSwap", (e) => {
	const target = (e as CustomEvent).detail?.target;
	if (target?.id === "plugin-grid") {
		startPreviews();
	}
});
