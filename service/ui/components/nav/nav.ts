import type Alpine from "alpinejs";

export function registerNav(alpine: typeof Alpine): void {
	alpine.data("mobileNav", () => ({
		open: false,
		toggle() {
			this.open = !this.open;
		},
		close() {
			this.open = false;
		},
	}));
}
