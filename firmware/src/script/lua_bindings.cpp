#include "script/lua_bindings.hpp"
#include "core/lua_includes.hpp"
#include "log/device_log.hpp"
#include "config/constants.hpp"

#include <cstring>
#include <cmath>
#include <cstdio>
#include <cstdlib>

#include "esp_timer.h"

static const char* TAG = "lua_bindings";

// Registry key for context pointer
static const char* CTX_REGISTRY_KEY = "LuaBindingContext";

static LuaBindingContext* get_ctx(lua_State* L) {
    lua_getfield(L, LUA_REGISTRYINDEX, CTX_REGISTRY_KEY);
    auto* ctx = static_cast<LuaBindingContext*>(lua_touserdata(L, -1));
    lua_pop(L, 1);
    return ctx;
}

void lua_set_binding_context(lua_State* L, LuaBindingContext* ctx) {
    lua_pushlightuserdata(L, ctx);
    lua_setfield(L, LUA_REGISTRYINDEX, CTX_REGISTRY_KEY);
}

// --- Helpers ---

static int find_station(const LuaBindingContext* ctx, const char* stop_id) {
    for (uint16_t i = 0; i < ctx->transit->station_count; i++) {
        if (strcmp(ctx->transit->stations[i].stop_id, stop_id) == 0) return i;
    }
    return -1;
}

static const char* find_config(const LuaBindingContext* ctx, const char* key) {
    for (pb_size_t i = 0; i < ctx->transit->config_count; i++) {
        if (strcmp(ctx->transit->config[i].key, key) == 0) return ctx->transit->config[i].value;
    }
    return nullptr;
}

// --- Lua C Functions ---

static int l_set_led(lua_State* L) {
    auto* ctx = get_ctx(L);
    int index = luaL_checkinteger(L, 1);
    int r = luaL_checkinteger(L, 2);
    int g = luaL_checkinteger(L, 3);
    int b = luaL_checkinteger(L, 4);
    if (index >= 0 && static_cast<uint32_t>(index) < ctx->led_count) {
        ctx->pixels[index].r = (r < 0) ? 0 : (r > 255) ? 255 : static_cast<uint8_t>(r);
        ctx->pixels[index].g = (g < 0) ? 0 : (g > 255) ? 255 : static_cast<uint8_t>(g);
        ctx->pixels[index].b = (b < 0) ? 0 : (b > 255) ? 255 : static_cast<uint8_t>(b);
    }
    return 0;
}

static int l_clear_leds(lua_State* L) {
    auto* ctx = get_ctx(L);
    for (auto& px : ctx->pixels) px = Rgb{};
    return 0;
}

static int l_led_count(lua_State* L) {
    auto* ctx = get_ctx(L);
    lua_pushinteger(L, ctx->led_count);
    return 1;
}

static int l_has_train(lua_State* L) {
    auto* ctx = get_ctx(L);
    int led_index = luaL_checkinteger(L, 1);
    if (led_index < 0 || static_cast<uint32_t>(led_index) >= ctx->led_count) {
        lua_pushboolean(L, 0);
        return 1;
    }
    const char* sid = ctx->board->board.led_map[led_index];
    if (sid[0] == '\0') {
        lua_pushboolean(L, 0);
        return 1;
    }
    int idx = find_station(ctx, sid);
    lua_pushboolean(L, idx >= 0 && ctx->transit->stations[idx].trains_count > 0);
    return 1;
}

static int l_has_status(lua_State* L) {
    auto* ctx = get_ctx(L);
    int led_index = luaL_checkinteger(L, 1);
    const char* status_str = luaL_checkstring(L, 2);

    if (led_index < 0 || static_cast<uint32_t>(led_index) >= ctx->led_count) {
        lua_pushboolean(L, 0);
        return 1;
    }

    const char* sid = ctx->board->board.led_map[led_index];
    if (sid[0] == '\0') {
        lua_pushboolean(L, 0);
        return 1;
    }

    subway_TrainStatus target;
    if (strcmp(status_str, "STOPPED_AT") == 0) target = subway_TrainStatus_STOPPED_AT;
    else if (strcmp(status_str, "INCOMING_AT") == 0) target = subway_TrainStatus_INCOMING_AT;
    else if (strcmp(status_str, "IN_TRANSIT_TO") == 0) target = subway_TrainStatus_IN_TRANSIT_TO;
    else { lua_pushboolean(L, 0); return 1; }

    int idx = find_station(ctx, sid);
    if (idx < 0) { lua_pushboolean(L, 0); return 1; }

    for (uint8_t t = 0; t < ctx->transit->stations[idx].trains_count; t++) {
        if (ctx->transit->stations[idx].trains[t].status == target) {
            lua_pushboolean(L, 1);
            return 1;
        }
    }
    lua_pushboolean(L, 0);
    return 1;
}

static int l_get_route(lua_State* L) {
    auto* ctx = get_ctx(L);
    int led_index = luaL_checkinteger(L, 1);
    if (led_index < 0 || static_cast<uint32_t>(led_index) >= ctx->led_count) {
        lua_pushnil(L);
        return 1;
    }
    const char* sid = ctx->board->board.led_map[led_index];
    if (sid[0] == '\0') { lua_pushnil(L); return 1; }

    int idx = find_station(ctx, sid);
    if (idx < 0 || ctx->transit->stations[idx].trains_count == 0) {
        lua_pushnil(L);
        return 1;
    }
    lua_pushstring(L, ctx->transit->stations[idx].trains[0].route);
    return 1;
}

static int l_get_routes(lua_State* L) {
    auto* ctx = get_ctx(L);
    int led_index = luaL_checkinteger(L, 1);
    lua_newtable(L);

    if (led_index < 0 || static_cast<uint32_t>(led_index) >= ctx->led_count) return 1;
    const char* sid = ctx->board->board.led_map[led_index];
    if (sid[0] == '\0') return 1;

    int idx = find_station(ctx, sid);
    if (idx < 0) return 1;

    for (uint8_t t = 0; t < ctx->transit->stations[idx].trains_count; t++) {
        lua_pushstring(L, ctx->transit->stations[idx].trains[t].route);
        lua_rawseti(L, -2, t + 1);
    }
    return 1;
}

static int l_get_station(lua_State* L) {
    auto* ctx = get_ctx(L);
    int led_index = luaL_checkinteger(L, 1);
    if (led_index < 0 || static_cast<uint32_t>(led_index) >= ctx->led_count) {
        lua_pushnil(L);
        return 1;
    }
    const char* sid = ctx->board->board.led_map[led_index];
    if (sid[0] == '\0') {
        lua_pushnil(L);
        return 1;
    }
    lua_pushstring(L, sid);
    return 1;
}

static int l_get_leds_for_station(lua_State* L) {
    auto* ctx = get_ctx(L);
    const char* station_id = luaL_checkstring(L, 1);
    lua_newtable(L);

    for (uint16_t i = 0; i < ctx->board->station_leds_count; i++) {
        if (strcmp(ctx->board->station_leds[i].station_id, station_id) == 0) {
            for (uint8_t j = 0; j < ctx->board->station_leds[i].count; j++) {
                lua_pushinteger(L, ctx->board->station_leds[i].led_indices[j]);
                lua_rawseti(L, -2, j + 1);
            }
            break;
        }
    }
    return 1;
}

static int l_get_string_config(lua_State* L) {
    auto* ctx = get_ctx(L);
    const char* key = luaL_checkstring(L, 1);
    const char* val = find_config(ctx, key);
    if (val) lua_pushstring(L, val);
    else lua_pushnil(L);
    return 1;
}

static int l_get_int_config(lua_State* L) {
    auto* ctx = get_ctx(L);
    const char* key = luaL_checkstring(L, 1);
    const char* val = find_config(ctx, key);
    if (val) lua_pushinteger(L, atoi(val));
    else lua_pushnil(L);
    return 1;
}

static int l_get_rgb_config(lua_State* L) {
    auto* ctx = get_ctx(L);
    const char* key = luaL_checkstring(L, 1);
    const char* hex = find_config(ctx, key);
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

static int l_get_time(lua_State* L) {
    lua_pushnumber(L, esp_timer_get_time() / 1e6);
    return 1;
}

static int l_hsv_to_rgb(lua_State* L) {
    double h = luaL_checknumber(L, 1);
    double s = luaL_checknumber(L, 2);
    double v = luaL_checknumber(L, 3);

    int i = static_cast<int>(floor(h * 6.0));
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

    lua_pushinteger(L, static_cast<int>(r * 255.0 + 0.5));
    lua_pushinteger(L, static_cast<int>(g * 255.0 + 0.5));
    lua_pushinteger(L, static_cast<int>(b * 255.0 + 0.5));
    return 3;
}

static int l_hex_to_rgb(lua_State* L) {
    const char* hex = luaL_checkstring(L, 1);
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

static int l_get_strip_info(lua_State* L) {
    auto* ctx = get_ctx(L);
    lua_newtable(L);
    for (uint8_t i = 0; i < ctx->board->board.strip_count; i++) {
        lua_pushinteger(L, ctx->board->board.strip_sizes[i]);
        lua_rawseti(L, -2, i + 1);
    }
    return 1;
}

static int l_led_to_strip(lua_State* L) {
    auto* ctx = get_ctx(L);
    int index = luaL_checkinteger(L, 1);
    uint32_t offset = 0;
    for (uint8_t s = 0; s < ctx->board->board.strip_count; s++) {
        if (static_cast<uint32_t>(index) < offset + ctx->board->board.strip_sizes[s]) {
            lua_pushinteger(L, s + 1);          // 1-based strip number
            lua_pushinteger(L, index - offset);  // pixel within strip
            return 2;
        }
        offset += ctx->board->board.strip_sizes[s];
    }
    lua_pushnil(L);
    lua_pushnil(L);
    return 2;
}

static int l_log(lua_State* L) {
    const char* msg = luaL_checkstring(L, 1);
    DLOG_I(TAG, "Lua: %s", msg);
    return 0;
}

// --- Registration ---

void lua_register_bindings(lua_State* L) {
    // LED Control
    lua_register(L, "set_led", l_set_led);
    lua_register(L, "clear_leds", l_clear_leds);
    lua_register(L, "led_count", l_led_count);

    // MTA State Queries
    lua_register(L, "has_train", l_has_train);
    lua_register(L, "has_status", l_has_status);
    lua_register(L, "get_route", l_get_route);
    lua_register(L, "get_routes", l_get_routes);
    lua_register(L, "get_station", l_get_station);
    lua_register(L, "get_leds_for_station", l_get_leds_for_station);

    // Config Queries
    lua_register(L, "get_string_config", l_get_string_config);
    lua_register(L, "get_int_config", l_get_int_config);
    lua_register(L, "get_rgb_config", l_get_rgb_config);

    // Timing
    lua_register(L, "get_time", l_get_time);

    // Color Utilities
    lua_register(L, "hsv_to_rgb", l_hsv_to_rgb);
    lua_register(L, "hex_to_rgb", l_hex_to_rgb);

    // Board Info
    lua_register(L, "get_strip_info", l_get_strip_info);
    lua_register(L, "led_to_strip", l_led_to_strip);

    // Logging
    lua_register(L, "log", l_log);

    // Status Constants
    lua_pushstring(L, "STOPPED_AT");
    lua_setglobal(L, "STOPPED_AT");
    lua_pushstring(L, "INCOMING_AT");
    lua_setglobal(L, "INCOMING_AT");
    lua_pushstring(L, "IN_TRANSIT_TO");
    lua_setglobal(L, "IN_TRANSIT_TO");
}
