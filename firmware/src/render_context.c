#include "render_context.h"
#include <string.h>

void render_context_init(render_context_t *ctx)
{
    memset(ctx, 0, sizeof(render_context_t));
    ctx->mutex = xSemaphoreCreateMutex();
    configASSERT(ctx->mutex != NULL);
}

void render_context_build_station_leds(render_context_t *ctx)
{
    ctx->station_leds_count = 0;

    /* Build inverted index: for each unique station in led_map,
       collect all LED indices that map to it */
    for (uint32_t i = 0; i < ctx->board.led_count && i < MAX_LEDS; i++) {
        if (ctx->board.led_map[i][0] == '\0') continue;

        /* Find existing entry or create new one */
        int found = -1;
        for (uint16_t j = 0; j < ctx->station_leds_count; j++) {
            if (strcmp(ctx->station_leds[j].station_id, ctx->board.led_map[i]) == 0) {
                found = j;
                break;
            }
        }

        if (found < 0) {
            if (ctx->station_leds_count >= PB_MAX_STATIONS) continue;
            found = ctx->station_leds_count++;
            strncpy(ctx->station_leds[found].station_id, ctx->board.led_map[i], PB_STOP_ID_LEN - 1);
            ctx->station_leds[found].count = 0;
        }

        if (ctx->station_leds[found].count < MAX_LEDS_PER_STATION) {
            ctx->station_leds[found].led_indices[ctx->station_leds[found].count++] = i;
        }
    }
}
