#pragma once
#include "../config/board_config.hpp"
#include "../core/channel.hpp"
#include "../data/diag_pad.hpp"
#include "../hal/led_driver.hpp"

class SerialHandler {
  public:
    struct Context {
        DiagPad* diag;
        BoardHwConfig* hw_config;
        LedDriver* led_driver;
        ScriptChannel* script_chan;
    };
    static void start(const Context& ctx);
};
