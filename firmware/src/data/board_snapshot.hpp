#pragma once
#include <cstdint>
#include <cstring>

#include "freertos/FreeRTOS.h"
#include "freertos/semphr.h"

extern "C" {
#include "subway.pb.h"
}

#include "config/constants.hpp"

struct BoardInfo {
    char board_id[32]{};
    uint32_t led_count = 0;
    uint32_t strip_sizes[kMaxStrips]{};
    uint8_t strip_count = 0;
    char led_map[kMaxLeds][kStopIdLen]{};
    char hash[kHashLen]{};
};

struct StationLedsEntry {
    char station_id[kStopIdLen]{};
    uint16_t led_indices[kMaxLedsPerStation]{};
    uint8_t count = 0;
};

struct BoardSnapshot {
    BoardInfo board;
    StationLedsEntry station_leds[kMaxStations]{};
    uint16_t station_leds_count = 0;
    uint32_t generation = 0;

    // Factory: decode subway_DeviceBoard + build inverted index
    static void from_proto(const subway_DeviceBoard& pb, BoardSnapshot& out);
};

// Single BoardSnapshot + mutex. Board topology changes ~once per boot,
// so mutex contention is negligible. Saves ~26KB vs double-buffered.
class BoardStore {
    BoardSnapshot snapshot_{};
    SemaphoreHandle_t mutex_ = nullptr;
public:
    void init() { mutex_ = xSemaphoreCreateMutex(); }

    // Writer: lock, modify snapshot directly, unlock.
    BoardSnapshot& lock_for_write() {
        xSemaphoreTake(mutex_, portMAX_DELAY);
        return snapshot_;
    }
    void unlock_write() { xSemaphoreGive(mutex_); }

    // Reader: lock, read, unlock. Read is fast (microseconds).
    const BoardSnapshot& lock_for_read() {
        xSemaphoreTake(mutex_, portMAX_DELAY);
        return snapshot_;
    }
    void unlock_read() { xSemaphoreGive(mutex_); }
};
