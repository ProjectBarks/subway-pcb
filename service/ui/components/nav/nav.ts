export function initNav(): void {
	const menuToggle = document.getElementById("mobile-menu-toggle");
	const mobileMenu = document.getElementById("mobile-menu");
	const menuIconOpen = document.getElementById("menu-icon-open");
	const menuIconClose = document.getElementById("menu-icon-close");

	if (!menuToggle || !mobileMenu) return;

	let isOpen = false;

	function open(): void {
		isOpen = true;
		mobileMenu!.style.maxHeight = `${mobileMenu!.scrollHeight}px`;
		mobileMenu!.style.opacity = "1";
		menuIconOpen?.classList.add("hidden");
		menuIconClose?.classList.remove("hidden");
	}

	function close(): void {
		isOpen = false;
		mobileMenu!.style.maxHeight = "0";
		mobileMenu!.style.opacity = "0";
		menuIconOpen?.classList.remove("hidden");
		menuIconClose?.classList.add("hidden");
	}

	menuToggle.addEventListener("click", () => {
		if (isOpen) {
			close();
		} else {
			open();
		}
	});

	// Close mobile menu when clicking a nav link
	mobileMenu.querySelectorAll("a").forEach((link) => {
		link.addEventListener("click", close);
	});
}
