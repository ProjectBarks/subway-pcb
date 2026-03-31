#include "data/board_snapshot.hpp"

#include <cstring>

void BoardSnapshot::from_proto(const subway_DeviceBoard& pb, BoardSnapshot& out) {
    // Zero-init the output
    out = BoardSnapshot{};

    // Copy board_id
    std::strncpy(out.board.board_id, pb.board_id, sizeof(out.board.board_id) - 1);

    // Copy led_count
    out.board.led_count = pb.led_count;

    // Copy strip_sizes
    out.board.strip_count =
        static_cast<uint8_t>(pb.strip_sizes_count < kMaxStrips ? pb.strip_sizes_count : kMaxStrips);
    for (uint8_t i = 0; i < out.board.strip_count; i++) {
        out.board.strip_sizes[i] = pb.strip_sizes[i];
    }

    // Copy hash
    std::strncpy(out.board.hash, pb.hash, kHashLen - 1);

    // Copy led_map entries: pb.led_map[i].key is the LED index,
    // pb.led_map[i].value is the stop_id string
    std::memset(out.board.led_map, 0, sizeof(out.board.led_map));
    for (pb_size_t i = 0; i < pb.led_map_count; i++) {
        uint32_t idx = pb.led_map[i].key;
        if (idx < kMaxLeds) {
            std::strncpy(out.board.led_map[idx], pb.led_map[i].value, kStopIdLen - 1);
        }
    }

    // Build inverted index: for each unique station in led_map,
    // collect all LED indices that map to it
    out.station_leds_count = 0;
    for (uint32_t i = 0; i < out.board.led_count && i < kMaxLeds; i++) {
        if (out.board.led_map[i][0] == '\0')
            continue;

        // Find existing entry or create new one
        int found = -1;
        for (uint16_t j = 0; j < out.station_leds_count; j++) {
            if (std::strcmp(out.station_leds[j].station_id, out.board.led_map[i]) == 0) {
                found = j;
                break;
            }
        }

        if (found < 0) {
            if (out.station_leds_count >= kMaxStations)
                continue;
            found = out.station_leds_count++;
            std::strncpy(out.station_leds[found].station_id, out.board.led_map[i], kStopIdLen - 1);
            out.station_leds[found].count = 0;
        }

        if (out.station_leds[found].count < kMaxLedsPerStation) {
            out.station_leds[found].led_indices[out.station_leds[found].count++] =
                static_cast<uint16_t>(i);
        }
    }
}
