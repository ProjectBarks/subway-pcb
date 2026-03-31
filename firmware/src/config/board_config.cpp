#include "config/board_config.hpp"

#include "log/device_log.hpp"

extern "C" {
#include "nvs_flash.h"
#include "nvs.h"
}

#include <cstring>

static const char* TAG = "board_config";

// Compile-time defaults (matching original board_config.c values)
static constexpr uint8_t  kDefaultStripGpios[]    = {16, 17, 18, 19, 21, 22, 23, 25, 26};
static constexpr uint16_t kDefaultStripLedCounts[] = {97, 102, 55, 81, 70, 21, 22, 19, 11};
static constexpr uint8_t  kDefaultNumStrips       = 9;
static constexpr uint16_t kDefaultTotalLeds       = 478;
static constexpr uint8_t  kDefaultSpiStripIdx     = 8;

void BoardHwConfig::load(BoardHwConfig& cfg)
{
    cfg = BoardHwConfig{};

    // Fill with compile-time defaults
    cfg.num_strips = kDefaultNumStrips;
    cfg.total_leds = kDefaultTotalLeds;
    cfg.spi_strip_index = kDefaultSpiStripIdx;
    for (int i = 0; i < kDefaultNumStrips; i++) {
        cfg.strip_gpios[i] = kDefaultStripGpios[i];
        cfg.strip_led_counts[i] = kDefaultStripLedCounts[i];
    }

    // Try NVS overrides
    nvs_handle_t nvs;
    if (nvs_open("subway", NVS_READONLY, &nvs) != ESP_OK) {
        DLOG_I(TAG, "Using defaults: %d strips, %d LEDs", cfg.num_strips, cfg.total_leds);
        return;
    }

    uint8_t num_strips = 0;
    if (nvs_get_u8(nvs, "hw_num_strips", &num_strips) == ESP_OK &&
        num_strips > 0 && num_strips <= kMaxStrips) {
        cfg.num_strips = num_strips;

        // Read GPIO blob
        size_t len = num_strips;
        uint8_t gpio_buf[kMaxStrips];
        if (nvs_get_blob(nvs, "hw_strip_gpios", gpio_buf, &len) == ESP_OK &&
            len == num_strips) {
            std::memcpy(cfg.strip_gpios, gpio_buf, num_strips);
        }

        // Read LED count blob (uint16_t array)
        len = num_strips * sizeof(uint16_t);
        uint16_t led_buf[kMaxStrips];
        if (nvs_get_blob(nvs, "hw_strip_leds", led_buf, &len) == ESP_OK &&
            len == num_strips * sizeof(uint16_t)) {
            std::memcpy(cfg.strip_led_counts, led_buf, len);
            // Recompute total
            cfg.total_leds = 0;
            for (int i = 0; i < num_strips; i++) {
                cfg.total_leds += cfg.strip_led_counts[i];
            }
        }

        uint8_t spi_idx = 0;
        if (nvs_get_u8(nvs, "hw_spi_strip", &spi_idx) == ESP_OK) {
            cfg.spi_strip_index = spi_idx;
        }
    }

    nvs_close(nvs);
    DLOG_I(TAG, "Board config: %d strips, %d LEDs, SPI strip=%d",
           cfg.num_strips, cfg.total_leds, cfg.spi_strip_index);
}
