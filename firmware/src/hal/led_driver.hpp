#pragma once
#include <cstdint>
#include <span>
#include "esp_err.h"
#include "core/types.hpp"
#include "config/constants.hpp"
#include "config/board_config.hpp"

struct DiagPad;  // forward declaration

class LedDriver {
    const BoardHwConfig* hw_ = nullptr;
    Rgb pixel_buf_[kMaxLeds]{};
    uint16_t strip_offsets_[kMaxStrips]{};

    esp_err_t refresh_strip_spi(int strip_idx);

public:
    esp_err_t init(const BoardHwConfig* hw);

    // Direct access to pixel buffer -- Lua writes here
    std::span<Rgb> pixel_buffer() { return {pixel_buf_, hw_ ? hw_->total_leds : 0u}; }

    uint32_t led_count() const { return hw_ ? hw_->total_leds : 0; }

    // Map global LED index to strip + pixel offset
    bool map_pixel(uint32_t global_index, uint8_t* out_strip, uint16_t* out_pixel) const;

    // Push all pixel data to strips. Updates diag strip_ok/strip_fail.
    esp_err_t refresh(DiagPad& diag);

    // Set all pixels black and refresh
    esp_err_t clear();
};
