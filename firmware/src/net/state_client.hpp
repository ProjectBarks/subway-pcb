#pragma once
#include <atomic>

#include "core/triple_buffer.hpp"
#include "core/channel.hpp"
#include "data/transit_snapshot.hpp"
#include "data/board_snapshot.hpp"
#include "data/diag_pad.hpp"
#include "net/http_client.hpp"

class StateClient {
public:
    // Start the state polling task. All references must outlive the task.
    static void start(DoubleBuffer<TransitSnapshot>& transit_buf,
                      BoardStore& board_store,
                      ScriptChannel& script_chan,
                      DiagPad& diag,
                      std::atomic<bool>& ota_active,
                      std::atomic<bool>& http_active);
};
