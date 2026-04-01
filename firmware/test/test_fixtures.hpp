#pragma once
#include <cstring>

#include "config/constants.hpp"
#include "core/types.hpp"
#include "data/board_snapshot.hpp"
#include "data/transit_snapshot.hpp"
#include "script/lua_bindings.hpp"

extern "C" {
#include "subway.pb.h"
}

/* Shared test fixtures used by both conformance and E2E tests.
 *
 * Board: 5 LEDs, 2 strips (3+2)
 *   LED 0,1 → A01 | LED 2 → B02 | LED 3 → (unmapped) | LED 4 → C03
 *
 * Stations:
 *   A01: train route "1", STOPPED_AT
 *   B02: train route "A", IN_TRANSIT_TO
 *
 * Config: brightness=200, color=#FF8800, empty="", name=test
 */

struct TestFixtures {
    TransitSnapshot transit{};
    BoardSnapshot board{};
    Rgb pixels[512]{};
    LuaBindingContext ctx{};

    void reset() {
        std::memset(&transit, 0, sizeof(transit));
        std::memset(&board, 0, sizeof(board));
        std::memset(pixels, 0, sizeof(pixels));

        // Board
        board.board.led_count = 5;
        board.board.strip_count = 2;
        board.board.strip_sizes[0] = 3;
        board.board.strip_sizes[1] = 2;

        std::strncpy(board.board.led_map[0], "A01", kStopIdLen);
        std::strncpy(board.board.led_map[1], "A01", kStopIdLen);
        std::strncpy(board.board.led_map[2], "B02", kStopIdLen);
        board.board.led_map[3][0] = '\0';
        std::strncpy(board.board.led_map[4], "C03", kStopIdLen);

        board.station_leds_count = 3;
        std::strncpy(board.station_leds[0].station_id, "A01", kStopIdLen);
        board.station_leds[0].led_indices[0] = 0;
        board.station_leds[0].led_indices[1] = 1;
        board.station_leds[0].count = 2;
        std::strncpy(board.station_leds[1].station_id, "B02", kStopIdLen);
        board.station_leds[1].led_indices[0] = 2;
        board.station_leds[1].count = 1;
        std::strncpy(board.station_leds[2].station_id, "C03", kStopIdLen);
        board.station_leds[2].led_indices[0] = 4;
        board.station_leds[2].count = 1;

        // Stations
        transit.station_count = 2;
        std::strncpy(transit.stations[0].stop_id, "A01", sizeof(transit.stations[0].stop_id));
        transit.stations[0].trains_count = 1;
        std::strncpy(transit.stations[0].trains[0].route, "1",
                     sizeof(transit.stations[0].trains[0].route));
        transit.stations[0].trains[0].status = subway_TrainStatus_STOPPED_AT;
        std::strncpy(transit.stations[1].stop_id, "B02", sizeof(transit.stations[1].stop_id));
        transit.stations[1].trains_count = 1;
        std::strncpy(transit.stations[1].trains[0].route, "A",
                     sizeof(transit.stations[1].trains[0].route));
        transit.stations[1].trains[0].status = subway_TrainStatus_IN_TRANSIT_TO;

        // Config
        transit.config_count = 4;
        std::strncpy(transit.config[0].key, "brightness", sizeof(transit.config[0].key));
        std::strncpy(transit.config[0].value, "200", sizeof(transit.config[0].value));
        std::strncpy(transit.config[1].key, "color", sizeof(transit.config[1].key));
        std::strncpy(transit.config[1].value, "#FF8800", sizeof(transit.config[1].value));
        std::strncpy(transit.config[2].key, "empty", sizeof(transit.config[2].key));
        std::strncpy(transit.config[2].value, "", sizeof(transit.config[2].value));
        std::strncpy(transit.config[3].key, "name", sizeof(transit.config[3].key));
        std::strncpy(transit.config[3].value, "test", sizeof(transit.config[3].value));

        // Context
        ctx.transit = &transit;
        ctx.board = &board;
        ctx.pixels = std::span<Rgb>(pixels, board.board.led_count);
        ctx.led_count = board.board.led_count;
    }

    // Build a subway_DeviceBoard protobuf matching the fixture board
    void build_proto_board(subway_DeviceBoard& pb) const {
        std::memset(&pb, 0, sizeof(pb));
        std::strncpy(pb.hash, "test-board-hash", sizeof(pb.hash) - 1);
        std::strncpy(pb.board_id, "test-board", sizeof(pb.board_id) - 1);
        pb.led_count = 5;
        pb.strip_sizes_count = 2;
        pb.strip_sizes[0] = 3;
        pb.strip_sizes[1] = 2;
        pb.led_map_count = 4;
        pb.led_map[0] = {0, "A01"};
        pb.led_map[1] = {1, "A01"};
        pb.led_map[2] = {2, "B02"};
        pb.led_map[3] = {4, "C03"};
    }

    // Build a subway_DeviceState protobuf with route "1" = #FF0000
    void build_proto_state(subway_DeviceState& pb) const {
        std::memset(&pb, 0, sizeof(pb));
        std::strncpy(pb.script_hash, "test-script-hash", sizeof(pb.script_hash) - 1);
        std::strncpy(pb.board_hash, "test-board-hash", sizeof(pb.board_hash) - 1);
        pb.timestamp = 1000;
        pb.stations_count = 2;
        std::strncpy(pb.stations[0].stop_id, "A01", sizeof(pb.stations[0].stop_id));
        pb.stations[0].trains_count = 1;
        std::strncpy(pb.stations[0].trains[0].route, "1",
                     sizeof(pb.stations[0].trains[0].route));
        pb.stations[0].trains[0].status = subway_TrainStatus_STOPPED_AT;
        std::strncpy(pb.stations[1].stop_id, "B02", sizeof(pb.stations[1].stop_id));
        pb.stations[1].trains_count = 1;
        std::strncpy(pb.stations[1].trains[0].route, "A",
                     sizeof(pb.stations[1].trains[0].route));
        pb.stations[1].trains[0].status = subway_TrainStatus_IN_TRANSIT_TO;
        pb.config_count = 2;
        std::strncpy(pb.config[0].key, "1", sizeof(pb.config[0].key));
        std::strncpy(pb.config[0].value, "#FF0000", sizeof(pb.config[0].value));
        std::strncpy(pb.config[1].key, "brightness", sizeof(pb.config[1].key));
        std::strncpy(pb.config[1].value, "200", sizeof(pb.config[1].value));
    }
};
