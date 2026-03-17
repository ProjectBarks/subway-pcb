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
#include "subway_client.h"

static const char *TAG = "main";
static ghota_client_handle_t *s_ghota = NULL;

/* Global flag: when true, subway_client pauses to give OTA exclusive HTTPS access */
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

static void cb_wifi_got_ip(void *pvParameter)
{
    ESP_LOGI(TAG, "WiFi connected — starting subway client + OTA");
    led_driver_set_pixel(0, 0, 0, 30, 0); /* green = connected */
    led_driver_refresh();

    subway_client_start();

    /* Start OTA update checker — register events AFTER wifi_manager creates the event loop */
    if (s_ghota) {
        esp_event_handler_register(GHOTA_EVENTS, ESP_EVENT_ANY_ID, &ghota_event_handler, NULL);
        ghota_start_update_timer(s_ghota);
        ESP_LOGI(TAG, "OTA checker started (every %d min, repo: ProjectBarks/subway-pcb)", OTA_CHECK_INTERVAL_MIN);
    } else {
        ESP_LOGE(TAG, "OTA not initialized — skipping update checker");
    }
}

void app_main(void)
{
    ESP_LOGI(TAG, "=== NYC Subway PCB v0.5.0 ===");

    /* NVS */
    esp_err_t ret = nvs_flash_init();
    if (ret == ESP_ERR_NVS_NO_FREE_PAGES || ret == ESP_ERR_NVS_NEW_VERSION_FOUND) {
        ESP_ERROR_CHECK(nvs_flash_erase());
        ret = nvs_flash_init();
    }
    ESP_ERROR_CHECK(ret);

    /* LED strips */
    ESP_ERROR_CHECK(led_driver_init());
    led_driver_set_pixel(0, 0, 30, 0, 0); /* red = booting */
    led_driver_refresh();

    /* Init OTA — checks GitHub releases for firmware updates */
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
        ESP_LOGI(TAG, "OTA initialized (ProjectBarks/subway-pcb, match: firmware.bin)");
    } else {
        ESP_LOGE(TAG, "OTA init FAILED — version may not be valid semver");
    }

    /* WiFi via captive portal — broadcasts "nyc-subway-pcb" AP if no saved credentials */
    wifi_manager_start();
    wifi_manager_set_callback(WM_EVENT_STA_GOT_IP, &cb_wifi_got_ip);

    /* Blue blink = waiting for WiFi config/connection */
    led_driver_set_pixel(0, 0, 0, 0, 30);
    led_driver_refresh();

    ESP_LOGI(TAG, "Waiting for WiFi (connect to 'nyc-subway-pcb' AP to configure)");
}
