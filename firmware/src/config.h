#ifndef CONFIG_H
#define CONFIG_H

#include <stdint.h>

/* Number of WS2812B LED strips */
#define NUM_STRIPS 9

/* Total LEDs across all strips */
#define TOTAL_LEDS 478

/* Total bytes of pixel data (478 * 3) */
#define TOTAL_PIXEL_BYTES 1434

/* GPIO pin assignments for each strip */
static const uint8_t STRIP_GPIOS[NUM_STRIPS] = {
    16, 17, 18, 19, 21, 22, 23, 25, 26
};

/* Number of LEDs on each strip */
static const uint16_t STRIP_LED_COUNTS[NUM_STRIPS] = {
    97, 102, 55, 81, 70, 21, 22, 19, 11
};

/* Strip index that uses SPI backend instead of RMT (GPIO 26, 11 LEDs) */
#define SPI_STRIP_INDEX 8

/* Server configuration */
#define DEFAULT_SERVER_URL "https://subway-pcb-production.up.railway.app/api/v1/pixels"
#define SERVER_URL_NVS_KEY "server_url"
#define SERVER_URL_MAX_LEN 128

/* Polling interval in seconds */
#define POLL_INTERVAL_SEC 1

/* Default LED brightness (0-255) */
#define DEFAULT_BRIGHTNESS 15

/* HTTP receive buffer size (slightly over 1428) */
#define HTTP_BUF_SIZE 1600

/* Subway client task config */
#define SUBWAY_CLIENT_TASK_STACK 8192
#define SUBWAY_CLIENT_TASK_PRIORITY 4

/* OTA update check interval in minutes */
#define OTA_CHECK_INTERVAL_MIN 5

#endif /* CONFIG_H */
