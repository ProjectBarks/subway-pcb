import "../../lib/types";
import { initNav } from "../nav/nav";
import { initToastHandler } from "../toast/toast";
import "../toast/toast.css";
import { Board } from "../../lib/board";

// Expose Board globally for any page that needs it
window.Board = Board;

initToastHandler();
initNav();
