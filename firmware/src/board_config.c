#include "board_config.h"
#include "device_log.h"
#include "esp_log.h"
#include "nvs_flash.h"
#include "nvs.h"
#include <string.h>

static const char *TAG = "board_config";

/* Compile-time defaults (matching original config.h values) */
static const uint8_t DEFAULT_STRIP_GPIOS[] = {16, 17, 18, 19, 21, 22, 23, 25, 26};
static const uint16_t DEFAULT_STRIP_LED_COUNTS[] = {97, 102, 55, 81, 70, 21, 22, 19, 11};
#define DEFAULT_NUM_STRIPS     9
#define DEFAULT_TOTAL_LEDS     478
#define DEFAULT_SPI_STRIP_IDX  8

void board_config_load(board_hw_config_t *cfg)
{
    memset(cfg, 0, sizeof(*cfg));

    /* Fill with compile-time defaults */
    cfg->num_strips = DEFAULT_NUM_STRIPS;
    cfg->total_leds = DEFAULT_TOTAL_LEDS;
    cfg->spi_strip_index = DEFAULT_SPI_STRIP_IDX;
    for (int i = 0; i < DEFAULT_NUM_STRIPS; i++) {
        cfg->strip_gpios[i] = DEFAULT_STRIP_GPIOS[i];
        cfg->strip_led_counts[i] = DEFAULT_STRIP_LED_COUNTS[i];
    }

    /* Try NVS overrides */
    nvs_handle_t nvs;
    if (nvs_open("subway", NVS_READONLY, &nvs) != ESP_OK) {
        DLOG_I(TAG, "Using defaults: %d strips, %d LEDs", cfg->num_strips, cfg->total_leds);
        return;
    }

    uint8_t num_strips = 0;
    if (nvs_get_u8(nvs, "hw_num_strips", &num_strips) == ESP_OK && num_strips > 0 && num_strips <= MAX_STRIPS) {
        cfg->num_strips = num_strips;

        /* Read GPIO blob */
        size_t len = num_strips;
        uint8_t gpio_buf[MAX_STRIPS];
        if (nvs_get_blob(nvs, "hw_strip_gpios", gpio_buf, &len) == ESP_OK && len == num_strips) {
            memcpy(cfg->strip_gpios, gpio_buf, num_strips);
        }

        /* Read LED count blob (uint16_t array) */
        len = num_strips * sizeof(uint16_t);
        uint16_t led_buf[MAX_STRIPS];
        if (nvs_get_blob(nvs, "hw_strip_leds", led_buf, &len) == ESP_OK && len == num_strips * sizeof(uint16_t)) {
            memcpy(cfg->strip_led_counts, led_buf, len);
            /* Recompute total */
            cfg->total_leds = 0;
            for (int i = 0; i < num_strips; i++) {
                cfg->total_leds += cfg->strip_led_counts[i];
            }
        }

        uint8_t spi_idx = 0;
        if (nvs_get_u8(nvs, "hw_spi_strip", &spi_idx) == ESP_OK) {
            cfg->spi_strip_index = spi_idx;
        }
    }

    nvs_close(nvs);
    DLOG_I(TAG, "Board config: %d strips, %d LEDs, SPI strip=%d",
             cfg->num_strips, cfg->total_leds, cfg->spi_strip_index);
}
