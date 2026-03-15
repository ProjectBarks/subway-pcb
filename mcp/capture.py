#!/usr/bin/env python3
# /// script
# requires-python = ">=3.10"
# dependencies = [
#     "opencv-python>=4.9",
#     "numpy>=1.26",
#     "mcp[cli]>=1.0",
# ]
# ///
"""
Capture and enhance PCB LED camera views.

Setup (run once):
    uv run capture.py setup              # Interactive: pick camera, click 4 corners
    uv run capture.py setup --camera 1   # Use specific camera

Capture (requires setup first):
    uv run capture.py                    # Capture, warp, enhance → saves to captures/
    uv run capture.py -o out.png         # Save to specific path
    uv run capture.py --show             # Preview result
    uv run capture.py --raw              # Also save the raw frame
    uv run capture.py --debug            # Save debug overlay
"""

import argparse
import json
import sys
import time
from pathlib import Path

import cv2
import numpy as np

# Physical PCB dimensions: 24.5cm wide x 29.5cm tall
PCB_WIDTH_CM = 24.5
PCB_HEIGHT_CM = 29.5
PX_PER_CM = 40
OUT_W = int(PCB_WIDTH_CM * PX_PER_CM)   # 980
OUT_H = int(PCB_HEIGHT_CM * PX_PER_CM)  # 1180

CONFIG_PATH = Path(__file__).parent / "config.json"


# ─── Config ──────────────────────────────────────────────────────────────────

def load_config() -> dict | None:
    if CONFIG_PATH.exists():
        with open(CONFIG_PATH) as f:
            return json.load(f)
    return None


def save_config(camera: int, corners: list[list[float]]):
    data = {"camera": camera, "corners": corners}
    with open(CONFIG_PATH, "w") as f:
        json.dump(data, f, indent=2)
    print(f"Config saved to {CONFIG_PATH}")


# ─── Setup: interactive corner selection ─────────────────────────────────────

def _scan_cameras() -> list[tuple[int, np.ndarray]]:
    """Scan cameras 0-7, return list of (index, preview_frame) for working ones."""
    found = []
    for i in range(8):
        cap = cv2.VideoCapture(i)
        if cap.isOpened():
            # Grab a few frames to let it warm up
            for _ in range(5):
                cap.read()
            ret, frame = cap.read()
            if ret and frame is not None:
                found.append((i, frame))
            cap.release()
    return found


def _camera_picker_ui(cameras: list[tuple[int, np.ndarray]]) -> int | None:
    """Show a visual camera picker grid. Returns selected camera index or None."""
    if not cameras:
        return None
    if len(cameras) == 1:
        return cameras[0][0]

    BG = (30, 30, 30)
    ACCENT = (0, 200, 255)
    THUMB_W, THUMB_H = 320, 180
    PAD = 15
    cols = min(len(cameras), 3)
    rows = (len(cameras) + cols - 1) // cols

    canvas_w = PAD + cols * (THUMB_W + PAD)
    canvas_h = 50 + PAD + rows * (THUMB_H + 40 + PAD)
    canvas = np.full((canvas_h, canvas_w, 3), BG, dtype=np.uint8)

    # Header
    cv2.putText(canvas, "Select Camera", (PAD, 35),
                cv2.FONT_HERSHEY_SIMPLEX, 0.8, ACCENT, 2, cv2.LINE_AA)

    # Draw thumbnails
    thumb_rects = []  # (x1,y1,x2,y2, cam_index)
    for idx, (cam_id, frame) in enumerate(cameras):
        r, c = divmod(idx, cols)
        x = PAD + c * (THUMB_W + PAD)
        y = 50 + PAD + r * (THUMB_H + 40 + PAD)
        thumb = cv2.resize(frame, (THUMB_W, THUMB_H))
        canvas[y:y + THUMB_H, x:x + THUMB_W] = thumb
        cv2.rectangle(canvas, (x - 1, y - 1), (x + THUMB_W, y + THUMB_H), (80, 80, 80), 1)
        label = f"Camera {cam_id}"
        cv2.putText(canvas, label, (x + 5, y + THUMB_H + 22),
                    cv2.FONT_HERSHEY_SIMPLEX, 0.55, (200, 200, 200), 1, cv2.LINE_AA)
        cv2.putText(canvas, f"Press [{cam_id}]", (x + THUMB_W - 85, y + THUMB_H + 22),
                    cv2.FONT_HERSHEY_SIMPLEX, 0.4, (120, 120, 120), 1, cv2.LINE_AA)
        thumb_rects.append((x, y, x + THUMB_W, y + THUMB_H, cam_id))

    selected = [None]

    def on_click(event, x, y, flags, param):
        if event == cv2.EVENT_LBUTTONDOWN:
            for x1, y1, x2, y2, cam_id in thumb_rects:
                if x1 <= x <= x2 and y1 <= y <= y2:
                    selected[0] = cam_id

    cv2.namedWindow("Select Camera", cv2.WINDOW_AUTOSIZE)
    cv2.setMouseCallback("Select Camera", on_click)
    cv2.imshow("Select Camera", canvas)

    valid_keys = {ord(str(cam_id)): cam_id for cam_id, _ in cameras}

    while True:
        key = cv2.waitKey(50) & 0xFF
        if key in valid_keys:
            selected[0] = valid_keys[key]
        if key == ord("q") or key == 27:
            break
        if selected[0] is not None:
            break

    cv2.destroyAllWindows()
    cv2.waitKey(100)  # let macOS flush the window
    return selected[0]


def run_setup(camera_index: int | None):
    """Interactive corner setup with live preview and zoom loupe."""
    if camera_index is None:
        print("Scanning cameras...")
        cameras = _scan_cameras()
        if not cameras:
            sys.exit("No cameras found")
        camera_index = _camera_picker_ui(cameras)
        if camera_index is None:
            sys.exit("No camera selected")
        print(f"Selected camera {camera_index}")

    cap = cv2.VideoCapture(camera_index)
    if not cap.isOpened():
        sys.exit(f"Error: Could not open camera {camera_index}")

    for _ in range(10):
        cap.read()

    corners = []
    LABELS = ["TL", "TR", "BR", "BL"]
    LABEL_FULL = ["Top-Left", "Top-Right", "Bottom-Right", "Bottom-Left"]
    frozen_frame = None
    mouse_x, mouse_y = 0, 0

    # Layout
    FRAME_PAD = 40    # padding around the camera frame so edge corners are clickable
    MAIN_W = 800      # width of the frame area (without padding)
    LOUPE_W = 400
    ZOOM = 6
    LOUPE_SRC = LOUPE_W // (2 * ZOOM)

    # Colors
    BG_DARK = (30, 30, 30)
    ACCENT = (0, 200, 255)     # warm yellow
    CORNER_COLORS = [
        (80, 80, 255),    # TL - red
        (80, 255, 80),    # TR - green
        (255, 80, 80),    # BR - blue
        (0, 220, 220),    # BL - yellow
    ]

    scale = [1.0]
    main_h = [0]

    def on_mouse(event, x, y, flags, param):
        nonlocal corners, frozen_frame, mouse_x, mouse_y
        # Ignore clicks on the sidebar (loupe/info panel)
        if x > MAIN_W + 2 * FRAME_PAD:
            return
        # Map display coords (with padding) back to original frame coords
        mouse_x = int((x - FRAME_PAD) / scale[0])
        mouse_y = int((y - FRAME_PAD) / scale[0])
        # Clamp to frame bounds (will be set properly once we read a frame)
        mouse_x = max(0, mouse_x)
        mouse_y = max(0, mouse_y)
        if event == cv2.EVENT_LBUTTONDOWN and len(corners) < 4:
            corners.append([mouse_x, mouse_y])
            print(f"  {LABELS[len(corners)-1]}: ({mouse_x}, {mouse_y})")
            if len(corners) == 1:
                _, frozen_frame = cap.read()

    cv2.namedWindow("PCB Corner Setup", cv2.WINDOW_AUTOSIZE)
    cv2.setMouseCallback("PCB Corner Setup", on_mouse)

    print("\n  PCB Corner Setup")
    print("  Click 4 corners: TL -> TR -> BR -> BL")
    print("  [R] Reset  [S] Save  [Q] Quit\n")

    while True:
        source = frozen_frame if frozen_frame is not None else None
        if source is None:
            ret, frame = cap.read()
            if not ret:
                continue
            source = frame

        fh, fw = source.shape[:2]
        scale[0] = MAIN_W / fw
        frame_h = int(fh * scale[0])
        main_h[0] = frame_h + 2 * FRAME_PAD

        # ── Main view: frame centered with dark padding around it ──
        scaled_frame = cv2.resize(source, (MAIN_W, frame_h))
        main_view = np.full((main_h[0], MAIN_W + 2 * FRAME_PAD, 3), BG_DARK, dtype=np.uint8)
        main_view[FRAME_PAD:FRAME_PAD + frame_h, FRAME_PAD:FRAME_PAD + MAIN_W] = scaled_frame

        # Header bar at very top (in padding area)
        if len(corners) < 4:
            txt = f"Click {LABEL_FULL[len(corners)]} corner ({len(corners)+1}/4)"
            cv2.putText(main_view, txt, (10, 25),
                        cv2.FONT_HERSHEY_SIMPLEX, 0.6, ACCENT, 1, cv2.LINE_AA)
        else:
            cv2.putText(main_view, "All corners set", (10, 25),
                        cv2.FONT_HERSHEY_SIMPLEX, 0.6, (80, 255, 80), 1, cv2.LINE_AA)
            mw = MAIN_W + 2 * FRAME_PAD
            cv2.putText(main_view, "[S] Save  [R] Reset", (mw - 250, 25),
                        cv2.FONT_HERSHEY_SIMPLEX, 0.5, (180, 180, 180), 1, cv2.LINE_AA)

        # Crosshair on main view (offset by padding)
        sx = int(mouse_x * scale[0]) + FRAME_PAD
        sy = int(mouse_y * scale[0]) + FRAME_PAD
        cv2.line(main_view, (sx - 20, sy), (sx - 5, sy), ACCENT, 1, cv2.LINE_AA)
        cv2.line(main_view, (sx + 5, sy), (sx + 20, sy), ACCENT, 1, cv2.LINE_AA)
        cv2.line(main_view, (sx, sy - 20), (sx, sy - 5), ACCENT, 1, cv2.LINE_AA)
        cv2.line(main_view, (sx, sy + 5), (sx, sy + 20), ACCENT, 1, cv2.LINE_AA)

        # Draw placed corners with filled circles + labels + connecting lines
        def to_display(cx, cy):
            return (int(cx * scale[0]) + FRAME_PAD, int(cy * scale[0]) + FRAME_PAD)

        if len(corners) >= 2:
            pts_draw = [to_display(*p) for p in corners]
            for i in range(len(pts_draw)):
                j = (i + 1) % len(pts_draw) if len(corners) == 4 else i + 1
                if j < len(pts_draw):
                    cv2.line(main_view, pts_draw[i], pts_draw[j], (0, 200, 0), 2, cv2.LINE_AA)
            if len(corners) == 4:
                cv2.line(main_view, pts_draw[3], pts_draw[0], (0, 200, 0), 2, cv2.LINE_AA)
                poly = np.array(pts_draw, dtype=np.int32).reshape((-1, 1, 2))
                fill_overlay = main_view.copy()
                cv2.fillPoly(fill_overlay, [poly], (0, 180, 0))
                main_view[:] = cv2.addWeighted(fill_overlay, 0.1, main_view, 0.9, 0)

        for i, (cx, cy) in enumerate(corners):
            dx, dy = to_display(cx, cy)
            cv2.circle(main_view, (dx, dy), 8, CORNER_COLORS[i], -1, cv2.LINE_AA)
            cv2.circle(main_view, (dx, dy), 8, (255, 255, 255), 1, cv2.LINE_AA)
            cv2.putText(main_view, LABELS[i], (dx + 12, dy + 5),
                        cv2.FONT_HERSHEY_SIMPLEX, 0.5, (255, 255, 255), 1, cv2.LINE_AA)

        # ── Loupe panel ──
        # Build loupe: always a fixed-size crop centered on cursor,
        # padded with black when near frame edges so it never stretches
        full_crop = np.zeros((LOUPE_SRC * 2, LOUPE_SRC * 2, 3), dtype=np.uint8)
        # Where in the source frame to read from
        src_x1 = max(0, mouse_x - LOUPE_SRC)
        src_y1 = max(0, mouse_y - LOUPE_SRC)
        src_x2 = min(fw, mouse_x + LOUPE_SRC)
        src_y2 = min(fh, mouse_y + LOUPE_SRC)
        # Where in the padded crop to paste
        dst_x1 = src_x1 - (mouse_x - LOUPE_SRC)
        dst_y1 = src_y1 - (mouse_y - LOUPE_SRC)
        dst_x2 = dst_x1 + (src_x2 - src_x1)
        dst_y2 = dst_y1 + (src_y2 - src_y1)
        if src_x2 > src_x1 and src_y2 > src_y1:
            full_crop[dst_y1:dst_y2, dst_x1:dst_x2] = source[src_y1:src_y2, src_x1:src_x2]
        loupe = cv2.resize(full_crop, (LOUPE_W, LOUPE_W), interpolation=cv2.INTER_NEAREST)

        # Crosshair in loupe — gap in center so you can see the pixel
        lc = LOUPE_W // 2
        gap = 8
        cv2.line(loupe, (lc - 30, lc), (lc - gap, lc), ACCENT, 1, cv2.LINE_AA)
        cv2.line(loupe, (lc + gap, lc), (lc + 30, lc), ACCENT, 1, cv2.LINE_AA)
        cv2.line(loupe, (lc, lc - 30), (lc, lc - gap), ACCENT, 1, cv2.LINE_AA)
        cv2.line(loupe, (lc, lc + gap), (lc, lc + 30), ACCENT, 1, cv2.LINE_AA)
        # Thin border around loupe
        cv2.rectangle(loupe, (0, 0), (LOUPE_W - 1, LOUPE_W - 1), (60, 60, 60), 1)

        # Coordinate badge at bottom of loupe
        badge_h = 28
        badge_bg = np.full_like(loupe[-badge_h:, :], BG_DARK)
        loupe[-badge_h:, :] = cv2.addWeighted(badge_bg, 0.8, loupe[-badge_h:, :], 0.2, 0)
        cv2.putText(loupe, f"({mouse_x}, {mouse_y})", (8, LOUPE_W - 8),
                    cv2.FONT_HERSHEY_SIMPLEX, 0.55, ACCENT, 1, cv2.LINE_AA)
        cv2.putText(loupe, f"{ZOOM}x", (LOUPE_W - 40, LOUPE_W - 8),
                    cv2.FONT_HERSHEY_SIMPLEX, 0.45, (120, 120, 120), 1, cv2.LINE_AA)

        # If all 4 corners placed, show a live warp preview in the loupe area
        if len(corners) == 4 and frozen_frame is not None:
            src_pts = order_points(np.array(corners, dtype=np.float32))
            preview = perspective_warp(frozen_frame, src_pts)
            preview = enhance_contrast(preview)
            # Scale preview to fit loupe panel
            ph = int(LOUPE_W * preview.shape[0] / preview.shape[1])
            loupe = cv2.resize(preview, (LOUPE_W, ph))
            # Add label
            cv2.putText(loupe, "Preview", (8, 20),
                        cv2.FONT_HERSHEY_SIMPLEX, 0.5, (0, 200, 255), 1, cv2.LINE_AA)

        # Build sidebar: loupe/preview on top, info panel below
        info_h = max(0, main_h[0] - loupe.shape[0])
        if info_h > 0:
            info_panel = np.full((info_h, LOUPE_W, 3), BG_DARK, dtype=np.uint8)
            # Corner status list
            y_off = 30
            cv2.putText(info_panel, "Corners", (10, y_off),
                        cv2.FONT_HERSHEY_SIMPLEX, 0.55, (200, 200, 200), 1, cv2.LINE_AA)
            y_off += 8
            cv2.line(info_panel, (10, y_off), (LOUPE_W - 10, y_off), (60, 60, 60), 1)
            y_off += 25
            for i in range(4):
                color = CORNER_COLORS[i] if i < len(corners) else (80, 80, 80)
                cv2.circle(info_panel, (20, y_off - 5), 5, color, -1, cv2.LINE_AA)
                if i < len(corners):
                    txt = f"{LABELS[i]}  ({corners[i][0]}, {corners[i][1]})"
                    cv2.putText(info_panel, txt, (35, y_off),
                                cv2.FONT_HERSHEY_SIMPLEX, 0.45, (220, 220, 220), 1, cv2.LINE_AA)
                else:
                    cv2.putText(info_panel, f"{LABELS[i]}  ---", (35, y_off),
                                cv2.FONT_HERSHEY_SIMPLEX, 0.45, (80, 80, 80), 1, cv2.LINE_AA)
                y_off += 28

            # Keybindings
            y_off += 15
            cv2.line(info_panel, (10, y_off), (LOUPE_W - 10, y_off), (60, 60, 60), 1)
            y_off += 25
            for key_txt, desc in [("[R]", "Reset"), ("[S]", "Save & quit"), ("[Q]", "Quit")]:
                cv2.putText(info_panel, key_txt, (15, y_off),
                            cv2.FONT_HERSHEY_SIMPLEX, 0.45, ACCENT, 1, cv2.LINE_AA)
                cv2.putText(info_panel, desc, (55, y_off),
                            cv2.FONT_HERSHEY_SIMPLEX, 0.45, (160, 160, 160), 1, cv2.LINE_AA)
                y_off += 24

            sidebar = np.vstack([loupe, info_panel])
        else:
            sidebar = loupe[:main_h[0]]

        # Ensure sidebar matches main view height exactly
        if sidebar.shape[0] < main_h[0]:
            pad_bottom = np.full((main_h[0] - sidebar.shape[0], LOUPE_W, 3), BG_DARK, dtype=np.uint8)
            sidebar = np.vstack([sidebar, pad_bottom])
        elif sidebar.shape[0] > main_h[0]:
            sidebar = sidebar[:main_h[0]]

        sep = np.full((main_h[0], 2, 3), (50, 50, 50), dtype=np.uint8)
        composite = np.hstack([main_view, sep, sidebar])

        cv2.imshow("PCB Corner Setup", composite)
        key = cv2.waitKey(30) & 0xFF

        if key == ord("r"):
            corners = []
            frozen_frame = None
            print("  Reset")
        elif key == ord("s") and len(corners) == 4:
            save_config(camera_index, corners)
            break
        elif key == ord("q") or key == 27:
            print("  Quit without saving")
            break

    cap.release()
    cv2.destroyAllWindows()


# ─── Corner refinement ───────────────────────────────────────────────────────

def order_points(pts: np.ndarray) -> np.ndarray:
    """Order 4 points as: top-left, top-right, bottom-right, bottom-left."""
    rect = np.zeros((4, 2), dtype=np.float32)
    s = pts.sum(axis=1)
    rect[0] = pts[np.argmin(s)]
    rect[2] = pts[np.argmax(s)]
    d = np.diff(pts, axis=1)
    rect[1] = pts[np.argmin(d)]
    rect[3] = pts[np.argmax(d)]
    return rect


def refine_corners(frame: np.ndarray, saved_corners: np.ndarray,
                   search_radius: int = 40) -> np.ndarray:
    """
    Refine saved corners using local edge detection near each corner.

    For each saved corner, crops a local region and looks for the actual
    PCB corner using Canny + goodFeaturesToTrack. Falls back to the saved
    position if refinement fails.
    """
    gray = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)
    h, w = gray.shape
    ordered = order_points(saved_corners)
    refined = ordered.copy()

    for i, (cx, cy) in enumerate(ordered):
        ci, cj = int(cx), int(cy)
        r = search_radius
        x1, y1 = max(0, ci - r), max(0, cj - r)
        x2, y2 = min(w, ci + r), min(h, cj + r)

        roi = gray[y1:y2, x1:x2]
        roi_blur = cv2.GaussianBlur(roi, (5, 5), 0)

        # Try multiple methods and collect candidates
        candidates = []

        # Method A: Canny + goodFeaturesToTrack (Harris)
        edges = cv2.Canny(roi_blur, 30, 100)
        edges = cv2.dilate(edges, None, iterations=1)
        for ql in [0.01, 0.05, 0.1]:
            feats = cv2.goodFeaturesToTrack(edges, 10, ql, 5,
                                           blockSize=5, useHarrisDetector=True)
            if feats is not None:
                pts = feats.reshape(-1, 2) + np.array([x1, y1], dtype=np.float32)
                dists = np.linalg.norm(pts - np.array([cx, cy]), axis=1)
                best_idx = np.argmin(dists)
                if dists[best_idx] < search_radius:
                    candidates.append(pts[best_idx])

        # Method B: Otsu threshold + contour vertex
        _, th = cv2.threshold(roi_blur, 0, 255, cv2.THRESH_BINARY_INV | cv2.THRESH_OTSU)
        contours, _ = cv2.findContours(th, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE)
        if contours:
            lc = max(contours, key=cv2.contourArea)
            peri = cv2.arcLength(lc, True)
            for eps in [0.003, 0.005, 0.008]:
                approx = cv2.approxPolyDP(lc, eps * peri, True)
                pts = approx.reshape(-1, 2).astype(np.float32) + np.array([x1, y1], dtype=np.float32)
                dists = np.linalg.norm(pts - np.array([cx, cy]), axis=1)
                best_idx = np.argmin(dists)
                if dists[best_idx] < search_radius:
                    candidates.append(pts[best_idx])

        if candidates:
            # Pick the candidate closest to the saved corner (small drift)
            all_pts = np.array(candidates)
            dists = np.linalg.norm(all_pts - np.array([cx, cy]), axis=1)
            refined[i] = all_pts[np.argmin(dists)]

    return refined


# ─── Warp & enhance ──────────────────────────────────────────────────────────

def perspective_warp(frame: np.ndarray, corners: np.ndarray) -> np.ndarray:
    """Warp the corner quad to a flat 1150x975 rectangle."""
    dst = np.array([
        [0, 0],
        [OUT_W - 1, 0],
        [OUT_W - 1, OUT_H - 1],
        [0, OUT_H - 1],
    ], dtype=np.float32)
    M = cv2.getPerspectiveTransform(order_points(corners), dst)
    return cv2.warpPerspective(frame, M, (OUT_W, OUT_H))


def enhance_contrast(image: np.ndarray) -> np.ndarray:
    """Push black background to true black, white traces/LEDs to bright white."""
    img = image.astype(np.float32)

    for c in range(3):
        ch = img[:, :, c]
        lo = np.percentile(ch, 2)
        hi = np.percentile(ch, 99)
        if hi - lo < 1:
            continue
        img[:, :, c] = np.clip((ch - lo) / (hi - lo) * 255.0, 0, 255)

    img = img.astype(np.uint8)

    # S-curve to crush blacks and boost whites
    x = np.linspace(0, 1, 256)
    lut = np.clip((np.tanh((x - 0.5) * 5) * 0.5 + 0.5) * 255, 0, 255).astype(np.uint8)
    img = cv2.LUT(img, lut)

    # CLAHE on lightness
    lab = cv2.cvtColor(img, cv2.COLOR_BGR2LAB)
    l, a, b = cv2.split(lab)
    l = cv2.createCLAHE(clipLimit=3.0, tileGridSize=(8, 8)).apply(l)
    img = cv2.cvtColor(cv2.merge([l, a, b]), cv2.COLOR_LAB2BGR)

    return img


# ─── Capture frame ───────────────────────────────────────────────────────────

def capture_frame(camera_index: int = 0) -> np.ndarray:
    cap = cv2.VideoCapture(camera_index)
    if not cap.isOpened():
        sys.exit(f"Error: Could not open camera {camera_index}")
    for _ in range(10):
        cap.read()
    time.sleep(0.3)
    ret, frame = cap.read()
    cap.release()
    if not ret or frame is None:
        sys.exit("Error: Failed to capture frame")
    return frame


# ─── Main ────────────────────────────────────────────────────────────────────

def main():
    parser = argparse.ArgumentParser(description="PCB LED camera capture")
    sub = parser.add_subparsers(dest="command")

    # Setup subcommand
    setup_p = sub.add_parser("setup", help="Interactive corner setup")
    setup_p.add_argument("--camera", type=int, default=None, help="Camera index (auto-detect if omitted)")

    # Capture is the default (no subcommand)
    parser.add_argument("--camera", type=int, default=None, help="Override camera index")
    parser.add_argument("--output", "-o", type=str, default=None)
    parser.add_argument("--show", action="store_true")
    parser.add_argument("--raw", action="store_true")
    parser.add_argument("--debug", action="store_true")
    parser.add_argument("--refine", action="store_true", help="Refine saved corners (default: use exact saved corners)")
    parser.add_argument("--format", choices=["normal", "sobel"], default="normal", help="Output format (default: normal)")
    parser.add_argument("--quality", type=int, default=95, help="JPEG quality (default: 95)")

    args = parser.parse_args()

    if args.command == "setup":
        run_setup(args.camera)
        return

    # ── Capture mode ──
    config = load_config()
    if config is None:
        sys.exit("Error: No setup config found. Run 'uv run capture.py setup' first.")

    camera = args.camera if args.camera is not None else config["camera"]
    saved_corners = np.array(config["corners"], dtype=np.float32)

    # Determine output path
    if args.output:
        output_path = Path(args.output)
    else:
        output_dir = Path(__file__).parent / "captures"
        output_dir.mkdir(exist_ok=True)
        output_path = output_dir / f"capture_{time.strftime('%Y%m%d_%H%M%S')}.jpg"

    print(f"Capturing from camera {camera}...")
    frame = capture_frame(camera)
    print(f"Captured {frame.shape[1]}x{frame.shape[0]} frame")

    if args.raw:
        raw_path = output_path.with_stem(output_path.stem + "_raw")
        cv2.imwrite(str(raw_path), frame)
        print(f"Raw frame saved to {raw_path}")

    # Refine corners from saved positions
    if args.refine:
        corners = refine_corners(frame, saved_corners)
        print("Refined corners from saved positions")
    else:
        corners = order_points(saved_corners)
        print("Using saved corners")

    labels = ["TL", "TR", "BR", "BL"]
    for label, pt in zip(labels, corners):
        print(f"  {label}: ({pt[0]:.0f}, {pt[1]:.0f})")

    if args.debug:
        debug = frame.copy()
        # Draw saved corners in yellow (small)
        for sx, sy in saved_corners:
            cv2.circle(debug, (int(sx), int(sy)), 6, (0, 255, 255), 1)
        # Draw refined corners in color (large)
        for i, (label, pt) in enumerate(zip(labels, corners)):
            color = [(0, 0, 255), (0, 255, 0), (255, 0, 0), (0, 255, 255)][i]
            cv2.circle(debug, (int(pt[0]), int(pt[1])), 10, color, -1)
            cv2.putText(debug, label, (int(pt[0]) + 14, int(pt[1]) + 5),
                        cv2.FONT_HERSHEY_SIMPLEX, 0.7, color, 2)
        pts_arr = corners.astype(np.int32).reshape((-1, 1, 2))
        cv2.polylines(debug, [pts_arr], True, (0, 255, 0), 2)
        debug_path = output_path.with_stem(output_path.stem + "_debug")
        cv2.imwrite(str(debug_path), debug)
        print(f"Debug image saved to {debug_path}")

    warped = perspective_warp(frame, corners)
    print(f"Warped to {warped.shape[1]}x{warped.shape[0]} ({PCB_WIDTH_CM}cm x {PCB_HEIGHT_CM}cm)")

    write_params = [cv2.IMWRITE_JPEG_QUALITY, args.quality]

    if args.format == "sobel":
        # Per-channel Sobel to preserve color
        channels = cv2.split(warped)
        sobel_channels = []
        for ch in channels:
            sx = cv2.Sobel(ch, cv2.CV_64F, 1, 0, ksize=3)
            sy = cv2.Sobel(ch, cv2.CV_64F, 0, 1, ksize=3)
            mag = np.sqrt(sx ** 2 + sy ** 2)
            sobel_channels.append(mag)
        sobel_color = cv2.merge(sobel_channels)
        mx = sobel_color.max()
        if mx > 0:
            sobel_color = sobel_color / mx * 255
        result = np.clip(sobel_color, 0, 255).astype(np.uint8)
    else:
        result = enhance_contrast(warped)

    cv2.imwrite(str(output_path), result, write_params)
    print(f"Saved to {output_path}")

    if args.show:
        cv2.imshow("Output", result)
        print("Press any key to close...")
        cv2.waitKey(0)
        cv2.destroyAllWindows()


# ─── MCP Server ──────────────────────────────────────────────────────────────

def _apply_sobel(image: np.ndarray) -> np.ndarray:
    channels = cv2.split(image)
    sobel_channels = []
    for ch in channels:
        sx = cv2.Sobel(ch, cv2.CV_64F, 1, 0, ksize=3)
        sy = cv2.Sobel(ch, cv2.CV_64F, 0, 1, ksize=3)
        sobel_channels.append(np.sqrt(sx ** 2 + sy ** 2))
    merged = cv2.merge(sobel_channels)
    mx = merged.max()
    if mx > 0:
        merged = merged / mx * 255
    return np.clip(merged, 0, 255).astype(np.uint8)


def run_mcp_server():
    """Start the MCP server with capture tools."""
    import base64
    from mcp.server.fastmcp import FastMCP

    mcp = FastMCP("subway-pcb")

    @mcp.tool()
    def capture_pcb(
        format: str = "normal",
        save: bool = True,
        quality: int = 95,
    ) -> str:
        """
        Capture a live photo of the PCB, perspective-transform it, and return the image.

        Args:
            format: "normal" for contrast-enhanced, "sobel" for color edge detection, "raw" for just the transform
            save: Whether to save the image to disk
            quality: JPEG quality (1-100)

        Returns the image as a base64-encoded JPEG.
        Requires running 'uv run capture.py setup' first to configure corners.
        """
        config = load_config()
        if config is None:
            return json.dumps({"error": "No setup config. Run 'uv run capture.py setup' first."})

        camera = config["camera"]
        corners = order_points(np.array(config["corners"], dtype=np.float32))

        frame = capture_frame(camera)
        warped = perspective_warp(frame, corners)

        if format == "sobel":
            result = _apply_sobel(warped)
        elif format == "raw":
            result = warped
        else:
            result = enhance_contrast(warped)

        _, buf = cv2.imencode(".jpg", result, [cv2.IMWRITE_JPEG_QUALITY, quality])
        b64 = base64.b64encode(buf).decode("utf-8")

        saved_path = ""
        if save:
            captures_dir = Path(__file__).parent / "captures"
            captures_dir.mkdir(exist_ok=True)
            ts = time.strftime("%Y%m%d_%H%M%S")
            suffix = f"_{format}" if format != "normal" else ""
            path = captures_dir / f"capture_{ts}{suffix}.jpg"
            cv2.imwrite(str(path), result, [cv2.IMWRITE_JPEG_QUALITY, quality])
            saved_path = str(path)

        return json.dumps({
            "image_base64": b64,
            "width": result.shape[1],
            "height": result.shape[0],
            "format": format,
            "saved_to": saved_path,
        })

    @mcp.tool()
    def get_setup_status() -> str:
        """Check if the PCB camera setup has been configured."""
        config = load_config()
        if config is None:
            return json.dumps({"configured": False, "message": "Run 'uv run capture.py setup' to configure."})
        return json.dumps({
            "configured": True,
            "camera": config["camera"],
            "corners": config["corners"],
        })

    mcp.run()


if __name__ == "__main__":
    import sys
    if len(sys.argv) > 1 and sys.argv[1] == "serve":
        sys.argv = [sys.argv[0]] + sys.argv[2:]  # strip "serve" for mcp arg parsing
        run_mcp_server()
    else:
        main()
