#pragma once
#include <atomic>
#include <cstdint>

#include "config/constants.hpp"

struct HttpClientConfig {
    const char* server_url;
    const char* device_id;
    const char* firmware_ver;
    const char* hardware;
};

struct HttpResponse {
    uint8_t* data;
    int len;
};

class HttpClient {
    HttpClientConfig cfg_{};
    std::atomic<bool>* http_active_ = nullptr;

    int do_request(const char* path, const char* method,
                   const uint8_t* body, int body_len, HttpResponse* resp);

public:
    // Static instance pointer for C callback trampoline.
    // Only one HttpClient instance exists (state task owns it).
    static HttpClient* s_instance_;

    // Buffer and length are public so the C event handler can write to them.
    // Only accessed from the networking task (single-threaded within a request).
    uint8_t buf_[kHttpBufSize]{};
    int buf_len_ = 0;

    void init(const HttpClientConfig& cfg, std::atomic<bool>& http_active);
    int get(const char* path, HttpResponse* resp);
    int post(const char* path, const uint8_t* body, int body_len, HttpResponse* resp);
};
