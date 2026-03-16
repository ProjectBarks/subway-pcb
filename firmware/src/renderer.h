#ifndef RENDERER_H
#define RENDERER_H

#include "subway.pb.h"

/**
 * Initialize the renderer (clears persistence state).
 */
void renderer_init(void);

/**
 * Update LED display from a decoded SubwayState protobuf.
 * Applies route colors, status brightness, persistence, and multi-train priority.
 */
void renderer_update(subway_SubwayState *state);

/**
 * Get the idle/dim color for stations with no active trains.
 */
void renderer_get_idle_color(uint8_t *r, uint8_t *g, uint8_t *b);

#endif /* RENDERER_H */
