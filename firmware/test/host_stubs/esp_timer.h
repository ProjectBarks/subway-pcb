#ifndef ESP_TIMER_H
#define ESP_TIMER_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>
#include <time.h>

static inline int64_t esp_timer_get_time(void)
{
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (int64_t)ts.tv_sec * 1000000 + ts.tv_nsec / 1000;
}

#ifdef __cplusplus
}
#endif

#endif
