#include "http_client.h"
#include "render_context.h"

#include <string.h>
#include <stdio.h>

#include "esp_log.h"
#include "esp_http_client.h"
#include "esp_crt_bundle.h"

static const char *TAG = "http_client";

#define HTTP_BUF_SIZE 16384

static uint8_t s_http_buf[HTTP_BUF_SIZE];
static int s_http_buf_len = 0;

static http_client_config_t s_cfg;
static render_context_t *s_ctx = NULL;

static esp_err_t http_event_handler(esp_http_client_event_t *evt)
{
    switch (evt->event_id) {
    case HTTP_EVENT_ON_DATA:
        if (s_http_buf_len + evt->data_len <= HTTP_BUF_SIZE) {
            memcpy(s_http_buf + s_http_buf_len, evt->data, evt->data_len);
            s_http_buf_len += evt->data_len;
        }
        break;
    default:
        break;
    }
    return ESP_OK;
}

void http_client_init(const http_client_config_t *cfg, render_context_t *ctx)
{
    s_cfg = *cfg;
    s_ctx = ctx;
    ESP_LOGI(TAG, "HTTP client initialized (server=%s, device=%s)", cfg->server_url, cfg->device_id);
}

static int do_request(const char *path, const char *method,
                      const uint8_t *body, int body_len,
                      http_response_t *resp)
{
    if (s_ctx) s_ctx->http_active = true;

    char url[384];
    snprintf(url, sizeof(url), "%s%s", s_cfg.server_url, path);

    s_http_buf_len = 0;

    esp_http_client_config_t http_cfg = {
        .url = url,
        .event_handler = http_event_handler,
        .timeout_ms = 10000,
        .crt_bundle_attach = esp_crt_bundle_attach,
    };
    esp_http_client_handle_t client = esp_http_client_init(&http_cfg);
    if (!client) {
        ESP_LOGE(TAG, "Failed to create HTTP client");
        if (s_ctx) s_ctx->http_active = false;
        return -1;
    }

    esp_http_client_set_header(client, "X-Device-ID", s_cfg.device_id);
    esp_http_client_set_header(client, "X-Firmware-Version", s_cfg.firmware_ver);
    esp_http_client_set_header(client, "X-Hardware", s_cfg.hardware);

    if (body && body_len > 0) {
        esp_http_client_set_method(client, HTTP_METHOD_POST);
        esp_http_client_set_header(client, "Content-Type", "application/x-protobuf");
        esp_http_client_set_post_field(client, (const char *)body, body_len);
    }

    esp_err_t err = esp_http_client_perform(client);
    int status = esp_http_client_get_status_code(client);
    esp_http_client_cleanup(client);

    if (s_ctx) s_ctx->http_active = false;

    if (err != ESP_OK || status != 200) {
        ESP_LOGE(TAG, "HTTP %s %s: err=%d(%s) status=%d",
                 method, path, err, esp_err_to_name(err), status);
        return -1;
    }

    if (resp) {
        resp->data = s_http_buf;
        resp->len = s_http_buf_len;
    }
    return 0;
}

int http_client_get(const char *path, http_response_t *resp)
{
    return do_request(path, "GET", NULL, 0, resp);
}

int http_client_post(const char *path, const uint8_t *body, int body_len, http_response_t *resp)
{
    return do_request(path, "POST", body, body_len, resp);
}
