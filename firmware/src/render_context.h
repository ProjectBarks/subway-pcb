#ifndef RENDER_CONTEXT_H
#define RENDER_CONTEXT_H

#include <stdint.h>
#include <stdbool.h>
#include "freertos/FreeRTOS.h"
#include "freertos/semphr.h"
#include "config.h"
#include "subway.pb.h"
#include <pb.h>

/* Derived from nanopb options — single source of truth for protobuf sizes */
#define PB_MAX_STATIONS       pb_arraysize(subway_DeviceState, stations)
#define PB_MAX_CONFIG         pb_arraysize(subway_DeviceState, config)
#define PB_HASH_LEN           pb_membersize(subway_DeviceState, script_hash)
#define PB_STOP_ID_LEN        pb_membersize(subway_Station, stop_id)

/* Firmware-only constants (not in protobuf schema) */
#define MAX_LEDS_PER_STATION  32

/* Board info (derived structure — not a protobuf mirror) */
typedef struct {
    char board_id[32];
    uint32_t led_count;
    uint32_t strip_sizes[MAX_STRIPS];
    uint8_t strip_count;
    char led_map[MAX_LEDS][PB_STOP_ID_LEN];
    char hash[PB_HASH_LEN];
} board_info_t;

/* Inverted index: station -> LED indices */
typedef struct {
    char station_id[PB_STOP_ID_LEN];
    uint16_t led_indices[MAX_LEDS_PER_STATION];
    uint8_t count;
} station_leds_entry_t;

/* Device diagnostics (written by lua_runtime + led_driver, read by state_client) */
typedef struct {
    uint32_t nonzero_pixels;
    uint32_t pushed_pixels;
    int strip_ok;
    int strip_fail;
    int lua_errors;
    uint32_t lua_mem;
    uint32_t first_lit_led;
    int last_reload;
    char last_lua_err[64];
    char fetch_err[64];
} device_diag_t;

/* Render context shared between tasks */
typedef struct render_context {
    SemaphoreHandle_t mutex;

    /* MTA state — uses nanopb types directly */
    subway_Station stations[PB_MAX_STATIONS];
    pb_size_t station_count;
    uint64_t timestamp;

    /* Config — uses nanopb type directly */
    subway_DeviceState_ConfigEntry config[PB_MAX_CONFIG];
    pb_size_t config_count;

    /* Board info (cached, updated on hash change) */
    board_info_t board;
    bool board_loaded;

    /* Inverted index for get_leds_for_station() */
    station_leds_entry_t station_leds[PB_MAX_STATIONS];
    uint16_t station_leds_count;

    /* Script state */
    char script_hash[PB_HASH_LEN];
    char board_hash[PB_HASH_LEN];
    bool script_changed;
    char *lua_source;

    /* Hash for change detection */
    char cached_script_hash[PB_HASH_LEN];
    char cached_board_hash[PB_HASH_LEN];

    /* Cross-task flags */
    volatile bool ota_active;
    volatile bool http_active;

    /* Diagnostics */
    device_diag_t diag;
} render_context_t;

/* Initialize the render context (call once at startup) */
void render_context_init(render_context_t *ctx);

/* Build the inverted station->LEDs index from board data */
void render_context_build_station_leds(render_context_t *ctx);

#endif /* RENDER_CONTEXT_H */
