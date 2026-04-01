#pragma once
#include <chrono>
#include <condition_variable>
#include <cstdlib>
#include <mutex>

class ScriptChannel {
    std::mutex mutex_;
    std::condition_variable cv_;
    char* pending_ = nullptr;

  public:
    ScriptChannel() = default;
    ~ScriptChannel() { free(pending_); }

    // Takes ownership of ptr (must be malloc/strdup'd).
    // Frees any previously pending script that hasn't been received.
    void send(char* ptr) {
        std::lock_guard lock(mutex_);
        free(pending_);
        pending_ = ptr;
        cv_.notify_one();
    }

    // Returns true if a script was received. Caller takes ownership of out.
    // timeout_ms=0 means non-blocking check.
    bool receive(char*& out, uint32_t timeout_ms = 0) {
        std::unique_lock lock(mutex_);
        if (timeout_ms > 0) {
            cv_.wait_for(lock, std::chrono::milliseconds(timeout_ms), [this] {
                return pending_ != nullptr;
            });
        }
        if (!pending_)
            return false;
        out = pending_;
        pending_ = nullptr;
        return true;
    }

    ScriptChannel(const ScriptChannel&) = delete;
    ScriptChannel& operator=(const ScriptChannel&) = delete;
};
