#include "log/device_log.hpp"

#include "esp_log.h"
#include "esp_timer.h"
#include "freertos/FreeRTOS.h"
#include "freertos/semphr.h"

#include <cstdarg>
#include <cstdio>
#include <cstring>

// Ring buffer size for remote logs
static constexpr int kLogRingSize = 2048;

static char s_ring[kLogRingSize];
static int s_ring_len = 0;
static bool s_remote_enabled = false;
static SemaphoreHandle_t s_mutex = nullptr;

void device_log_init() {
    s_mutex = xSemaphoreCreateMutex();
    s_ring_len = 0;
    s_remote_enabled = false;
}

static const char* level_char(LogLevel level) {
    switch (level) {
    case LogLevel::Error:
        return "E";
    case LogLevel::Warn:
        return "W";
    case LogLevel::Info:
        return "I";
    case LogLevel::Debug:
        return "D";
    default:
        return "?";
    }
}

void device_log(LogLevel level, const char* tag, const char* fmt, ...) {
    char msg[128];
    va_list args;
    va_start(args, fmt);
    vsnprintf(msg, sizeof(msg), fmt, args);
    va_end(args);

    // Always output to serial via ESP_LOG
    switch (level) {
    case LogLevel::Error:
        ESP_LOGE(tag, "%s", msg);
        break;
    case LogLevel::Warn:
        ESP_LOGW(tag, "%s", msg);
        break;
    case LogLevel::Info:
        ESP_LOGI(tag, "%s", msg);
        break;
    case LogLevel::Debug:
        ESP_LOGD(tag, "%s", msg);
        break;
    }

    // Buffer for remote if enabled
    if (!s_remote_enabled || !s_mutex)
        return;

    uint32_t uptime_ms = static_cast<uint32_t>(esp_timer_get_time() / 1000);

    char line[160];
    int line_len = snprintf(line,
                            sizeof(line),
                            "[%lu][%s][%s] %s\n",
                            static_cast<unsigned long>(uptime_ms),
                            level_char(level),
                            tag,
                            msg);
    if (line_len <= 0)
        return;
    if (line_len >= static_cast<int>(sizeof(line)))
        line_len = sizeof(line) - 1;

    xSemaphoreTake(s_mutex, portMAX_DELAY);
    if (s_ring_len + line_len < kLogRingSize) {
        std::memcpy(s_ring + s_ring_len, line, line_len);
        s_ring_len += line_len;
    }
    // If buffer is full, silently drop -- oldest logs are preserved
    xSemaphoreGive(s_mutex);
}

void device_log_set_remote(bool enabled) {
    s_remote_enabled = enabled;
    if (!enabled && s_mutex) {
        // Clear buffer when disabling
        xSemaphoreTake(s_mutex, portMAX_DELAY);
        s_ring_len = 0;
        xSemaphoreGive(s_mutex);
    }
}

int device_log_drain(char* buf, int buf_size) {
    if (!s_mutex || !buf || buf_size <= 0)
        return 0;

    xSemaphoreTake(s_mutex, portMAX_DELAY);
    int copy_len = s_ring_len < buf_size ? s_ring_len : buf_size - 1;
    if (copy_len > 0) {
        std::memcpy(buf, s_ring, copy_len);
        buf[copy_len] = '\0';
    }
    s_ring_len = 0;
    xSemaphoreGive(s_mutex);

    return copy_len;
}
