/* Stress tests for firmware subsystems:
 *   - Lua memory pressure (OOM recovery, string bomb, GC reclaim, small allocs)
 *   - Protobuf robustness (truncated, empty, garbage, max stations/config)
 *   - BoardSnapshot limits (max size, all same station, empty board)
 *   - DoubleBuffer SPSC concurrency
 *   - ScriptChannel concurrency */

#include <cstring>
#include <thread>
#include <atomic>

#include "test_utils.hpp"
#include "proto/codec.hpp"
#include "core/triple_buffer.hpp"
#include "core/channel.hpp"

extern "C" {
#include <pb_encode.h>
}

/* Include lua_bindings.cpp directly (same pattern as conformance test) */
#include "script/lua_bindings.cpp"

// --- Custom allocator with configurable cap (mirrors firmware) ---

static int32_t s_stress_mem_used = 0;
static int32_t s_stress_mem_cap = 40 * 1024;

static void* stress_alloc(void* ud, void* ptr, size_t osize, size_t nsize) {
    (void)ud;
    if (nsize == 0) {
        s_stress_mem_used -= static_cast<int32_t>(osize);
        if (s_stress_mem_used < 0) s_stress_mem_used = 0;
        free(ptr);
        return nullptr;
    }
    int32_t delta = static_cast<int32_t>(nsize) - static_cast<int32_t>(osize);
    if (s_stress_mem_used + delta > s_stress_mem_cap)
        return nullptr;
    void* new_ptr = realloc(ptr, nsize);
    if (new_ptr) {
        s_stress_mem_used += delta;
        if (s_stress_mem_used < 0) s_stress_mem_used = 0;
    }
    return new_ptr;
}

static lua_State* create_capped_lua() {
    s_stress_mem_used = 0;
    lua_State* L = lua_newstate(stress_alloc, nullptr);
    if (!L) return nullptr;
    luaL_requiref(L, "_G", luaopen_base, 1);
    luaL_requiref(L, "math", luaopen_math, 1);
    luaL_requiref(L, "string", luaopen_string, 1);
    luaL_requiref(L, "table", luaopen_table, 1);
    lua_pop(L, 4);
    lua_register_bindings(L);
    return L;
}

static TestResults r;

// ================================================================
// Lua Memory Pressure
// ================================================================

static void test_oom_recovery_loop() {
    for (int cycle = 0; cycle < 100; cycle++) {
        lua_State* L = create_capped_lua();
        check(r, L != nullptr, "OOM loop: create VM");
        if (!L) return;

        // Script that allocates until OOM inside pcall
        int rc = luaL_dostring(L, R"(
            local ok, err = pcall(function()
                local t = {}
                for i = 1, 1000000 do t[i] = string.rep("x", 1024) end
            end)
            _oom_caught = not ok
        )");
        if (rc == LUA_OK) {
            lua_getglobal(L, "_oom_caught");
            check(r, lua_toboolean(L, -1), "OOM loop: pcall caught OOM");
            lua_pop(L, 1);
        }
        lua_close(L);
    }
    int32_t mem_after = s_stress_mem_used;
    check(r, mem_after == 0, "OOM loop: mem resets to 0 after close");
}

static void test_string_bomb() {
    s_stress_mem_cap = 40 * 1024;
    lua_State* L = create_capped_lua();
    check(r, L != nullptr, "string bomb: create VM");
    if (!L) return;

    int rc = luaL_dostring(L, R"(
        local ok, err = pcall(function()
            local s = "x"
            for i = 1, 30 do s = s .. s end
        end)
        _bomb_caught = not ok
    )");
    check(r, rc == LUA_OK, "string bomb: script ran");
    if (rc == LUA_OK) {
        lua_getglobal(L, "_bomb_caught");
        check(r, lua_toboolean(L, -1), "string bomb: allocator rejected");
        lua_pop(L, 1);
    }
    lua_close(L);
}

static void test_gc_reclaim() {
    s_stress_mem_cap = 40 * 1024;
    lua_State* L = create_capped_lua();
    check(r, L != nullptr, "GC reclaim: create VM");
    if (!L) return;

    int rc = luaL_dostring(L, R"(
        -- Allocate ~10KB (VM overhead uses ~15KB of the 40KB cap)
        local big = string.rep("x", 10000)
        big = nil
        collectgarbage("collect")
        -- Should succeed again after GC freed it
        local big2 = string.rep("y", 10000)
        _gc_ok = (big2 ~= nil and #big2 == 10000)
    )");
    check(r, rc == LUA_OK, "GC reclaim: script ran");
    if (rc == LUA_OK) {
        lua_getglobal(L, "_gc_ok");
        check(r, lua_toboolean(L, -1), "GC reclaim: re-alloc after GC");
        lua_pop(L, 1);
    }
    lua_close(L);
}

static void test_many_small_allocs() {
    s_stress_mem_cap = 40 * 1024;
    lua_State* L = create_capped_lua();
    check(r, L != nullptr, "small allocs: create VM");
    if (!L) return;

    int rc = luaL_dostring(L, R"(
        local ok, err = pcall(function()
            local t = {}
            for i = 1, 100000 do t[i] = {} end
        end)
        _alloc_hit_limit = not ok
    )");
    check(r, rc == LUA_OK, "small allocs: script ran");
    if (rc == LUA_OK) {
        lua_getglobal(L, "_alloc_hit_limit");
        check(r, lua_toboolean(L, -1), "small allocs: hit cap");
        lua_pop(L, 1);
    }
    lua_close(L);
}

// ================================================================
// Protobuf Robustness
// ================================================================

static void test_proto_truncated() {
    uint8_t buf[10] = {0x08, 0x05, 0x10, 0x01, 0x1A, 0x03, 0x41, 0x30, 0x31, 0x00};
    subway_DeviceBoard dst{};
    check(r, !codec::decode_board(buf, 10, dst), "truncated: decode returns false");
}

static void test_proto_empty() {
    // Empty protobuf is valid (all fields take defaults) — just verify no crash
    subway_DeviceBoard dst{};
    uint8_t empty = 0;
    codec::decode_board(&empty, 0, dst);
    check(r, dst.led_count == 0, "empty: led_count == 0");
    check(r, dst.led_map_count == 0, "empty: led_map_count == 0");
}

static void test_proto_garbage() {
    uint8_t buf[1024];
    memset(buf, 0xDE, sizeof(buf));
    subway_DeviceBoard dst{};
    bool ok = codec::decode_board(buf, sizeof(buf), dst);
    // Don't crash — result doesn't matter as long as no segfault
    (void)ok;
    r.pass++;
}

static void test_proto_max_stations() {
    subway_DeviceState src{};
    src.stations_count = kMaxStations;
    for (uint32_t i = 0; i < kMaxStations; i++) {
        snprintf(src.stations[i].stop_id, sizeof(src.stations[i].stop_id), "S%03u", i);
        src.stations[i].trains_count = 0;
    }
    uint8_t buf[32768];
    pb_ostream_t os = pb_ostream_from_buffer(buf, sizeof(buf));
    check(r, pb_encode(&os, subway_DeviceState_fields, &src), "max stations: encode");
    subway_DeviceState dst{};
    check(r, codec::decode_state(buf, static_cast<int>(os.bytes_written), dst),
          "max stations: decode");
    check(r, dst.stations_count == kMaxStations, "max stations: count matches");
}

static void test_proto_max_config() {
    subway_DeviceState src{};
    src.config_count = kMaxConfig;
    for (uint32_t i = 0; i < kMaxConfig; i++) {
        snprintf(src.config[i].key, sizeof(src.config[i].key), "k%u", i);
        snprintf(src.config[i].value, sizeof(src.config[i].value), "v%u", i);
    }
    uint8_t buf[32768];
    pb_ostream_t os = pb_ostream_from_buffer(buf, sizeof(buf));
    check(r, pb_encode(&os, subway_DeviceState_fields, &src), "max config: encode");
    subway_DeviceState dst{};
    check(r, codec::decode_state(buf, static_cast<int>(os.bytes_written), dst),
          "max config: decode");
    check(r, dst.config_count == kMaxConfig, "max config: count matches");
}

// ================================================================
// BoardSnapshot Limits
// ================================================================

static void test_board_max_size() {
    subway_DeviceBoard pb{};
    memset(&pb, 0, sizeof(pb));
    strncpy(pb.board_id, "max-board", sizeof(pb.board_id) - 1);
    strncpy(pb.hash, "max-hash", sizeof(pb.hash) - 1);
    pb.led_count = kMaxLeds;
    pb.strip_sizes_count = kMaxStrips;
    for (uint32_t i = 0; i < kMaxStrips; i++)
        pb.strip_sizes[i] = kMaxLeds / kMaxStrips;

    // Map all LEDs to unique stations (up to kMaxStations)
    pb.led_map_count = kMaxLeds < 300 ? kMaxLeds : 300;
    for (uint32_t i = 0; i < pb.led_map_count; i++) {
        pb.led_map[i].key = i;
        snprintf(pb.led_map[i].value, sizeof(pb.led_map[i].value), "S%03u", i % kMaxStations);
    }

    BoardSnapshot snap{};
    BoardSnapshot::from_proto(pb, snap);
    check(r, snap.board.led_count == kMaxLeds, "max board: led_count");
    check(r, snap.board.strip_count == kMaxStrips, "max board: strip_count");
}

static void test_board_all_same_station() {
    subway_DeviceBoard pb{};
    memset(&pb, 0, sizeof(pb));
    strncpy(pb.board_id, "same-station", sizeof(pb.board_id) - 1);
    pb.led_count = kMaxLeds;
    pb.strip_sizes_count = 1;
    pb.strip_sizes[0] = kMaxLeds;
    pb.led_map_count = kMaxLeds < 300 ? kMaxLeds : 300;
    for (uint32_t i = 0; i < pb.led_map_count; i++) {
        pb.led_map[i].key = i;
        strncpy(pb.led_map[i].value, "A01", sizeof(pb.led_map[i].value) - 1);
    }

    BoardSnapshot snap{};
    BoardSnapshot::from_proto(pb, snap);
    check(r, snap.board.led_count == kMaxLeds, "same station: led_count");
    // A01 should have count capped at kMaxLedsPerStation
    bool found = false;
    for (uint16_t i = 0; i < snap.station_leds_count; i++) {
        if (strcmp(snap.station_leds[i].station_id, "A01") == 0) {
            found = true;
            check(r, snap.station_leds[i].count <= kMaxLedsPerStation,
                  "same station: capped at kMaxLedsPerStation");
        }
    }
    check(r, found, "same station: A01 in index");
}

static void test_board_empty() {
    subway_DeviceBoard pb{};
    memset(&pb, 0, sizeof(pb));
    strncpy(pb.board_id, "empty", sizeof(pb.board_id) - 1);
    pb.led_count = 0;
    pb.strip_sizes_count = 0;
    pb.led_map_count = 0;

    BoardSnapshot snap{};
    BoardSnapshot::from_proto(pb, snap);
    check(r, snap.board.led_count == 0, "empty board: led_count == 0");
    check(r, snap.station_leds_count == 0, "empty board: no stations");
}

// ================================================================
// DoubleBuffer Concurrency
// ================================================================

static void test_double_buffer_spsc() {
    DoubleBuffer<uint32_t> buf;
    buf.write_buffer() = 0;
    buf.publish();

    std::atomic<bool> done{false};
    std::atomic<uint32_t> max_seen{0};
    bool monotonic = true;

    std::thread reader([&] {
        uint32_t last = 0;
        while (!done.load(std::memory_order_relaxed)) {
            uint32_t val = buf.read();
            if (val < last) monotonic = false;
            if (val > last) last = val;
            max_seen.store(last, std::memory_order_relaxed);
        }
        // Final read
        uint32_t val = buf.read();
        if (val > max_seen.load()) max_seen.store(val);
    });

    for (uint32_t i = 1; i <= 1000; i++) {
        buf.write_buffer() = i;
        buf.publish();
    }
    done.store(true, std::memory_order_relaxed);
    reader.join();

    check(r, monotonic, "SPSC 1000: monotonic reads");
    check(r, max_seen.load() == 1000, "SPSC 1000: reader saw final value");
}

static void test_double_buffer_rapid_writer() {
    // Use a sentinel pattern: each write is (i * 3 + 7). Reader checks the
    // value matches that pattern (not a torn or partial write).
    DoubleBuffer<uint64_t> buf;
    buf.write_buffer() = 7; // 0 * 3 + 7
    buf.publish();

    std::atomic<bool> done{false};
    bool valid = true;

    std::thread reader([&] {
        while (!done.load(std::memory_order_relaxed)) {
            uint64_t val = buf.read();
            // Check pattern: (val - 7) should be divisible by 3
            if ((val < 7) || ((val - 7) % 3 != 0)) valid = false;
        }
    });

    auto start = std::chrono::steady_clock::now();
    uint64_t writes = 0;
    while (std::chrono::steady_clock::now() - start < std::chrono::milliseconds(50)) {
        writes++;
        buf.write_buffer() = writes * 3 + 7;
        buf.publish();
    }
    done.store(true, std::memory_order_relaxed);
    reader.join();

    check(r, valid, "rapid writer: reader always sees valid data");
    check(r, writes > 0, "rapid writer: wrote > 0 values");
}

// ================================================================
// ScriptChannel Concurrency
// ================================================================

static void test_channel_send_replaces() {
    ScriptChannel chan;
    chan.send(strdup("A"));
    chan.send(strdup("B"));

    char* out = nullptr;
    bool got = chan.receive(out);
    check(r, got, "channel replace: received");
    check(r, out != nullptr && strcmp(out, "B") == 0, "channel replace: got B (A freed)");
    free(out);

    // Nothing left
    char* out2 = nullptr;
    check(r, !chan.receive(out2), "channel replace: empty after receive");
}

static void test_channel_1000_cycles() {
    ScriptChannel chan;
    for (int i = 0; i < 1000; i++) {
        char buf[32];
        snprintf(buf, sizeof(buf), "script_%d", i);
        chan.send(strdup(buf));
        char* out = nullptr;
        bool got = chan.receive(out);
        check(r, got, "channel 1000: received");
        free(out);
    }
    // Channel should be empty
    char* out = nullptr;
    check(r, !chan.receive(out), "channel 1000: empty at end");
}

// ================================================================
// Main
// ================================================================

struct NamedTest {
    const char* name;
    void (*fn)();
};

int main(int argc, char* argv[]) {
    const char* junit_path = parse_junit_arg(argc, argv);

    NamedTest tests[] = {
        // Lua Memory Pressure
        {"oom_recovery_loop",            test_oom_recovery_loop},
        {"string_bomb",                  test_string_bomb},
        {"gc_reclaim",                   test_gc_reclaim},
        {"many_small_allocs",            test_many_small_allocs},
        // Protobuf Robustness
        {"proto_truncated",              test_proto_truncated},
        {"proto_empty",                  test_proto_empty},
        {"proto_garbage",                test_proto_garbage},
        {"proto_max_stations",           test_proto_max_stations},
        {"proto_max_config",             test_proto_max_config},
        // BoardSnapshot Limits
        {"board_max_size",               test_board_max_size},
        {"board_all_same_station",       test_board_all_same_station},
        {"board_empty",                  test_board_empty},
        // DoubleBuffer Concurrency
        {"double_buffer_spsc",           test_double_buffer_spsc},
        {"double_buffer_rapid_writer",   test_double_buffer_rapid_writer},
        // ScriptChannel Concurrency
        {"channel_send_replaces",        test_channel_send_replaces},
        {"channel_1000_cycles",          test_channel_1000_cycles},
    };
    constexpr int n = sizeof(tests) / sizeof(tests[0]);

    printf("Running stress tests...\n");
    const char* names[n];
    int fails[n];
    for (int i = 0; i < n; i++) {
        names[i] = tests[i].name;
        int before = r.fail;
        tests[i].fn();
        fails[i] = r.fail - before;
    }

    printf("\n%s: %d passed, %d failed\n",
           r.fail > 0 ? "FAILED" : "PASSED", r.pass, r.fail);

    if (junit_path)
        write_junit_xml(junit_path, "stress", names, fails, n);

    return r.fail > 0 ? 1 : 0;
}
