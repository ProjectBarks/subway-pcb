#pragma once
#include <cstdint>
#include "config/constants.hpp"

struct BoardHwConfig {
    uint8_t  num_strips = 0;
    uint8_t  strip_gpios[kMaxStrips]{};
    uint16_t strip_led_counts[kMaxStrips]{};
    uint16_t total_leds = 0;
    uint8_t  spi_strip_index = 0;

    // Load from compile-time defaults + NVS overrides
    static void load(BoardHwConfig& cfg);
};
