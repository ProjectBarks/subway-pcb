/* End-to-end functional tests for the firmware data pipeline:
 *   protobuf bytes → codec decode → BoardSnapshot/TransitSnapshot → Lua render → pixel output
 *
 * Lua scripts are loaded from service/internal/lua/ (the real production scripts). */

#include <cstring>
#include <cmath>

#include "test_utils.hpp"
#include "test_fixtures.hpp"
#include "proto/codec.hpp"

extern "C" {
#include <pb_encode.h>
}

// Path to production Lua scripts (relative to firmware/test/)
static constexpr const char* LUA_DIR = "../../service/internal/lua";

static TestResults r;

// --- Helpers ---

static BoardSnapshot decode_board(const TestFixtures& fix) {
    subway_DeviceBoard pb{};
    fix.build_proto_board(pb);
    uint8_t buf[4096];
    pb_ostream_t os = pb_ostream_from_buffer(buf, sizeof(buf));
    pb_encode(&os, subway_DeviceBoard_fields, &pb);
    subway_DeviceBoard dec{};
    codec::decode_board(buf, static_cast<int>(os.bytes_written), dec);
    BoardSnapshot snap{};
    BoardSnapshot::from_proto(dec, snap);
    return snap;
}

static TransitSnapshot decode_state(const TestFixtures& fix) {
    subway_DeviceState pb{};
    fix.build_proto_state(pb);
    uint8_t buf[8192];
    pb_ostream_t os = pb_ostream_from_buffer(buf, sizeof(buf));
    pb_encode(&os, subway_DeviceState_fields, &pb);
    subway_DeviceState dec{};
    codec::decode_state(buf, static_cast<int>(os.bytes_written), dec);
    TransitSnapshot transit{};
    memcpy(transit.stations.data(), dec.stations, sizeof(subway_Station) * dec.stations_count);
    transit.station_count = dec.stations_count;
    memcpy(transit.config.data(), dec.config,
           sizeof(subway_DeviceState_ConfigEntry) * dec.config_count);
    transit.config_count = dec.config_count;
    return transit;
}

// --- Tests ---

static void test_protobuf_roundtrip_board() {
    TestFixtures fix;
    fix.reset();
    subway_DeviceBoard src{};
    fix.build_proto_board(src);
    uint8_t buf[4096];
    pb_ostream_t os = pb_ostream_from_buffer(buf, sizeof(buf));
    check(r, pb_encode(&os, subway_DeviceBoard_fields, &src), "encode DeviceBoard");
    check(r, os.bytes_written > 0, "encoded bytes > 0");
    subway_DeviceBoard dst{};
    check(r, codec::decode_board(buf, static_cast<int>(os.bytes_written), dst), "decode DeviceBoard");
    check(r, dst.led_count == 5, "board led_count == 5");
    check(r, dst.strip_sizes_count == 2, "strip_sizes_count == 2");
    check(r, strcmp(dst.board_id, "test-board") == 0, "board_id matches");
}

static void test_protobuf_roundtrip_state() {
    TestFixtures fix;
    fix.reset();
    subway_DeviceState src{};
    fix.build_proto_state(src);
    uint8_t buf[8192];
    pb_ostream_t os = pb_ostream_from_buffer(buf, sizeof(buf));
    check(r, pb_encode(&os, subway_DeviceState_fields, &src), "encode DeviceState");
    subway_DeviceState dst{};
    check(r, codec::decode_state(buf, static_cast<int>(os.bytes_written), dst), "decode DeviceState");
    check(r, dst.stations_count == 2, "stations_count == 2");
    check(r, strcmp(dst.stations[0].stop_id, "A01") == 0, "station 0 is A01");
    check(r, dst.stations[0].trains[0].status == subway_TrainStatus_STOPPED_AT, "A01 STOPPED_AT");
    check(r, dst.config_count == 2, "config_count == 2");
}

static void test_board_snapshot_inverted_index() {
    TestFixtures fix;
    fix.reset();
    auto snap = decode_board(fix);
    check(r, snap.board.led_count == 5, "led_count == 5");
    check(r, strcmp(snap.board.led_map[0], "A01") == 0, "led_map[0] == A01");
    check(r, snap.board.led_map[3][0] == '\0', "led_map[3] unmapped");
    check(r, snap.station_leds_count == 3, "3 stations in index");

    bool found = false;
    for (uint16_t i = 0; i < snap.station_leds_count; i++) {
        if (strcmp(snap.station_leds[i].station_id, "A01") == 0) {
            found = true;
            check(r, snap.station_leds[i].count == 2, "A01 has 2 LEDs");
        }
    }
    check(r, found, "A01 in inverted index");
}

static void test_track_lua_e2e() {
    TestFixtures fix;
    fix.reset();
    auto board = decode_board(fix);
    auto transit = decode_state(fix);

    Rgb pixels[512]{};
    LuaBindingContext ctx{&transit, &board, {pixels, board.board.led_count}, board.board.led_count};

    lua_State* L = create_test_lua();
    check(r, L != nullptr, "Lua state created");
    lua_set_binding_context(L, &ctx);

    // Load the REAL track.lua from service/internal/lua/
    char path[512];
    snprintf(path, sizeof(path), "%s/track.lua", LUA_DIR);
    char* src = read_file(path);
    check(r, src != nullptr, "track.lua read from disk");
    if (!src) return;

    check(r, luaL_dostring(L, src) == 0, "track.lua loaded");
    free(src);

    lua_getglobal(L, "render");
    check(r, lua_pcall(L, 0, 0, 0) == 0, "render() succeeded");

    // LED 0 → A01, route "1" STOPPED_AT, config "1"=#FF0000, brightness=200
    int expected_r = static_cast<int>(floor(255.0 * 200.0 / 255.0));
    check(r, pixels[0].r == expected_r, "LED 0 (A01) red");
    check(r, pixels[0].g == 0, "LED 0 (A01) green");
    check(r, pixels[0].b == 0, "LED 0 (A01) blue");
    check(r, pixels[1].r == expected_r, "LED 1 (A01) red");
    check(r, pixels[2].r == 0, "LED 2 (B02) black");
    check(r, pixels[3].r == 0, "LED 3 (unmapped) black");
    check(r, pixels[4].r == 0, "LED 4 (C03) black");

    lua_close(L);
}

static void test_empty_state_all_black() {
    TestFixtures fix;
    fix.reset();
    // Override: clear all stations
    fix.transit.station_count = 0;
    fix.transit.config_count = 0;
    fix.ctx.transit = &fix.transit;

    lua_State* L = create_test_lua();
    lua_set_binding_context(L, &fix.ctx);

    char path[512];
    snprintf(path, sizeof(path), "%s/track.lua", LUA_DIR);
    char* src = read_file(path);
    if (src) { luaL_dostring(L, src); free(src); }
    lua_getglobal(L, "render");
    lua_pcall(L, 0, 0, 0);

    bool all_black = true;
    for (uint32_t i = 0; i < fix.ctx.led_count; i++)
        if (fix.pixels[i].r || fix.pixels[i].g || fix.pixels[i].b) { all_black = false; break; }
    check(r, all_black, "empty state all-black");
    lua_close(L);
}

static void test_malformed_script_no_crash() {
    TestFixtures fix;
    fix.reset();

    lua_State* L = create_test_lua();
    lua_set_binding_context(L, &fix.ctx);

    check(r, luaL_dostring(L, "function render() this is not valid lua end") != 0,
          "malformed script errors");

    lua_getglobal(L, "render");
    if (lua_isfunction(L, -1)) {
        check(r, lua_pcall(L, 0, 0, 0) != 0, "malformed render errors");
    } else {
        lua_pop(L, 1);
        r.pass++;
    }

    bool all_black = true;
    for (uint32_t i = 0; i < fix.ctx.led_count; i++)
        if (fix.pixels[i].r || fix.pixels[i].g || fix.pixels[i].b) { all_black = false; break; }
    check(r, all_black, "pixels black after error");
    lua_close(L);
}

int main() {
    printf("Running E2E tests...\n");
    test_protobuf_roundtrip_board();
    test_protobuf_roundtrip_state();
    test_board_snapshot_inverted_index();
    test_track_lua_e2e();
    test_empty_state_all_black();
    test_malformed_script_no_crash();
    printf("\n%s: %d passed, %d failed\n",
           r.fail > 0 ? "FAILED" : "PASSED", r.pass, r.fail);
    return r.fail > 0 ? 1 : 0;
}
