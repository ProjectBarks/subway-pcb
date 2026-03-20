import "../types";
import { initNav } from "../global/nav";
import { initToastHandler } from "../global/toast";
import { Board } from "../lib/board";

// Expose Board globally for any page that needs it
window.Board = Board;

initToastHandler();
initNav();
