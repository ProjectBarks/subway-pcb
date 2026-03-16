#include "led_driver.h"
#include "config.h"
#include "esp_log.h"
#include "led_strip.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"

static const char *TAG = "led_driver";

static led_strip_handle_t s_strips[NUM_STRIPS];

esp_err_t led_driver_init(void)
{
    esp_err_t ret;

    for (int i = 0; i < NUM_STRIPS; i++) {
        if (i == SPI_STRIP_INDEX) {
            /* Skip SPI strip - potential conflict */
            s_strips[i] = NULL;
            continue;
        }

        led_strip_config_t strip_config = {
            .strip_gpio_num = STRIP_GPIOS[i],
            .max_leds = STRIP_LED_COUNTS[i],
            .led_pixel_format = LED_PIXEL_FORMAT_GRB,
            .led_model = LED_MODEL_WS2812,
        };

        {
            /* Use RMT backend for the other 8 strips */
            led_strip_rmt_config_t rmt_config = {
                .resolution_hz = 8 * 1000 * 1000, /* 8 MHz — wider timing margins for WS2812B */
                .flags = {
                    .with_dma = false,
                },
            };
            ret = led_strip_new_rmt_device(&strip_config, &rmt_config, &s_strips[i]);
        }

        if (ret != ESP_OK) {
            ESP_LOGE(TAG, "Failed to init strip %d (GPIO %d): %s",
                     i, STRIP_GPIOS[i], esp_err_to_name(ret));
            return ret;
        }

        /* Clear strip on init */
        led_strip_clear(s_strips[i]);
    }

    ESP_LOGI(TAG, "Initialized %d LED strips (%d total LEDs)", NUM_STRIPS, TOTAL_LEDS);
    return ESP_OK;
}

esp_err_t led_driver_set_pixel(uint8_t strip, uint16_t pixel, uint8_t r, uint8_t g, uint8_t b)
{
    if (strip >= NUM_STRIPS || !s_strips[strip]) {
        return ESP_ERR_INVALID_ARG;
    }
    if (pixel >= STRIP_LED_COUNTS[strip]) {
        return ESP_ERR_INVALID_ARG;
    }
    /* Scale brightness locally — server sends full-range colors for
     * better WS2812B signal resilience (higher values = fewer visible bit flips) */
    r = (uint8_t)((uint16_t)r * DEFAULT_BRIGHTNESS / 255);
    g = (uint8_t)((uint16_t)g * DEFAULT_BRIGHTNESS / 255);
    b = (uint8_t)((uint16_t)b * DEFAULT_BRIGHTNESS / 255);
    return led_strip_set_pixel(s_strips[strip], pixel, r, g, b);
}

esp_err_t led_driver_refresh(void)
{
    esp_err_t ret;
    for (int i = 0; i < NUM_STRIPS; i++) {
        if (!s_strips[i]) continue;
        ret = led_strip_refresh(s_strips[i]);
        if (ret != ESP_OK) {
            ESP_LOGW(TAG, "Failed to refresh strip %d: %s", i, esp_err_to_name(ret));
        }
    }
    return ESP_OK;
}

static void refresh_task(void *pvParameters)
{
    /* Continuously re-push pixel data to fix WS2812B signal glitches.
     * Any bit corruption (causing cyan/wrong colors) self-corrects
     * within 50ms instead of persisting until the next server poll. */
    while (1) {
        for (int i = 0; i < NUM_STRIPS; i++) {
            if (!s_strips[i]) continue;
            led_strip_refresh(s_strips[i]);
            vTaskDelay(pdMS_TO_TICKS(1));
        }
        vTaskDelay(pdMS_TO_TICKS(50));
    }
}

void led_driver_start_refresh_task(void)
{
    xTaskCreate(refresh_task, "led_refresh", 2048, NULL, 3, NULL);
    ESP_LOGI(TAG, "LED refresh task started (50ms cycle)");
}

esp_err_t led_driver_clear(void)
{
    for (int i = 0; i < NUM_STRIPS; i++) {
        if (!s_strips[i]) continue;
        led_strip_clear(s_strips[i]);
    }
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
