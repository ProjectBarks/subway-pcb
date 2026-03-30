#ifndef CONFIG_H
#define CONFIG_H

#include <stdint.h>

/* Buffer ceilings — used for static array sizing */
#define MAX_STRIPS 16
#define MAX_LEDS   512

/* Server configuration */
#define DEFAULT_SERVER_URL "https://device.pcb.nyc"
#define SERVER_URL_NVS_KEY "server_url"
#define SERVER_URL_MAX_LEN 128

/* Firmware version string (sent as X-Firmware-Version header) */
#define FIRMWARE_VERSION "0.7.0"

/* Polling interval in milliseconds */
#define POLL_INTERVAL_MS 1000

/* Default LED brightness (0-255) */
#define DEFAULT_BRIGHTNESS 10

/* OTA update check interval in minutes */
#define OTA_CHECK_INTERVAL_MIN 60

#endif /* CONFIG_H */
