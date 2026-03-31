#pragma once
#include <cstdint>
#include <span>
#include "core/types.hpp"
#include "data/transit_snapshot.hpp"
#include "data/board_snapshot.hpp"

// Forward declaration
struct lua_State;

// Per-frame context set in Lua registry, read by all binding functions
struct LuaBindingContext {
    const TransitSnapshot* transit = nullptr;
    const BoardSnapshot* board = nullptr;
    std::span<Rgb> pixels;
    uint32_t led_count = 0;
};

// Register all 20 C functions + status constants in a Lua state
void lua_register_bindings(lua_State* L);

// Set the per-frame binding context (stored in Lua registry)
void lua_set_binding_context(lua_State* L, LuaBindingContext* ctx);
