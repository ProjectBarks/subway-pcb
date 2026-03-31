#pragma once
#include <array>
#include <cstdint>

// nanopb generated types
extern "C" {
#include "subway.pb.h"
}

#include "config/constants.hpp"

struct TransitSnapshot {
    std::array<subway_Station, kMaxStations> stations{};
    pb_size_t station_count = 0;
    uint64_t timestamp = 0;
    std::array<subway_DeviceState_ConfigEntry, kMaxConfig> config{};
    pb_size_t config_count = 0;
    uint32_t board_generation = 0;
};
