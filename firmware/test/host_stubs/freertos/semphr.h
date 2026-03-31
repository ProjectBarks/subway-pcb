#ifndef SEMPHR_H
#define SEMPHR_H

#include "FreeRTOS.h"

#ifdef __cplusplus
extern "C" {
#endif

#define xSemaphoreTake(sem, timeout) pdTRUE
#define xSemaphoreGive(sem) pdTRUE
#define xSemaphoreCreateMutex() ((SemaphoreHandle_t)1)

#ifdef __cplusplus
}
#endif

#endif
