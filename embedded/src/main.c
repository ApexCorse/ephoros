#include "can.h"
#include "network.h"

void app_main(void) {
	ESP_ERROR_CHECK(network_start());
	ESP_ERROR_CHECK(can_start());

	/* Application business logic starts here. */
}
