#pragma once
#include <atomic>
#include <cstdint>
#include <climits>
#include <cstring>

#include "freertos/FreeRTOS.h"
#include "freertos/semphr.h"

struct DiagPad {
    std::atomic<uint32_t> nonzero_pixels{0};
    std::atomic<uint32_t> pushed_pixels{0};
    std::atomic<int32_t>  strip_ok{0};
    std::atomic<int32_t>  strip_fail{0};
    std::atomic<int32_t>  lua_errors{0};
    std::atomic<uint32_t> lua_mem{0};
    std::atomic<uint32_t> first_lit_led{UINT32_MAX};
    std::atomic<int32_t>  last_reload{0};
    std::atomic<uint32_t> frame_time_us{0};
    std::atomic<uint32_t> stack_hwm_render{0};
    std::atomic<uint32_t> stack_hwm_state{0};

    // String fields: written rarely, read ~1/s
    SemaphoreHandle_t str_mutex = nullptr;
    char last_lua_err[64]{};
    char fetch_err[64]{};

    void init() {
        str_mutex = xSemaphoreCreateMutex();
    }

    void set_lua_err(const char* err) {
        if (xSemaphoreTake(str_mutex, pdMS_TO_TICKS(50)) == pdTRUE) {
            strncpy(last_lua_err, err, sizeof(last_lua_err) - 1);
            last_lua_err[sizeof(last_lua_err) - 1] = '\0';
            xSemaphoreGive(str_mutex);
        }
    }

    void set_fetch_err(const char* err) {
        if (xSemaphoreTake(str_mutex, pdMS_TO_TICKS(50)) == pdTRUE) {
            strncpy(fetch_err, err, sizeof(fetch_err) - 1);
            fetch_err[sizeof(fetch_err) - 1] = '\0';
            xSemaphoreGive(str_mutex);
        }
    }
};
