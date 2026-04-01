#pragma once
#include <cstdio>
#include <cstdlib>

extern "C" {
#include "lua.h"
#include "lauxlib.h"
#include "lualib.h"
}

#include "script/lua_bindings.hpp"

// --- File reading ---

inline char* read_file(const char* path) {
    FILE* f = fopen(path, "rb");
    if (!f) return nullptr;
    fseek(f, 0, SEEK_END);
    long len = ftell(f);
    fseek(f, 0, SEEK_SET);
    auto* buf = static_cast<char*>(malloc(len + 1));
    if (!buf) { fclose(f); return nullptr; }
    fread(buf, 1, len, f);
    buf[len] = '\0';
    fclose(f);
    return buf;
}

// --- Minimal test framework ---

struct TestResults {
    int pass = 0;
    int fail = 0;
};

inline void check(TestResults& r, bool cond, const char* label) {
    if (cond) {
        r.pass++;
    } else {
        r.fail++;
        printf("  FAIL: %s\n", label);
    }
}

// --- Lua VM with standard libs + bindings ---

inline lua_State* create_test_lua() {
    lua_State* L = luaL_newstate();
    if (!L) return nullptr;
    luaL_requiref(L, "_G", luaopen_base, 1);
    luaL_requiref(L, "math", luaopen_math, 1);
    luaL_requiref(L, "string", luaopen_string, 1);
    luaL_requiref(L, "table", luaopen_table, 1);
    luaL_requiref(L, "utf8", luaopen_utf8, 1);
    lua_pop(L, 5);
    lua_register_bindings(L);
    return L;
}

// --- Lua _results reader (for conformance scripts) ---

inline int check_lua_results(lua_State* L, const char* test_name) {
    lua_getglobal(L, "_results");
    if (!lua_istable(L, -1)) {
        printf("  FAIL: %s — _results not found\n", test_name);
        lua_pop(L, 1);
        return 1;
    }

    lua_getfield(L, -1, "pass");
    int pass = static_cast<int>(lua_tointeger(L, -1));
    lua_pop(L, 1);

    lua_getfield(L, -1, "fail");
    int fail = static_cast<int>(lua_tointeger(L, -1));
    lua_pop(L, 1);

    if (fail > 0) {
        printf("  FAIL: %s — %d passed, %d failed:\n", test_name, pass, fail);
        lua_getfield(L, -1, "errors");
        if (lua_istable(L, -1)) {
            int len = static_cast<int>(luaL_len(L, -1));
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
