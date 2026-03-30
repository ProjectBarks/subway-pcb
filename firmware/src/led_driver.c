#include "led_driver.h"
#include "render_context.h"
#include "device_log.h"
#include "esp_log.h"
#include "led_strip.h"
#include <string.h>

static const char *TAG = "led_driver";

static render_context_t *s_ctx = NULL;
static const board_hw_config_t *s_hw = NULL;

/* Time-multiplexed SPI: create SPI device for one strip, send data, destroy it,
 * move to next strip. Uses DMA so each transfer is WiFi-immune. */

/* Pixel buffer — sized to compile-time ceiling */
static uint8_t s_pixel_buf[MAX_LEDS * 3];
static uint16_t s_strip_offsets[MAX_STRIPS];

esp_err_t led_driver_init(render_context_t *ctx, const board_hw_config_t *hw)
{
    s_ctx = ctx;
    s_hw = hw;

    /* Compute strip offsets into flat pixel buffer */
    uint16_t offset = 0;
    for (int i = 0; i < hw->num_strips; i++) {
        s_strip_offsets[i] = offset;
        offset += hw->strip_led_counts[i];
    }

    memset(s_pixel_buf, 0, sizeof(s_pixel_buf));

    DLOG_I(TAG, "LED driver initialized (%d strips, %d LEDs, SPI time-multiplexed)",
             hw->num_strips, hw->total_leds);
    return ESP_OK;
}

esp_err_t led_driver_set_pixel(uint8_t strip, uint16_t pixel, uint8_t r, uint8_t g, uint8_t b)
{
    if (!s_hw || strip >= s_hw->num_strips || pixel >= s_hw->strip_led_counts[strip]) {
        return ESP_ERR_INVALID_ARG;
    }
    int idx = (s_strip_offsets[strip] + pixel) * 3;
    s_pixel_buf[idx + 0] = r;
    s_pixel_buf[idx + 1] = g;
    s_pixel_buf[idx + 2] = b;
    return ESP_OK;
}

/* Send one strip via SPI DMA: create device, set pixels, refresh, destroy.
 * Each strip gets exclusive use of the SPI bus during its transfer. */
static esp_err_t refresh_strip_spi(int strip_idx)
{
    led_strip_config_t cfg = {
        .strip_gpio_num = s_hw->strip_gpios[strip_idx],
        .max_leds = s_hw->strip_led_counts[strip_idx],
        .led_pixel_format = LED_PIXEL_FORMAT_GRB,
        .led_model = LED_MODEL_WS2812,
    };
    led_strip_spi_config_t spi_cfg = {
        .spi_bus = SPI2_HOST,
        .flags = { .with_dma = true },
    };

    led_strip_handle_t handle = NULL;
    esp_err_t ret = led_strip_new_spi_device(&cfg, &spi_cfg, &handle);
    if (ret != ESP_OK) return ret;

    /* Load pixel data */
    int offset = s_strip_offsets[strip_idx];
    for (uint16_t p = 0; p < s_hw->strip_led_counts[strip_idx]; p++) {
        int idx = (offset + p) * 3;
        led_strip_set_pixel(handle, p, s_pixel_buf[idx], s_pixel_buf[idx+1], s_pixel_buf[idx+2]);
    }

    ret = led_strip_refresh(handle);

    /* Tear down — frees the SPI bus for the next strip */
    led_strip_del(handle);

    return ret;
}

esp_err_t led_driver_refresh(void)
{
    if (!s_hw) return ESP_ERR_INVALID_STATE;

    int ok = 0, fail = 0;
    for (int i = 0; i < s_hw->num_strips; i++) {
        esp_err_t ret = refresh_strip_spi(i);
        if (ret != ESP_OK) {
            fail++;
        } else {
            ok++;
        }
    }
    if (s_ctx) {
        s_ctx->diag.strip_ok = ok;
        s_ctx->diag.strip_fail = fail;
    }
    return ESP_OK;
}

esp_err_t led_driver_clear(void)
{
    if (!s_hw) return ESP_ERR_INVALID_STATE;
    memset(s_pixel_buf, 0, s_hw->total_leds * 3);
    return led_driver_refresh();
}
