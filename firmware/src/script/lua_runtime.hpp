#pragma once
#include "core/channel.hpp"
#include "core/triple_buffer.hpp"
#include "data/board_snapshot.hpp"
#include "data/diag_pad.hpp"
#include "data/transit_snapshot.hpp"
#include "hal/led_driver.hpp"

#include <atomic>

class LuaRuntime {
  public:
    // Start the render task. All references must outlive the task.
    static void start(DoubleBuffer<TransitSnapshot>& transit_buf,
                      BoardStore& board_store,
                      ScriptChannel& script_chan,
                      DiagPad& diag,
                      std::atomic<bool>& http_active,
                      LedDriver& led);
};
