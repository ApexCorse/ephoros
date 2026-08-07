#pragma once

#include "esp_err.h"

/**
 * Initialize the configured network backend and start connecting.
 *
 * This function returns once Wi-Fi association or cellular PPP startup has
 * been requested. Connection progress is reported through the ESP event loop.
 */
esp_err_t network_start(void);
