#ifndef FREERTOS_H
#define FREERTOS_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>

typedef void* SemaphoreHandle_t;
typedef uint32_t TickType_t;
typedef int BaseType_t;

#define portMAX_DELAY 0xFFFFFFFF
#define pdMS_TO_TICKS(ms) ((TickType_t)(ms))
#define pdTRUE 1
#define pdFALSE 0
#define configSTACK_DEPTH_TYPE uint32_t

#ifdef __cplusplus
}
#endif

#endif
