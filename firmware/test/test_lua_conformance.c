/* Host conformance test for the Lua runtime API.
 * Compiles lua_runtime.c on the host and runs shared Lua test scripts
 * to verify behavior matches the TypeScript (wasmoon) implementation. */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <dirent.h>

/* Include lua_runtime.c directly to access static functions & globals */
#include "lua_runtime.c"

static render_context_t test_ctx;

static void setup_fixtures(void)
{
    render_context_init(&test_ctx);

    /* Board: 5 LEDs, 2 strips (3 + 2) */
    test_ctx.board.led_count = 5;
    test_ctx.board.strip_count = 2;
    test_ctx.board.strip_sizes[0] = 3;
    test_ctx.board.strip_sizes[1] = 2;

    /* LED map: 0,1 -> "A01", 2 -> "B02", 3 -> "" (unmapped), 4 -> "C03" */
    strncpy(test_ctx.board.led_map[0], "A01", PB_STOP_ID_LEN);
    strncpy(test_ctx.board.led_map[1], "A01", PB_STOP_ID_LEN);
    strncpy(test_ctx.board.led_map[2], "B02", PB_STOP_ID_LEN);
    test_ctx.board.led_map[3][0] = '\0';
    strncpy(test_ctx.board.led_map[4], "C03", PB_STOP_ID_LEN);

    render_context_build_station_leds(&test_ctx);

    /* Stations: A01 STOPPED_AT route "1", B02 IN_TRANSIT_TO route "A" */
    test_ctx.station_count = 2;
    strncpy(test_ctx.stations[0].stop_id, "A01", sizeof(test_ctx.stations[0].stop_id));
    test_ctx.stations[0].trains_count = 1;
    strncpy(test_ctx.stations[0].trains[0].route, "1", sizeof(test_ctx.stations[0].trains[0].route));
    test_ctx.stations[0].trains[0].status = subway_TrainStatus_STOPPED_AT;

    strncpy(test_ctx.stations[1].stop_id, "B02", sizeof(test_ctx.stations[1].stop_id));
    test_ctx.stations[1].trains_count = 1;
    strncpy(test_ctx.stations[1].trains[0].route, "A", sizeof(test_ctx.stations[1].trains[0].route));
    test_ctx.stations[1].trains[0].status = subway_TrainStatus_IN_TRANSIT_TO;

    /* Config: brightness=200, color=#FF8800, empty="", name=test */
    test_ctx.config_count = 4;
    strncpy(test_ctx.config[0].key, "brightness", sizeof(test_ctx.config[0].key));
    strncpy(test_ctx.config[0].value, "200", sizeof(test_ctx.config[0].value));
    strncpy(test_ctx.config[1].key, "color", sizeof(test_ctx.config[1].key));
    strncpy(test_ctx.config[1].value, "#FF8800", sizeof(test_ctx.config[1].value));
    strncpy(test_ctx.config[2].key, "empty", sizeof(test_ctx.config[2].key));
    strncpy(test_ctx.config[2].value, "", sizeof(test_ctx.config[2].value));
    strncpy(test_ctx.config[3].key, "name", sizeof(test_ctx.config[3].key));
    strncpy(test_ctx.config[3].value, "test", sizeof(test_ctx.config[3].value));

    /* Point context and snapshot */
    s_ctx = &test_ctx;
    memcpy(s_snap_stations, test_ctx.stations, sizeof(subway_Station) * test_ctx.station_count);
    s_snap_station_count = test_ctx.station_count;
    memcpy(s_snap_config, test_ctx.config, sizeof(subway_DeviceState_ConfigEntry) * test_ctx.config_count);
    s_snap_config_count = test_ctx.config_count;
    s_led_count = test_ctx.board.led_count;
    s_snap_strip_count = test_ctx.board.strip_count;
    for (uint8_t i = 0; i < s_snap_strip_count && i < MAX_STRIPS; i++)
        s_snap_strip_sizes[i] = test_ctx.board.strip_sizes[i];
}

static int check_results(lua_State *L, const char *test_name)
{
    lua_getglobal(L, "_results");
    if (!lua_istable(L, -1)) {
        printf("  FAIL: %s — _results not found\n", test_name);
        lua_pop(L, 1);
        return 1;
    }

    lua_getfield(L, -1, "pass");
    int pass = (int)lua_tointeger(L, -1);
    lua_pop(L, 1);

    lua_getfield(L, -1, "fail");
    int fail = (int)lua_tointeger(L, -1);
    lua_pop(L, 1);

    if (fail > 0) {
        printf("  FAIL: %s — %d passed, %d failed:\n", test_name, pass, fail);
        lua_getfield(L, -1, "errors");
        if (lua_istable(L, -1)) {
            int len = (int)luaL_len(L, -1);
            for (int i = 1; i <= len; i++) {
                lua_rawgeti(L, -1, i);
                printf("    - %s\n", lua_tostring(L, -1));
                lua_pop(L, 1);
            }
        }
        lua_pop(L, 1);
    } else {
        printf("  OK: %s — %d passed\n", test_name, pass);
    }

    lua_pop(L, 1);
    return fail;
}

static void reset_results(lua_State *L)
{
    luaL_dostring(L, "_results = { pass = 0, fail = 0, errors = {} }");
}

int main(int argc, char *argv[])
{
    if (argc < 2) {
        fprintf(stderr, "Usage: %s <conformance-dir>\n", argv[0]);
        return 1;
    }

    const char *dir = argv[1];
    char path[512];

    setup_fixtures();

    /* Use standard allocator (no 40KB ESP32 cap) for host testing */
    lua_State *L = luaL_newstate();
    if (!L) { fprintf(stderr, "Failed to create Lua state\n"); return 1; }
    luaL_requiref(L, "_G", luaopen_base, 1);
    luaL_requiref(L, "math", luaopen_math, 1);
    luaL_requiref(L, "string", luaopen_string, 1);
    luaL_requiref(L, "table", luaopen_table, 1);
    luaL_requiref(L, "utf8", luaopen_utf8, 1);
    lua_pop(L, 5);
    register_lua_functions(L);

    /* Load helpers */
    snprintf(path, sizeof(path), "%s/helpers.lua", dir);
    if (luaL_dofile(L, path) != LUA_OK) {
        fprintf(stderr, "Failed to load helpers.lua: %s\n", lua_tostring(L, -1));
        lua_close(L);
        return 1;
    }

    /* Collect and sort test_*.lua files */
    DIR *d = opendir(dir);
    if (!d) { fprintf(stderr, "Cannot open directory: %s\n", dir); lua_close(L); return 1; }

    char test_files[32][256];
    int test_count = 0;
    struct dirent *entry;
    while ((entry = readdir(d)) != NULL && test_count < 32) {
        if (strncmp(entry->d_name, "test_", 5) == 0 &&
            strcmp(entry->d_name + strlen(entry->d_name) - 4, ".lua") == 0) {
            strncpy(test_files[test_count], entry->d_name, 255);
            test_count++;
        }
    }
    closedir(d);

    for (int i = 0; i < test_count - 1; i++)
        for (int j = i + 1; j < test_count; j++)
            if (strcmp(test_files[i], test_files[j]) > 0) {
                char tmp[256];
                strncpy(tmp, test_files[i], 255);
                strncpy(test_files[i], test_files[j], 255);
                strncpy(test_files[j], tmp, 255);
            }

    printf("Running %d conformance test files...\n", test_count);
    int total_failures = 0;
    for (int i = 0; i < test_count; i++) {
        reset_results(L);
        snprintf(path, sizeof(path), "%s/%s", dir, test_files[i]);
        if (luaL_dofile(L, path) != LUA_OK) {
            printf("  ERROR: %s — %s\n", test_files[i], lua_tostring(L, -1));
            lua_pop(L, 1);
            total_failures++;
        } else {
            total_failures += check_results(L, test_files[i]);
        }
    }

    lua_close(L);
    printf("\n%s: %d file(s), %d failure(s)\n",
           total_failures > 0 ? "FAILED" : "PASSED", test_count, total_failures);
    return total_failures > 0 ? 1 : 0;
}
