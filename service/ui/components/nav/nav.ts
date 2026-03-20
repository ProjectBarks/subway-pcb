export function initNav(): void {
	const menuToggle = document.getElementById("mobile-menu-toggle");
	const mobileMenu = document.getElementById("mobile-menu");
	const menuIconOpen = document.getElementById("menu-icon-open");
	const menuIconClose = document.getElementById("menu-icon-close");

	if (!menuToggle || !mobileMenu) return;

	menuToggle.addEventListener("click", () => {
		const isOpen = !mobileMenu.classList.contains("hidden");
		mobileMenu.classList.toggle("hidden", isOpen);

		// Toggle icons
		if (menuIconOpen && menuIconClose) {
			menuIconOpen.classList.toggle("hidden", !isOpen);
			menuIconClose.classList.toggle("hidden", isOpen);
		}
	});

	// Close mobile menu when clicking a nav link
	mobileMenu.querySelectorAll("a").forEach((link) => {
		link.addEventListener("click", () => {
			mobileMenu.classList.add("hidden");
			if (menuIconOpen && menuIconClose) {
				menuIconOpen.classList.remove("hidden");
				menuIconClose.classList.add("hidden");
			}
		});
	});
}
