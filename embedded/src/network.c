#include <string.h>

#include "board_config.h"
#include "esp_check.h"
#include "esp_event.h"
#include "esp_log.h"
#include "esp_netif.h"
#include "nvs_flash.h"
#include "sdkconfig.h"

#if CONFIG_EPHOROS_NETWORK_CELLULAR
#include "esp_modem_api.h"
#include "esp_netif_ppp.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#else
#include "esp_wifi.h"
#endif

static const char *TAG = "network";

#if CONFIG_EPHOROS_NETWORK_CELLULAR

static esp_modem_dce_t *s_dce;

static void on_ip_event(void *arg, esp_event_base_t event_base,
						int32_t event_id, void *event_data) {
	(void)arg;
	(void)event_base;
	(void)event_data;
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

static esp_err_t start_cellular(void) {
	gpio_config_t power_config = {
		.pin_bit_mask = 1ULL << BOARD_POWER_EN,
		.mode = GPIO_MODE_OUTPUT,
		.pull_up_en = GPIO_PULLUP_DISABLE,
		.pull_down_en = GPIO_PULLDOWN_DISABLE,
		.intr_type = GPIO_INTR_DISABLE,
	};

	ESP_RETURN_ON_ERROR(gpio_config(&power_config), TAG, "configure modem power");
	ESP_RETURN_ON_ERROR(gpio_set_level(BOARD_POWER_EN, 1), TAG,
						"enable modem power");
	vTaskDelay(pdMS_TO_TICKS(CONFIG_EPHOROS_MODEM_POWER_ON_DELAY_MS));

#if CONFIG_EPHOROS_MODEM_PULSE_PWRKEY
	ESP_RETURN_ON_ERROR(gpio_set_direction(MODEM_PWRKEY, GPIO_MODE_OUTPUT), TAG,
						"configure modem PWRKEY");
	ESP_RETURN_ON_ERROR(gpio_set_level(MODEM_PWRKEY, 0), TAG,
						"assert modem PWRKEY");
	vTaskDelay(pdMS_TO_TICKS(CONFIG_EPHOROS_MODEM_PWRKEY_PULSE_MS));
	ESP_RETURN_ON_ERROR(gpio_set_level(MODEM_PWRKEY, 1), TAG,
						"release modem PWRKEY");
#endif

	ESP_ERROR_CHECK(esp_event_handler_register(
		IP_EVENT, ESP_EVENT_ANY_ID, &on_ip_event, NULL));

	esp_netif_config_t ppp_config = ESP_NETIF_DEFAULT_PPP();
	esp_netif_t *ppp_netif = esp_netif_new(&ppp_config);
	if (ppp_netif == NULL) {
		ESP_LOGE(TAG, "failed to create PPP network interface");
		return ESP_ERR_NO_MEM;
	}

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
		return ESP_FAIL;
	}

	if (esp_modem_sync(s_dce) != ESP_OK) {
		ESP_LOGE(TAG, "modem did not respond to AT; check power and UART pins");
		return ESP_ERR_TIMEOUT;
	}

	ESP_RETURN_ON_ERROR(unlock_sim_if_needed(s_dce), TAG, "unlock SIM");
	ESP_LOGI(TAG, "starting PPP with APN '%s'", CONFIG_EPHOROS_MODEM_APN);
	return esp_modem_set_mode(s_dce, ESP_MODEM_MODE_DATA);
}

#else

static void on_wifi_event(void *arg, esp_event_base_t event_base,
						  int32_t event_id, void *event_data) {
	(void)arg;
	if (event_base == WIFI_EVENT && event_id == WIFI_EVENT_STA_START) {
		ESP_ERROR_CHECK(esp_wifi_connect());
	} else if (event_base == WIFI_EVENT && event_id == WIFI_EVENT_STA_DISCONNECTED) {
		ESP_LOGW(TAG, "Wi-Fi disconnected; retrying");
		ESP_ERROR_CHECK(esp_wifi_connect());
	} else if (event_base == IP_EVENT && event_id == IP_EVENT_STA_GOT_IP) {
		const ip_event_got_ip_t *event = (const ip_event_got_ip_t *)event_data;
		ESP_LOGI(TAG, "Wi-Fi connected; IP address: " IPSTR,
				 IP2STR(&event->ip_info.ip));
	}
}

static esp_err_t start_wifi(void) {
	esp_netif_create_default_wifi_sta();
	wifi_init_config_t wifi_config = WIFI_INIT_CONFIG_DEFAULT();
	ESP_RETURN_ON_ERROR(esp_wifi_init(&wifi_config), TAG, "initialize Wi-Fi");
	ESP_RETURN_ON_ERROR(esp_event_handler_register(
		WIFI_EVENT, ESP_EVENT_ANY_ID, &on_wifi_event, NULL), TAG,
		"register Wi-Fi event handler");
	ESP_RETURN_ON_ERROR(esp_event_handler_register(
		IP_EVENT, IP_EVENT_STA_GOT_IP, &on_wifi_event, NULL), TAG,
		"register IP event handler");

	wifi_config_t station_config = {
		.sta = {
			.ssid = CONFIG_EPHOROS_WIFI_SSID,
			.password = CONFIG_EPHOROS_WIFI_PASSWORD,
			.threshold.authmode = WIFI_AUTH_WPA2_PSK,
		},
	};
	ESP_RETURN_ON_ERROR(esp_wifi_set_mode(WIFI_MODE_STA), TAG,
						"set Wi-Fi station mode");
	ESP_RETURN_ON_ERROR(esp_wifi_set_config(WIFI_IF_STA, &station_config), TAG,
						"configure Wi-Fi station");
	ESP_RETURN_ON_ERROR(esp_wifi_start(), TAG, "start Wi-Fi");
	ESP_LOGI(TAG, "starting Wi-Fi station with SSID '%s'", CONFIG_EPHOROS_WIFI_SSID);
	return ESP_OK;
}

#endif

esp_err_t network_start(void) {
	esp_err_t err = nvs_flash_init();
	if (err == ESP_ERR_NVS_NO_FREE_PAGES ||
		err == ESP_ERR_NVS_NEW_VERSION_FOUND) {
		ESP_RETURN_ON_ERROR(nvs_flash_erase(), TAG, "erase incompatible NVS");
		err = nvs_flash_init();
	}
	if (err != ESP_OK) {
		return err;
	}

	err = esp_netif_init();
	if (err != ESP_OK && err != ESP_ERR_INVALID_STATE) {
		return err;
	}

	err = esp_event_loop_create_default();
	if (err != ESP_OK && err != ESP_ERR_INVALID_STATE) {
		return err;
	}

#if CONFIG_EPHOROS_NETWORK_CELLULAR
	return start_cellular();
#else
	return start_wifi();
#endif
}
