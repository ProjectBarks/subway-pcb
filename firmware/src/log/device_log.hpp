#pragma once
#include <cstdarg>

enum class LogLevel {
    Error = 0,
    Warn,
    Info,
    Debug,
};

// Initialize the log system (call once at startup)
void device_log_init();

// Log a message. Always outputs to serial (ESP_LOG*).
// When remote is enabled, also buffers for server transmission.
void device_log(LogLevel level, const char* tag, const char* fmt, ...)
    __attribute__((format(printf, 3, 4)));

// Enable/disable remote log collection
void device_log_set_remote(bool enabled);

// Drain collected logs into buf. Returns bytes written. Clears ring buffer.
int device_log_drain(char* buf, int buf_size);

// Convenience macros
#define DLOG_E(tag, fmt, ...) device_log(LogLevel::Error, tag, fmt, ##__VA_ARGS__)
#define DLOG_W(tag, fmt, ...) device_log(LogLevel::Warn,  tag, fmt, ##__VA_ARGS__)
#define DLOG_I(tag, fmt, ...) device_log(LogLevel::Info,  tag, fmt, ##__VA_ARGS__)
#define DLOG_D(tag, fmt, ...) device_log(LogLevel::Debug, tag, fmt, ##__VA_ARGS__)
