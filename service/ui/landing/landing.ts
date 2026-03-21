import { initBoardViewer } from "../lib/board-viewer";
import "./landing.css";

// --- Scroll-triggered fade-up animations ---
function initScrollAnimations(): void {
	const elements = document.querySelectorAll<HTMLElement>(
		'[data-animate="fade-up"]',
	);

	const observer = new IntersectionObserver(
		(entries) => {
			for (const entry of entries) {
				if (entry.isIntersecting) {
					(entry.target as HTMLElement).classList.add("animate-visible");
					observer.unobserve(entry.target);
				}
			}
		},
		{ threshold: 0.3 },
	);

	for (const el of elements) {
		el.classList.add("animate-on-scroll");
		observer.observe(el);
	}
}

// --- Smooth scroll for "Learn More" anchor ---
function initSmoothScroll(): void {
	const link = document.querySelector<HTMLAnchorElement>(
		'a[href="#how-it-works"]',
	);
	if (!link) return;

	link.addEventListener("click", (e) => {
		e.preventDefault();
		document
			.getElementById("how-it-works")
			?.scrollIntoView({ behavior: "smooth" });
	});
}

// --- 3D hero board with live LED data ---
async function initHero(): Promise<void> {
	const container = document.getElementById("hero-board");
	if (!container) return;

	const isMobile = window.innerWidth < 768;
	await initBoardViewer(container, {
		boardUrl: "/static/dist/boards/nyc-subway/v1/board.json",
		mode: "hero",
		pixelsUrl: "/api/v1/pixels",
		camera: { distance: isMobile ? 220 : 300 },
	});
}

// --- Init ---
initScrollAnimations();
initSmoothScroll();
initHero();
