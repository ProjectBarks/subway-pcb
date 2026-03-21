import {
	type BoardData,
	drawLeds,
	drawOffState,
	loadBoardData,
	setupCanvas,
} from "./led-preview";
import { LuaRunner, type StationState } from "./lua-runner";

export interface PreviewCleanup {
	destroy(): void;
}

interface CardState {
	el: HTMLElement;
	canvas: HTMLCanvasElement;
	boardUrl: string;
	luaSource: string;
	config: Record<string, string>;
	board: BoardData | null;
	rendered: boolean;
	enterHandler: () => void;
	leaveHandler: () => void;
}

let sharedRunner: LuaRunner | null = null;
let mtaState: StationState[] = [];
let mtaFetched = false;
let activeAnimCard: CardState | null = null;
let animFrameId = 0;

async function ensureRunner(): Promise<LuaRunner> {
	if (!sharedRunner) {
		sharedRunner = new LuaRunner();
		await sharedRunner.init();
	}
	return sharedRunner;
}

async function fetchMtaState(): Promise<StationState[]> {
	if (mtaFetched) return mtaState;
	mtaFetched = true;
	try {
		const resp = await fetch("/api/v1/state?format=json");
		if (resp.ok) {
			const data = await resp.json();
			mtaState = data.stations ?? [];
		}
	} catch {
		// keep empty array
	}
	return mtaState;
}

function stopAnimation(): void {
	if (animFrameId) {
		cancelAnimationFrame(animFrameId);
		animFrameId = 0;
	}
	activeAnimCard = null;
}

async function configureRunner(
	runner: LuaRunner,
	card: CardState,
): Promise<boolean> {
	if (!card.board) return false;

	runner.setLedMap(card.board.ledMap);
	runner.setStripSizes(card.board.strips);
	runner.setConfig(card.config);

	const stations = await fetchMtaState();
	runner.setMtaState(stations);

	if (!card.luaSource) return false;
	const err = await runner.loadScript(card.luaSource);
	return err === null;
}

async function renderOneFrame(
	runner: LuaRunner,
	card: CardState,
): Promise<void> {
	if (!card.board) return;
	const pixels = await runner.render();
	const { ctx, w, h } = setupCanvas(card.canvas);
	drawLeds(ctx, pixels, card.board, w, h);
	card.rendered = true;
	card.el.classList.add("preview-ready");
}

async function startAnimation(card: CardState): Promise<void> {
	stopAnimation();
	activeAnimCard = card;

	const runner = await ensureRunner();
	const ok = await configureRunner(runner, card);
	if (!ok || activeAnimCard !== card) return;

	const loop = async () => {
		if (activeAnimCard !== card) return;
		const pixels = await runner.render();
		if (activeAnimCard !== card) return;
		const { ctx, w, h } = setupCanvas(card.canvas);
		drawLeds(ctx, pixels, card.board!, w, h);
		animFrameId = requestAnimationFrame(loop);
	};
	animFrameId = requestAnimationFrame(loop);
}

async function renderInitialFrame(card: CardState): Promise<void> {
	try {
		card.board = await loadBoardData(card.boardUrl);
	} catch {
		return;
	}

	if (!card.luaSource) {
		drawOffState(card.canvas, card.board);
		card.rendered = true;
		card.el.classList.add("preview-ready");
		return;
	}

	try {
		const runner = await ensureRunner();
		const ok = await configureRunner(runner, card);
		if (ok) {
			await renderOneFrame(runner, card);
		} else {
			drawOffState(card.canvas, card.board);
			card.rendered = true;
			card.el.classList.add("preview-ready");
		}
	} catch {
		drawOffState(card.canvas, card.board);
		card.rendered = true;
		card.el.classList.add("preview-ready");
	}
}

function parseConfig(raw: string | null | undefined): Record<string, string> {
	if (!raw) return {};
	try {
		return JSON.parse(raw);
	} catch {
		return {};
	}
}

export async function initPreviews(
	selector = "[data-preview-card]",
): Promise<PreviewCleanup> {
	const elements = document.querySelectorAll<HTMLElement>(selector);
	const cards: CardState[] = [];

	for (const el of elements) {
		const canvas = el.querySelector<HTMLCanvasElement>("canvas");
		if (!canvas) continue;

		const boardUrl =
			el.dataset.boardUrl || "/static/dist/boards/nyc-subway/v1/board.json";
		const luaSource = el.dataset.luaSource || "";
		const config = parseConfig(el.dataset.config);

		const card: CardState = {
			el,
			canvas,
			boardUrl,
			luaSource,
			config,
			board: null,
			rendered: false,
			enterHandler: () => {},
			leaveHandler: () => {},
		};

		card.enterHandler = () => {
			startAnimation(card);
		};
		card.leaveHandler = () => {
			if (activeAnimCard === card) {
				stopAnimation();
			}
		};

		el.addEventListener("mouseenter", card.enterHandler);
		el.addEventListener("mouseleave", card.leaveHandler);
		cards.push(card);
	}

	// Render initial frames using IntersectionObserver for deferred loading
	const pending = new Set(cards);

	const renderVisible = async (visibleCards: CardState[]) => {
		for (const card of visibleCards) {
			if (card.rendered) continue;
			pending.delete(card);
			await renderInitialFrame(card);
		}
	};

	if (typeof IntersectionObserver !== "undefined") {
		const observer = new IntersectionObserver(
			(entries) => {
				const visible = entries
					.filter((e) => e.isIntersecting)
					.map((e) => cards.find((c) => c.el === e.target))
					.filter((c): c is CardState => c != null && !c.rendered);

				if (visible.length > 0) {
					renderVisible(visible);
				}
			},
			{ rootMargin: "200px" },
		);

		for (const card of cards) {
			observer.observe(card.el);
		}

		return {
			destroy() {
				stopAnimation();
				observer.disconnect();
				for (const card of cards) {
					card.el.removeEventListener("mouseenter", card.enterHandler);
					card.el.removeEventListener("mouseleave", card.leaveHandler);
					card.el.classList.remove("preview-ready");
				}
			},
		};
	}

	// Fallback: render all cards sequentially
	await renderVisible(cards);

	return {
		destroy() {
			stopAnimation();
			for (const card of cards) {
				card.el.removeEventListener("mouseenter", card.enterHandler);
				card.el.removeEventListener("mouseleave", card.leaveHandler);
				card.el.classList.remove("preview-ready");
			}
		},
	};
}
