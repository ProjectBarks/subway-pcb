#ifndef SEMPHR_H
#define SEMPHR_H

#include "FreeRTOS.h"

#define xSemaphoreTake(sem, timeout) pdTRUE
#define xSemaphoreGive(sem) pdTRUE
#define xSemaphoreCreateMutex() ((SemaphoreHandle_t)1)

#endif
