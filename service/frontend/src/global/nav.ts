export function initNavBackdrop(): void {
  const sidebar = document.getElementById("sidebar");
  const backdrop = document.getElementById("nav-backdrop");
  if (sidebar && backdrop) {
    const observer = new MutationObserver(() => {
      const hidden = sidebar.classList.contains("-translate-x-full");
      backdrop.classList.toggle("hidden", hidden);
    });
    observer.observe(sidebar, {
      attributes: true,
      attributeFilter: ["class"],
    });
  }
}
