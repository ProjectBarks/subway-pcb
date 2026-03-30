#include "state_client.h"
#include "http_client.h"
#include "device_log.h"
#include "config.h"

#include <string.h>
#include <stdio.h>
#include <stdlib.h>

#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "esp_log.h"
#include "esp_mac.h"
#include "nvs_flash.h"
#include "nvs.h"

#include "esp_system.h"
#include "esp_heap_caps.h"
#include "pb_decode.h"
#include "pb_encode.h"
#include "subway.pb.h"

static const char *TAG = "state_client";

static render_context_t *s_ctx = NULL;

static void init_http_client(render_context_t *ctx)
{
    /* Read device MAC */
    static char device_id[18];
    uint8_t mac[6];
    esp_read_mac(mac, ESP_MAC_WIFI_STA);
    snprintf(device_id, sizeof(device_id), "%02x:%02x:%02x:%02x:%02x:%02x",
             mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]);

    /* Read NVS config */
    static char server_url[128];
    static char hardware[32];
    strncpy(server_url, DEFAULT_SERVER_URL, sizeof(server_url) - 1);
    strncpy(hardware, "nyc-subway/v1", sizeof(hardware) - 1);

    nvs_handle_t nvs;
    if (nvs_open("subway", NVS_READONLY, &nvs) == ESP_OK) {
        size_t len = sizeof(server_url);
        nvs_get_str(nvs, "server_url", server_url, &len);
        len = sizeof(hardware);
        nvs_get_str(nvs, "hardware", hardware, &len);
        nvs_close(nvs);
    }

    http_client_config_t cfg = {
        .server_url = server_url,
        .device_id = device_id,
        .firmware_ver = FIRMWARE_VERSION,
        .hardware = hardware,
    };
    http_client_init(&cfg, ctx);
}

static bool s_last_state_ok = false;

static bool fetch_state(render_context_t *ctx, bool board_fetched, bool script_fetched)
{
    /* Build diagnostics protobuf (reports previous cycle's results).
     * Static to keep ~2.3KB (mostly logs[2048]) off the 16KB task stack. */
    static subway_DeviceDiagnostics diag;
    memset(&diag, 0, sizeof(diag));

    diag.has_system = true;
    diag.system.reset_reason = (int32_t)esp_reset_reason();
    diag.system.free_heap = (uint32_t)esp_get_free_heap_size();
    diag.system.largest_block = (uint32_t)heap_caps_get_largest_free_block(MALLOC_CAP_8BIT);

    diag.has_fetch = true;
    diag.fetch.state_ok = s_last_state_ok;
    diag.fetch.board_fetched = board_fetched;
    diag.fetch.script_fetched = script_fetched;
    diag.fetch.script_hash_match = (strcmp(ctx->script_hash, ctx->cached_script_hash) == 0);

    diag.has_lua = true;
    diag.lua.errors = ctx->diag.lua_errors;
    diag.lua.mem = ctx->diag.lua_mem;
    diag.lua.last_reload = ctx->diag.last_reload;

    diag.has_render = true;
    diag.render.nonzero_pixels = ctx->diag.nonzero_pixels;
    diag.render.first_lit_led = ctx->diag.first_lit_led;

    /* Copy error string */
    const char *err_src = ctx->diag.last_lua_err[0] ? ctx->diag.last_lua_err :
                          ctx->diag.fetch_err[0] ? ctx->diag.fetch_err : "";
    strncpy(diag.error, err_src, sizeof(diag.error) - 1);

    /* Drain remote logs if enabled */
    device_log_drain(diag.logs, sizeof(diag.logs));

    /* Encode diagnostics */
    uint8_t diag_buf[512];
    pb_ostream_t ostream = pb_ostream_from_buffer(diag_buf, sizeof(diag_buf));
    pb_encode(&ostream, subway_DeviceDiagnostics_fields, &diag);

    /* POST diagnostics, receive state */
    http_response_t resp;
    if (http_client_post("/api/v1/device-state", diag_buf, ostream.bytes_written, &resp) != 0)
        return false;

    /* Temporary decode buffer — allocated per cycle, freed after copy to render context */
    subway_DeviceState *state = calloc(1, sizeof(subway_DeviceState));
    if (!state) { DLOG_E(TAG, "OOM: DeviceState"); return false; }

    pb_istream_t stream = pb_istream_from_buffer(resp.data, resp.len);
    if (!pb_decode(&stream, subway_DeviceState_fields, state)) {
        DLOG_E(TAG, "Failed to decode DeviceState (len=%d): %s", resp.len, PB_GET_ERROR(&stream));
        free(state);
        return false;
    }

    /* Copy decoded state into render context under mutex.
     * render_context now uses the same nanopb types, so this is a direct memcpy. */
    xSemaphoreTake(s_ctx->mutex, portMAX_DELAY);

    pb_size_t ns = state->stations_count < PB_MAX_STATIONS ? state->stations_count : PB_MAX_STATIONS;
    memcpy(s_ctx->stations, state->stations, sizeof(subway_Station) * ns);
    s_ctx->station_count = ns;
    s_ctx->timestamp = state->timestamp;

    pb_size_t nc = state->config_count < PB_MAX_CONFIG ? state->config_count : PB_MAX_CONFIG;
    memcpy(s_ctx->config, state->config, sizeof(subway_DeviceState_ConfigEntry) * nc);
    s_ctx->config_count = nc;

    memcpy(s_ctx->script_hash, state->script_hash, PB_HASH_LEN);
    memcpy(s_ctx->board_hash, state->board_hash, PB_HASH_LEN);

    xSemaphoreGive(s_ctx->mutex);

    pb_size_t log_sc = s_ctx->station_count;
    pb_size_t log_cc = s_ctx->config_count;
    free(state);

    DLOG_I(TAG, "State: %d stations, %d config entries", log_sc, log_cc);
    return true;
}

static bool fetch_board(void)
{
    http_response_t resp;
    if (http_client_get("/api/v1/device-board", &resp) != 0) {
        DLOG_E(TAG, "Board fetch failed");
        return false;
    }

    subway_DeviceBoard *buf = calloc(1, sizeof(subway_DeviceBoard));
    if (!buf) { ESP_LOGE(TAG, "OOM: DeviceBoard"); return false; }

    pb_istream_t stream = pb_istream_from_buffer(resp.data, resp.len);
    if (!pb_decode(&stream, subway_DeviceBoard_fields, buf)) {
        DLOG_E(TAG, "Board decode failed: %s", PB_GET_ERROR(&stream));
        free(buf);
        return false;
    }

    xSemaphoreTake(s_ctx->mutex, portMAX_DELAY);
    strncpy(s_ctx->board.board_id, buf->board_id, sizeof(s_ctx->board.board_id) - 1);
    s_ctx->board.led_count = buf->led_count;
    s_ctx->board.strip_count = buf->strip_sizes_count;
    for (pb_size_t i = 0; i < buf->strip_sizes_count && i < MAX_STRIPS; i++) {
        s_ctx->board.strip_sizes[i] = buf->strip_sizes[i];
    }
    strncpy(s_ctx->board.hash, buf->hash, PB_HASH_LEN - 1);
    memset(s_ctx->board.led_map, 0, sizeof(s_ctx->board.led_map));
    for (pb_size_t i = 0; i < buf->led_map_count; i++) {
        uint32_t idx = buf->led_map[i].key;
        if (idx < MAX_LEDS) {
            strncpy(s_ctx->board.led_map[idx], buf->led_map[i].value, PB_STOP_ID_LEN - 1);
        }
    }
    s_ctx->board_loaded = true;

    render_context_build_station_leds(s_ctx);
    strncpy(s_ctx->cached_board_hash, buf->hash, PB_HASH_LEN - 1);
    xSemaphoreGive(s_ctx->mutex);

    DLOG_I(TAG, "Board loaded: %lu LEDs, %d strips",
             (unsigned long)buf->led_count, (int)buf->strip_sizes_count);
    free(buf);
    return true;
}

static bool fetch_script(void)
{
    http_response_t resp;
    if (http_client_get("/api/v1/device-script", &resp) != 0) {
        DLOG_E(TAG, "Script fetch failed");
        return false;
    }

    subway_DeviceScript *script_buf = calloc(1, sizeof(subway_DeviceScript));
    if (!script_buf) { ESP_LOGE(TAG, "OOM: DeviceScript"); return false; }

    pb_istream_t stream = pb_istream_from_buffer(resp.data, resp.len);
    if (!pb_decode(&stream, subway_DeviceScript_fields, script_buf)) {
        DLOG_E(TAG, "Script decode failed: %s", PB_GET_ERROR(&stream));
        free(script_buf);
        return false;
    }

    char *src = strdup(script_buf->lua_source);

    xSemaphoreTake(s_ctx->mutex, portMAX_DELAY);
    strncpy(s_ctx->cached_script_hash, script_buf->hash, PB_HASH_LEN - 1);
    free(s_ctx->lua_source);
    s_ctx->lua_source = src;
    s_ctx->script_changed = true;
    xSemaphoreGive(s_ctx->mutex);

    DLOG_I(TAG, "Script loaded: %s (%d bytes)",
             script_buf->plugin_name, (int)strlen(script_buf->lua_source));
    free(script_buf);
    return true;
}

static void state_task(void *arg)
{
    render_context_t *ctx = (render_context_t *)arg;
    s_ctx = ctx;

    init_http_client(ctx);

    bool board_fetched = false;
    bool script_fetched = false;

    while (1) {
        if (ctx->ota_active) {
            vTaskDelay(pdMS_TO_TICKS(1000));
            continue;
        }

        /* Check remote logging config */
        for (uint8_t i = 0; i < ctx->config_count; i++) {
            if (strcmp(ctx->config[i].key, "remote_log") == 0) {
                device_log_set_remote(strcmp(ctx->config[i].value, "1") == 0);
                break;
            }
        }

        bool state_ok = fetch_state(ctx, board_fetched, script_fetched);
        s_last_state_ok = state_ok;

        if (state_ok) {
            xSemaphoreTake(ctx->mutex, portMAX_DELAY);
            bool board_changed = !board_fetched ||
                                 strcmp(ctx->board_hash, ctx->cached_board_hash) != 0;
            bool script_changed = !script_fetched ||
                                  strcmp(ctx->script_hash, ctx->cached_script_hash) != 0;
            xSemaphoreGive(ctx->mutex);

            if (board_changed) {
                DLOG_W(TAG, "Fetching board... heap=%lu", (unsigned long)esp_get_free_heap_size());
                if (fetch_board()) {
                    board_fetched = true;
                    DLOG_I(TAG, "Board OK. heap=%lu", (unsigned long)esp_get_free_heap_size());
                } else {
                    DLOG_E(TAG, "Board FAILED. heap=%lu", (unsigned long)esp_get_free_heap_size());
                }
            }

            if (script_changed && board_fetched) {
                DLOG_W(TAG, "Fetching script... heap=%lu", (unsigned long)esp_get_free_heap_size());
                if (fetch_script()) {
                    script_fetched = true;
                    DLOG_I(TAG, "Script updated");
                }
            }
        }

        vTaskDelay(pdMS_TO_TICKS(POLL_INTERVAL_MS));
    }
}

void state_client_start(render_context_t *ctx)
{
    xTaskCreate(state_task, "state_task", 16384, ctx, 3, NULL);
}
