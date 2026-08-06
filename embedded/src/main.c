#include "esp_modem_config.h"
#include "esp_modem_dce_config.h"
#include <assert.h>
#include <driver/gpio.h>
#include <freertos/FreeRTOS.h>
#include <freertos/task.h>
#include <esp_netif.h>
#include <esp_event_loop.h>
#include <esp_modem_api.h>

#define BOARD_POWER_EN GPIO_NUM_12
#define MODEM_TX_PIN   GPIO_NUM_26
#define MODEM_RX_PIN   GPIO_NUM_27
#define MODEM_PWRKEY   GPIO_NUM_4
#define MODEM_RESET    GPIO_NUM_5
#define MODEM_DTR      GPIO_NUM_25

void app_main() {
	gpio_config_t power_config = {
		.pin_bit_mask = 1ULL << BOARD_POWER_EN,
		.mode = GPIO_MODE_OUTPUT,
		.pull_up_en = GPIO_PULLUP_DISABLE,
		.pull_down_en = GPIO_PULLDOWN_DISABLE,
		.intr_type = GPIO_INTR_DISABLE,
	};

	ESP_ERROR_CHECK(gpio_config(&power_config));
	ESP_ERROR_CHECK(gpio_set_level(BOARD_POWER_EN, 1));
	vTaskDelay(pdMS_TO_TICKS(500));

	// Check if modem is already live at this point,
	// if not, uncomment
	// gpio_set_direction(MODEM_PWRKEY, GPIO_MODE_OUTPUT);
	// gpio_set_level(MODEM_PWRKEY, 0);
	// vTaskDelay(pdMS_TO_TICKS(1000));
	// gpio_set_level(MODEM_PWRKEY, 1);

	ESP_ERROR_CHECK(esp_netif_init());
	ESP_ERROR_CHECK(esp_event_loop_create_default());

	esp_netif_config_t ppp_config = ESP_NETIF_DEFAULT_PPP();
	esp_netif_t *ppp_netif = esp_netif_new(&ppp_config);
	assert(ppp_netif != NULL);

	esp_modem_dte_config_t dte_config = ESP_MODEM_DTE_DEFAULT_CONFIG();
	dte_config.uart_config.tx_io_num = MODEM_TX_PIN;
	dte_config.uart_config.rx_io_num = MODEM_RX_PIN;
	dte_config.uart_config.baud_rate = 115200; 
	// Maybe change this
	dte_config.uart_config.flow_control = ESP_MODEM_FLOW_CONTROL_NONE;

	esp_modem_dce_config_t dce_config = ESP_MODEM_DCE_DEFAULT_CONFIG("APN"); //TODO: change APN
	
	//TODO: configure this
	// esp_modem_dce_t *dce = esp_modem_new_dev(
	//    ESP_MODEM_DCE_CUSTOM,
	//    &dte_config,
	//    &dce_config,
	//    ppp_netif
	// );
}
