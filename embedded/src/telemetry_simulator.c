#include "sdkconfig.h"

#if CONFIG_EPHOROS_CAN_SIMULATOR

#include "telemetry_simulator.h"

#include <stdlib.h>
#include <string.h>

#include "esp_log.h"
#include "esp_random.h"
#include "freertos/task.h"
#include "telemetry_catalog.h"

static const char *TAG = "telemetry_sim";
static bool s_started;

esp_err_t telemetry_simulator_start(void) {
	if (s_started) {
		return ESP_ERR_INVALID_STATE;
	}
	if (EPHOROS_TELEMETRY_SIMULATOR_TOPIC_COUNT == 0) {
		return ESP_ERR_INVALID_STATE;
	}

	unsigned int seed = CONFIG_EPHOROS_CAN_SIMULATOR_SEED;
	if (seed == 0) {
		seed = esp_random();
	}
	srand(seed);
	s_started = true;
	ESP_LOGI(TAG, "simulating %d DBC topics every %d ms",
			 EPHOROS_TELEMETRY_SIMULATOR_TOPIC_COUNT,
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
	const size_t topic_index = (size_t)rand() % EPHOROS_TELEMETRY_SIMULATOR_TOPIC_COUNT;
	strlcpy(signal->topic, ephoros_telemetry_simulator_topics[topic_index],
			sizeof(signal->topic));
	strlcpy(signal->name, "simulated", sizeof(signal->name));
	strlcpy(signal->unit, "V", sizeof(signal->unit));
	signal->value = ((float)rand() / (float)RAND_MAX) * 1000.0f - 500.0f;
	return true;
}

#endif
