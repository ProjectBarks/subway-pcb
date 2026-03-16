#ifndef SUBWAY_CLIENT_H
#define SUBWAY_CLIENT_H

#include "esp_err.h"

/**
 * Start the subway client FreeRTOS task.
 * Polls the server for SubwayState protobuf and updates the renderer.
 * Should be called after WiFi is connected.
 */
esp_err_t subway_client_start(void);

#endif /* SUBWAY_CLIENT_H */
