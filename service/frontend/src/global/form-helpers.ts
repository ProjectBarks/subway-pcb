export function updatePreviewColor(_input: HTMLInputElement): void {
  if (!window._previewRenderer) return;

  const colors: Record<string, string> = {};
  document
    .querySelectorAll<HTMLInputElement>("input[data-route-key]")
    .forEach((el) => {
      colors[el.dataset.routeKey!] = el.value;
    });

  window._previewRenderer.setThemeColors(colors);
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

export function collectConfigToThemeForm(form: HTMLFormElement): void {
  // Remove old val_ inputs
  form
    .querySelectorAll<HTMLInputElement>('input[name^="val_"]')
    .forEach((e) => e.remove());
  // Copy current config values from the config form
  const configForm = document.getElementById(
    "mode-config-form",
  ) as HTMLFormElement | null;
  if (!configForm) return;
  const inputs =
    configForm.querySelectorAll<HTMLInputElement | HTMLSelectElement>(
      "input[name], select[name]",
    );
  inputs.forEach((el) => {
    const hidden = document.createElement("input");
    hidden.type = "hidden";
    hidden.name = "val_" + el.name;
    hidden.value = el.value;
    form.appendChild(hidden);
  });
}
