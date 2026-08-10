#pragma once

#include <stdbool.h>

#include "can.h"
#include "esp_err.h"

/** Start the build-time-DBC-configured synthetic telemetry source. */
esp_err_t telemetry_simulator_start(void);

/** Wait for the next generated signal. */
bool telemetry_simulator_next(can_decoded_signal_t *signal, TickType_t timeout);
