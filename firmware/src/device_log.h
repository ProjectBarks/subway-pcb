#ifndef DEVICE_LOG_H
#define DEVICE_LOG_H

#include <stdbool.h>

typedef enum {
    LOG_LEVEL_ERROR = 0,
    LOG_LEVEL_WARN,
    LOG_LEVEL_INFO,
    LOG_LEVEL_DEBUG,
} device_log_level_t;

/* Initialize the log system (call once at startup). */
void device_log_init(void);

/* Log a message. Always outputs to serial (ESP_LOG*).
 * When remote is enabled, also buffers for server transmission. */
void device_log(device_log_level_t level, const char *tag, const char *fmt, ...)
    __attribute__((format(printf, 3, 4)));

/* Enable/disable remote log collection. */
void device_log_set_remote(bool enabled);

/* Drain collected logs into buf. Returns bytes written. Clears the ring buffer. */
int device_log_drain(char *buf, int buf_size);

/* Convenience macros — drop-in replacements for ESP_LOG* */
#define DLOG_E(tag, fmt, ...) device_log(LOG_LEVEL_ERROR, tag, fmt, ##__VA_ARGS__)
#define DLOG_W(tag, fmt, ...) device_log(LOG_LEVEL_WARN,  tag, fmt, ##__VA_ARGS__)
#define DLOG_I(tag, fmt, ...) device_log(LOG_LEVEL_INFO,  tag, fmt, ##__VA_ARGS__)
#define DLOG_D(tag, fmt, ...) device_log(LOG_LEVEL_DEBUG, tag, fmt, ##__VA_ARGS__)

#endif /* DEVICE_LOG_H */
