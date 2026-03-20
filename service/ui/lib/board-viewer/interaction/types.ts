export interface InteractionMode {
	init(): void;
	update(dt: number): void;
	dispose(): void;
}
