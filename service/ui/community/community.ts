import Prism from "prismjs";
import "prismjs/components/prism-lua";
import "../editor/prism-lua.css";
import Alpine from "alpinejs";
import { initPreviews, type PreviewCleanup } from "../lib/preview-controller";

function highlightLua(code: string): string {
	return Prism.highlight(code, Prism.languages.lua, "lua");
}

// Register Alpine component for the code modal
Alpine.data("codeModal", () => ({
	open: false,
	name: "",
	author: "",
	code: "",
	codeHtml: "",

	show(pluginId: unknown, pluginName: unknown, pluginAuthor: unknown) {
		this.name = String(pluginName);
		this.author = String(pluginAuthor);
		this.codeHtml = "Loading...";
		this.open = true;
		fetch(`/api/v1/plugins/${pluginId}`)
			.then((r) => r.json())
			.then((data: { data?: { lua_source?: string } }) => {
				this.code = data.data?.lua_source || "-- No source available";
				this.codeHtml = highlightLua(this.code);
			})
			.catch(() => {
				this.code = "-- Failed to load source";
				this.codeHtml = this.code;
			});
	},

	copyCode() {
		navigator.clipboard.writeText(this.code).then(() => {
			document.body.dispatchEvent(
				new CustomEvent("showtoast", {
					detail: { message: "Code copied to clipboard", type: "success" },
					bubbles: true,
				}),
			);
		});
	},
}));

// Preview controller lifecycle
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
