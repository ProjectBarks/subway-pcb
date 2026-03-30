#ifndef BOARD_CONFIG_H
#define BOARD_CONFIG_H

#include <stdint.h>
#include "config.h"

/* Runtime board hardware configuration.
 * Loaded from compile-time defaults, overridable via NVS. */
typedef struct {
    uint8_t  num_strips;
    uint8_t  strip_gpios[MAX_STRIPS];
    uint16_t strip_led_counts[MAX_STRIPS];
    uint16_t total_leds;
    uint8_t  spi_strip_index;
} board_hw_config_t;

/* Load board config: fills with compile-time defaults, then overrides from NVS. */
void board_config_load(board_hw_config_t *cfg);

#endif /* BOARD_CONFIG_H */
