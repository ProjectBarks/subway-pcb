#ifndef RENDER_CONTEXT_H
#define RENDER_CONTEXT_H

#include <stdint.h>
#include <stdbool.h>
#include "freertos/FreeRTOS.h"
#include "freertos/semphr.h"
#include "config.h"

/* Maximum sizes */
#define MAX_STATIONS      200
#define MAX_TRAINS_PER_STATION 4
#define MAX_ROUTE_LEN     8
#define MAX_STOP_ID_LEN   8
#define MAX_CONFIG_ENTRIES 80
#define MAX_CONFIG_KEY_LEN 32
#define MAX_CONFIG_VAL_LEN 32
#define MAX_SCRIPT_SIZE   16384
#define MAX_HASH_LEN      65

/* Train status enum (matches protobuf) */
typedef enum {
    TRAIN_STATUS_NONE       = 0,
    TRAIN_STATUS_STOPPED_AT = 1,
    TRAIN_STATUS_INCOMING_AT = 2,
    TRAIN_STATUS_IN_TRANSIT_TO = 3,
} train_status_t;

/* A single train at a station */
typedef struct {
    char route[MAX_ROUTE_LEN];
    train_status_t status;
} train_t;

/* A station with its trains */
typedef struct {
    char stop_id[MAX_STOP_ID_LEN];
    train_t trains[MAX_TRAINS_PER_STATION];
    uint8_t train_count;
} station_t;

/* Config key-value pair */
typedef struct {
    char key[MAX_CONFIG_KEY_LEN];
    char value[MAX_CONFIG_VAL_LEN];
} config_entry_t;

/* Board info (cached, rarely changes) */
typedef struct {
    char board_id[32];
    uint32_t led_count;
    uint32_t strip_sizes[16];
    uint8_t strip_count;
    char led_map[MAX_LEDS][MAX_STOP_ID_LEN];  /* LED index -> station ID */
    char hash[MAX_HASH_LEN];
} board_info_t;

/* Inverted index: station -> LED indices */
typedef struct {
    char station_id[MAX_STOP_ID_LEN];
    uint16_t led_indices[32]; /* up to 32 LEDs per station */
    uint8_t count;
} station_leds_entry_t;

/* Render context shared between tasks */
typedef struct {
    SemaphoreHandle_t mutex;

    /* MTA state (updated every cycle by state_task) */
    station_t stations[MAX_STATIONS];
    uint16_t station_count;
    uint64_t timestamp;

    /* Config (updated every cycle) */
    config_entry_t config[MAX_CONFIG_ENTRIES];
    uint8_t config_count;

    /* Board info (cached, updated on hash change) */
    board_info_t board;
    bool board_loaded;

    /* Inverted index for get_leds_for_station() */
    station_leds_entry_t station_leds[MAX_STATIONS];
    uint16_t station_leds_count;

    /* Script state */
    char script_hash[MAX_HASH_LEN];
    char board_hash[MAX_HASH_LEN];
    bool script_changed;
    char *lua_source;  /* heap-allocated, set by state_client, consumed by lua_runtime */

    /* Lua source (stored in SPIFFS, hash for change detection) */
    char cached_script_hash[MAX_HASH_LEN];
    char cached_board_hash[MAX_HASH_LEN];

    /* Render diagnostics (written by lua_runtime, read by state_client) */
    uint32_t diag_nonzero_pixels;  /* non-zero pixels after Lua render() */
    uint32_t diag_pushed_pixels;   /* pixels mapped to strips */
    int diag_strip_ok;             /* strips refreshed successfully */
    int diag_strip_fail;           /* strips that failed refresh */
    int diag_lua_errors;           /* consecutive Lua errors */
    uint32_t diag_lua_mem;         /* Lua VM memory usage in bytes */
    uint32_t diag_first_lit_led;   /* index of first non-zero LED */
    int diag_last_reload;          /* 0=none, 1=ok, -1=failed */
    char diag_last_lua_err[64];    /* last Lua error (truncated) */
    char diag_fetch_err[64];       /* last fetch_board/script error info */
} render_context_t;

/* Initialize the render context (call once at startup) */
void render_context_init(render_context_t *ctx);

/* Build the inverted station->LEDs index from board data */
void render_context_build_station_leds(render_context_t *ctx);

#endif /* RENDER_CONTEXT_H */
