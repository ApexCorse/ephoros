#include "can.h"
#include "network.h"
#include "telemetry.h"

void app_main(void) {
	ESP_ERROR_CHECK(network_start());
	ESP_ERROR_CHECK(can_start());
	ESP_ERROR_CHECK(telemetry_start());
}
