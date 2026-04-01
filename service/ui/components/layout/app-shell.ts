import "htmx.org";
import intersect from "@alpinejs/intersect";
import Alpine from "alpinejs";
import { registerNav } from "../nav/nav";
import { registerToastStore } from "../toast/toast";

Alpine.plugin(intersect);
registerToastStore(Alpine);
registerNav(Alpine);

// Start after all deferred module scripts have registered their components.
document.addEventListener("DOMContentLoaded", () => Alpine.start());
