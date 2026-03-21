#include "lua_runtime.h"
#include "led_driver.h"
#include "config.h"

#include <string.h>
#include <math.h>
#include <stdio.h>

#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "esp_log.h"
#include "esp_timer.h"

#include "lua.h"
#include "lauxlib.h"
#include "lualib.h"

static const char *TAG = "lua_runtime";

/* Maximum Lua memory (80KB) */
#define LUA_MAX_MEM (80 * 1024)

/* Maximum instructions per render call */
#define LUA_MAX_INSTRUCTIONS 100000

/* Consecutive failure limit before fallback */
#define MAX_CONSECUTIVE_FAILURES 5

/* Render context pointer (set at task start) */
static render_context_t *s_ctx = NULL;

/* LED pixel buffer */
static uint8_t s_pixels[MAX_LEDS * 3];
static uint32_t s_led_count = MAX_LEDS;

/* Snapshot of render context for current frame */
static station_t s_snap_stations[MAX_STATIONS];
static uint16_t s_snap_station_count;
static config_entry_t s_snap_config[MAX_CONFIG_ENTRIES];
static uint8_t s_snap_config_count;
static board_info_t s_snap_board;
static station_leds_entry_t s_snap_station_leds[MAX_STATIONS];
static uint16_t s_snap_station_leds_count;

/* Custom allocator to cap memory */
static size_t s_lua_mem_used = 0;

static void *lua_custom_alloc(void *ud, void *ptr, size_t osize, size_t nsize)
{
    (void)ud;
    if (nsize == 0) {
        s_lua_mem_used -= osize;
        free(ptr);
        return NULL;
    }
    if (s_lua_mem_used - osize + nsize > LUA_MAX_MEM) {
        return NULL; /* OOM — Lua will raise error */
    }
    void *new_ptr = realloc(ptr, nsize);
    if (new_ptr) {
        s_lua_mem_used = s_lua_mem_used - osize + nsize;
    }
    return new_ptr;
}

/* Instruction hook to prevent infinite loops */
static void lua_instruction_hook(lua_State *L, lua_Debug *ar)
{
    (void)ar;
    luaL_error(L, "script exceeded instruction limit");
}

/* ─── Helper: find station data by stop_id ─── */
static int find_station(const char *stop_id)
{
    for (uint16_t i = 0; i < s_snap_station_count; i++) {
        if (strcmp(s_snap_stations[i].stop_id, stop_id) == 0) {
            return i;
        }
    }
    return -1;
}

/* ─── Helper: find config value by key ─── */
static const char *find_config(const char *key)
{
    for (uint8_t i = 0; i < s_snap_config_count; i++) {
        if (strcmp(s_snap_config[i].key, key) == 0) {
            return s_snap_config[i].value;
        }
    }
    return NULL;
}

/* ─── Lua C Functions ─── */

static int l_set_led(lua_State *L)
{
    int index = luaL_checkinteger(L, 1);
    int r = luaL_checkinteger(L, 2);
    int g = luaL_checkinteger(L, 3);
    int b = luaL_checkinteger(L, 4);
    if (index >= 0 && (uint32_t)index < s_led_count) {
        s_pixels[index * 3 + 0] = (r < 0) ? 0 : (r > 255) ? 255 : r;
        s_pixels[index * 3 + 1] = (g < 0) ? 0 : (g > 255) ? 255 : g;
        s_pixels[index * 3 + 2] = (b < 0) ? 0 : (b > 255) ? 255 : b;
    }
    return 0;
}

static int l_clear_leds(lua_State *L)
{
    (void)L;
    memset(s_pixels, 0, sizeof(s_pixels));
    return 0;
}

static int l_led_count(lua_State *L)
{
    lua_pushinteger(L, s_led_count);
    return 1;
}

static int l_has_train(lua_State *L)
{
    int led_index = luaL_checkinteger(L, 1);
    if (led_index < 0 || (uint32_t)led_index >= s_led_count) {
        lua_pushboolean(L, 0);
        return 1;
    }
    const char *sid = s_snap_board.led_map[led_index];
    if (sid[0] == '\0') {
        lua_pushboolean(L, 0);
        return 1;
    }
    int idx = find_station(sid);
    lua_pushboolean(L, idx >= 0 && s_snap_stations[idx].train_count > 0);
    return 1;
}

static int l_has_status(lua_State *L)
{
    int led_index = luaL_checkinteger(L, 1);
    const char *status_str = luaL_checkstring(L, 2);

    if (led_index < 0 || (uint32_t)led_index >= s_led_count) {
        lua_pushboolean(L, 0);
        return 1;
    }

    const char *sid = s_snap_board.led_map[led_index];
    if (sid[0] == '\0') {
        lua_pushboolean(L, 0);
        return 1;
    }

    train_status_t target;
    if (strcmp(status_str, "STOPPED_AT") == 0) target = TRAIN_STATUS_STOPPED_AT;
    else if (strcmp(status_str, "INCOMING_AT") == 0) target = TRAIN_STATUS_INCOMING_AT;
    else if (strcmp(status_str, "IN_TRANSIT_TO") == 0) target = TRAIN_STATUS_IN_TRANSIT_TO;
    else { lua_pushboolean(L, 0); return 1; }

    int idx = find_station(sid);
    if (idx < 0) { lua_pushboolean(L, 0); return 1; }

    for (uint8_t t = 0; t < s_snap_stations[idx].train_count; t++) {
        if (s_snap_stations[idx].trains[t].status == target) {
            lua_pushboolean(L, 1);
            return 1;
        }
    }
    lua_pushboolean(L, 0);
    return 1;
}

static int l_get_route(lua_State *L)
{
    int led_index = luaL_checkinteger(L, 1);
    if (led_index < 0 || (uint32_t)led_index >= s_led_count) {
        lua_pushnil(L);
        return 1;
    }
    const char *sid = s_snap_board.led_map[led_index];
    if (sid[0] == '\0') { lua_pushnil(L); return 1; }

    int idx = find_station(sid);
    if (idx < 0 || s_snap_stations[idx].train_count == 0) {
        lua_pushnil(L);
        return 1;
    }
    lua_pushstring(L, s_snap_stations[idx].trains[0].route);
    return 1;
}

static int l_get_routes(lua_State *L)
{
    int led_index = luaL_checkinteger(L, 1);
    lua_newtable(L);

    if (led_index < 0 || (uint32_t)led_index >= s_led_count) return 1;
    const char *sid = s_snap_board.led_map[led_index];
    if (sid[0] == '\0') return 1;

    int idx = find_station(sid);
    if (idx < 0) return 1;

    for (uint8_t t = 0; t < s_snap_stations[idx].train_count; t++) {
        lua_pushstring(L, s_snap_stations[idx].trains[t].route);
        lua_rawseti(L, -2, t + 1);
    }
    return 1;
}

static int l_get_station(lua_State *L)
{
    int led_index = luaL_checkinteger(L, 1);
    if (led_index < 0 || (uint32_t)led_index >= s_led_count) {
        lua_pushnil(L);
        return 1;
    }
    const char *sid = s_snap_board.led_map[led_index];
    if (sid[0] == '\0') {
        lua_pushnil(L);
        return 1;
    }
    lua_pushstring(L, sid);
    return 1;
}

static int l_get_leds_for_station(lua_State *L)
{
    const char *station_id = luaL_checkstring(L, 1);
    lua_newtable(L);

    for (uint16_t i = 0; i < s_snap_station_leds_count; i++) {
        if (strcmp(s_snap_station_leds[i].station_id, station_id) == 0) {
            for (uint8_t j = 0; j < s_snap_station_leds[i].count; j++) {
                lua_pushinteger(L, s_snap_station_leds[i].led_indices[j]);
                lua_rawseti(L, -2, j + 1);
            }
            break;
        }
    }
    return 1;
}

static int l_get_string_config(lua_State *L)
{
    const char *key = luaL_checkstring(L, 1);
    const char *val = find_config(key);
    if (val) lua_pushstring(L, val);
    else lua_pushnil(L);
    return 1;
}

static int l_get_int_config(lua_State *L)
{
    const char *key = luaL_checkstring(L, 1);
    const char *val = find_config(key);
    if (val) lua_pushinteger(L, atoi(val));
    else lua_pushnil(L);
    return 1;
}

static int l_get_rgb_config(lua_State *L)
{
    const char *key = luaL_checkstring(L, 1);
    const char *hex = find_config(key);
    if (!hex || strlen(hex) < 7 || hex[0] != '#') {
        lua_pushnil(L);
        return 1;
    }
    unsigned int r, g, b;
    sscanf(hex + 1, "%02x%02x%02x", &r, &g, &b);
    lua_pushinteger(L, r);
    lua_pushinteger(L, g);
    lua_pushinteger(L, b);
    return 3;
}

static int l_get_time(lua_State *L)
{
    lua_pushnumber(L, esp_timer_get_time() / 1e6);
    return 1;
}

static int l_hsv_to_rgb(lua_State *L)
{
    double h = luaL_checknumber(L, 1);
    double s = luaL_checknumber(L, 2);
    double v = luaL_checknumber(L, 3);

    int i = (int)floor(h * 6.0);
    double f = h * 6.0 - i;
    double p = v * (1.0 - s);
    double q = v * (1.0 - f * s);
    double t = v * (1.0 - (1.0 - f) * s);
    double r, g, b;

    switch (i % 6) {
        case 0: r = v; g = t; b = p; break;
        case 1: r = q; g = v; b = p; break;
        case 2: r = p; g = v; b = t; break;
        case 3: r = p; g = q; b = v; break;
        case 4: r = t; g = p; b = v; break;
        default: r = v; g = p; b = q; break;
    }

    lua_pushinteger(L, (int)(r * 255.0 + 0.5));
    lua_pushinteger(L, (int)(g * 255.0 + 0.5));
    lua_pushinteger(L, (int)(b * 255.0 + 0.5));
    return 3;
}

static int l_hex_to_rgb(lua_State *L)
{
    const char *hex = luaL_checkstring(L, 1);
    if (strlen(hex) < 7 || hex[0] != '#') {
        lua_pushnil(L);
        return 1;
    }
    unsigned int r, g, b;
    sscanf(hex + 1, "%02x%02x%02x", &r, &g, &b);
    lua_pushinteger(L, r);
    lua_pushinteger(L, g);
    lua_pushinteger(L, b);
    return 3;
}

static int l_get_strip_info(lua_State *L)
{
    lua_newtable(L);
    for (uint8_t i = 0; i < s_snap_board.strip_count; i++) {
        lua_pushinteger(L, s_snap_board.strip_sizes[i]);
        lua_rawseti(L, -2, i + 1);
    }
    return 1;
}

static int l_led_to_strip(lua_State *L)
{
    int index = luaL_checkinteger(L, 1);
    uint32_t offset = 0;
    for (uint8_t s = 0; s < s_snap_board.strip_count; s++) {
        if ((uint32_t)index < offset + s_snap_board.strip_sizes[s]) {
            lua_pushinteger(L, s + 1);       /* 1-based strip number */
            lua_pushinteger(L, index - offset); /* pixel within strip */
            return 2;
        }
        offset += s_snap_board.strip_sizes[s];
    }
    lua_pushnil(L);
    lua_pushnil(L);
    return 2;
}

static int l_log(lua_State *L)
{
    const char *msg = luaL_checkstring(L, 1);
    ESP_LOGI(TAG, "Lua: %s", msg);
    return 0;
}

/* Register all C functions in the Lua state */
static void register_lua_functions(lua_State *L)
{
    /* LED Control */
    lua_register(L, "set_led", l_set_led);
    lua_register(L, "clear_leds", l_clear_leds);
    lua_register(L, "led_count", l_led_count);

    /* MTA State Queries */
    lua_register(L, "has_train", l_has_train);
    lua_register(L, "has_status", l_has_status);
    lua_register(L, "get_route", l_get_route);
    lua_register(L, "get_routes", l_get_routes);
    lua_register(L, "get_station", l_get_station);
    lua_register(L, "get_leds_for_station", l_get_leds_for_station);

    /* Config Queries */
    lua_register(L, "get_string_config", l_get_string_config);
    lua_register(L, "get_int_config", l_get_int_config);
    lua_register(L, "get_rgb_config", l_get_rgb_config);

    /* Timing */
    lua_register(L, "get_time", l_get_time);

    /* Color Utilities */
    lua_register(L, "hsv_to_rgb", l_hsv_to_rgb);
    lua_register(L, "hex_to_rgb", l_hex_to_rgb);

    /* Board Info */
    lua_register(L, "get_strip_info", l_get_strip_info);
    lua_register(L, "led_to_strip", l_led_to_strip);

    /* Logging */
    lua_register(L, "log", l_log);

    /* Status Constants */
    lua_pushstring(L, "STOPPED_AT");
    lua_setglobal(L, "STOPPED_AT");
    lua_pushstring(L, "INCOMING_AT");
    lua_setglobal(L, "INCOMING_AT");
    lua_pushstring(L, "IN_TRANSIT_TO");
    lua_setglobal(L, "IN_TRANSIT_TO");
}

/* Fallback script for error recovery */
static const char *FALLBACK_SCRIPT =
    "function render()\n"
    "    local t = get_time()\n"
    "    local i = math.floor(t) % led_count()\n"
    "    set_led(i, 0, 0, 30)\n"
    "end\n";

static void render_task(void *arg)
{
    render_context_t *ctx = (render_context_t *)arg;
    s_ctx = ctx;

    ESP_LOGI(TAG, "Render task started");

    /* Create Lua state with custom allocator */
    lua_State *L = lua_newstate(lua_custom_alloc, NULL);
    if (!L) {
        ESP_LOGE(TAG, "Failed to create Lua state!");
        vTaskDelete(NULL);
        return;
    }

    /* Open safe standard libraries */
    luaL_requiref(L, "_G", luaopen_base, 1);
    luaL_requiref(L, "math", luaopen_math, 1);
    luaL_requiref(L, "string", luaopen_string, 1);
    luaL_requiref(L, "table", luaopen_table, 1);
    luaL_requiref(L, "utf8", luaopen_utf8, 1);
    lua_pop(L, 5);

    /* Set instruction hook for safety */
    lua_sethook(L, lua_instruction_hook, LUA_MASKCOUNT, LUA_MAX_INSTRUCTIONS);

    /* Register our C API */
    register_lua_functions(L);

    /* Load fallback script initially */
    bool script_loaded = false;
    int consecutive_failures = 0;

    if (luaL_dostring(L, FALLBACK_SCRIPT) != LUA_OK) {
        ESP_LOGE(TAG, "Failed to load fallback: %s", lua_tostring(L, -1));
        lua_pop(L, 1);
    } else {
        script_loaded = true;
    }

    /* Main render loop */
    while (1) {
        /* Check for script changes */
        xSemaphoreTake(ctx->mutex, portMAX_DELAY);
        bool need_reload = ctx->script_changed;
        if (need_reload) {
            ctx->script_changed = false;
        }
        xSemaphoreGive(ctx->mutex);

        if (need_reload) {
            /* TODO: Read new script from SPIFFS and reload Lua VM */
            /* For now, this is a placeholder. When SPIFFS caching is
               implemented in state_client.c, read the script from there. */
            ESP_LOGI(TAG, "Script change detected (reload not yet implemented)");
            consecutive_failures = 0;
        }

        /* Snapshot render context */
        xSemaphoreTake(ctx->mutex, portMAX_DELAY);
        memcpy(s_snap_stations, ctx->stations, sizeof(station_t) * ctx->station_count);
        s_snap_station_count = ctx->station_count;
        memcpy(s_snap_config, ctx->config, sizeof(config_entry_t) * ctx->config_count);
        s_snap_config_count = ctx->config_count;
        memcpy(&s_snap_board, &ctx->board, sizeof(board_info_t));
        memcpy(s_snap_station_leds, ctx->station_leds, sizeof(station_leds_entry_t) * ctx->station_leds_count);
        s_snap_station_leds_count = ctx->station_leds_count;
        s_led_count = ctx->board.led_count > 0 ? ctx->board.led_count : MAX_LEDS;
        xSemaphoreGive(ctx->mutex);

        /* Clear pixel buffer */
        memset(s_pixels, 0, s_led_count * 3);

        /* Call Lua render() */
        if (script_loaded) {
            lua_getglobal(L, "render");
            if (lua_pcall(L, 0, 0, 0) != LUA_OK) {
                const char *err = lua_tostring(L, -1);
                ESP_LOGW(TAG, "Lua render error: %s", err ? err : "unknown");
                lua_pop(L, 1);
                consecutive_failures++;

                if (consecutive_failures >= MAX_CONSECUTIVE_FAILURES) {
                    ESP_LOGW(TAG, "Too many failures, loading fallback script");
                    if (luaL_dostring(L, FALLBACK_SCRIPT) == LUA_OK) {
                        consecutive_failures = 0;
                    }
                }
            } else {
                consecutive_failures = 0;
            }
        }

        /* Push pixels to LED driver */
        for (uint32_t i = 0; i < s_led_count && i < MAX_LEDS; i++) {
            uint32_t strip, pixel;
            /* Convert flat index to strip/pixel */
            uint32_t offset = 0;
            bool found = false;
            for (uint8_t si = 0; si < s_snap_board.strip_count; si++) {
                if (i < offset + s_snap_board.strip_sizes[si]) {
                    strip = si;
                    pixel = i - offset;
                    found = true;
                    break;
                }
                offset += s_snap_board.strip_sizes[si];
            }
            if (found) {
                led_driver_set_pixel(strip, pixel,
                                     s_pixels[i * 3], s_pixels[i * 3 + 1], s_pixels[i * 3 + 2]);
            }
        }
        led_driver_refresh();

        /* GC */
        lua_gc(L, LUA_GCCOLLECT, 0);

        /* Sleep for render interval (~30ms = ~33fps) */
        vTaskDelay(pdMS_TO_TICKS(30));
    }

    lua_close(L);
    vTaskDelete(NULL);
}

void lua_runtime_start(render_context_t *ctx)
{
    xTaskCreate(render_task, "render_task", 16384, ctx, 5, NULL);
}
