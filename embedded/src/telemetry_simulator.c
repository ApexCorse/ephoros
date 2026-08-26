#include "sdkconfig.h"

#if CONFIG_EPHOROS_CAN_SIMULATOR

#include "telemetry_simulator.h"

#include <math.h>
#include <stdlib.h>
#include <string.h>

#include "esp_log.h"
#include "esp_random.h"
#include "freertos/task.h"
#include "telemetry_catalog.h"

static const char *TAG = "telemetry_sim";
static bool s_started;

static float random_positive_unit_interval(void) {
	return ((float)rand() + 1.0f) / ((float)RAND_MAX + 2.0f);
}

static float generate_bounded_gaussian(float minimum, float maximum) {
	if (minimum == maximum) {
		return minimum;
	}

	const float mean = minimum / 2.0f + maximum / 2.0f;
	const float standard_deviation = maximum / 6.0f - minimum / 6.0f;
	const float two_pi = 6.28318530717958647692f;

	// The configured range spans six standard deviations. Rejection produces
	// a truncated Gaussian without concentrating outliers at either bound.
	for (int attempt = 0; attempt < 16; ++attempt) {
		const float u1 = random_positive_unit_interval();
		const float u2 = random_positive_unit_interval();
		const float standard_normal = sqrtf(-2.0f * logf(u1)) * cosf(two_pi * u2);
		const float value = mean + standard_normal * standard_deviation;

		if (value >= minimum && value <= maximum) {
			return value;
		}
	}

	// Reaching this fallback has probability below 10^-40 for valid bounds.
	return mean;
}

esp_err_t telemetry_simulator_start(void) {
	if (s_started) {
		return ESP_ERR_INVALID_STATE;
	}
	if (EPHOROS_TELEMETRY_SIMULATOR_SIGNAL_COUNT == 0) {
		return ESP_ERR_INVALID_STATE;
	}

	unsigned int seed = CONFIG_EPHOROS_CAN_SIMULATOR_SEED;
	if (seed == 0) {
		seed = esp_random();
	}
	srand(seed);
	s_started = true;
	ESP_LOGI(TAG, "simulating %d DBC signals every %d ms",
			 EPHOROS_TELEMETRY_SIMULATOR_SIGNAL_COUNT,
			 CONFIG_EPHOROS_CAN_SIMULATOR_INTERVAL_MS);
	return ESP_OK;
}

bool telemetry_simulator_next(can_decoded_signal_t *signal, TickType_t timeout) {
	if (!s_started || signal == NULL) {
		return false;
	}

	const TickType_t delay = pdMS_TO_TICKS(CONFIG_EPHOROS_CAN_SIMULATOR_INTERVAL_MS);
	if (timeout != portMAX_DELAY && timeout < delay) {
		vTaskDelay(timeout);
		return false;
	}
	vTaskDelay(delay);

	memset(signal, 0, sizeof(*signal));
	const size_t signal_index = (size_t)rand() % EPHOROS_TELEMETRY_SIMULATOR_SIGNAL_COUNT;
	const ephoros_telemetry_simulator_signal_t *simulated =
		&ephoros_telemetry_simulator_signals[signal_index];
	strlcpy(signal->topic, simulated->topic, sizeof(signal->topic));
	strlcpy(signal->name, simulated->name, sizeof(signal->name));
	strlcpy(signal->unit, simulated->unit, sizeof(signal->unit));
	signal->value = generate_bounded_gaussian(simulated->min, simulated->max);
	return true;
}

#endif
