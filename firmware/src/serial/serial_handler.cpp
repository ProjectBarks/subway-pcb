#include "serial_handler.hpp"

#include "cJSON.h"

#include <cstdio>
#include <cstring>
#include <driver/uart.h>
#include <esp_mac.h>
#include <esp_random.h>
#include <esp_spiffs.h>
#include <esp_system.h>
#include <esp_timer.h>
#include <esp_wifi.h>
#include <nvs.h>
#include <nvs_flash.h>

extern "C" {
void wifi_manager_scan_async();
bool wifi_manager_lock_json_buffer(uint32_t ticks);
void wifi_manager_unlock_json_buffer();
char* wifi_manager_get_ap_list_json();
}

#include "../config/constants.hpp"
#include "../log/device_log.hpp"

#include <esp_log.h>
#include <rom/uart.h>

static const char* TAG = "serial";

// ---------------------------------------------------------------------------
// Static state
// ---------------------------------------------------------------------------
enum class TransferState { kIdle, kReceivingScript };

static TransferState s_transfer_state = TransferState::kIdle;
static char* s_script_buf = nullptr;
static size_t s_script_len = 0;
static size_t s_script_alloc = 0;
static uint32_t s_transfer_start_ms = 0;

static char s_factory_nonce[8] = {};
static uint32_t s_nonce_timestamp_ms = 0;

static char s_line_buf[kSerialLineBufSize];
static size_t s_line_pos = 0;

// ---------------------------------------------------------------------------
// Context pointer (set once at task start)
// ---------------------------------------------------------------------------
static SerialHandler::Context s_ctx;

// ---------------------------------------------------------------------------
// Response helper
// ---------------------------------------------------------------------------
static void send_response(bool ok,
                          const char* cmd,
                          uint16_t seq,
                          const char* data_json,
                          const char* error = nullptr,
                          const char* code = nullptr) {
    cJSON* root = cJSON_CreateObject();
    if (!root)
        return;

    cJSON_AddBoolToObject(root, "ok", ok);
    cJSON_AddStringToObject(root, "cmd", cmd);
    cJSON_AddNumberToObject(root, "seq", seq);

    if (data_json) {
        cJSON* parsed = cJSON_Parse(data_json);
        if (parsed) {
            cJSON_AddItemToObject(root, "data", parsed);
        } else {
            // Treat as a plain string
            cJSON_AddStringToObject(root, "data", data_json);
        }
    }

    if (error) {
        cJSON_AddStringToObject(root, "error", error);
    }
    if (code) {
        cJSON_AddStringToObject(root, "code", code);
    }

    char* printed = cJSON_PrintUnformatted(root);
    cJSON_Delete(root);
    if (!printed)
        return;

    // Build complete response line in one buffer
    // Prepend \n to break out of any partial ESP_LOG line
    size_t json_len = strlen(printed);
    size_t total = 1 + 2 + json_len + 1; // \n + \x01> + json + \n
    char* line = (char*)malloc(total);
    if (line) {
        line[0] = '\n';
        line[1] = '\x01';
        line[2] = '>';
        memcpy(line + 3, printed, json_len);
        line[total - 1] = '\n';
        // Suppress logs, drain FIFO, then write response
        esp_log_level_set("*", ESP_LOG_NONE);
        vTaskDelay(pdMS_TO_TICKS(10));
        uart_write_bytes(UART_NUM_0, line, total);
        uart_wait_tx_done(UART_NUM_0, pdMS_TO_TICKS(200));
        esp_log_level_set("*", ESP_LOG_INFO);
        free(line);
    }

    cJSON_free(printed);
}

// ---------------------------------------------------------------------------
// NVS helpers
// ---------------------------------------------------------------------------
static bool nvs_write_str(const char* ns, const char* key, const char* value) {
    nvs_handle_t h;
    if (nvs_open(ns, NVS_READWRITE, &h) != ESP_OK)
        return false;
    esp_err_t err = nvs_set_str(h, key, value);
    if (err == ESP_OK)
        err = nvs_commit(h);
    nvs_close(h);
    return err == ESP_OK;
}

// Write a string as a fixed-size zero-padded NVS blob (matches wifi_manager format).
static bool nvs_write_blob(const char* ns, const char* key, const char* value, size_t blob_sz) {
    uint8_t buf[64] = {};
    size_t len = strlen(value);
    if (len >= blob_sz)
        len = blob_sz - 1;
    memcpy(buf, value, len);

    nvs_handle_t h;
    if (nvs_open(ns, NVS_READWRITE, &h) != ESP_OK)
        return false;
    esp_err_t err = nvs_set_blob(h, key, buf, blob_sz);
    if (err == ESP_OK)
        err = nvs_commit(h);
    nvs_close(h);
    return err == ESP_OK;
}

static bool nvs_read_str(const char* ns, const char* key, char* buf, size_t buf_len) {
    nvs_handle_t h;
    if (nvs_open(ns, NVS_READONLY, &h) != ESP_OK) {
        buf[0] = '\0';
        return false;
    }
    size_t len = buf_len;
    // Try string first, fall back to blob (wifi_manager stores as blob).
    esp_err_t err = nvs_get_str(h, key, buf, &len);
    if (err == ESP_ERR_NVS_TYPE_MISMATCH) {
        len = buf_len;
        err = nvs_get_blob(h, key, buf, &len);
        if (err == ESP_OK && len < buf_len)
            buf[len] = '\0';
    }
    nvs_close(h);
    if (err != ESP_OK) {
        buf[0] = '\0';
        return false;
    }
    return true;
}

// ---------------------------------------------------------------------------
// Command handlers
// ---------------------------------------------------------------------------
static void
handle_ping(const char* /*value*/, uint16_t seq, const SerialHandler::Context& /*ctx*/) {
    // Build data object inline
    char data[64];
    snprintf(data,
             sizeof(data),
             "{\"pong\":true,\"v\":%d,\"fw\":\"%s\"}",
             kSerialProtocolVersion,
             kFirmwareVersion);
    send_response(true, "PING", seq, data);
}

static void
handle_get_info(const char* /*value*/, uint16_t seq, const SerialHandler::Context& ctx) {
    uint8_t mac[6] = {};
    esp_read_mac(mac, ESP_MAC_WIFI_STA);

    char mac_str[18];
    snprintf(mac_str,
             sizeof(mac_str),
             "%02X:%02X:%02X:%02X:%02X:%02X",
             mac[0],
             mac[1],
             mac[2],
             mac[3],
             mac[4],
             mac[5]);

    wifi_ap_record_t ap;
    bool wifi_connected = (esp_wifi_sta_get_ap_info(&ap) == ESP_OK);

    char ssid_str[33] = {};
    if (wifi_connected) {
        memcpy(ssid_str, ap.ssid, sizeof(ap.ssid));
    }

    cJSON* data = cJSON_CreateObject();
    if (!data)
        return;

    cJSON_AddStringToObject(data, "mac", mac_str);
    cJSON_AddStringToObject(data, "fw", kFirmwareVersion);
    cJSON_AddNumberToObject(data, "proto", kSerialProtocolVersion);
    cJSON_AddNumberToObject(data, "heap", (double)esp_get_free_heap_size());
    cJSON_AddNumberToObject(data, "heap_min", (double)esp_get_minimum_free_heap_size());
    cJSON_AddBoolToObject(data, "wifi", wifi_connected);
    if (wifi_connected) {
        cJSON_AddStringToObject(data, "ssid", ssid_str);
    }
    cJSON_AddNumberToObject(data, "leds", ctx.led_driver->led_count());
    cJSON_AddNumberToObject(data, "strips", ctx.hw_config->num_strips);
    cJSON_AddNumberToObject(data, "reset_reason", (double)esp_reset_reason());
    cJSON_AddNumberToObject(data, "uptime_ms", (double)(uint32_t)(esp_timer_get_time() / 1000));

    char* printed = cJSON_PrintUnformatted(data);
    cJSON_Delete(data);
    if (!printed)
        return;

    send_response(true, "GET INFO", seq, printed);
    cJSON_free(printed);
}

static void
handle_get_wifi(const char* /*value*/, uint16_t seq, const SerialHandler::Context& /*ctx*/) {
    wifi_ap_record_t ap;
    bool connected = (esp_wifi_sta_get_ap_info(&ap) == ESP_OK);

    char ssid_buf[33] = {};
    char ip_str[16] = "0.0.0.0";
    int8_t rssi = 0;

    if (connected) {
        memcpy(ssid_buf, ap.ssid, sizeof(ap.ssid));
        rssi = ap.rssi;

        esp_netif_t* netif = esp_netif_get_handle_from_ifkey("WIFI_STA_DEF");
        if (netif) {
            esp_netif_ip_info_t ip_info;
            if (esp_netif_get_ip_info(netif, &ip_info) == ESP_OK) {
                snprintf(ip_str, sizeof(ip_str), IPSTR, IP2STR(&ip_info.ip));
            }
        }
    }

    // Read stored SSID from NVS
    char stored_ssid[33] = {};
    nvs_read_str("wifi_manager", "ssid", stored_ssid, sizeof(stored_ssid));

    cJSON* data = cJSON_CreateObject();
    if (!data)
        return;

    cJSON_AddStringToObject(data, "ssid", connected ? ssid_buf : stored_ssid);
    cJSON_AddBoolToObject(data, "connected", connected);
    cJSON_AddStringToObject(data, "ip", ip_str);
    cJSON_AddNumberToObject(data, "rssi", rssi);

    char* printed = cJSON_PrintUnformatted(data);
    cJSON_Delete(data);
    if (!printed)
        return;

    send_response(true, "GET WIFI", seq, printed);
    cJSON_free(printed);
}

static void
handle_get_diag(const char* /*value*/, uint16_t seq, const SerialHandler::Context& ctx) {
    DiagPad* d = ctx.diag;
    uint32_t frame_us = d->frame_time_us.load(std::memory_order_relaxed);
    float fps = (frame_us > 0) ? (1000000.0f / (float)frame_us) : 0.0f;

    cJSON* data = cJSON_CreateObject();
    if (!data)
        return;

    cJSON_AddNumberToObject(data, "heap", (double)esp_get_free_heap_size());
    cJSON_AddNumberToObject(data, "heap_min", (double)esp_get_minimum_free_heap_size());
    cJSON_AddNumberToObject(data, "lua_errors", d->lua_errors.load(std::memory_order_relaxed));
    cJSON_AddNumberToObject(data, "lua_mem", d->lua_mem.load(std::memory_order_relaxed));
    cJSON_AddNumberToObject(data, "render_fps", fps);
    cJSON_AddNumberToObject(data, "nonzero_px", d->nonzero_pixels.load(std::memory_order_relaxed));
    cJSON_AddNumberToObject(data, "uptime_ms", (double)(uint32_t)(esp_timer_get_time() / 1000));

    char* printed = cJSON_PrintUnformatted(data);
    cJSON_Delete(data);
    if (!printed)
        return;

    send_response(true, "GET DIAG", seq, printed);
    cJSON_free(printed);
}

static void
handle_set_wifi_ssid(const char* value, uint16_t seq, const SerialHandler::Context& /*ctx*/) {
    if (!value || value[0] == '\0') {
        send_response(false, "SET WIFI_SSID", seq, nullptr, "missing value", "ERR_MISSING_VALUE");
        return;
    }
    if (nvs_write_blob("wifi_manager", "ssid", value, 32)) {
        send_response(true, "SET WIFI_SSID", seq, "\"ok\"");
    } else {
        send_response(false, "SET WIFI_SSID", seq, nullptr, "nvs write failed", "ERR_NVS");
    }
}

static void
handle_set_wifi_pass(const char* value, uint16_t seq, const SerialHandler::Context& /*ctx*/) {
    if (!value || value[0] == '\0') {
        send_response(false, "SET WIFI_PASS", seq, nullptr, "missing value", "ERR_MISSING_VALUE");
        return;
    }
    if (nvs_write_blob("wifi_manager", "password", value, 64)) {
        send_response(true, "SET WIFI_PASS", seq, "\"ok\"");
    } else {
        send_response(false, "SET WIFI_PASS", seq, nullptr, "nvs write failed", "ERR_NVS");
    }
}

static void
handle_set_server_url(const char* value, uint16_t seq, const SerialHandler::Context& /*ctx*/) {
    if (!value || value[0] == '\0') {
        send_response(false, "SET SERVER_URL", seq, nullptr, "missing value", "ERR_MISSING_VALUE");
        return;
    }
    if (strlen(value) >= kServerUrlMaxLen) {
        send_response(false, "SET SERVER_URL", seq, nullptr, "url too long", "ERR_TOO_LONG");
        return;
    }
    if (nvs_write_str("subway", "server_url", value)) {
        send_response(true, "SET SERVER_URL", seq, "\"ok\"");
    } else {
        send_response(false, "SET SERVER_URL", seq, nullptr, "nvs write failed", "ERR_NVS");
    }
}

static void
handle_wifi_scan(const char* /*value*/, uint16_t seq, const SerialHandler::Context& /*ctx*/) {
    // Trigger scan through wifi-manager (which owns the SCAN_DONE handler).
    // Direct esp_wifi_scan_start returns empty because the wifi-manager
    // consumes results in its SCAN_DONE callback before we can read them.
    wifi_manager_scan_async();
    vTaskDelay(pdMS_TO_TICKS(5000)); // wait for scan + processing

    // Read cached JSON directly — no parsing needed, just pass through
    if (wifi_manager_lock_json_buffer(pdMS_TO_TICKS(1000))) {
        char* ap_json = wifi_manager_get_ap_list_json();
        if (ap_json && ap_json[0] != '\0') {
            send_response(true, "DO WIFI_SCAN", seq, ap_json);
        } else {
            send_response(true, "DO WIFI_SCAN", seq, "[]");
        }
        wifi_manager_unlock_json_buffer();
    } else {
        send_response(false, "DO WIFI_SCAN", seq, nullptr, "scan busy", "BUSY");
    }
}

static void
handle_wifi_apply(const char* /*value*/, uint16_t seq, const SerialHandler::Context& /*ctx*/) {
    send_response(true, "DO WIFI_APPLY", seq, "\"reconnecting\"");
    DLOG_I(TAG, "Disconnecting WiFi for reconnect...");
    esp_wifi_disconnect();
    vTaskDelay(pdMS_TO_TICKS(500));
    esp_wifi_connect();
}

static void
handle_led_test(const char* /*value*/, uint16_t seq, const SerialHandler::Context& ctx) {
    // Send a brief white flash by pushing a test script to LuaRuntime.
    // We can't directly write pixels because the render task also writes
    // to the pixel buffer every frame — that's a data race.
    // Instead, push a short script that flashes white for 0.5s then reverts.
    static const char* test_script = "local t = get_time()\n"
                                     "function render()\n"
                                     "  if get_time() - t < 0.5 then\n"
                                     "    for i = 1, led_count() do set_led(i, 255, 255, 255) end\n"
                                     "  else\n"
                                     "    for i = 1, led_count() do set_led(i, 0, 0, 0) end\n"
                                     "  end\n"
                                     "end\n";
    size_t len = strlen(test_script);
    char* buf = (char*)malloc(len + 1);
    if (buf) {
        memcpy(buf, test_script, len + 1);
        ctx.script_chan->send(buf); // ownership transferred
    }
    send_response(true, "DO LED_TEST", seq, "\"flashing\"");
}

static void
handle_reboot(const char* /*value*/, uint16_t seq, const SerialHandler::Context& /*ctx*/) {
    send_response(true, "DO REBOOT", seq, "\"rebooting\"");
    vTaskDelay(pdMS_TO_TICKS(200)); // give UART time to flush
    esp_restart();
}

static void
handle_factory_reset(const char* value, uint16_t seq, const SerialHandler::Context& /*ctx*/) {
    uint32_t now_ms = (uint32_t)(esp_timer_get_time() / 1000);

    // Step 1: No nonce provided -- generate one
    if (!value || value[0] == '\0') {
        uint32_t rnd = esp_random();
        snprintf(s_factory_nonce, sizeof(s_factory_nonce), "%04X", (unsigned)(rnd & 0xFFFF));
        s_nonce_timestamp_ms = now_ms;

        char data[64];
        snprintf(data,
                 sizeof(data),
                 "{\"nonce\":\"%s\",\"ttl\":%d}",
                 s_factory_nonce,
                 kFactoryResetNonceTtlSec);
        send_response(true, "DO FACTORY_RESET", seq, data);
        return;
    }

    // Step 2: Nonce provided -- verify
    if (s_factory_nonce[0] == '\0') {
        send_response(false, "DO FACTORY_RESET", seq, nullptr, "no pending nonce", "ERR_NO_NONCE");
        return;
    }
    if ((now_ms - s_nonce_timestamp_ms) > (uint32_t)(kFactoryResetNonceTtlSec * 1000)) {
        s_factory_nonce[0] = '\0';
        send_response(
            false, "DO FACTORY_RESET", seq, nullptr, "nonce expired", "ERR_NONCE_EXPIRED");
        return;
    }
    if (strcmp(value, s_factory_nonce) != 0) {
        s_factory_nonce[0] = '\0';
        send_response(
            false, "DO FACTORY_RESET", seq, nullptr, "nonce mismatch", "ERR_NONCE_MISMATCH");
        return;
    }

    // Nonce valid -- erase NVS and SPIFFS, reboot
    s_factory_nonce[0] = '\0';
    DLOG_W(TAG, "Factory reset confirmed -- erasing NVS + SPIFFS");

    // Delete local script
    remove("/scripts/local.lua");

    // Erase NVS
    nvs_flash_erase();

    send_response(true, "DO FACTORY_RESET", seq, "\"erased, rebooting\"");
    vTaskDelay(pdMS_TO_TICKS(200));
    esp_restart();
}

static void
handle_script_begin(const char* /*value*/, uint16_t seq, const SerialHandler::Context& /*ctx*/) {
    if (s_transfer_state == TransferState::kReceivingScript) {
        // Abort previous incomplete transfer
        free(s_script_buf);
        s_script_buf = nullptr;
        s_script_len = 0;
    }

    // Start with a 4KB buffer and realloc if needed during transfer
    static constexpr size_t kInitScriptBuf = 4096;
    s_script_buf = (char*)malloc(kInitScriptBuf);
    if (!s_script_buf) {
        send_response(false, "DO SCRIPT_BEGIN", seq, nullptr, "alloc failed", "ERR_OOM");
        return;
    }
    s_script_alloc = kInitScriptBuf;

    s_script_len = 0;
    s_script_buf[0] = '\0';
    s_transfer_state = TransferState::kReceivingScript;
    s_transfer_start_ms = (uint32_t)(esp_timer_get_time() / 1000);

    DLOG_I(TAG, "Script transfer started (buf=%u)", (unsigned)s_script_alloc);
    char resp[64];
    snprintf(resp,
             sizeof(resp),
             "{\"max_size\":%d,\"timeout\":%d}",
             (int)kScriptMaxSize,
             kScriptTransferTimeoutSec);
    send_response(true, "DO SCRIPT_BEGIN", seq, resp);
}

static void
handle_script_clear(const char* /*value*/, uint16_t seq, const SerialHandler::Context& /*ctx*/) {
    if (remove("/scripts/local.lua") == 0) {
        DLOG_I(TAG, "Deleted /scripts/local.lua");
        send_response(true, "DO SCRIPT_CLEAR", seq, "\"deleted\"");
    } else {
        // File might not exist, which is fine
        send_response(true, "DO SCRIPT_CLEAR", seq, "\"no_file\"");
    }
}

// ---------------------------------------------------------------------------
// Dispatch table
// ---------------------------------------------------------------------------
struct CommandEntry {
    const char* verb;     // "GET", "SET", "DO", or nullptr for bare commands
    const char* resource; // "INFO", "WIFI_SSID", etc.
    void (*handler)(const char* value, uint16_t seq, const SerialHandler::Context& ctx);
};

static const CommandEntry s_commands[] = {
    {nullptr, "PING", handle_ping},
    {"GET", "INFO", handle_get_info},
    {"GET", "WIFI", handle_get_wifi},
    {"GET", "DIAG", handle_get_diag},
    {"SET", "WIFI_SSID", handle_set_wifi_ssid},
    {"SET", "WIFI_PASS", handle_set_wifi_pass},
    {"SET", "SERVER_URL", handle_set_server_url},
    {"DO", "WIFI_SCAN", handle_wifi_scan},
    {"DO", "WIFI_APPLY", handle_wifi_apply},
    {"DO", "LED_TEST", handle_led_test},
    {"DO", "REBOOT", handle_reboot},
    {"DO", "FACTORY_RESET", handle_factory_reset},
    {"DO", "SCRIPT_BEGIN", handle_script_begin},
    {"DO", "SCRIPT_CLEAR", handle_script_clear},
};
static constexpr int kNumCommands = sizeof(s_commands) / sizeof(s_commands[0]);

// ---------------------------------------------------------------------------
// Script transfer mode handler
// ---------------------------------------------------------------------------
static void handle_transfer_line(const char* line) {
    // Check for SCRIPT_END (with optional " #seq" or "#seq")
    uint16_t seq = 0;
    bool is_end = false;

    if (strncmp(line, "SCRIPT_END", 10) == 0) {
        const char* rest = line + 10;
        if (*rest == '\0') {
            is_end = true;
        } else if (*rest == '#') {
            is_end = true;
            seq = (uint16_t)atoi(rest + 1);
        } else if (*rest == ' ' && *(rest + 1) == '#') {
            is_end = true;
            seq = (uint16_t)atoi(rest + 2);
        }
    }

    if (is_end) {
        // Save to SPIFFS
        s_script_buf[s_script_len] = '\0';

        FILE* f = fopen("/scripts/local.lua", "w");
        if (!f) {
            send_response(false, "SCRIPT_END", seq, nullptr, "spiffs write failed", "ERR_SPIFFS");
            free(s_script_buf);
            s_script_buf = nullptr;
            s_script_len = 0;
            s_transfer_state = TransferState::kIdle;
            return;
        }

        fwrite(s_script_buf, 1, s_script_len, f);
        fclose(f);

        DLOG_I(TAG, "Script saved: %u bytes", (unsigned)s_script_len);

        // Push to ScriptChannel (transfer ownership of a copy)
        char* copy = (char*)malloc(s_script_len + 1);
        if (copy) {
            memcpy(copy, s_script_buf, s_script_len + 1);
            s_ctx.script_chan->send(copy);
        }

        char data[64];
        snprintf(data, sizeof(data), "{\"size\":%u}", (unsigned)s_script_len);
        send_response(true, "SCRIPT_END", seq, data);

        free(s_script_buf);
        s_script_buf = nullptr;
        s_script_len = 0;
        s_transfer_state = TransferState::kIdle;
        return;
    }

    // Accumulate line into buffer
    size_t line_len = strlen(line);
    size_t needed = s_script_len + line_len + 2; // +1 for \n, +1 for \0
    if (needed > kScriptMaxSize) {
        send_response(false, "SCRIPT_END", 0, nullptr, "script too large", "ERR_TOO_LARGE");
        free(s_script_buf);
        s_script_buf = nullptr;
        s_script_len = 0;
        s_script_alloc = 0;
        s_transfer_state = TransferState::kIdle;
        return;
    }

    // Grow buffer if needed (doubles up to kScriptMaxSize)
    if (needed > s_script_alloc) {
        size_t new_alloc = s_script_alloc * 2;
        if (new_alloc < needed)
            new_alloc = needed;
        if (new_alloc > kScriptMaxSize)
            new_alloc = kScriptMaxSize;
        char* new_buf = (char*)realloc(s_script_buf, new_alloc);
        if (!new_buf) {
            send_response(
                false, "SCRIPT_END", 0, nullptr, "alloc failed during transfer", "ERR_OOM");
            free(s_script_buf);
            s_script_buf = nullptr;
            s_script_len = 0;
            s_script_alloc = 0;
            s_transfer_state = TransferState::kIdle;
            return;
        }
        s_script_buf = new_buf;
        s_script_alloc = new_alloc;
    }

    if (s_script_len > 0) {
        s_script_buf[s_script_len++] = '\n';
    }
    memcpy(s_script_buf + s_script_len, line, line_len);
    s_script_len += line_len;
}

// ---------------------------------------------------------------------------
// Command parsing and dispatch
// ---------------------------------------------------------------------------
static void dispatch_command(const char* line) {
    // Strip \r if present
    char buf[kSerialLineBufSize];
    strncpy(buf, line, sizeof(buf) - 1);
    buf[sizeof(buf) - 1] = '\0';

    size_t len = strlen(buf);
    while (len > 0 && (buf[len - 1] == '\r' || buf[len - 1] == '\n')) {
        buf[--len] = '\0';
    }
    if (len == 0)
        return;

    // Extract #seq suffix
    uint16_t seq = 0;
    char* hash_pos = strrchr(buf, '#');
    if (hash_pos) {
        seq = (uint16_t)atoi(hash_pos + 1);
        *hash_pos = '\0';
        len = strlen(buf);
        // Trim trailing space before #
        while (len > 0 && buf[len - 1] == ' ') {
            buf[--len] = '\0';
        }
    }

    // Split into tokens: verb [resource [value...]]
    char* saveptr = nullptr;
    char* token1 = strtok_r(buf, " ", &saveptr);
    if (!token1)
        return;

    char* token2 = strtok_r(nullptr, " ", &saveptr);
    // Remainder is the value (may contain spaces)
    char* value = saveptr; // points to rest of string after token2

    // Try bare command first (verb=nullptr, resource=token1)
    for (int i = 0; i < kNumCommands; i++) {
        const auto& entry = s_commands[i];
        if (entry.verb == nullptr && strcasecmp(entry.resource, token1) == 0) {
            entry.handler(token2, seq, s_ctx);
            return;
        }
    }

    // Try verb+resource
    if (token2) {
        for (int i = 0; i < kNumCommands; i++) {
            const auto& entry = s_commands[i];
            if (entry.verb && strcasecmp(entry.verb, token1) == 0 &&
                strcasecmp(entry.resource, token2) == 0) {
                entry.handler(value, seq, s_ctx);
                return;
            }
        }
    }

    // Unknown command
    char err_msg[128];
    snprintf(err_msg,
             sizeof(err_msg),
             "unknown command: %s%s%s",
             token1,
             token2 ? " " : "",
             token2 ? token2 : "");
    send_response(false, token1, seq, nullptr, err_msg, "ERR_UNKNOWN_CMD");
}

// ---------------------------------------------------------------------------
// Main task
// ---------------------------------------------------------------------------
static void serial_task(void* /*param*/) {
    // Configure UART
    uart_config_t uart_cfg = {};
    uart_cfg.baud_rate = 115200;
    uart_cfg.data_bits = UART_DATA_8_BITS;
    uart_cfg.parity = UART_PARITY_DISABLE;
    uart_cfg.stop_bits = UART_STOP_BITS_1;
    uart_cfg.flow_ctrl = UART_HW_FLOWCTRL_DISABLE;
    uart_cfg.source_clk = UART_SCLK_DEFAULT;

    // Suppress noisy log tags that flood UART0 and interleave with responses
    esp_log_level_set("gpio", ESP_LOG_NONE);
    esp_log_level_set("wifi_manager", ESP_LOG_WARN);
    esp_log_level_set("wifi", ESP_LOG_WARN);
    esp_log_level_set("wifi_init", ESP_LOG_WARN);

    esp_err_t err;
    err = uart_param_config(UART_NUM_0, &uart_cfg);
    DLOG_I(TAG, "uart_param_config: %s", esp_err_to_name(err));

    // Delete existing driver if present (console may have installed one)
    uart_driver_delete(UART_NUM_0);

    err = uart_driver_install(UART_NUM_0, 1024, 1024, 0, nullptr, 0);
    DLOG_I(TAG, "uart_driver_install: %s", esp_err_to_name(err));

    DLOG_I(TAG, "Serial handler ready");

    s_line_pos = 0;
    uint8_t byte;

    while (true) {
        int n = uart_read_bytes(UART_NUM_0, &byte, 1, pdMS_TO_TICKS(100));

        // Check transfer timeout
        if (s_transfer_state == TransferState::kReceivingScript) {
            uint32_t elapsed_ms = (uint32_t)(esp_timer_get_time() / 1000) - s_transfer_start_ms;
            if (elapsed_ms > (uint32_t)(kScriptTransferTimeoutSec * 1000)) {
                DLOG_W(TAG, "Script transfer timed out");
                send_response(false, "SCRIPT_END", 0, nullptr, "transfer timeout", "ERR_TIMEOUT");
                free(s_script_buf);
                s_script_buf = nullptr;
                s_script_len = 0;
                s_transfer_state = TransferState::kIdle;
            }
        }

        if (n <= 0)
            continue;

        // Accumulate into line buffer
        if (byte == '\n') {
            s_line_buf[s_line_pos] = '\0';

            if (s_line_pos > 0) {
                if (s_transfer_state == TransferState::kReceivingScript) {
                    handle_transfer_line(s_line_buf);
                } else {
                    dispatch_command(s_line_buf);
                }
            }

            s_line_pos = 0;
        } else if (byte != '\r') {
            if (s_line_pos < sizeof(s_line_buf) - 1) {
                s_line_buf[s_line_pos++] = (char)byte;
            }
            // else: line too long, silently drop extra bytes
        }
    }
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------
void SerialHandler::start(const Context& ctx) {
    s_ctx = ctx;

    xTaskCreate(serial_task, "serial", 4096, nullptr, 2, nullptr);
    DLOG_I(TAG, "Serial handler task started");
}
