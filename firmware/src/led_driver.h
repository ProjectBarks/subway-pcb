#ifndef LED_DRIVER_H
#define LED_DRIVER_H

#include <stdint.h>
#include "esp_err.h"

/**
 * Initialize strip 0 only (for status indicator during boot).
 */
esp_err_t led_driver_init(void);

/**
 * Initialize remaining strips 1-7 (call after WiFi is connected).
 * Strip 8 (SPI) is skipped.
 */
esp_err_t led_driver_init_remaining(void);

/**
 * Set a single pixel on the given strip.
 * Changes are not visible until led_driver_refresh() is called.
 */
esp_err_t led_driver_set_pixel(uint8_t strip, uint16_t pixel, uint8_t r, uint8_t g, uint8_t b);

/**
 * Push all pixel data to the LED strips.
 */
esp_err_t led_driver_refresh(void);

/**
 * Set all pixels on all strips to black.
 * Calls refresh internally.
 */
esp_err_t led_driver_clear(void);

/**
 * Start a background task that refreshes all strips every 50ms.
 * Fixes WS2812B signal corruption (cyan glitch) by re-pushing data continuously.
 */
void led_driver_start_refresh_task(void);

#endif /* LED_DRIVER_H */
