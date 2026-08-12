#pragma once

#include "esp_err.h"

/** Start the CAN-to-transport telemetry fan-out and its sink workers. */
esp_err_t telemetry_start(void);
