import type Alpine from "alpinejs";

interface ToastItem {
	id: number;
	message: string;
	type: string;
}

export function registerToastStore(alpine: typeof Alpine): void {
	alpine.store("toast", {
		items: [] as ToastItem[],
		show(message: string, type = "success") {
			const id = Date.now();
			(this as unknown as { items: ToastItem[] }).items.push({
				id,
				message,
				type,
			});
			setTimeout(() => {
				const self = this as unknown as { items: ToastItem[] };
				self.items = self.items.filter((t) => t.id !== id);
			}, 3500);
		},
	});
}
