#!/usr/bin/env python3
"""Resilient ESP32 serial logger. Reconnects automatically, writes to serial.log"""

import serial, time, re, sys, os

PORT = "/dev/cu.usbserial-210"
BAUD = 115200
LOGFILE = os.path.join(os.path.dirname(os.path.abspath(__file__)), "serial.log")

def main():
    print(f"Logging {PORT} -> {LOGFILE}")
    print("Press Ctrl+C to stop")

    with open(LOGFILE, "a") as f:
        f.write(f"\n--- Logger started {time.strftime('%Y-%m-%d %H:%M:%S')} ---\n")
        f.flush()

        while True:
            try:
                ser = serial.Serial(PORT, BAUD, timeout=1)
                f.write(f"[CONNECTED {time.strftime('%H:%M:%S')}]\n")
                f.flush()
                print(f"[{time.strftime('%H:%M:%S')}] Connected")

                while True:
                    line = ser.readline()
                    if line:
                        text = re.sub(r'\x1b\[[0-9;]*m', '', line.decode('utf-8', errors='replace').rstrip())
                        if text:
                            ts = time.strftime('%H:%M:%S')
                            f.write(f"[{ts}] {text}\n")
                            f.flush()

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
