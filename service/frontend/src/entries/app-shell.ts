import "../types";
import { Board } from "../lib/board";
import { initToastHandler } from "../global/toast";
import { initNavBackdrop } from "../global/nav";

// Expose Board globally for any page that needs it
window.Board = Board;

initToastHandler();
initNavBackdrop();
