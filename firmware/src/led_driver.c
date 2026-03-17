#include "led_driver.h"
#include "config.h"
#include "esp_log.h"
#include "led_strip.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include <string.h>

static const char *TAG = "led_driver";

static led_strip_handle_t s_strips[NUM_STRIPS];

/* Time-multiplexed SPI: create SPI device for one strip, send data, destroy it,
 * move to next strip. Uses DMA so each transfer is WiFi-immune.
 * We only update ~1/second so the sequential overhead is negligible. */

/* Pixel buffer — store all pixel data so we can re-send via SPI on refresh */
static uint8_t s_pixel_buf[TOTAL_LEDS * 3];
static uint16_t s_strip_offsets[NUM_STRIPS];

esp_err_t led_driver_init(void)
{
    /* Compute strip offsets into flat pixel buffer */
    uint16_t offset = 0;
    for (int i = 0; i < NUM_STRIPS; i++) {
        s_strip_offsets[i] = offset;
        offset += STRIP_LED_COUNTS[i];
    }

    memset(s_pixel_buf, 0, sizeof(s_pixel_buf));
    memset(s_strips, 0, sizeof(s_strips));

    ESP_LOGI(TAG, "LED driver initialized (%d strips, %d LEDs, SPI time-multiplexed)", NUM_STRIPS, TOTAL_LEDS);
    return ESP_OK;
}

esp_err_t led_driver_set_pixel(uint8_t strip, uint16_t pixel, uint8_t r, uint8_t g, uint8_t b)
{
    if (strip >= NUM_STRIPS || pixel >= STRIP_LED_COUNTS[strip]) {
        return ESP_ERR_INVALID_ARG;
    }
    /* Scale brightness locally — server sends full-range colors */
    int idx = (s_strip_offsets[strip] + pixel) * 3;
    s_pixel_buf[idx + 0] = (uint8_t)((uint16_t)r * DEFAULT_BRIGHTNESS / 255);
    s_pixel_buf[idx + 1] = (uint8_t)((uint16_t)g * DEFAULT_BRIGHTNESS / 255);
    s_pixel_buf[idx + 2] = (uint8_t)((uint16_t)b * DEFAULT_BRIGHTNESS / 255);
    return ESP_OK;
}

/* Send one strip via SPI DMA: create device, set pixels, refresh, destroy.
 * Each strip gets exclusive use of the SPI bus during its transfer. */
static esp_err_t refresh_strip_spi(int strip_idx)
{
    led_strip_config_t cfg = {
        .strip_gpio_num = STRIP_GPIOS[strip_idx],
        .max_leds = STRIP_LED_COUNTS[strip_idx],
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
    for (uint16_t p = 0; p < STRIP_LED_COUNTS[strip_idx]; p++) {
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
    for (int i = 0; i < NUM_STRIPS; i++) {
        esp_err_t ret = refresh_strip_spi(i);
        if (ret != ESP_OK) {
            ESP_LOGW(TAG, "Strip %d refresh failed: %s", i, esp_err_to_name(ret));
        }
    }
    return ESP_OK;
}

void led_driver_start_refresh_task(void) { /* unused */ }

esp_err_t led_driver_clear(void)
{
    memset(s_pixel_buf, 0, sizeof(s_pixel_buf));
    return led_driver_refresh();
}

void led_driver_boot_animation(void)
{
    ESP_LOGI(TAG, "Playing boot animation");
    const uint8_t brightness = DEFAULT_BRIGHTNESS;

    /* Sweep a window of 5 lit pixels across each strip sequentially */
    const int window = 5;
    for (int i = 0; i < NUM_STRIPS; i++) {
        uint16_t count = STRIP_LED_COUNTS[i];
        for (uint16_t p = 0; p < count + window; p++) {
            /* Light leading edge */
            if (p < count) {
                led_strip_set_pixel(s_strips[i], p, brightness, brightness, brightness);
            }
            /* Clear trailing edge */
            if (p >= window && (p - window) < count) {
                led_strip_set_pixel(s_strips[i], p - window, 0, 0, 0);
            }
            led_strip_refresh(s_strips[i]);
            vTaskDelay(pdMS_TO_TICKS(8));
        }
        /* Ensure strip is fully cleared */
        led_strip_clear(s_strips[i]);
        led_strip_refresh(s_strips[i]);
    }

    ESP_LOGI(TAG, "Boot animation complete");
}
