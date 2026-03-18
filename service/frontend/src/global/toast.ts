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

export function initToastHandler(): void {
  document.body.addEventListener("htmx:afterSwap", ((
    evt: HtmxAfterSwapEvent,
  ) => {
    const toast = evt.detail.target.querySelector("[data-toast]");
    if (toast) {
      const container = document.getElementById("toast-container");
      if (container) {
        container.appendChild(toast);
        setTimeout(() => toast.remove(), 3500);
      }
    }
  }) as EventListener);

  document.body.addEventListener("showToast", ((evt: ShowToastEvent) => {
    const container = document.getElementById("toast-container");
    if (!container) return;
    const msg = evt.detail.message || "";
    const type = evt.detail.type || "success";
    const div = document.createElement("div");
    div.className =
      "glass border rounded-lg px-4 py-3 text-sm shadow-lg " +
      (type === "error"
        ? "border-red-500/50 text-red-400"
        : "border-green-500/50 text-green-400");
    div.style.animation = "toast-fade 3.5s ease forwards";
    div.textContent = msg;
    container.appendChild(div);
    setTimeout(() => div.remove(), 3500);
  }) as EventListener);
}
