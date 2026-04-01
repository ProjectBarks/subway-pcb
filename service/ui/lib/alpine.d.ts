declare module "alpinejs" {
	interface Alpine {
		start(): void;
		plugin(plugin: unknown): void;
		// biome-ignore lint/suspicious/noExplicitAny: Alpine data callbacks use dynamic this
		data(name: string, callback: () => any): void;
		store(name: string, value: Record<string, unknown>): void;
		store(name: string): Record<string, unknown>;
	}
	const Alpine: Alpine;
	export default Alpine;
}

declare module "@alpinejs/intersect" {
	const plugin: unknown;
	export default plugin;
}
