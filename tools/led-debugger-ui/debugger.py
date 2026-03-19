#!/usr/bin/env python3
"""
LED Debugger — Click on the web board view to light up physical LEDs.

Usage:
    python3 debugger.py [--port /dev/cu.usbserial-210] [--http 8090]

Reuses board.js/leds.json/board.svg from server/static/ for the PCB view.
Adds click-to-light serial control on top.
"""

import argparse
import json
import os
import threading
from http.server import HTTPServer, SimpleHTTPRequestHandler
import serial
import time

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "../.."))
STATIC_DIR = os.path.join(PROJECT_ROOT, "server/static")


class LEDController:
    def __init__(self, port, baud=115200):
        self.port = port
        self.baud = baud
        self.ser = None
        self.lock = threading.Lock()

    def connect(self):
        try:
            self.ser = serial.Serial(self.port, self.baud, timeout=1)
            time.sleep(2)
            while self.ser.in_waiting:
                self.ser.readline()
            self.ser.write(b"ping\n")
            resp = self.ser.readline().decode().strip()
            if resp == "pong":
                print(f"[serial] Connected to {self.port}")
                return True
            print(f"[serial] Unexpected: {resp}")
        except Exception as e:
            print(f"[serial] Failed: {e}")
        return False

    def send(self, cmd):
        with self.lock:
            if not self.ser:
                return "not connected"
            try:
                self.ser.write((cmd + "\n").encode())
                return self.ser.readline().decode().strip()
            except Exception as e:
                return f"error: {e}"


class DebugHandler(SimpleHTTPRequestHandler):
    controller: "LEDController | None" = None

    def do_GET(self):
        if self.path == "/":
            self._serve_file(os.path.join(SCRIPT_DIR, "index.html"), "text/html")
        elif self.path.startswith("/static/"):
            filepath = os.path.join(STATIC_DIR, self.path[8:])
            if os.path.isfile(filepath):
                ext = os.path.splitext(filepath)[1]
                ct = {".js": "text/javascript", ".json": "application/json",
                      ".svg": "image/svg+xml", ".html": "text/html"}.get(ext, "application/octet-stream")
                self._serve_file(filepath, ct)
            else:
                self.send_error(404)
        else:
            self.send_error(404)

    def do_POST(self):
        if self.path == "/api/led":
            body = json.loads(self.rfile.read(int(self.headers.get("Content-Length", 0))))
            action = body.get("action", "")
            if action == "on" and self.controller:
                resp = self.controller.send(
                    f"on {body['strip']} {body['pixel']} {body.get('r',255)} {body.get('g',255)} {body.get('b',255)}")
                self._json({"ok": resp == "ok", "resp": resp})
            elif action == "clear" and self.controller:
                self._json({"ok": self.controller.send("clear") == "ok"})
            else:
                self._json({"ok": False})
        else:
            self.send_error(404)

    def _serve_file(self, path, content_type):
        self.send_response(200)
        self.send_header("Content-Type", content_type)
        self.end_headers()
        with open(path, "rb") as f:
            self.wfile.write(f.read())

    def _json(self, data):
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(data).encode())

    def log_message(self, format: str, *args: object) -> None:
        try:
            if args and isinstance(args[0], str) and "/api/" in args[0]:
                return
        except Exception:
            pass
        super().log_message(format, *args)


def main():
    parser = argparse.ArgumentParser(description="LED Debugger")
    parser.add_argument("--port", default="/dev/cu.usbserial-210")
    parser.add_argument("--http", type=int, default=8090)
    parser.add_argument("--no-serial", action="store_true")
    args = parser.parse_args()

    controller = LEDController(args.port)
    if not args.no_serial:
        if not controller.connect():
            print(f"Warning: no serial on {args.port}, web-only mode")

    DebugHandler.controller = controller
    server = HTTPServer(("", args.http), DebugHandler)
    print(f"LED Debugger at http://localhost:{args.http}/")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nStopped.")


if __name__ == "__main__":
    main()
