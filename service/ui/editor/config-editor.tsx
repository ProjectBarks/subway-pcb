import type { ConfigField } from "./types";

export function ConfigFieldEditor({
	fields,
	onChange,
}: {
	fields: ConfigField[];
	onChange: (fields: ConfigField[]) => void;
}) {
	const addField = () => {
		onChange([
			...fields,
			{
				key: "",
				label: "",
				type: "color",
				default: "#ffffff",
				group: "",
				min: "",
				max: "",
				options: [],
			},
		]);
	};

	const updateField = (index: number, updates: Partial<ConfigField>) => {
		const next = fields.map((f, i) => (i === index ? { ...f, ...updates } : f));
		onChange(next);
	};

	const removeField = (index: number) => {
		onChange(fields.filter((_, i) => i !== index));
	};

	return (
		<div class="p-5 bg-bg-primary space-y-4">
			<div class="flex items-center justify-between">
				<div>
					<h3 class="text-text-primary mb-1">Config Fields</h3>
					<p class="text-text-secondary text-sm">
						Define settings that users can customize when using this plugin
					</p>
				</div>
				<button
					type="button"
					onClick={addField}
					class="btn-primary flex items-center gap-2"
				>
					<svg
						class="size-4"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
					>
						<path d="M5 12h14" />
						<path d="M12 5v14" />
					</svg>
					Add Field
				</button>
			</div>
			{fields.length === 0 ? (
				<div class="text-center py-10 border border-dashed border-border-subtle rounded-xl">
					<p class="text-text-muted text-sm">No config fields defined yet</p>
					<p class="text-text-muted text-xs mt-1">
						Add fields to let users customize colors, numbers, and options
					</p>
				</div>
			) : (
				<div class="space-y-3">
					{fields.map((field, i) => (
						<ConfigFieldRow
							key={i}
							field={field}
							onChange={(u) => updateField(i, u)}
							onDelete={() => removeField(i)}
						/>
					))}
				</div>
			)}
		</div>
	);
}

export function ConfigFieldRow({
	field,
	onChange,
	onDelete,
}: {
	field: ConfigField;
	onChange: (u: Partial<ConfigField>) => void;
	onDelete: () => void;
}) {
	return (
		<div class="p-4 bg-bg-surface border border-border-subtle rounded-xl space-y-3">
			<div class="flex items-start gap-3">
				<div class="flex-1 grid grid-cols-2 gap-3">
					<div>
						<label class="text-text-muted text-xs block mb-1">Key</label>
						<input
							type="text"
							value={field.key}
							onInput={(e) =>
								onChange({ key: (e.target as HTMLInputElement).value })
							}
							placeholder="e.g. bg_color"
							class="w-full h-9 px-3 bg-bg-primary border border-border-subtle rounded-lg text-text-primary text-sm focus:outline-none focus:ring-1 focus:ring-accent-gold"
						/>
					</div>
					<div>
						<label class="text-text-muted text-xs block mb-1">Label</label>
						<input
							type="text"
							value={field.label}
							onInput={(e) =>
								onChange({ label: (e.target as HTMLInputElement).value })
							}
							placeholder="e.g. Background"
							class="w-full h-9 px-3 bg-bg-primary border border-border-subtle rounded-lg text-text-primary text-sm focus:outline-none focus:ring-1 focus:ring-accent-gold"
						/>
					</div>
				</div>
				<button
					type="button"
					onClick={onDelete}
					class="mt-5 size-9 rounded-lg hover:bg-status-error/10 flex items-center justify-center flex-shrink-0"
					title="Delete field"
				>
					<svg
						class="size-4 text-status-error"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
					>
						<path d="M3 6h18" />
						<path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6" />
						<path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2" />
					</svg>
				</button>
			</div>
			<div class="grid grid-cols-3 gap-3">
				<div>
					<label class="text-text-muted text-xs block mb-1">Type</label>
					<select
						value={field.type}
						onChange={(e) => {
							const type = (e.target as HTMLSelectElement)
								.value as ConfigField["type"];
							const updates: Partial<ConfigField> = { type };
							if (type === "color") updates.default = "#ffffff";
							else if (type === "number") updates.default = "0";
							else updates.default = "";
							onChange(updates);
						}}
						class="w-full h-9 px-3 bg-bg-primary border border-border-subtle rounded-lg text-text-primary text-sm focus:outline-none focus:ring-1 focus:ring-accent-gold"
					>
						<option value="color">Color</option>
						<option value="number">Number</option>
						<option value="select">Select</option>
					</select>
				</div>
				<div>
					<label class="text-text-muted text-xs block mb-1">Default</label>
					{field.type === "color" ? (
						<div class="flex items-center gap-2">
							<input
								type="color"
								value={field.default}
								onInput={(e) =>
									onChange({ default: (e.target as HTMLInputElement).value })
								}
								class="h-9 w-9 rounded-lg border border-border-subtle bg-transparent cursor-pointer"
							/>
							<span class="text-text-secondary text-xs font-mono">
								{field.default}
							</span>
						</div>
					) : (
						<input
							type={field.type === "number" ? "number" : "text"}
							value={field.default}
							onInput={(e) =>
								onChange({ default: (e.target as HTMLInputElement).value })
							}
							class="w-full h-9 px-3 bg-bg-primary border border-border-subtle rounded-lg text-text-primary text-sm focus:outline-none focus:ring-1 focus:ring-accent-gold"
						/>
					)}
				</div>
				<div>
					<label class="text-text-muted text-xs block mb-1">Group</label>
					<input
						type="text"
						value={field.group}
						onInput={(e) =>
							onChange({ group: (e.target as HTMLInputElement).value })
						}
						placeholder="e.g. Colors"
						class="w-full h-9 px-3 bg-bg-primary border border-border-subtle rounded-lg text-text-primary text-sm focus:outline-none focus:ring-1 focus:ring-accent-gold"
					/>
				</div>
			</div>
			{field.type === "number" && (
				<div class="grid grid-cols-2 gap-3">
					<div>
						<label class="text-text-muted text-xs block mb-1">Min</label>
						<input
							type="number"
							value={field.min}
							onInput={(e) =>
								onChange({ min: (e.target as HTMLInputElement).value })
							}
							class="w-full h-9 px-3 bg-bg-primary border border-border-subtle rounded-lg text-text-primary text-sm focus:outline-none focus:ring-1 focus:ring-accent-gold"
						/>
					</div>
					<div>
						<label class="text-text-muted text-xs block mb-1">Max</label>
						<input
							type="number"
							value={field.max}
							onInput={(e) =>
								onChange({ max: (e.target as HTMLInputElement).value })
							}
							class="w-full h-9 px-3 bg-bg-primary border border-border-subtle rounded-lg text-text-primary text-sm focus:outline-none focus:ring-1 focus:ring-accent-gold"
						/>
					</div>
				</div>
			)}
			{field.type === "select" && (
				<div>
					<label class="text-text-muted text-xs block mb-1">
						Options (one per line)
					</label>
					<textarea
						value={(field.options || []).join("\n")}
						onInput={(e) =>
							onChange({
								options: (e.target as HTMLTextAreaElement).value
									.split("\n")
									.filter(Boolean),
							})
						}
						rows={3}
						placeholder={"option1\noption2\noption3"}
						class="w-full p-3 bg-bg-primary border border-border-subtle rounded-lg text-text-primary text-sm resize-none focus:outline-none focus:ring-1 focus:ring-accent-gold"
					/>
				</div>
			)}
		</div>
	);
}
