#ifndef HTTP_CLIENT_H
#define HTTP_CLIENT_H

#include <stdint.h>

/* Forward declaration */
struct render_context;

typedef struct {
    const char *server_url;
    const char *device_id;
    const char *firmware_ver;
    const char *hardware;
} http_client_config_t;

typedef struct {
    uint8_t *data;
    int len;
} http_response_t;

/* Initialize the HTTP client (call once after NVS config is read).
 * Stores ctx pointer to set http_active flag during TLS operations. */
void http_client_init(const http_client_config_t *cfg, struct render_context *ctx);

/* GET request. Returns 0 on success, -1 on failure. */
int http_client_get(const char *path, http_response_t *resp);

/* POST request with protobuf body. Returns 0 on success, -1 on failure. */
int http_client_post(const char *path, const uint8_t *body, int body_len, http_response_t *resp);

#endif /* HTTP_CLIENT_H */
