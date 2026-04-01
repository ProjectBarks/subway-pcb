#include "hal/led_driver.hpp"

#include "data/diag_pad.hpp"
#include "esp_log.h"
#include "led_strip.h"
#include "log/device_log.hpp"

#include <algorithm>
#include <cstring>

static const char* TAG = "led_driver";

esp_err_t LedDriver::init(const BoardHwConfig* hw) {
    hw_ = hw;

    // Compute strip offsets into flat pixel buffer
    uint16_t offset = 0;
    for (int i = 0; i < hw->num_strips; i++) {
        strip_offsets_[i] = offset;
        offset += hw->strip_led_counts[i];
    }

    std::fill(std::begin(pixel_buf_), std::end(pixel_buf_), Rgb{});

    DLOG_I(TAG,
           "LED driver initialized (%d strips, %d LEDs, SPI time-multiplexed)",
           hw->num_strips,
           hw->total_leds);
    return ESP_OK;
}

esp_err_t LedDriver::refresh_strip_spi(int strip_idx) const {
    led_strip_config_t cfg = {
        .strip_gpio_num = hw_->strip_gpios[strip_idx],
        .max_leds = hw_->strip_led_counts[strip_idx],
        .led_pixel_format = LED_PIXEL_FORMAT_GRB,
        .led_model = LED_MODEL_WS2812,
    };
    led_strip_spi_config_t spi_cfg = {
        .spi_bus = SPI2_HOST,
        .flags = {.with_dma = true},
    };

    led_strip_handle_t handle = nullptr;
    esp_err_t ret = led_strip_new_spi_device(&cfg, &spi_cfg, &handle);
    if (ret != ESP_OK)
        return ret;

    // Load pixel data from Rgb buffer
    uint16_t offset = strip_offsets_[strip_idx];
    for (uint16_t p = 0; p < hw_->strip_led_counts[strip_idx]; p++) {
        const Rgb& px = pixel_buf_[offset + p];
        led_strip_set_pixel(handle, p, px.r, px.g, px.b);
    }

    ret = led_strip_refresh(handle);

    // Tear down -- frees the SPI bus for the next strip
    led_strip_del(handle);

    return ret;
}

bool LedDriver::map_pixel(uint32_t global_index, uint8_t* out_strip, uint16_t* out_pixel) const {
    if (!hw_)
        return false;
    for (uint8_t i = 0; i < hw_->num_strips; i++) {
        uint16_t offset = strip_offsets_[i];
        if (global_index >= offset && global_index < offset + hw_->strip_led_counts[i]) {
            *out_strip = i;
            *out_pixel = static_cast<uint16_t>(global_index - offset);
            return true;
        }
    }
    return false;
}

esp_err_t LedDriver::refresh(DiagPad& diag) {
    if (!hw_)
        return ESP_ERR_INVALID_STATE;

    int ok = 0;
    int fail = 0;
    for (int i = 0; i < hw_->num_strips; i++) {
        esp_err_t ret = refresh_strip_spi(i);
        if (ret != ESP_OK) {
            ESP_LOGW(TAG, "Strip %d SPI refresh failed: %s", i, esp_err_to_name(ret));
            fail++;
        } else {
            ok++;
        }
    }

    diag.strip_ok.store(ok, std::memory_order_relaxed);
    diag.strip_fail.store(fail, std::memory_order_relaxed);

    return ESP_OK;
}

esp_err_t LedDriver::clear() {
    if (!hw_)
        return ESP_ERR_INVALID_STATE;

    std::fill(pixel_buf_, pixel_buf_ + hw_->total_leds, Rgb{});

    // Push zeroed pixels to all strips directly (no diag update needed)
    for (int i = 0; i < hw_->num_strips; i++) {
        esp_err_t ret = refresh_strip_spi(i);
        if (ret != ESP_OK) {
            ESP_LOGW(TAG, "Strip %d SPI clear failed: %s", i, esp_err_to_name(ret));
        }
    }

    return ESP_OK;
}
