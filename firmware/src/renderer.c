#include "renderer.h"
#include "led_driver.h"
#include "station_map.h"
#include "config.h"

#include "esp_log.h"
#include "esp_timer.h"
#include <string.h>

static const char *TAG = "renderer";

/* Route color table: RGB for each subway_Route enum value */
typedef struct {
    uint8_t r, g, b;
} rgb_t;

static const rgb_t ROUTE_COLORS[] = {
    [subway_Route_ROUTE_UNKNOWN] = { 20,  20,  20},  /* Dim White */
    [subway_Route_ROUTE_1]       = {238,  53,  46},  /* Red */
    [subway_Route_ROUTE_2]       = {238,  53,  46},  /* Red */
    [subway_Route_ROUTE_3]       = {238,  53,  46},  /* Red */
    [subway_Route_ROUTE_4]       = {  0, 147,  60},  /* Green */
    [subway_Route_ROUTE_5]       = {  0, 147,  60},  /* Green */
    [subway_Route_ROUTE_6]       = {  0, 147,  60},  /* Green */
    [subway_Route_ROUTE_7]       = {185,  51, 173},  /* Purple */
    [subway_Route_ROUTE_A]       = {  0,  57, 166},  /* Blue */
    [subway_Route_ROUTE_B]       = {255,  99,  25},  /* Orange */
    [subway_Route_ROUTE_C]       = {  0,  57, 166},  /* Blue */
    [subway_Route_ROUTE_D]       = {255,  99,  25},  /* Orange */
    [subway_Route_ROUTE_E]       = {  0,  57, 166},  /* Blue */
    [subway_Route_ROUTE_F]       = {255,  99,  25},  /* Orange */
    [subway_Route_ROUTE_G]       = {108, 190,  69},  /* Light Green */
    [subway_Route_ROUTE_J]       = {153, 102,  51},  /* Brown */
    [subway_Route_ROUTE_L]       = {167, 169, 172},  /* Gray */
    [subway_Route_ROUTE_M]       = {255,  99,  25},  /* Orange */
    [subway_Route_ROUTE_N]       = {252, 204,  10},  /* Yellow */
    [subway_Route_ROUTE_Q]       = {252, 204,  10},  /* Yellow */
    [subway_Route_ROUTE_R]       = {252, 204,  10},  /* Yellow */
    [subway_Route_ROUTE_W]       = {252, 204,  10},  /* Yellow */
    [subway_Route_ROUTE_Z]       = {153, 102,  51},  /* Brown */
    [subway_Route_ROUTE_S]       = {128, 129, 131},  /* Shuttle Gray */
    [subway_Route_ROUTE_FS]      = {128, 129, 131},  /* Shuttle Gray */
    [subway_Route_ROUTE_GS]      = {128, 129, 131},  /* Shuttle Gray */
    [subway_Route_ROUTE_SI]      = {  0,  57, 166},  /* Blue */
};

/* Status brightness multiplier (percentage out of 255) */
static inline uint8_t status_brightness(subway_TrainStatus status)
{
    switch (status) {
        case subway_TrainStatus_STOPPED_AT:      return 255;  /* 100% */
        case subway_TrainStatus_INCOMING_AT:     return 179;  /* ~70% */
        case subway_TrainStatus_IN_TRANSIT_TO:   return 102;  /* ~40% */
        default:                                 return 0;
    }
}

/* Priority: higher = wins when multiple trains at same LED.
 * STOPPED_AT > INCOMING_AT > IN_TRANSIT_TO */
static inline int status_priority(subway_TrainStatus status)
{
    switch (status) {
        case subway_TrainStatus_STOPPED_AT:      return 3;
        case subway_TrainStatus_INCOMING_AT:     return 2;
        case subway_TrainStatus_IN_TRANSIT_TO:   return 1;
        default:                                 return 0;
    }
}

/* Per-LED persistence tracking */
typedef struct {
    int64_t last_change_us;   /* microsecond timestamp of last color change */
    uint8_t r, g, b;         /* currently displayed color */
    subway_Route route;
    subway_TrainStatus status;
    bool active;             /* true if a train was assigned this frame */
} led_state_t;

/* Flat array indexed by (strip_base_offset + pixel).
 * We compute strip base offsets from STRIP_LED_COUNTS. */
static led_state_t s_led_state[TOTAL_LEDS];
static uint16_t s_strip_offsets[NUM_STRIPS];

void renderer_init(void)
{
    memset(s_led_state, 0, sizeof(s_led_state));

    /* Precompute strip base offsets into flat array */
    uint16_t offset = 0;
    for (int i = 0; i < NUM_STRIPS; i++) {
        s_strip_offsets[i] = offset;
        offset += STRIP_LED_COUNTS[i];
    }

    ESP_LOGI(TAG, "Renderer initialized (%d LEDs)", TOTAL_LEDS);
}

void renderer_get_idle_color(uint8_t *r, uint8_t *g, uint8_t *b)
{
    /* Dim white for idle stations */
    *r = 3;
    *g = 3;
    *b = 3;
}

static inline uint16_t led_flat_index(uint8_t strip, uint16_t pixel)
{
    return s_strip_offsets[strip] + pixel;
}

static inline void apply_brightness(const rgb_t *color, uint8_t bright_pct,
                                    uint8_t *r, uint8_t *g, uint8_t *b)
{
    /* Apply status brightness, then scale by global brightness */
    uint16_t br = DEFAULT_BRIGHTNESS;
    *r = (uint8_t)(((uint16_t)color->r * bright_pct * br) / (255 * 255));
    *g = (uint8_t)(((uint16_t)color->g * bright_pct * br) / (255 * 255));
    *b = (uint8_t)(((uint16_t)color->b * bright_pct * br) / (255 * 255));
}

void renderer_update(subway_SubwayState *state)
{
    if (!state) return;

    int64_t now_us = esp_timer_get_time();
    int64_t persist_us = (int64_t)LED_PERSIST_SEC * 1000000LL;

    /* Temporary array to track which LEDs are active this frame.
     * We use the led_state active flag, reset it each frame. */
    for (int i = 0; i < TOTAL_LEDS; i++) {
        s_led_state[i].active = false;
    }

    /* Temporary best-priority tracking per LED for this frame */
    static int8_t frame_priority[TOTAL_LEDS];
    static subway_Route frame_route[TOTAL_LEDS];
    static subway_TrainStatus frame_status[TOTAL_LEDS];
    memset(frame_priority, 0, sizeof(frame_priority));

    /* Process each station in the state */
    for (pb_size_t s = 0; s < state->stations_count; s++) {
        subway_Station *station = &state->stations[s];

        /* Find LED positions for this station */
        led_pos_t positions[MAX_LEDS_PER_STATION];
        int n_leds = station_map_find(station->stop_id, positions, MAX_LEDS_PER_STATION);
        if (n_leds == 0) continue;

        /* Find the highest-priority train at this station */
        subway_Route best_route = subway_Route_ROUTE_UNKNOWN;
        subway_TrainStatus best_status = subway_TrainStatus_STATUS_NONE;
        int best_pri = 0;

        for (pb_size_t t = 0; t < station->trains_count; t++) {
            subway_Train *train = &station->trains[t];
            int pri = status_priority(train->status);
            if (pri > best_pri) {
                best_pri = pri;
                best_route = train->route;
                best_status = train->status;
            }
        }

        if (best_pri == 0) continue;

        /* Apply to all LED positions for this station */
        int apply_count = n_leds < MAX_LEDS_PER_STATION ? n_leds : MAX_LEDS_PER_STATION;
        for (int l = 0; l < apply_count; l++) {
            uint16_t idx = led_flat_index(positions[l].strip, positions[l].pixel);
            if (idx >= TOTAL_LEDS) continue;

            s_led_state[idx].active = true;

            /* Multi-train resolution: highest priority wins across all stations
             * that share this LED position */
            if (best_pri > frame_priority[idx]) {
                frame_priority[idx] = best_pri;
                frame_route[idx] = best_route;
                frame_status[idx] = best_status;
            }
        }
    }

    /* Now render each LED */
    uint8_t idle_r, idle_g, idle_b;
    renderer_get_idle_color(&idle_r, &idle_g, &idle_b);

    for (int i = 0; i < TOTAL_LEDS; i++) {
        uint8_t r, g, b;

        if (s_led_state[i].active && frame_priority[i] > 0) {
            /* Compute desired color */
            subway_Route route = frame_route[i];
            subway_TrainStatus status = frame_status[i];

            if (route < 0 || route > subway_Route_ROUTE_SI) {
                route = subway_Route_ROUTE_UNKNOWN;
            }

            const rgb_t *color = &ROUTE_COLORS[route];
            uint8_t bright = status_brightness(status);
            apply_brightness(color, bright, &r, &g, &b);

            /* Persistence check: only change if 10s elapsed or same train */
            bool should_change = true;
            if (s_led_state[i].r != 0 || s_led_state[i].g != 0 || s_led_state[i].b != 0) {
                /* LED was previously showing something */
                if (s_led_state[i].route != route || s_led_state[i].status != status) {
                    /* Different train/status - check persistence timer */
                    if ((now_us - s_led_state[i].last_change_us) < persist_us) {
                        should_change = false;
                    }
                }
            }

            if (should_change) {
                s_led_state[i].r = r;
                s_led_state[i].g = g;
                s_led_state[i].b = b;
                s_led_state[i].route = route;
                s_led_state[i].status = status;
                s_led_state[i].last_change_us = now_us;
            }
        } else {
            /* No active train - check persistence before going idle */
            if (s_led_state[i].r != idle_r || s_led_state[i].g != idle_g || s_led_state[i].b != idle_b) {
                if ((now_us - s_led_state[i].last_change_us) >= persist_us) {
                    s_led_state[i].r = idle_r;
                    s_led_state[i].g = idle_g;
                    s_led_state[i].b = idle_b;
                    s_led_state[i].route = subway_Route_ROUTE_UNKNOWN;
                    s_led_state[i].status = subway_TrainStatus_STATUS_NONE;
                    s_led_state[i].last_change_us = now_us;
                }
            }
        }
    }

    /* Count active LEDs for logging */
    int active_count = 0;
    for (int i = 0; i < TOTAL_LEDS; i++) {
        if (s_led_state[i].r > 0 || s_led_state[i].g > 0 || s_led_state[i].b > 0) {
            active_count++;
        }
    }
    ESP_LOGI(TAG, "Rendering %d active LEDs", active_count);

    /* Push colors to LED driver */
    uint16_t flat_idx = 0;
    for (int strip = 0; strip < NUM_STRIPS; strip++) {
        for (uint16_t pixel = 0; pixel < STRIP_LED_COUNTS[strip]; pixel++) {
            led_driver_set_pixel(strip, pixel,
                                 s_led_state[flat_idx].r,
                                 s_led_state[flat_idx].g,
                                 s_led_state[flat_idx].b);
            flat_idx++;
        }
    }
    led_driver_refresh();
}
