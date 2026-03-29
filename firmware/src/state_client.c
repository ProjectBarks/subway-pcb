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
#include "esp_system.h"
#include "esp_heap_caps.h"
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

static int http_fetch(const char *path, const char *diag)
{
    char url[384];
    snprintf(url, sizeof(url), "%s%s%s", s_server_url, path, diag ? diag : "");

    s_http_buf_len = 0;

    esp_http_client_config_t cfg = {
        .url = url,
        .event_handler = http_event_handler,
        .timeout_ms = 10000,
        .crt_bundle_attach = esp_crt_bundle_attach,
    };
    esp_http_client_handle_t client = esp_http_client_init(&cfg);
    if (!client) {
        ESP_LOGE(TAG, "Failed to create HTTP client");
        return -1;
    }
    esp_http_client_set_header(client, "X-Device-ID", s_device_id);
    esp_http_client_set_header(client, "X-Firmware-Version", FIRMWARE_VERSION);
    esp_http_client_set_header(client, "X-Hardware", s_hardware);

    esp_err_t err = esp_http_client_perform(client);
    int status = esp_http_client_get_status_code(client);
    esp_http_client_cleanup(client);

    if (err != ESP_OK || status != 200) {
        ESP_LOGE(TAG, "HTTP FAILED %s: err=%d(%s) status=%d", path, err, esp_err_to_name(err), status);
        return -1;
    }

    return s_http_buf_len;
}

/* Persistent decode buffer — avoids 17KB alloc/free every 3s which fragments heap */
static subway_DeviceState *s_state_buf = NULL;

static bool fetch_state(const char *diag)
{
    int len = http_fetch("/api/v1/device-state", diag);
    if (len <= 0) return false;

    if (!s_state_buf) {
        s_state_buf = calloc(1, sizeof(subway_DeviceState));
        if (!s_state_buf) { ESP_LOGE(TAG, "OOM: DeviceState"); return false; }
    }
    memset(s_state_buf, 0, sizeof(subway_DeviceState));

    pb_istream_t stream = pb_istream_from_buffer(s_http_buf, len);
    if (!pb_decode(&stream, subway_DeviceState_fields, s_state_buf)) {
        ESP_LOGE(TAG, "Failed to decode DeviceState (len=%d)", len);
        return false;
    }
    subway_DeviceState *state = s_state_buf;

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
    return true;
}

/* DeviceBoard — stripped to isolate crash */
static bool fetch_board(void)
{
    /* Step 1: HTTP fetch */
    int len = http_fetch("/api/v1/device-board", NULL);
    snprintf(s_ctx->diag_fetch_err, sizeof(s_ctx->diag_fetch_err), "board:step1_len%d", len);
    if (len <= 0) return false;

    /* Step 2: calloc */
    subway_DeviceBoard *buf = calloc(1, sizeof(subway_DeviceBoard));
    snprintf(s_ctx->diag_fetch_err, sizeof(s_ctx->diag_fetch_err),
             "board:step2_alloc%s", buf ? "ok" : "fail");
    if (!buf) return false;

    /* Step 3: decode */
    pb_istream_t stream = pb_istream_from_buffer(s_http_buf, len);
    bool decoded = pb_decode(&stream, subway_DeviceBoard_fields, buf);
    snprintf(s_ctx->diag_fetch_err, sizeof(s_ctx->diag_fetch_err),
             "board:step3_dec%s_leds%lu_strips%d",
             decoded ? "ok" : "fail",
             decoded ? (unsigned long)buf->led_count : 0,
             decoded ? (int)buf->strip_sizes_count : 0);
    if (!decoded) { free(buf); return false; }

    /* Step 4: copy to context */
    xSemaphoreTake(s_ctx->mutex, portMAX_DELAY);
    strncpy(s_ctx->board.board_id, buf->board_id, sizeof(s_ctx->board.board_id) - 1);
    s_ctx->board.led_count = buf->led_count;
    s_ctx->board.strip_count = buf->strip_sizes_count;
    for (pb_size_t i = 0; i < buf->strip_sizes_count && i < 16; i++) {
        s_ctx->board.strip_sizes[i] = buf->strip_sizes[i];
    }
    strncpy(s_ctx->board.hash, buf->hash, MAX_HASH_LEN - 1);
    memset(s_ctx->board.led_map, 0, sizeof(s_ctx->board.led_map));
    for (pb_size_t i = 0; i < buf->led_map_count; i++) {
        uint32_t idx = buf->led_map[i].key;
        if (idx < MAX_LEDS) {
            strncpy(s_ctx->board.led_map[idx], buf->led_map[i].value, MAX_STOP_ID_LEN - 1);
        }
    }
    s_ctx->board_loaded = true;
    snprintf(s_ctx->diag_fetch_err, sizeof(s_ctx->diag_fetch_err), "board:step4_copy_ok");

    /* Step 5: build index */
    render_context_build_station_leds(s_ctx);
    strncpy(s_ctx->cached_board_hash, buf->hash, MAX_HASH_LEN - 1);
    xSemaphoreGive(s_ctx->mutex);
    snprintf(s_ctx->diag_fetch_err, sizeof(s_ctx->diag_fetch_err), "board:step5_idx_ok");

    /* Step 6: free + done */
    free(buf);
    s_ctx->diag_fetch_err[0] = '\0';  /* clear so lua errors show through */
    return true;
}

/* DeviceScript is 17KB — allocate on demand and free after use (only fetched on change) */
static bool fetch_script(void)
{
    int len = http_fetch("/api/v1/device-script", NULL);
    if (len <= 0) {
        snprintf(s_ctx->diag_fetch_err, sizeof(s_ctx->diag_fetch_err), "script:http_len%d", len);
        return false;
    }

    subway_DeviceScript *s_script_buf = calloc(1, sizeof(subway_DeviceScript));
    if (!s_script_buf) { ESP_LOGE(TAG, "OOM: DeviceScript"); return false; }

    pb_istream_t stream = pb_istream_from_buffer(s_http_buf, len);
    if (!pb_decode(&stream, subway_DeviceScript_fields, s_script_buf)) {
        snprintf(s_ctx->diag_fetch_err, sizeof(s_ctx->diag_fetch_err),
                 "script:len%d,b0x%02x%02x%02x%02x,%s",
                 len, s_http_buf[0], s_http_buf[1], s_http_buf[2], s_http_buf[3],
                 PB_GET_ERROR(&stream));
        ESP_LOGE(TAG, "Script decode fail: %s", s_ctx->diag_fetch_err);
        free(s_script_buf);
        return false;
    }
    subway_DeviceScript *script = s_script_buf;

    /* Copy Lua source to heap for render task to consume */
    char *src = strdup(script->lua_source);

    xSemaphoreTake(s_ctx->mutex, portMAX_DELAY);
    strncpy(s_ctx->cached_script_hash, script->hash, MAX_HASH_LEN - 1);
    free(s_ctx->lua_source);  /* free previous if any */
    s_ctx->lua_source = src;
    s_ctx->script_changed = true;
    xSemaphoreGive(s_ctx->mutex);

    ESP_LOGI(TAG, "Script loaded: %s (%d bytes)",
             script->plugin_name, (int)strlen(script->lua_source));
    free(s_script_buf);
    return true;
}

static void state_task(void *arg)
{
    render_context_t *ctx = (render_context_t *)arg;
    s_ctx = ctx;

    read_device_id();
    read_nvs_config();

    ESP_LOGI(TAG, "State task started (device=%s, url=%s)", s_device_id, s_server_url);

    /* Diagnostics now go via query string on device-state requests */

    /* Initial fetch of board and script */
    bool board_fetched = false;
    bool script_fetched = false;

    while (1) {
        if (g_ota_active) {
            vTaskDelay(pdMS_TO_TICKS(1000));
            continue;
        }

        /* Build diag from previous cycle, pass as query string on this fetch */
        static char diag[256] = "";
        bool state_ok = fetch_state(diag);

        /* Update diag for NEXT cycle — includes this cycle's results */
        int sh_match = (strcmp(ctx->script_hash, ctx->cached_script_hash) == 0) ? 1 : 0;
        snprintf(diag, sizeof(diag),
                 "?d=st%d,bf%d,sf%d,shm%d,px%lu,lerr%d,lmem%lu,heap%lu,maxblk%lu,rld%d,first%lu",
                 state_ok ? 1 : 0,
                 board_fetched ? 1 : 0,
                 script_fetched ? 1 : 0,
                 sh_match,
                 (unsigned long)ctx->diag_nonzero_pixels,
                 ctx->diag_lua_errors,
                 (unsigned long)ctx->diag_lua_mem,
                 (unsigned long)esp_get_free_heap_size(),
                 (unsigned long)heap_caps_get_largest_free_block(MALLOC_CAP_8BIT),
                 ctx->diag_last_reload,
                 (unsigned long)ctx->diag_first_lit_led);
        /* Append errors if any (URL-safe: replace bad chars) */
        {
            const char *err_src = ctx->diag_fetch_err[0] ? ctx->diag_fetch_err : ctx->diag_last_lua_err;
            if (err_src[0]) {
                char err_safe[50];
                strncpy(err_safe, err_src, sizeof(err_safe) - 1);
                err_safe[sizeof(err_safe) - 1] = '\0';
                for (char *p = err_safe; *p; p++) {
                    if (*p == ' ') *p = '_';
                    if (*p == '?' || *p == '&' || *p == '=' || *p == '#') *p = '.';
                }
                size_t dlen = strlen(diag);
                snprintf(diag + dlen, sizeof(diag) - dlen, ",err=%s", err_safe);
            }
        }

        if (state_ok) {
            /* Check if board needs updating */
            xSemaphoreTake(ctx->mutex, portMAX_DELAY);
            bool board_changed = !board_fetched ||
                                 strcmp(ctx->board_hash, ctx->cached_board_hash) != 0;
            bool script_changed = !script_fetched ||
                                  strcmp(ctx->script_hash, ctx->cached_script_hash) != 0;
            xSemaphoreGive(ctx->mutex);

            if (board_changed) {
                ESP_LOGW(TAG, "Fetching board... heap=%lu", (unsigned long)esp_get_free_heap_size());
                if (fetch_board()) {
                    board_fetched = true;
                    ESP_LOGW(TAG, "Board OK. heap=%lu", (unsigned long)esp_get_free_heap_size());
                } else {
                    ESP_LOGE(TAG, "Board FAILED. heap=%lu", (unsigned long)esp_get_free_heap_size());
                }
            }

            if (script_changed && board_fetched) {
                ESP_LOGW(TAG, "Fetching script... heap=%lu", (unsigned long)esp_get_free_heap_size());
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
    xTaskCreate(state_task, "state_task", 16384, ctx, 3, NULL);
}
