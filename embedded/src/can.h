#pragma once

#include "esp_err.h"

/**
 * Start the ESP32 TWAI receiver and its frame-processing task.
 *
 * GPIO pins, bitrate, and receive mode are configured through menuconfig.
 */
esp_err_t can_start(void);
