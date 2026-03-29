#include "state_client.h"
#include "config.h"

#include <string.h>
#include <stdio.h>

#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "esp_log.h"
#include "esp_http_client.h"
#include "esp_mac.h"
#include "nvs_flash.h"
#include "nvs.h"

#include "esp_crt_bundle.h"
#include "pb_decode.h"
#include "subway.pb.h"

static const char *TAG = "state_client";

/* Global flag from main.c — pause during OTA */
extern volatile bool g_ota_active;

static char s_device_id[18] = {0};   /* "aa:bb:cc:dd:ee:ff" */
static char s_hardware[32] = {0};     /* board model from NVS */
static char s_server_url[128] = {0};

static render_context_t *s_ctx = NULL;

/* Response buffer for HTTP fetches */
#define STATE_HTTP_BUF_SIZE 16384
static uint8_t s_http_buf[STATE_HTTP_BUF_SIZE];
static int s_http_buf_len = 0;

static esp_err_t http_event_handler(esp_http_client_event_t *evt)
{
    switch (evt->event_id) {
    case HTTP_EVENT_ON_DATA:
        if (s_http_buf_len + evt->data_len <= STATE_HTTP_BUF_SIZE) {
            memcpy(s_http_buf + s_http_buf_len, evt->data, evt->data_len);
            s_http_buf_len += evt->data_len;
        }
        break;
    default:
        break;
    }
    return ESP_OK;
}

static void read_device_id(void)
{
    uint8_t mac[6];
    esp_read_mac(mac, ESP_MAC_WIFI_STA);
    snprintf(s_device_id, sizeof(s_device_id), "%02x:%02x:%02x:%02x:%02x:%02x",
             mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]);
}

static void read_nvs_config(void)
{
    nvs_handle_t nvs;
    if (nvs_open("subway", NVS_READONLY, &nvs) == ESP_OK) {
        size_t len = sizeof(s_server_url);
        if (nvs_get_str(nvs, "server_url", s_server_url, &len) != ESP_OK) {
            strncpy(s_server_url, DEFAULT_SERVER_URL, sizeof(s_server_url) - 1);
        }
        len = sizeof(s_hardware);
        if (nvs_get_str(nvs, "hardware", s_hardware, &len) != ESP_OK) {
            strncpy(s_hardware, "nyc-subway/v1", sizeof(s_hardware) - 1);
        }
        nvs_close(nvs);
    } else {
        strncpy(s_server_url, DEFAULT_SERVER_URL, sizeof(s_server_url) - 1);
        strncpy(s_hardware, "nyc-subway/v1", sizeof(s_hardware) - 1);
    }
    ESP_LOGI(TAG, "Server URL: %s, Hardware: %s", s_server_url, s_hardware);
}

/* Copy decoded protobuf station data into render context.
 * nanopb generates static arrays for all repeated fields (via .options max_count),
 * so no callbacks are needed — pb_decode fills the structs directly. */

static int http_fetch(const char *path)
{
    char url[256];
    snprintf(url, sizeof(url), "%s%s", s_server_url, path);

    s_http_buf_len = 0;

    esp_http_client_config_t cfg = {
        .url = url,
        .event_handler = http_event_handler,
        .timeout_ms = 10000,
        .crt_bundle_attach = esp_crt_bundle_attach,
    };
    esp_http_client_handle_t client = esp_http_client_init(&cfg);

    esp_http_client_set_header(client, "X-Device-ID", s_device_id);
    esp_http_client_set_header(client, "X-Firmware-Version", FIRMWARE_VERSION);
    esp_http_client_set_header(client, "X-Hardware", s_hardware);

    ESP_LOGW(TAG, "HTTP fetch: %s", url);
    esp_err_t err = esp_http_client_perform(client);
    int status = esp_http_client_get_status_code(client);
    esp_http_client_cleanup(client);

    if (err != ESP_OK || status != 200) {
        ESP_LOGE(TAG, "HTTP FAILED %s: err=%d(%s) status=%d", path, err, esp_err_to_name(err), status);
        return -1;
    }

    return s_http_buf_len;
}

static bool fetch_state(void)
{
    int len = http_fetch("/api/v1/device-state");
    if (len <= 0) return false;

    /* Heap-allocate — struct is ~15KB (stations[200]), too large for stack */
    subway_DeviceState *state = calloc(1, sizeof(subway_DeviceState));
    if (!state) { ESP_LOGE(TAG, "OOM: DeviceState"); return false; }

    pb_istream_t stream = pb_istream_from_buffer(s_http_buf, len);
    if (!pb_decode(&stream, subway_DeviceState_fields, state)) {
        ESP_LOGE(TAG, "Failed to decode DeviceState");
        free(state);
        return false;
    }

    /* Update render context under mutex */
    xSemaphoreTake(s_ctx->mutex, portMAX_DELAY);

    s_ctx->station_count = 0;
    for (pb_size_t i = 0; i < state->stations_count && i < MAX_STATIONS; i++) {
        strncpy(s_ctx->stations[i].stop_id, state->stations[i].stop_id, MAX_STOP_ID_LEN - 1);
        s_ctx->stations[i].train_count = 0;
        for (pb_size_t j = 0; j < state->stations[i].trains_count && j < MAX_TRAINS_PER_STATION; j++) {
            strncpy(s_ctx->stations[i].trains[j].route, state->stations[i].trains[j].route, MAX_ROUTE_LEN - 1);
            s_ctx->stations[i].trains[j].status = (train_status_t)state->stations[i].trains[j].status;
            s_ctx->stations[i].train_count++;
        }
        s_ctx->station_count++;
    }
    s_ctx->timestamp = state->timestamp;

    s_ctx->config_count = 0;
    for (pb_size_t i = 0; i < state->config_count && i < MAX_CONFIG_ENTRIES; i++) {
        strncpy(s_ctx->config[i].key, state->config[i].key, MAX_CONFIG_KEY_LEN - 1);
        strncpy(s_ctx->config[i].value, state->config[i].value, MAX_CONFIG_VAL_LEN - 1);
        s_ctx->config_count++;
    }

    strncpy(s_ctx->script_hash, state->script_hash, MAX_HASH_LEN - 1);
    strncpy(s_ctx->board_hash, state->board_hash, MAX_HASH_LEN - 1);

    xSemaphoreGive(s_ctx->mutex);

    ESP_LOGI(TAG, "State: %d stations, %d config entries",
             s_ctx->station_count, s_ctx->config_count);
    free(state);
    return true;
}

static bool fetch_board(void)
{
    int len = http_fetch("/api/v1/device-board");
    if (len <= 0) return false;

    /* Heap-allocate — struct is ~6KB, too large for task stack */
    subway_DeviceBoard *board = calloc(1, sizeof(subway_DeviceBoard));
    if (!board) { ESP_LOGE(TAG, "OOM: DeviceBoard"); return false; }

    pb_istream_t stream = pb_istream_from_buffer(s_http_buf, len);
    if (!pb_decode(&stream, subway_DeviceBoard_fields, board)) {
        ESP_LOGE(TAG, "Failed to decode DeviceBoard");
        free(board);
        return false;
    }

    xSemaphoreTake(s_ctx->mutex, portMAX_DELAY);

    strncpy(s_ctx->board.board_id, board->board_id, sizeof(s_ctx->board.board_id) - 1);
    s_ctx->board.led_count = board->led_count;
    s_ctx->board.strip_count = board->strip_sizes_count;
    for (pb_size_t i = 0; i < board->strip_sizes_count && i < 16; i++) {
        s_ctx->board.strip_sizes[i] = board->strip_sizes[i];
    }
    strncpy(s_ctx->board.hash, board->hash, MAX_HASH_LEN - 1);

    memset(s_ctx->board.led_map, 0, sizeof(s_ctx->board.led_map));
    for (pb_size_t i = 0; i < board->led_map_count; i++) {
        uint32_t idx = board->led_map[i].key;
        if (idx < MAX_LEDS) {
            strncpy(s_ctx->board.led_map[idx], board->led_map[i].value, MAX_STOP_ID_LEN - 1);
        }
    }

    s_ctx->board_loaded = true;
    render_context_build_station_leds(s_ctx);
    strncpy(s_ctx->cached_board_hash, board->hash, MAX_HASH_LEN - 1);

    xSemaphoreGive(s_ctx->mutex);

    ESP_LOGI(TAG, "Board loaded: %s, %lu LEDs, %d strips",
             board->board_id, (unsigned long)board->led_count, board->strip_sizes_count);
    free(board);
    return true;
}

static bool fetch_script(void)
{
    int len = http_fetch("/api/v1/device-script");
    if (len <= 0) return false;

    /* Heap-allocate — struct is ~17KB (lua_source[16384]), way too large for stack */
    subway_DeviceScript *script = calloc(1, sizeof(subway_DeviceScript));
    if (!script) { ESP_LOGE(TAG, "OOM: DeviceScript"); return false; }

    pb_istream_t stream = pb_istream_from_buffer(s_http_buf, len);
    if (!pb_decode(&stream, subway_DeviceScript_fields, script)) {
        ESP_LOGE(TAG, "Failed to decode DeviceScript");
        free(script);
        return false;
    }

    xSemaphoreTake(s_ctx->mutex, portMAX_DELAY);
    strncpy(s_ctx->cached_script_hash, script->hash, MAX_HASH_LEN - 1);
    s_ctx->script_changed = true;
    xSemaphoreGive(s_ctx->mutex);

    ESP_LOGI(TAG, "Script loaded: %s (%d bytes)",
             script->plugin_name, (int)strlen(script->lua_source));
    free(script);
    return true;
}

static void state_task(void *arg)
{
    render_context_t *ctx = (render_context_t *)arg;
    s_ctx = ctx;

    read_device_id();
    read_nvs_config();

    ESP_LOGI(TAG, "State task started (device=%s, url=%s)", s_device_id, s_server_url);

    /* Ping health endpoint to confirm connectivity — shows in server logs */
    {
        char url[256];
        snprintf(url, sizeof(url), "%s/health", s_server_url);
        s_http_buf_len = 0;
        esp_http_client_config_t cfg = {
            .url = url,
            .event_handler = http_event_handler,
            .timeout_ms = 15000,
            .crt_bundle_attach = esp_crt_bundle_attach,
        };
        esp_http_client_handle_t client = esp_http_client_init(&cfg);
        esp_http_client_set_header(client, "X-Device-ID", s_device_id);
        esp_http_client_set_header(client, "X-Firmware-Version", FIRMWARE_VERSION);
        esp_http_client_set_header(client, "X-Hardware", s_hardware);
        esp_err_t err = esp_http_client_perform(client);
        int status = esp_http_client_get_status_code(client);
        esp_http_client_cleanup(client);
        ESP_LOGW(TAG, "HEALTH PING: err=%d(%s) status=%d bytes=%d",
                 err, esp_err_to_name(err), status, s_http_buf_len);
    }

    /* Initial fetch of board and script */
    bool board_fetched = false;
    bool script_fetched = false;

    while (1) {
        if (g_ota_active) {
            vTaskDelay(pdMS_TO_TICKS(1000));
            continue;
        }

        /* Fetch state (every cycle) */
        bool state_ok = fetch_state();

        if (state_ok) {
            /* Check if board needs updating */
            xSemaphoreTake(ctx->mutex, portMAX_DELAY);
            bool board_changed = !board_fetched ||
                                 strcmp(ctx->board_hash, ctx->cached_board_hash) != 0;
            bool script_changed = !script_fetched ||
                                  strcmp(ctx->script_hash, ctx->cached_script_hash) != 0;
            xSemaphoreGive(ctx->mutex);

            if (board_changed) {
                if (fetch_board()) {
                    board_fetched = true;
                    ESP_LOGI(TAG, "Board data updated");
                }
            }

            if (script_changed) {
                if (fetch_script()) {
                    script_fetched = true;
                    ESP_LOGI(TAG, "Script updated");
                }
            }
        }

        vTaskDelay(pdMS_TO_TICKS(POLL_INTERVAL_MS));
    }
}

void state_client_start(render_context_t *ctx)
{
    xTaskCreate(state_task, "state_task", 12288, ctx, 3, NULL);
}
