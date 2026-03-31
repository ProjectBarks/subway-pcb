#ifndef QUEUE_H
#define QUEUE_H

#include "FreeRTOS.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef void* QueueHandle_t;

#define xQueueCreate(len, size) ((QueueHandle_t)1)
#define xQueueSend(q, item, timeout) pdTRUE
#define xQueueReceive(q, item, timeout) pdFALSE
#define xQueueOverwrite(q, item) pdTRUE
#define vQueueDelete(q) ((void)(q))

#ifdef __cplusplus
}
#endif

#endif
