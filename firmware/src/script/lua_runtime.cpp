#include "script/lua_runtime.hpp"

#include "config/constants.hpp"
#include "core/lua_includes.hpp"
#include "esp_task_wdt.h"
#include "esp_timer.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "log/device_log.hpp"
#include "script/lua_bindings.hpp"

#include <climits>
#include <cstdlib>
#include <cstring>

static const char* TAG = "lua_runtime";

// --- Task argument struct ---

struct RenderTaskArgs {
    DoubleBuffer<TransitSnapshot>* transit_buf;
    BoardStore* board_store;
    ScriptChannel* script_chan;
    DiagPad* diag;
    std::atomic<bool>* http_active;
    LedDriver* led;
};

// --- Custom allocator (40KB cap) ---

static int32_t s_lua_mem_used = 0;

static void* lua_custom_alloc(void* ud, void* ptr, size_t osize, size_t nsize) {
    (void)ud;
    if (nsize == 0) {
        s_lua_mem_used -= static_cast<int32_t>(osize);
        if (s_lua_mem_used < 0)
            s_lua_mem_used = 0;
        free(ptr);
        return nullptr;
    }
    int32_t delta = static_cast<int32_t>(nsize) - static_cast<int32_t>(osize);
    if (s_lua_mem_used + delta > static_cast<int32_t>(kLuaMaxMem)) {
        return nullptr; // OOM -- Lua will raise error
    }
    void* new_ptr = realloc(ptr, nsize);
    if (new_ptr) {
        s_lua_mem_used += delta;
        if (s_lua_mem_used < 0)
            s_lua_mem_used = 0;
    }
    return new_ptr;
}

// --- Instruction hook to prevent infinite loops ---

static void lua_instruction_hook(lua_State* L, lua_Debug* ar) {
    (void)ar;
    luaL_error(L, "script exceeded instruction limit");
}

// --- Fallback script: bright chase pattern ---

static const char* FALLBACK_SCRIPT = "function render()\n"
                                     "    local t = get_time()\n"
                                     "    local n = led_count()\n"
                                     "    local pos = math.floor(t * 5) % n\n"
                                     "    for i = 0, 9 do\n"
                                     "        set_led((pos + i) % n, 0, 0, 255)\n"
                                     "    end\n"
                                     "end\n";

// --- Create a fresh Lua VM with libraries and C API registered ---

static lua_State* create_lua_state() {
    s_lua_mem_used = 0;
    lua_State* L = lua_newstate(lua_custom_alloc, nullptr);
    if (!L)
        return nullptr;

    luaL_requiref(L, "_G", luaopen_base, 1);
    luaL_requiref(L, "math", luaopen_math, 1);
    luaL_requiref(L, "string", luaopen_string, 1);
    luaL_requiref(L, "table", luaopen_table, 1);
    luaL_requiref(L, "utf8", luaopen_utf8, 1);
    lua_pop(L, 5);

    if constexpr (kLuaMaxInstructions > 0) {
        lua_sethook(L, lua_instruction_hook, LUA_MASKCOUNT, kLuaMaxInstructions);
    }
    lua_register_bindings(L);
    return L;
}

// --- Render task ---

static void render_task(void* arg) {
    auto* args = static_cast<RenderTaskArgs*>(arg);

    DLOG_I(TAG, "Render task started");

    // Register with task watchdog (30s timeout — generous for ~33fps loop)
    esp_task_wdt_add(nullptr);

    lua_State* L = create_lua_state();
    if (!L) {
        DLOG_E(TAG, "Failed to create Lua state!");
        vTaskDelete(nullptr);
        return;
    }

    bool script_loaded = false;
    int consecutive_failures = 0;

    if (luaL_dostring(L, FALLBACK_SCRIPT) != LUA_OK) {
        DLOG_E(TAG, "Failed to load fallback: %s", lua_tostring(L, -1));
        lua_pop(L, 1);
    } else {
        script_loaded = true;
    }

    // Main render loop
    while (true) {
        // Check for new script via ScriptChannel
        char* new_source = nullptr;
        if (args->script_chan->receive(new_source)) {
            if (new_source && new_source[0]) {
                // Destroy old VM and create fresh one -- prevents memory
                // fragmentation from accumulating across script reloads
                lua_close(L);
                L = create_lua_state();

                if (L && luaL_dostring(L, new_source) == LUA_OK) {
                    script_loaded = true;
                    consecutive_failures = 0;
                    args->diag->last_reload.store(1, std::memory_order_relaxed);
                    DLOG_I(
                        TAG, "Loaded new script (%d bytes)", static_cast<int>(strlen(new_source)));
                } else {
                    args->diag->last_reload.store(-1, std::memory_order_relaxed);
                    if (L) {
                        DLOG_W(TAG, "Script load failed: %s", lua_tostring(L, -1));
                        lua_pop(L, 1);
                    } else {
                        DLOG_E(TAG, "Failed to recreate Lua state");
                        L = create_lua_state();
                    }
                    if (L) {
                        if (luaL_dostring(L, FALLBACK_SCRIPT) == LUA_OK) {
                            script_loaded = true;
                        }
                    }
                    consecutive_failures = 0;
                }
            }
            free(new_source);
        }

        // If Lua VM is dead, skip rendering but keep the loop alive
        // so we can pick up new scripts from the channel
        if (!L) {
            L = create_lua_state();
            if (L && luaL_dostring(L, FALLBACK_SCRIPT) == LUA_OK) {
                script_loaded = true;
                DLOG_W(TAG, "Recovered Lua VM with fallback");
            }
            if (!L) {
                vTaskDelay(pdMS_TO_TICKS(1000)); // back off on persistent OOM
                continue;
            }
        }

        // Read transit data (lock-free double buffer)
        const auto& transit = args->transit_buf->read();

        // Lock board data for this frame. Held during Lua render (~1ms).
        // Board writes are rare (~once per boot) so contention is negligible.
        args->board_store->lock_for_read();
        const auto& board = args->board_store->snapshot();

        // Set up per-frame binding context
        auto pixels = args->led->pixel_buffer();
        uint32_t led_count = args->led->led_count();

        LuaBindingContext ctx;
        ctx.transit = &transit;
        ctx.board = &board;
        ctx.pixels = pixels;
        ctx.led_count = led_count;
        lua_set_binding_context(L, &ctx);

        // GC before render -- free temporary allocations from previous frame
        // so max Lua memory is available during the render call
        lua_gc(L, LUA_GCCOLLECT, 0);

        // Clear pixel buffer
        for (auto& px : pixels)
            px = Rgb{};

        // Call Lua render()
        int64_t frame_start = esp_timer_get_time();

        if (script_loaded) {
            lua_getglobal(L, "render");
            if (lua_pcall(L, 0, 0, 0) != LUA_OK) {
                const char* err = lua_tostring(L, -1);
                DLOG_W(TAG, "Lua render error: %s", err ? err : "unknown");
                if (err) {
                    args->diag->set_lua_err(err);
                }
                lua_pop(L, 1);
                consecutive_failures++;

                if (consecutive_failures >= static_cast<int>(kMaxConsecutiveFailures)) {
                    DLOG_W(TAG, "Too many failures, loading fallback script");
                    if (luaL_dostring(L, FALLBACK_SCRIPT) == LUA_OK) {
                        consecutive_failures = 0;
                    }
                }
            } else {
                consecutive_failures = 0;
            }
        }

        // Release board mutex after Lua render
        args->board_store->unlock_read();

        int64_t frame_end = esp_timer_get_time();
        uint32_t frame_us = static_cast<uint32_t>(frame_end - frame_start);

        // Count non-zero pixels and find first lit LED after Lua render
        uint32_t nonzero_pixels = 0;
        uint32_t first_lit = UINT32_MAX;
        for (uint32_t i = 0; i < led_count; i++) {
            if (pixels[i].r || pixels[i].g || pixels[i].b) {
                nonzero_pixels++;
                if (first_lit == UINT32_MAX)
                    first_lit = i;
            }
        }

        // Skip LED refresh during HTTP -- TLS + SPI DMA concurrent current draw causes brownout
        if (!args->http_active->load(std::memory_order_acquire)) {
            args->led->refresh(*args->diag);
        }

        // Update diagnostics (all atomic, no mutex needed)
        args->diag->nonzero_pixels.store(nonzero_pixels, std::memory_order_relaxed);
        args->diag->pushed_pixels.store(led_count, std::memory_order_relaxed);
        args->diag->lua_errors.store(consecutive_failures, std::memory_order_relaxed);
        args->diag->lua_mem.store(static_cast<uint32_t>(s_lua_mem_used), std::memory_order_relaxed);
        args->diag->first_lit_led.store(first_lit, std::memory_order_relaxed);
        args->diag->frame_time_us.store(frame_us, std::memory_order_relaxed);

        // Feed watchdog + sleep for render interval (~30ms = ~33fps)
        esp_task_wdt_reset();
        vTaskDelay(pdMS_TO_TICKS(kRenderIntervalMs));
    }

    // Unreachable, but for completeness
    lua_close(L);
    vTaskDelete(nullptr);
}

// --- Public API ---

void LuaRuntime::start(DoubleBuffer<TransitSnapshot>& transit_buf,
                       BoardStore& board_store,
                       ScriptChannel& script_chan,
                       DiagPad& diag,
                       std::atomic<bool>& http_active,
                       LedDriver& led) {
    static RenderTaskArgs args;
    args.transit_buf = &transit_buf;
    args.board_store = &board_store;
    args.script_chan = &script_chan;
    args.diag = &diag;
    args.http_active = &http_active;
    args.led = &led;

    xTaskCreate(render_task, "render_task", 8192, &args, 5, nullptr);
}
