/** Collect all named config inputs (color, number, select) into a key-value map */
export function collectConfig(): Record<string, string> {
	const config: Record<string, string> = {};
	document
		.querySelectorAll<HTMLInputElement | HTMLSelectElement>(
			"input[type='color'][name], input[type='number'][name], select[name]",
		)
		.forEach((el) => {
			if (el.name) config[el.name] = el.value;
		});
	return config;
}

export function updatePreviewColor(_input: HTMLInputElement): void {
	if (!window._luaRunner) return;
	window._luaRunner.setConfig(collectConfig());
}

export function collectRouteColorsToForm(form: HTMLFormElement): void {
	// Remove any existing color_ hidden inputs
	form
		.querySelectorAll<HTMLInputElement>('input[name^="color_"]')
		.forEach((el) => el.remove());
	// Add current color values from all pickers
	document
		.querySelectorAll<HTMLInputElement>(".route-color-input")
		.forEach((input) => {
			const hidden = document.createElement("input");
			hidden.type = "hidden";
			hidden.name = input.name;
			hidden.value = input.value;
			form.appendChild(hidden);
		});
}

export function collectConfigToPresetForm(form: HTMLFormElement): void {
	// Remove old val_ inputs
	form
		.querySelectorAll<HTMLInputElement>('input[name^="val_"]')
		.forEach((e) => e.remove());
	// Copy current config values from the config form
	const configForm = document.getElementById(
		"plugin-config-form",
	) as HTMLFormElement | null;
	if (!configForm) return;
	const inputs = configForm.querySelectorAll<
		HTMLInputElement | HTMLSelectElement
	>("input[name], select[name]");
	inputs.forEach((el) => {
		const hidden = document.createElement("input");
		hidden.type = "hidden";
		hidden.name = `val_${el.name}`;
		hidden.value = el.value;
		form.appendChild(hidden);
	});
}
