#pragma once
#include <cstdlib>
#include "freertos/FreeRTOS.h"
#include "freertos/queue.h"

class ScriptChannel {
    QueueHandle_t q_;
public:
    ScriptChannel() : q_(xQueueCreate(1, sizeof(char*))) {}
    ~ScriptChannel() {
        char* remaining = nullptr;
        while (xQueueReceive(q_, &remaining, 0) == pdTRUE) {
            free(remaining);
        }
        vQueueDelete(q_);
    }

    // Takes ownership of ptr (must be malloc/strdup'd)
    void send(char* ptr) {
        char* old = nullptr;
        if (xQueueReceive(q_, &old, 0) == pdTRUE) {
            free(old);  // drain displaced entry to prevent leak
        }
        xQueueSend(q_, &ptr, portMAX_DELAY);
    }

    bool receive(char*& out, TickType_t timeout = 0) {
        return xQueueReceive(q_, &out, timeout) == pdTRUE;
    }

    // Non-copyable
    ScriptChannel(const ScriptChannel&) = delete;
    ScriptChannel& operator=(const ScriptChannel&) = delete;
};
