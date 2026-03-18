#!/usr/bin/env python3
# /// script
# requires-python = ">=3.10"
# dependencies = []
# ///
"""Standalone board viewer — serves a 2D canvas that polls the pixel API.

Usage:
    uv run serve.py                              # defaults: localhost:8888, production API
    uv run serve.py --port 9000                  # custom port
    uv run serve.py --api http://localhost:8080   # local backend
"""

import argparse
import http.server
import os
import urllib.request
import sys

DEFAULT_API = "https://subway-pcb-production.up.railway.app"
DEFAULT_DEVICE = "b0:b2:1c:41:2c:28"


class ProxyHandler(http.server.SimpleHTTPRequestHandler):
    api_base = DEFAULT_API
    device = DEFAULT_DEVICE

    def do_GET(self):
        if self.path == "/api/v1/pixels":
            self._proxy(f"{self.api_base}/api/v1/pixels?device={self.device.replace(':', '%3A')}")
        elif self.path.startswith("/api/v1/state"):
            self._proxy(f"{self.api_base}{self.path}")
        elif self.path.startswith("/static/"):
            # Serve static data files from this directory
            fname = self.path.replace("/static/", "")
            fpath = os.path.join(os.path.dirname(__file__), fname)
            if os.path.exists(fpath):
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                with open(fpath, "rb") as f:
                    self.wfile.write(f.read())
            else:
                self.send_error(404)
        else:
            super().do_GET()

    def _proxy(self, url):
        try:
            req = urllib.request.Request(url)
            with urllib.request.urlopen(req, timeout=5) as resp:
                data = resp.read()
                self.send_response(200)
                self.send_header("Content-Type", resp.headers.get("Content-Type", "application/octet-stream"))
                self.send_header("Content-Length", str(len(data)))
                self.end_headers()
                self.wfile.write(data)
        except Exception as e:
            self.send_error(502, f"Proxy error: {e}")

    def log_message(self, format, *args):
        pass  # quiet


def main():
    parser = argparse.ArgumentParser(description="Standalone board viewer with API proxy")
    parser.add_argument("--port", type=int, default=8888)
    parser.add_argument("--api", default=DEFAULT_API, help="Backend API base URL")
    parser.add_argument("--device", default=DEFAULT_DEVICE, help="Device MAC address")
    args = parser.parse_args()

    ProxyHandler.api_base = args.api.rstrip("/")
    ProxyHandler.device = args.device

    os.chdir(os.path.dirname(os.path.abspath(__file__)))
    server = http.server.HTTPServer(("", args.port), ProxyHandler)
    print(f"Board viewer: http://localhost:{args.port}")
    print(f"  API: {args.api}")
    print(f"  Device: {args.device}")
    server.serve_forever()


if __name__ == "__main__":
    main()
