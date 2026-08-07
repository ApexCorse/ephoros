#include <assert.h>
#include <string.h>

#include "driver/gpio.h"
#include "esp_event.h"
#include "esp_log.h"
#include "esp_modem_api.h"
#include "esp_netif.h"
#include "esp_netif_ppp.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "sdkconfig.h"

#define BOARD_POWER_EN GPIO_NUM_12
#define MODEM_TX_PIN   GPIO_NUM_26
#define MODEM_RX_PIN   GPIO_NUM_27
#define MODEM_PWRKEY   GPIO_NUM_4
#define MODEM_RESET    GPIO_NUM_5
#define MODEM_DTR      GPIO_NUM_25

static const char *TAG = "cellular";
static esp_modem_dce_t *s_dce;

static void on_ip_event(
	void *arg,
	esp_event_base_t event_base,
	int32_t event_id,
	void *event_data
) {
	if (event_id == IP_EVENT_PPP_GOT_IP) {
		ESP_LOGI(TAG, "PPP connected; the default route is now cellular");
	} else if (event_id == IP_EVENT_PPP_LOST_IP) {
		ESP_LOGW(TAG, "PPP connection lost");
	}
}

static const char *pdp_protocol_type(void) {
#if CONFIG_EPHOROS_MODEM_PDP_TYPE_IPV4V6
	return "IPV4V6";
#elif CONFIG_EPHOROS_MODEM_PDP_TYPE_IPV6
	return "IPV6";
#else
	return "IP";
#endif
}

static esp_err_t unlock_sim_if_needed(esp_modem_dce_t *dce) {
	if (strlen(CONFIG_EPHOROS_MODEM_SIM_PIN) == 0) {
		return ESP_OK;
	}

	bool pin_ok = false;
	esp_err_t err = esp_modem_read_pin(dce, &pin_ok);
	if (err != ESP_OK || pin_ok) {
		return err;
	}

	return esp_modem_set_pin(dce, CONFIG_EPHOROS_MODEM_SIM_PIN);
}

void app_main(void) {
	gpio_config_t power_config = {
		.pin_bit_mask = 1ULL << BOARD_POWER_EN,
		.mode = GPIO_MODE_OUTPUT,
		.pull_up_en = GPIO_PULLUP_DISABLE,
		.pull_down_en = GPIO_PULLDOWN_DISABLE,
		.intr_type = GPIO_INTR_DISABLE,
	};

	ESP_ERROR_CHECK(gpio_config(&power_config));
	ESP_ERROR_CHECK(gpio_set_level(BOARD_POWER_EN, 1));
	vTaskDelay(pdMS_TO_TICKS(CONFIG_EPHOROS_MODEM_POWER_ON_DELAY_MS));

#if CONFIG_EPHOROS_MODEM_PULSE_PWRKEY
	ESP_ERROR_CHECK(gpio_set_direction(MODEM_PWRKEY, GPIO_MODE_OUTPUT));
	ESP_ERROR_CHECK(gpio_set_level(MODEM_PWRKEY, 0));
	vTaskDelay(pdMS_TO_TICKS(CONFIG_EPHOROS_MODEM_PWRKEY_PULSE_MS));
	ESP_ERROR_CHECK(gpio_set_level(MODEM_PWRKEY, 1));
#endif

	ESP_ERROR_CHECK(esp_netif_init());
	ESP_ERROR_CHECK(esp_event_loop_create_default());
	ESP_ERROR_CHECK(esp_event_handler_register(
		IP_EVENT, ESP_EVENT_ANY_ID, &on_ip_event, NULL
	));

	esp_netif_config_t ppp_config = ESP_NETIF_DEFAULT_PPP();
	esp_netif_t *ppp_netif = esp_netif_new(&ppp_config);
	assert(ppp_netif != NULL);

	esp_modem_dte_config_t dte_config = ESP_MODEM_DTE_DEFAULT_CONFIG();
	dte_config.uart_config.tx_io_num = MODEM_TX_PIN;
	dte_config.uart_config.rx_io_num = MODEM_RX_PIN;
	dte_config.uart_config.baud_rate = CONFIG_EPHOROS_MODEM_UART_BAUD_RATE;
	dte_config.uart_config.flow_control = ESP_MODEM_FLOW_CONTROL_NONE;

	esp_modem_dce_config_t dce_config =
		ESP_MODEM_DCE_DEFAULT_CONFIG(CONFIG_EPHOROS_MODEM_APN);
	dce_config.context_id = CONFIG_EPHOROS_MODEM_PDP_CONTEXT_ID;
	dce_config.protocol_type = pdp_protocol_type();

#if CONFIG_EPHOROS_MODEM_DCE_SIM7600_COMPAT
	s_dce = esp_modem_new_dev(
		ESP_MODEM_DCE_SIM7600, &dte_config, &dce_config, ppp_netif
	);
#else
	s_dce = esp_modem_new(&dte_config, &dce_config, ppp_netif);
#endif

	if (s_dce == NULL) {
		ESP_LOGE(TAG, "failed to create the modem DCE");
		return;
	}

	if (esp_modem_sync(s_dce) != ESP_OK) {
		ESP_LOGE(TAG, "modem did not respond to AT; check power and UART pins");
		return;
	}

	ESP_ERROR_CHECK(unlock_sim_if_needed(s_dce));
	ESP_LOGI(TAG, "starting PPP with APN '%s'", CONFIG_EPHOROS_MODEM_APN);
	ESP_ERROR_CHECK(esp_modem_set_mode(s_dce, ESP_MODEM_MODE_DATA));
}
