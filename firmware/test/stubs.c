#include <stdio.h>
#include <stdarg.h>
#include <string.h>
#include <stdbool.h>
#include "esp_err.h"
#include "device_log.h"
#include "render_context.h"
#include "board_config.h"
#include "lua.h"
#include "lauxlib.h"

/* device_log — just printf */
void device_log(device_log_level_t level, const char *tag, const char *fmt, ...)
{
    (void)level;
    va_list ap;
    va_start(ap, fmt);
    printf("[%s] ", tag);
    vprintf(fmt, ap);
    printf("\n");
    va_end(ap);
}

void device_log_init(void) {}
void device_log_set_remote(bool enabled) { (void)enabled; }
int device_log_drain(char *buf, int buf_size) { (void)buf; (void)buf_size; return 0; }

/* led_driver — no-ops */
esp_err_t led_driver_init(struct render_context *ctx, const board_hw_config_t *hw)
{
    (void)ctx; (void)hw;
    return ESP_OK;
}

esp_err_t led_driver_set_pixel(uint8_t strip, uint16_t pixel, uint8_t r, uint8_t g, uint8_t b)
{
    (void)strip; (void)pixel; (void)r; (void)g; (void)b;
    return ESP_OK;
}

esp_err_t led_driver_refresh(void) { return ESP_OK; }
esp_err_t led_driver_clear(void) { return ESP_OK; }

bool led_driver_map_pixel(uint32_t global_index, uint8_t *out_strip, uint16_t *out_pixel)
{
    (void)global_index; (void)out_strip; (void)out_pixel;
    return false;
}

/* board_config — no-op */
void board_config_load(board_hw_config_t *cfg) { (void)cfg; }

/* Lua library openers referenced by linit.c but stripped from this build */
int luaopen_io(lua_State *L) { (void)L; return 0; }
int luaopen_os(lua_State *L) { (void)L; return 0; }
int luaopen_package(lua_State *L) { (void)L; return 0; }
int luaopen_debug(lua_State *L) { (void)L; return 0; }

/* render_context */
void render_context_init(render_context_t *ctx)
{
    memset(ctx, 0, sizeof(*ctx));
}

void render_context_build_station_leds(render_context_t *ctx)
{
    ctx->station_leds_count = 0;
    for (uint32_t i = 0; i < ctx->board.led_count && i < MAX_LEDS; i++) {
        const char *sid = ctx->board.led_map[i];
        if (sid[0] == '\0') continue;

        /* Find existing entry */
        int found = -1;
        for (uint16_t j = 0; j < ctx->station_leds_count; j++) {
            if (strcmp(ctx->station_leds[j].station_id, sid) == 0) {
                found = j;
                break;
            }
        }
        if (found < 0) {
            if (ctx->station_leds_count >= PB_MAX_STATIONS) continue;
            found = ctx->station_leds_count++;
            strncpy(ctx->station_leds[found].station_id, sid, PB_STOP_ID_LEN - 1);
            ctx->station_leds[found].count = 0;
        }
        if (ctx->station_leds[found].count < MAX_LEDS_PER_STATION) {
            ctx->station_leds[found].led_indices[ctx->station_leds[found].count++] = i;
        }
    }
}
