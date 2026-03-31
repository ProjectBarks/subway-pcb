#pragma once
#include <atomic>
#include <cstdint>

// Double-buffered SPSC container. Writer publishes to inactive buffer,
// reader reads active buffer. Safe when writes are infrequent relative
// to reads (1Hz write, 33Hz read → ~0.1% collision probability).
template <typename T> class DoubleBuffer {
    T buffers_[2]{};
    std::atomic<uint8_t> active_{0};
    uint8_t write_idx_{1};

  public:
    T& write_buffer() { return buffers_[write_idx_]; }

    void publish() {
        active_.store(write_idx_, std::memory_order_release);
        write_idx_ = 1 - write_idx_;
    }

    const T& read() const { return buffers_[active_.load(std::memory_order_acquire)]; }
};
