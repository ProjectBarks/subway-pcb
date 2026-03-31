#include "net/http_client.hpp"

#include "esp_crt_bundle.h"
#include "esp_http_client.h"
#include "esp_log.h"
#include "log/device_log.hpp"

#include <cstdio>
#include <cstring>

static const char* TAG = "http_client";

// Static instance pointer for C callback trampoline
HttpClient* HttpClient::s_instance_ = nullptr;

static esp_err_t http_event_handler(esp_http_client_event_t* evt) {
    auto* self = HttpClient::s_instance_;
    if (!self)
        return ESP_OK;

    switch (evt->event_id) {
    case HTTP_EVENT_ON_DATA:
        if (self->buf_len_ + evt->data_len <= kHttpBufSize) {
            std::memcpy(self->buf_ + self->buf_len_, evt->data, evt->data_len);
            self->buf_len_ += evt->data_len;
        } else {
            DLOG_W(TAG,
                   "Response truncated: %d + %d > %u",
                   self->buf_len_,
                   static_cast<int>(evt->data_len),
                   static_cast<unsigned>(kHttpBufSize));
        }
        break;
    default:
        break;
    }
    return ESP_OK;
}

void HttpClient::init(const HttpClientConfig& cfg, std::atomic<bool>& http_active) {
    cfg_ = cfg;
    http_active_ = &http_active;
    s_instance_ = this;
    DLOG_I(TAG, "HTTP client initialized (server=%s, device=%s)", cfg.server_url, cfg.device_id);
}

int HttpClient::do_request(
    const char* path, const char* method, const uint8_t* body, int body_len, HttpResponse* resp) {
    if (http_active_) {
        http_active_->store(true, std::memory_order_release);
    }

    char url[384];
    std::snprintf(url, sizeof(url), "%s%s", cfg_.server_url, path);

    buf_len_ = 0;

    esp_http_client_config_t http_cfg{};
    http_cfg.url = url;
    http_cfg.event_handler = http_event_handler;
    http_cfg.timeout_ms = 10000;
    http_cfg.crt_bundle_attach = esp_crt_bundle_attach;

    esp_http_client_handle_t client = esp_http_client_init(&http_cfg);
    if (!client) {
        DLOG_E(TAG, "Failed to create HTTP client");
        if (http_active_) {
            http_active_->store(false, std::memory_order_release);
        }
        return -1;
    }

    esp_http_client_set_header(client, "X-Device-ID", cfg_.device_id);
    esp_http_client_set_header(client, "X-Firmware-Version", cfg_.firmware_ver);
    esp_http_client_set_header(client, "X-Hardware", cfg_.hardware);

    if (body && body_len > 0) {
        esp_http_client_set_method(client, HTTP_METHOD_POST);
        esp_http_client_set_header(client, "Content-Type", "application/x-protobuf");
        esp_http_client_set_post_field(client, reinterpret_cast<const char*>(body), body_len);
    }

    esp_err_t err = esp_http_client_perform(client);
    int status = esp_http_client_get_status_code(client);
    esp_http_client_cleanup(client);

    if (http_active_) {
        http_active_->store(false, std::memory_order_release);
    }

    if (err != ESP_OK || status != 200) {
        DLOG_E(TAG,
               "HTTP %s %s: err=%d(%s) status=%d",
               method,
               path,
               err,
               esp_err_to_name(err),
               status);
        return -1;
    }

    if (resp) {
        resp->data = buf_;
        resp->len = buf_len_;
    }
    return 0;
}

int HttpClient::get(const char* path, HttpResponse* resp) {
    return do_request(path, "GET", nullptr, 0, resp);
}

int HttpClient::post(const char* path, const uint8_t* body, int body_len, HttpResponse* resp) {
    return do_request(path, "POST", body, body_len, resp);
}
