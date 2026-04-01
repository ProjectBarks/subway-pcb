#pragma once
#include <atomic>
#include <chrono>
#include <climits>
#include <cstdint>
#include <cstring>
#include <mutex>

struct DiagPad {
    std::atomic<uint32_t> nonzero_pixels{0};
    std::atomic<uint32_t> pushed_pixels{0};
    std::atomic<int32_t> strip_ok{0};
    std::atomic<int32_t> strip_fail{0};
    std::atomic<int32_t> lua_errors{0};
    std::atomic<uint32_t> lua_mem{0};
    std::atomic<uint32_t> first_lit_led{UINT32_MAX};
    std::atomic<int32_t> last_reload{0};
    std::atomic<uint32_t> frame_time_us{0};
    std::atomic<uint32_t> stack_hwm_render{0};
    std::atomic<uint32_t> stack_hwm_state{0};

    // String fields: written rarely, read ~1/s
    std::timed_mutex str_mutex;
    char last_lua_err[64]{};
    char fetch_err[64]{};

    void init() {}

    void set_lua_err(const char* err) {
        if (str_mutex.try_lock_for(std::chrono::milliseconds(50))) {
            strncpy(last_lua_err, err, sizeof(last_lua_err) - 1);
            last_lua_err[sizeof(last_lua_err) - 1] = '\0';
            str_mutex.unlock();
        }
    }

    void set_fetch_err(const char* err) {
        if (str_mutex.try_lock_for(std::chrono::milliseconds(50))) {
            strncpy(fetch_err, err, sizeof(fetch_err) - 1);
            fetch_err[sizeof(fetch_err) - 1] = '\0';
            str_mutex.unlock();
        }
    }
};
