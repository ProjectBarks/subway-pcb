interface HtmxAfterSwapEvent extends Event {
	detail: {
		target: HTMLElement;
	};
}

interface ShowToastEvent extends Event {
	detail: {
		message?: string;
		type?: string;
	};
}

const TOAST_DURATION = 3500;
const EXIT_ANIMATION_DURATION = 300;

function dismissToast(toast: HTMLElement): void {
	toast.classList.remove("animate-toast-in");
	toast.classList.add("animate-toast-out");
	setTimeout(() => toast.remove(), EXIT_ANIMATION_DURATION);
}

function createToastElement(message: string, type: string): HTMLDivElement {
	const div = document.createElement("div");
	div.className =
		"alert shadow-lg animate-toast-in" +
		(type === "error" ? " alert-destructive" : "");
	div.textContent = message;
	return div;
}

export function initToastHandler(): void {
	document.body.addEventListener("htmx:afterSwap", ((
		evt: HtmxAfterSwapEvent,
	) => {
		const toast = evt.detail.target.querySelector(
			"[data-toast]",
		) as HTMLElement | null;
		if (toast) {
			const container = document.getElementById("toast-container");
			if (container) {
				toast.classList.add("animate-toast-in");
				container.appendChild(toast);
				setTimeout(() => dismissToast(toast), TOAST_DURATION);
			}
		}
	}) as EventListener);

	document.body.addEventListener("showToast", ((evt: ShowToastEvent) => {
		const container = document.getElementById("toast-container");
		if (!container) return;
		const msg = evt.detail.message || "";
		const type = evt.detail.type || "success";
		const div = createToastElement(msg, type);
		container.appendChild(div);
		setTimeout(() => dismissToast(div), TOAST_DURATION);
	}) as EventListener);
}
