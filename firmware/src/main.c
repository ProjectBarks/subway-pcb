#include <stdio.h>
#include <string.h>

#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "esp_system.h"
#include "esp_log.h"
#include "esp_wifi.h"
#include "esp_netif.h"
#include "nvs_flash.h"
#include "esp_heap_caps.h"

#include "wifi_manager.h"
#include "esp_ghota.h"

#include "config.h"
#include "led_driver.h"
#include "render_context.h"
#include "state_client.h"
#include "lua_runtime.h"

static const char *TAG = "main";
static ghota_client_handle_t *s_ghota = NULL;
static render_context_t s_render_ctx;

/* Global flag: when true, state_client pauses to give OTA exclusive HTTPS access */
volatile bool g_ota_active = false;

static void ghota_event_handler(void *arg, esp_event_base_t base, int32_t id, void *data)
{
    ghota_event_e event = (ghota_event_e)id;
    ESP_LOGI(TAG, "OTA event: %s", ghota_get_event_str(event));

    switch (event) {
        case GHOTA_EVENT_START_CHECK:
        case GHOTA_EVENT_START_UPDATE:
            g_ota_active = true;
            ESP_LOGW(TAG, "Free heap: %lu bytes", (unsigned long)esp_get_free_heap_size());
            break;
        case GHOTA_EVENT_NOUPDATE_AVAILABLE:
        case GHOTA_EVENT_UPDATE_FAILED:
        case GHOTA_EVENT_FINISH_UPDATE:
            g_ota_active = false;
            break;
        default:
            break;
    }
}

static volatile bool s_wifi_connected = false;

static void cb_wifi_got_ip(void *pvParameter)
{
    /* Keep this callback lightweight — runs on wifi_manager's 4KB stack.
     * Just set a flag; app_main loop handles the heavy lifting. */
    ESP_LOGI(TAG, "WiFi connected");
    s_wifi_connected = true;
}

void app_main(void)
{
    ESP_LOGW(TAG, "=== NYC Subway PCB v0.6.0 === reset_reason=%d", (int)esp_reset_reason());
    ESP_LOGW(TAG, "HEAP: %lu free, %lu min, %lu largest block",
             (unsigned long)esp_get_free_heap_size(),
             (unsigned long)esp_get_minimum_free_heap_size(),
             (unsigned long)heap_caps_get_largest_free_block(MALLOC_CAP_8BIT));

    /* NVS */
    esp_err_t ret = nvs_flash_init();
    if (ret == ESP_ERR_NVS_NO_FREE_PAGES || ret == ESP_ERR_NVS_NEW_VERSION_FOUND) {
        ESP_ERROR_CHECK(nvs_flash_erase());
        ret = nvs_flash_init();
    }
    ESP_ERROR_CHECK(ret);
    ESP_LOGW(TAG, "HEAP after NVS: %lu free", (unsigned long)esp_get_free_heap_size());

    /* Initialize render context */
    render_context_init(&s_render_ctx);

    /* Turn LEDs off immediately — they latch the last frame across resets,
     * drawing current that can starve the WiFi radio during PHY cal. */
    ESP_ERROR_CHECK(led_driver_init());
    led_driver_clear();
    ESP_LOGW(TAG, "LEDs cleared. heap=%lu", (unsigned long)esp_get_free_heap_size());

    /* Now start WiFi — LEDs are dark so the radio gets clean power */
    ESP_LOGW(TAG, "Starting WiFi manager...");
    wifi_manager_start();
    wifi_manager_set_callback(WM_EVENT_STA_GOT_IP, &cb_wifi_got_ip);

    ESP_LOGW(TAG, "Waiting for WiFi. heap=%lu",
             (unsigned long)esp_get_free_heap_size());

    /* Poll for WiFi connection (flag set by lightweight callback) */
    int wifi_wait = 0;
    while (!s_wifi_connected) {
        vTaskDelay(pdMS_TO_TICKS(1000));
        wifi_wait++;
        if (wifi_wait % 5 == 0) {
            ESP_LOGW(TAG, "Still waiting for WiFi... (%ds)", wifi_wait);
        }
    }

    ESP_LOGW(TAG, "WiFi connected. heap=%lu",
             (unsigned long)esp_get_free_heap_size());

    /* OTA — WiFi radio is stable */
    ghota_config_t ghota_config = {
        .filenamematch = "firmware.bin",
        .storagenamematch = "",
        .storagepartitionname = "",
        .hostname = "api.github.com",
        .orgname = "ProjectBarks",
        .reponame = "subway-pcb",
        .updateInterval = OTA_CHECK_INTERVAL_MIN,
    };
    s_ghota = ghota_init(&ghota_config);
    if (s_ghota) {
        esp_event_handler_register(GHOTA_EVENTS, ESP_EVENT_ANY_ID, &ghota_event_handler, NULL);
        ghota_start_update_timer(s_ghota);
        ESP_LOGI(TAG, "OTA checker started (every %d min)", OTA_CHECK_INTERVAL_MIN);
    }

    /* Start state polling + Lua render */
    state_client_start(&s_render_ctx);
    lua_runtime_start(&s_render_ctx);

    ESP_LOGW(TAG, "All services running. heap=%lu min=%lu",
             (unsigned long)esp_get_free_heap_size(),
             (unsigned long)esp_get_minimum_free_heap_size());
}
