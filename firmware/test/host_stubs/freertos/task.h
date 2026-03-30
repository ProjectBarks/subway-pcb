#ifndef TASK_H
#define TASK_H

#include "FreeRTOS.h"

typedef void* TaskHandle_t;
typedef void (*TaskFunction_t)(void*);

#define xTaskCreate(fn, name, stack, param, prio, handle) ((void)(fn), (void)(name), (void)(stack), (void)(param), (void)(prio), (void)(handle), pdTRUE)
#define vTaskDelay(ticks) ((void)(ticks))
#define vTaskDelete(handle) ((void)(handle))

#endif
