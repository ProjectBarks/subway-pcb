import { useRef } from "preact/hooks";
import Prism from "prismjs";
import "prismjs/components/prism-lua";

export function CodeEditor({
	value,
	onChange,
}: {
	value: string;
	onChange: (code: string) => void;
}) {
	const textareaRef = useRef<HTMLTextAreaElement>(null);
	const preRef = useRef<HTMLPreElement>(null);

	const syncScroll = () => {
		if (textareaRef.current && preRef.current) {
			preRef.current.scrollTop = textareaRef.current.scrollTop;
			preRef.current.scrollLeft = textareaRef.current.scrollLeft;
		}
	};

	const handleKeyDown = (e: KeyboardEvent) => {
		if (e.key === "Tab") {
			e.preventDefault();
			const textarea = e.target as HTMLTextAreaElement;
			const start = textarea.selectionStart;
			const end = textarea.selectionEnd;
			const newValue = `${value.substring(0, start)}  ${value.substring(end)}`;
			onChange(newValue);
			requestAnimationFrame(() => {
				textarea.selectionStart = textarea.selectionEnd = start + 2;
			});
		}
	};

	const highlighted = Prism.highlight(value, Prism.languages.lua, "lua");

	return (
		<div class="relative h-full" style="min-height: 300px">
			<pre
				ref={preRef}
				class="absolute inset-0 p-5 bg-black text-sm pointer-events-none overflow-auto whitespace-pre-wrap break-words"
				style="font-family: Monaco, 'Courier New', monospace; tab-size: 2;"
				dangerouslySetInnerHTML={{ __html: `${highlighted}\n` }}
			/>
			<textarea
				ref={textareaRef}
				value={value}
				onInput={(e) => onChange((e.target as HTMLTextAreaElement).value)}
				onScroll={syncScroll}
				onKeyDown={handleKeyDown}
				class="absolute inset-0 p-5 bg-transparent text-transparent caret-white resize-none focus:outline-none"
				style="font-family: Monaco, 'Courier New', monospace; font-size: 14px; line-height: 1.5; tab-size: 2; -webkit-text-fill-color: transparent;"
				spellcheck={false}
			/>
		</div>
	);
}
