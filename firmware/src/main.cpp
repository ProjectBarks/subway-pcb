#include "config/board_config.hpp"
#include "config/constants.hpp"
#include "core/channel.hpp"
#include "core/triple_buffer.hpp"
#include "data/board_snapshot.hpp"
#include "data/diag_pad.hpp"
#include "data/transit_snapshot.hpp"
#include "esp_ghota.h"
#include "esp_heap_caps.h"
#include "esp_log.h"
#include "esp_netif.h"
#include "esp_spiffs.h"
#include "esp_system.h"
#include "esp_wifi.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "hal/led_driver.hpp"
#include "log/device_log.hpp"
#include "net/state_client.hpp"
#include "nvs_flash.h"
#include "script/lua_runtime.hpp"
#include "serial/serial_handler.hpp"
#include "wifi_manager.h"

#include <atomic>
#include <cstdio>
#include <cstring>

static const char* TAG = "main";

// All shared state — static in file scope, process lifetime
static BoardHwConfig s_hw_config;
static LedDriver s_led_driver;
static DoubleBuffer<TransitSnapshot> s_transit_buf;
static BoardStore s_board_store;
static ScriptChannel s_script_chan;
static DiagPad s_diag;
static std::atomic<bool> s_ota_active{false};
static std::atomic<bool> s_http_active{false};

static ghota_client_handle_t* s_ghota = nullptr;

static void ghota_event_handler(void* arg, esp_event_base_t base, int32_t id, void* data) {
    ghota_event_e event = static_cast<ghota_event_e>(id);
    DLOG_I(TAG, "OTA event: %s", ghota_get_event_str(event));

    switch (event) {
    case GHOTA_EVENT_START_CHECK:
    case GHOTA_EVENT_START_UPDATE:
        s_ota_active.store(true, std::memory_order_release);
        DLOG_W(TAG, "Free heap: %lu bytes", (unsigned long)esp_get_free_heap_size());
        break;
    case GHOTA_EVENT_NOUPDATE_AVAILABLE:
    case GHOTA_EVENT_UPDATE_FAILED:
    case GHOTA_EVENT_FINISH_UPDATE:
        s_ota_active.store(false, std::memory_order_release);
        break;
    default:
        break;
    }
}

static std::atomic<bool> s_wifi_connected{false};

static void cb_wifi_got_ip(void* pvParameter) {
    DLOG_I(TAG, "WiFi connected");
    s_wifi_connected.store(true, std::memory_order_release);
}

static bool load_local_script(ScriptChannel& chan) {
    esp_vfs_spiffs_conf_t conf = {
        .base_path = "/scripts",
        .partition_label = "storage",
        .max_files = 2,
        .format_if_mount_failed = true,
    };
    esp_err_t ret = esp_vfs_spiffs_register(&conf);
    if (ret != ESP_OK) {
        DLOG_W(TAG, "SPIFFS mount failed: %s", esp_err_to_name(ret));
        return false;
    }

    FILE* f = fopen("/scripts/local.lua", "r");
    if (!f)
        return false;

    fseek(f, 0, SEEK_END);
    long size = ftell(f);
    fseek(f, 0, SEEK_SET);

    if (size <= 0 || size > (long)kScriptMaxSize) {
        fclose(f);
        return false;
    }

    char* buf = (char*)malloc(size + 1);
    if (!buf) {
        fclose(f);
        return false;
    }

    fread(buf, 1, size, f);
    buf[size] = '\0';
    fclose(f);

    DLOG_I(TAG, "Loaded local script: %ld bytes", size);
    chan.send(buf); // ownership transferred
    return true;
}

extern "C" void app_main(void) {
    DLOG_W(TAG,
           "=== NYC Subway PCB v%s === reset_reason=%d",
           kFirmwareVersion,
           (int)esp_reset_reason());
    DLOG_W(TAG,
           "HEAP: %lu free, %lu min, %lu largest block",
           (unsigned long)esp_get_free_heap_size(),
           (unsigned long)esp_get_minimum_free_heap_size(),
           (unsigned long)heap_caps_get_largest_free_block(MALLOC_CAP_8BIT));

    // 1. Initialize logging
    device_log_init();

    // 2. NVS (with erase recovery)
    esp_err_t ret = nvs_flash_init();
    if (ret == ESP_ERR_NVS_NO_FREE_PAGES || ret == ESP_ERR_NVS_NEW_VERSION_FOUND) {
        ESP_ERROR_CHECK(nvs_flash_erase());
        ret = nvs_flash_init();
    }
    ESP_ERROR_CHECK(ret);
    DLOG_W(TAG, "HEAP after NVS: %lu free", (unsigned long)esp_get_free_heap_size());

    // 3. Load board hardware config
    BoardHwConfig::load(s_hw_config);

    // 4. LEDs dark BEFORE WiFi (current draw)
    ESP_ERROR_CHECK(s_led_driver.init(&s_hw_config));
    s_led_driver.clear();
    DLOG_W(TAG, "LEDs cleared. heap=%lu", (unsigned long)esp_get_free_heap_size());

    // 5. Mount SPIFFS + load local script (if any)
    bool has_local_script = load_local_script(s_script_chan);
    if (has_local_script) {
        DLOG_I(TAG, "Local script queued for execution");
    }

    // 6. Start serial handler -- USB available immediately
    SerialHandler::start({
        .diag = &s_diag,
        .hw_config = &s_hw_config,
        .led_driver = &s_led_driver,
        .script_chan = &s_script_chan,
    });

    // 7. Initialize shared state mutexes
    s_diag.init();
    s_board_store.init();

    // 8. Start Lua render -- board renders immediately (with local script if loaded)
    LuaRuntime::start(
        s_transit_buf, s_board_store, s_script_chan, s_diag, s_http_active, s_led_driver);

    // 9. WiFi
    DLOG_W(TAG, "Starting WiFi manager...");
    wifi_manager_start();
    wifi_manager_set_callback(WM_EVENT_STA_GOT_IP, &cb_wifi_got_ip);

    // 10. Wait for WiFi with timeout (not infinite)
    DLOG_W(TAG,
           "Waiting for WiFi (timeout %ds). heap=%lu",
           kWifiTimeoutSec,
           (unsigned long)esp_get_free_heap_size());
    int wifi_wait = 0;
    while (!s_wifi_connected.load(std::memory_order_acquire)) {
        vTaskDelay(pdMS_TO_TICKS(1000));
        wifi_wait++;
        if (wifi_wait % 5 == 0) {
            DLOG_W(TAG, "Still waiting for WiFi... (%ds)", wifi_wait);
        }
        if (wifi_wait >= kWifiTimeoutSec) {
            DLOG_W(TAG, "WiFi timeout after %ds -- continuing without network", wifi_wait);
            break;
        }
    }

    bool wifi_ok = s_wifi_connected.load(std::memory_order_acquire);

    if (wifi_ok) {
        DLOG_W(TAG, "WiFi connected. heap=%lu", (unsigned long)esp_get_free_heap_size());

        // 11. OTA (only if WiFi connected)
        ghota_config_t ghota_config = {};
        std::strncpy(
            ghota_config.filenamematch, "firmware.bin", sizeof(ghota_config.filenamematch) - 1);
        ghota_config.hostname = const_cast<char*>("api.github.com");
        ghota_config.orgname = const_cast<char*>("ProjectBarks");
        ghota_config.reponame = const_cast<char*>("subway-pcb");
        ghota_config.updateInterval = kOtaCheckIntervalMin;
        s_ghota = ghota_init(&ghota_config);
        if (s_ghota) {
            esp_event_handler_register(
                GHOTA_EVENTS, ESP_EVENT_ANY_ID, &ghota_event_handler, nullptr);
            ghota_start_update_timer(s_ghota);
            DLOG_I(TAG, "OTA checker started (every %lu min)", (unsigned long)kOtaCheckIntervalMin);
        }

        // 12. State polling (only if WiFi connected)
        StateClient::start(
            s_transit_buf, s_board_store, s_script_chan, s_diag, s_ota_active, s_http_active);
    } else {
        DLOG_W(TAG, "Running in offline mode -- no OTA or state polling");
    }

    DLOG_W(TAG,
           "All services running. heap=%lu min=%lu wifi=%s",
           (unsigned long)esp_get_free_heap_size(),
           (unsigned long)esp_get_minimum_free_heap_size(),
           wifi_ok ? "yes" : "no");
}
