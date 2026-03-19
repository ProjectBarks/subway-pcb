import "../types";
import { initNavBackdrop } from "../global/nav";
import { initToastHandler } from "../global/toast";
import { Board } from "../lib/board";

// Expose Board globally for any page that needs it
window.Board = Board;

initToastHandler();
initNavBackdrop();
