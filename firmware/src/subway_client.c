#include "subway_client.h"
#include "led_driver.h"
#include "config.h"
#include "subway.pb.h"

#include "esp_log.h"
#include "esp_http_client.h"
#include "nvs_flash.h"
#include "nvs.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include <pb_decode.h>
#include <string.h>

static const char *TAG = "subway_client";


/* Static buffer for HTTP response */
static uint8_t s_http_buf[HTTP_BUF_SIZE];

/* Server URL */
static char s_server_url[SERVER_URL_MAX_LEN];

static TaskHandle_t s_task_handle = NULL;

static void load_server_url(void)
{
    nvs_handle_t nvs;
    esp_err_t err = nvs_open("subway", NVS_READONLY, &nvs);
    if (err == ESP_OK) {
        size_t len = sizeof(s_server_url);
        err = nvs_get_str(nvs, SERVER_URL_NVS_KEY, s_server_url, &len);
        nvs_close(nvs);
        if (err == ESP_OK && len > 0) {
            ESP_LOGI(TAG, "Server URL from NVS: %s", s_server_url);
            return;
        }
    }
    strncpy(s_server_url, DEFAULT_SERVER_URL, sizeof(s_server_url) - 1);
    s_server_url[sizeof(s_server_url) - 1] = '\0';
    ESP_LOGI(TAG, "Using default URL: %s", s_server_url);
}

static int fetch_pixels(uint8_t *buf, int buf_size)
{
    esp_http_client_config_t config = {
        .url = s_server_url,
        .timeout_ms = 5000,
    };

    esp_http_client_handle_t client = esp_http_client_init(&config);
    if (!client) return -1;

    esp_err_t err = esp_http_client_open(client, 0);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "HTTP open: %s", esp_err_to_name(err));
        esp_http_client_cleanup(client);
        return -1;
    }

    esp_http_client_fetch_headers(client);
    int status = esp_http_client_get_status_code(client);
    if (status != 200) {
        ESP_LOGW(TAG, "HTTP %d", status);
        esp_http_client_close(client);
        esp_http_client_cleanup(client);
        return -1;
    }

    int total = 0;
    while (total < buf_size) {
        int n = esp_http_client_read(client, (char *)(buf + total), buf_size - total);
        if (n <= 0) break;
        total += n;
    }

    esp_http_client_close(client);
    esp_http_client_cleanup(client);
    return total > 0 ? total : -1;
}

/* Apply decoded PixelFrame to LED strips */
static void apply_pixels(const subway_PixelFrame *frame)
{
    if (frame->pixels.size < TOTAL_PIXEL_BYTES) {
        ESP_LOGW(TAG, "Short pixel data: %d < %d", (int)frame->pixels.size, TOTAL_PIXEL_BYTES);
        return;
    }

    const uint8_t *p = frame->pixels.bytes;
    int offset = 0;

    for (int strip = 0; strip < NUM_STRIPS; strip++) {
        for (uint16_t pixel = 0; pixel < STRIP_LED_COUNTS[strip]; pixel++) {
            int idx = offset * 3;
            led_driver_set_pixel(strip, pixel, p[idx], p[idx+1], p[idx+2]);
            offset++;
        }
    }

    led_driver_refresh();
}

static void subway_client_task(void *pvParameters)
{
    load_server_url();

    int backoff_sec = POLL_INTERVAL_SEC;

    /* PixelFrame is small enough for stack (~1.5KB) */
    subway_PixelFrame frame;

    while (1) {
        int len = fetch_pixels(s_http_buf, sizeof(s_http_buf));
        if (len > 0) {
            frame = (subway_PixelFrame)subway_PixelFrame_init_zero;
            pb_istream_t stream = pb_istream_from_buffer(s_http_buf, (size_t)len);
            bool ok = pb_decode(&stream, subway_PixelFrame_fields, &frame);

            if (ok) {
                ESP_LOGI(TAG, "Frame seq=%lu, %lu LEDs, %d bytes pixels",
                         (unsigned long)frame.sequence,
                         (unsigned long)frame.led_count,
                         (int)frame.pixels.size);
                apply_pixels(&frame);
                backoff_sec = POLL_INTERVAL_SEC;
            } else {
                ESP_LOGE(TAG, "Decode failed: %s", PB_GET_ERROR(&stream));
                backoff_sec = backoff_sec < 60 ? backoff_sec * 2 : 60;
            }
        } else {
            ESP_LOGW(TAG, "Fetch failed, retry in %ds", backoff_sec);
            backoff_sec = backoff_sec < 60 ? backoff_sec * 2 : 60;
        }

        vTaskDelay(pdMS_TO_TICKS(backoff_sec * 1000));
    }
}

esp_err_t subway_client_start(void)
{
    if (s_task_handle) return ESP_OK;

    BaseType_t ret = xTaskCreate(subway_client_task, "subway_client",
                                  SUBWAY_CLIENT_TASK_STACK, NULL,
                                  SUBWAY_CLIENT_TASK_PRIORITY, &s_task_handle);
    if (ret != pdPASS) {
        ESP_LOGE(TAG, "Failed to create task");
        return ESP_FAIL;
    }
    ESP_LOGI(TAG, "Started");
    return ESP_OK;
}
