#include "net/state_client.hpp"

#include "config/constants.hpp"
#include "esp_heap_caps.h"
#include "esp_log.h"
#include "esp_mac.h"
#include "esp_system.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "log/device_log.hpp"
#include "nvs.h"
#include "nvs_flash.h"
#include "proto/codec.hpp"

#include <cstdio>
#include <cstdlib>
#include <cstring>

extern "C" {
#include "subway.pb.h"
}

static const char* TAG = "state_client";

// -----------------------------------------------------------------------
// Task argument bundle — passed via xTaskCreate, deleted after capture
// -----------------------------------------------------------------------
struct StateTaskArgs {
    DoubleBuffer<TransitSnapshot>* transit_buf;
    BoardStore* board_store;
    ScriptChannel* script_chan;
    DiagPad* diag;
    std::atomic<bool>* ota_active;
    std::atomic<bool>* http_active;
};

// Static decode buffer for state (24KB BSS — always needed, avoids heap churn)
static subway_DeviceState s_state_decode;

// -----------------------------------------------------------------------
// Helpers local to the task
// -----------------------------------------------------------------------

static void init_http_client(HttpClient& http, std::atomic<bool>& http_active) {
    // Read device MAC
    static char device_id[18];
    uint8_t mac[6];
    esp_read_mac(mac, ESP_MAC_WIFI_STA);
    std::snprintf(device_id,
                  sizeof(device_id),
                  "%02x:%02x:%02x:%02x:%02x:%02x",
                  mac[0],
                  mac[1],
                  mac[2],
                  mac[3],
                  mac[4],
                  mac[5]);

    // Read NVS config
    static char server_url[kServerUrlMaxLen];
    static char hardware[32];
    std::strncpy(server_url, kDefaultServerUrl, sizeof(server_url) - 1);
    server_url[sizeof(server_url) - 1] = '\0';
    std::strncpy(hardware, "nyc-subway/v1", sizeof(hardware) - 1);
    hardware[sizeof(hardware) - 1] = '\0';

    nvs_handle_t nvs;
    if (nvs_open("subway", NVS_READONLY, &nvs) == ESP_OK) {
        size_t len = sizeof(server_url);
        nvs_get_str(nvs, "server_url", server_url, &len);
        len = sizeof(hardware);
        nvs_get_str(nvs, "hardware", hardware, &len);
        nvs_close(nvs);
    }

    HttpClientConfig cfg{};
    cfg.server_url = server_url;
    cfg.device_id = device_id;
    cfg.firmware_ver = kFirmwareVersion;
    cfg.hardware = hardware;
    http.init(cfg, http_active);
}

// -----------------------------------------------------------------------
// Build diagnostics protobuf from DiagPad + system info
// -----------------------------------------------------------------------
static int build_diagnostics(DiagPad& diag,
                             bool last_state_ok,
                             bool board_fetched,
                             bool script_fetched,
                             const char last_server_script_hash[kHashLen],
                             const char cached_script_hash[kHashLen],
                             uint8_t* out_buf,
                             int out_buf_size) {
    // Static to keep ~2.3KB (mostly logs[2048]) off the 16KB task stack
    static subway_DeviceDiagnostics dd;
    std::memset(&dd, 0, sizeof(dd));

    dd.has_system = true;
    dd.system.reset_reason = static_cast<int32_t>(esp_reset_reason());
    dd.system.free_heap = static_cast<uint32_t>(esp_get_free_heap_size());
    dd.system.largest_block =
        static_cast<uint32_t>(heap_caps_get_largest_free_block(MALLOC_CAP_8BIT));

    dd.has_fetch = true;
    dd.fetch.state_ok = last_state_ok;
    dd.fetch.board_fetched = board_fetched;
    dd.fetch.script_fetched = script_fetched;
    // Does the server's script hash match our cached (fetched) script hash?
    dd.fetch.script_hash_match =
        (std::strncmp(last_server_script_hash, cached_script_hash, 64) == 0);

    dd.has_lua = true;
    dd.lua.errors = diag.lua_errors.load(std::memory_order_relaxed);
    dd.lua.mem = diag.lua_mem.load(std::memory_order_relaxed);
    dd.lua.last_reload = diag.last_reload.load(std::memory_order_relaxed);

    dd.has_render = true;
    dd.render.nonzero_pixels = diag.nonzero_pixels.load(std::memory_order_relaxed);
    dd.render.first_lit_led = diag.first_lit_led.load(std::memory_order_relaxed);

    // Copy error string (prefer lua error, fall back to fetch error)
    if (xSemaphoreTake(diag.str_mutex, pdMS_TO_TICKS(50)) == pdTRUE) {
        const char* err_src = diag.last_lua_err[0] ? diag.last_lua_err
                              : diag.fetch_err[0]  ? diag.fetch_err
                                                   : "";
        std::strncpy(dd.error, err_src, sizeof(dd.error) - 1);
        dd.error[sizeof(dd.error) - 1] = '\0';
        xSemaphoreGive(diag.str_mutex);
    }

    // Drain remote logs if enabled
    device_log_drain(dd.logs, sizeof(dd.logs));

    return codec::encode_diagnostics(dd, out_buf, out_buf_size);
}

// -----------------------------------------------------------------------
// FreeRTOS task function
// -----------------------------------------------------------------------
static void state_task(void* arg) {
    auto* args = static_cast<StateTaskArgs*>(arg);

    // Capture references from the heap-allocated args
    auto& transit_buf = *args->transit_buf;
    auto& board_store = *args->board_store;
    auto& script_chan = *args->script_chan;
    auto& diag_pad = *args->diag;
    auto* ota_active = args->ota_active;
    auto* http_active = args->http_active;
    delete args;

    // Init HTTP client — static so the 16KB buffer lives in BSS, not on the task stack
    static HttpClient http;
    init_http_client(http, *http_active);

    // Task-local state
    char cached_board_hash[kHashLen]{};
    char cached_script_hash[kHashLen]{};
    char last_server_script_hash[kHashLen]{}; // script_hash from last state response
    uint32_t current_board_gen = 0;
    bool board_fetched = false;
    bool script_fetched = false;
    bool last_state_ok = false;

    // Private config cache for remote_log check (read from previous cycle)
    subway_DeviceState_ConfigEntry last_config[kMaxConfig]{};
    pb_size_t last_config_count = 0;

    while (true) {
        // Yield while OTA is in progress
        if (ota_active->load(std::memory_order_acquire)) {
            vTaskDelay(pdMS_TO_TICKS(kPollIntervalMs));
            continue;
        }

        // Check remote logging config from *previous* cycle's config cache
        for (pb_size_t i = 0; i < last_config_count; i++) {
            if (std::strcmp(last_config[i].key, "remote_log") == 0) {
                device_log_set_remote(std::strcmp(last_config[i].value, "1") == 0);
                break;
            }
        }

        // ---- Build & send diagnostics, receive state ----
        uint8_t diag_buf[512];
        int diag_len = build_diagnostics(diag_pad,
                                         last_state_ok,
                                         board_fetched,
                                         script_fetched,
                                         last_server_script_hash,
                                         cached_script_hash,
                                         diag_buf,
                                         sizeof(diag_buf));

        bool state_ok = false;
        for (uint32_t attempt = 0; attempt < kHttpMaxRetries; attempt++) {
            HttpResponse resp{};
            if (http.post("/api/v1/device-state", diag_buf, diag_len, &resp) != 0) {
                if (attempt < kHttpMaxRetries - 1) {
                    int backoff_ms = static_cast<int>((1u << attempt) * 1000);
                    DLOG_W(TAG,
                           "State fetch failed, retry in %dms (%lu/%lu)",
                           backoff_ms,
                           (unsigned long)(attempt + 1),
                           (unsigned long)kHttpMaxRetries);
                    vTaskDelay(pdMS_TO_TICKS(backoff_ms));
                }
                continue;
            }

            if (!codec::decode_state(resp.data, resp.len, s_state_decode)) {
                DLOG_E(TAG, "Failed to decode DeviceState (len=%d)", resp.len);
                break; // decode error is not retryable
            }

            // ---- Publish to DoubleBuffer (lock-free) ----
            pb_size_t ns = s_state_decode.stations_count < kMaxStations
                               ? s_state_decode.stations_count
                               : kMaxStations;
            pb_size_t nc =
                s_state_decode.config_count < kMaxConfig ? s_state_decode.config_count : kMaxConfig;

            auto& wb = transit_buf.write_buffer();
            std::memcpy(wb.stations.data(), s_state_decode.stations, sizeof(subway_Station) * ns);
            wb.station_count = ns;
            wb.timestamp = s_state_decode.timestamp;
            std::memcpy(wb.config.data(),
                        s_state_decode.config,
                        sizeof(subway_DeviceState_ConfigEntry) * nc);
            wb.config_count = nc;
            wb.board_generation = current_board_gen;
            transit_buf.publish();

            // Cache config for next cycle's remote_log check
            std::memcpy(
                last_config, s_state_decode.config, sizeof(subway_DeviceState_ConfigEntry) * nc);
            last_config_count = nc;

            // Cache server's script hash for next cycle's diagnostics
            std::strncpy(last_server_script_hash, s_state_decode.script_hash, kHashLen - 1);
            last_server_script_hash[kHashLen - 1] = '\0';

            DLOG_I(TAG,
                   "State: %u stations, %u config entries",
                   static_cast<unsigned>(ns),
                   static_cast<unsigned>(nc));
            state_ok = true;
            break;
        }
        last_state_ok = state_ok;

        if (!state_ok) {
            vTaskDelay(pdMS_TO_TICKS(kPollIntervalMs));
            continue;
        }

        // ---- Check hash changes (all task-local, no locks) ----
        bool board_changed =
            !board_fetched || std::strncmp(s_state_decode.board_hash, cached_board_hash, 64) != 0;
        bool script_changed = !script_fetched ||
                              std::strncmp(s_state_decode.script_hash, cached_script_hash, 64) != 0;

        // ---- Fetch board if changed ----
        if (board_changed) {
            DLOG_W(TAG,
                   "Fetching board... heap=%lu",
                   static_cast<unsigned long>(esp_get_free_heap_size()));

            HttpResponse resp{};
            if (http.get("/api/v1/device-board", &resp) == 0) {
                auto* board_buf =
                    static_cast<subway_DeviceBoard*>(calloc(1, sizeof(subway_DeviceBoard)));
                if (board_buf && codec::decode_board(resp.data, resp.len, *board_buf)) {
                    auto& bw = board_store.lock_for_write();
                    BoardSnapshot::from_proto(*board_buf, bw);
                    current_board_gen++;
                    bw.generation = current_board_gen;
                    board_store.unlock_write();

                    std::strncpy(cached_board_hash, board_buf->hash, kHashLen - 1);
                    cached_board_hash[kHashLen - 1] = '\0';
                    board_fetched = true;

                    DLOG_I(TAG,
                           "Board OK. heap=%lu",
                           static_cast<unsigned long>(esp_get_free_heap_size()));
                } else {
                    DLOG_E(TAG,
                           "Board FAILED. heap=%lu",
                           static_cast<unsigned long>(esp_get_free_heap_size()));
                }
                free(board_buf);
            } else {
                DLOG_E(TAG,
                       "Board fetch failed. heap=%lu",
                       static_cast<unsigned long>(esp_get_free_heap_size()));
            }
        }

        // ---- Fetch script if changed and board is loaded ----
        if (script_changed && board_fetched) {
            DLOG_W(TAG,
                   "Fetching script... heap=%lu",
                   static_cast<unsigned long>(esp_get_free_heap_size()));

            HttpResponse resp{};
            if (http.get("/api/v1/device-script", &resp) == 0) {
                // Use lightweight ScriptInfo (~16.5KB) instead of full
                // DeviceScript (~20.6KB) to reduce heap pressure
                auto* si = static_cast<codec::ScriptInfo*>(calloc(1, sizeof(codec::ScriptInfo)));
                if (!si) {
                    DLOG_E(TAG,
                           "OOM: ScriptInfo (%u bytes, heap=%lu)",
                           static_cast<unsigned>(sizeof(codec::ScriptInfo)),
                           static_cast<unsigned long>(esp_get_free_heap_size()));
                } else if (codec::decode_script_info(resp.data, resp.len, *si)) {
                    char* src = strdup(si->lua_source);
                    if (src) {
                        script_chan.send(src); // ownership transfers
                        std::strncpy(cached_script_hash, si->hash, kHashLen - 1);
                        cached_script_hash[kHashLen - 1] = '\0';
                        script_fetched = true;
                        DLOG_I(TAG,
                               "Script updated: %s (%d bytes)",
                               si->plugin_name,
                               static_cast<int>(std::strlen(si->lua_source)));
                    } else {
                        DLOG_E(TAG, "OOM: strdup lua_source");
                    }
                    free(si);
                } else {
                    DLOG_E(TAG, "Script decode failed (len=%d)", resp.len);
                    free(si);
                }
            } else {
                DLOG_E(TAG, "Script fetch failed");
            }
        }

        vTaskDelay(pdMS_TO_TICKS(kPollIntervalMs));
    }
}

// -----------------------------------------------------------------------
// Public API
// -----------------------------------------------------------------------
void StateClient::start(DoubleBuffer<TransitSnapshot>& transit_buf,
                        BoardStore& board_store,
                        ScriptChannel& script_chan,
                        DiagPad& diag,
                        std::atomic<bool>& ota_active,
                        std::atomic<bool>& http_active) {
    auto* args = new StateTaskArgs{
        &transit_buf, &board_store, &script_chan, &diag, &ota_active, &http_active};
    xTaskCreate(state_task, "state_task", 16384, args, 3, nullptr);
}
