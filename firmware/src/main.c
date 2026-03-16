#include <stdio.h>
#include <string.h>

#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "esp_system.h"
#include "esp_log.h"
#include "esp_wifi.h"
#include "esp_netif.h"
#include "nvs_flash.h"

#include "wifi_manager.h"

#include "config.h"
#include "led_driver.h"
#include "subway_client.h"

static const char *TAG = "main";

static void cb_wifi_got_ip(void *pvParameter)
{
    ESP_LOGI(TAG, "WiFi connected — starting subway client");
    led_driver_set_pixel(0, 0, 0, 30, 0); /* green = connected */
    led_driver_refresh();
    subway_client_start();
}

void app_main(void)
{
    ESP_LOGI(TAG, "=== NYC Subway PCB ===");

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

    /* WiFi via captive portal — broadcasts "nyc-subway-pcb" AP if no saved credentials */
    wifi_manager_start();
    wifi_manager_set_callback(WM_EVENT_STA_GOT_IP, &cb_wifi_got_ip);

    /* Blue blink = waiting for WiFi config/connection */
    led_driver_set_pixel(0, 0, 0, 0, 30);
    led_driver_refresh();

    ESP_LOGI(TAG, "Waiting for WiFi (connect to 'nyc-subway-pcb' AP to configure)");
}
