import Prism from "prismjs";
import "prismjs/components/prism-lua";
import "../editor/prism-lua.css";

export function highlightLua(code: string): string {
	return Prism.highlight(code, Prism.languages.lua, "lua");
}

(window as unknown as Record<string, unknown>).highlightLua = highlightLua;
