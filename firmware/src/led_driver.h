#ifndef LED_DRIVER_H
#define LED_DRIVER_H

#include <stdbool.h>
#include <stdint.h>
#include "esp_err.h"
#include "board_config.h"

/* Forward declaration — avoid circular include */
struct render_context;

/**
 * Initialize LED driver with board hardware config.
 * Stores ctx pointer for writing diagnostics (strip_ok/strip_fail).
 */
esp_err_t led_driver_init(struct render_context *ctx, const board_hw_config_t *hw);

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
 * Map a global LED index to its strip and pixel offset.
 * Returns true if the index is valid, false otherwise.
 */
bool led_driver_map_pixel(uint32_t global_index, uint8_t *out_strip, uint16_t *out_pixel);

/**
 * Set all pixels on all strips to black.
 * Calls refresh internally.
 */
esp_err_t led_driver_clear(void);

#endif /* LED_DRIVER_H */
