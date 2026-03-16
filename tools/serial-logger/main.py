#!/usr/bin/env python3
"""Resilient ESP32 serial logger. Reconnects automatically."""

import argparse
import glob
import os
import re
import time

import serial


def find_port():
    ports = glob.glob("/dev/cu.usbserial-*")
    return ports[0] if ports else None


def main():
    parser = argparse.ArgumentParser(description="ESP32 Serial Logger")
    parser.add_argument("--port", default=None, help="Serial port (auto-detect if omitted)")
    parser.add_argument("--baud", type=int, default=115200)
    parser.add_argument("--output", "-o", default="serial.log")
    args = parser.parse_args()

    port = args.port or find_port()
    if not port:
        print("No USB serial device found")
        return

    logfile = os.path.abspath(args.output)
    print(f"Logging {port} -> {logfile}")
    print("Press Ctrl+C to stop")

    with open(logfile, "a") as f:
        f.write(f"\n--- Logger started {time.strftime('%Y-%m-%d %H:%M:%S')} ---\n")
        f.flush()

        while True:
            try:
                ser = serial.Serial(port, args.baud, timeout=1)
                f.write(f"[CONNECTED {time.strftime('%H:%M:%S')}]\n")
                f.flush()
                print(f"[{time.strftime('%H:%M:%S')}] Connected")

                while True:
                    line = ser.readline()
                    if line:
                        text = re.sub(r'\x1b\[[0-9;]*m', '', line.decode('utf-8', errors='replace').rstrip())
                        if text:
                            ts = time.strftime('%H:%M:%S')
                            entry = f"[{ts}] {text}"
                            f.write(entry + "\n")
                            f.flush()
                            print(entry)

            except (serial.SerialException, OSError):
                f.write(f"[DISCONNECTED {time.strftime('%H:%M:%S')}]\n")
                f.flush()
                print(f"[{time.strftime('%H:%M:%S')}] Disconnected, retrying...")

            except KeyboardInterrupt:
                print("\nStopped.")
                return

            time.sleep(2)


if __name__ == "__main__":
    main()
