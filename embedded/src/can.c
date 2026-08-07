#include <inttypes.h>
#include <stdlib.h>

#include "can.h"
#include "esp_log.h"
#include "esp_twai.h"
#include "esp_twai_onchip.h"
#include "freertos/FreeRTOS.h"
#include "freertos/queue.h"
#include "freertos/task.h"
#include "sdkconfig.h"

#if CONFIG_EPHOROS_CAN_VERA_DECODER
#include "vera_espidf.h"
#endif

static const char *TAG = "can";

typedef struct {
	twai_frame_header_t header;
	uint8_t data[TWAI_FRAME_MAX_LEN];
} can_rx_frame_t;

static QueueHandle_t s_rx_queue;

static bool IRAM_ATTR can_rx_callback(twai_node_handle_t node,
							  const twai_rx_done_event_data_t *event,
							  void *user_ctx) {
	(void)event;
	(void)user_ctx;

	can_rx_frame_t received = {0};
	twai_frame_t frame = {
		.buffer = received.data,
		.buffer_len = sizeof(received.data),
	};
	BaseType_t higher_priority_task_woken = pdFALSE;

	if (twai_node_receive_from_isr(node, &frame) != ESP_OK) {
		return false;
	}

	received.header = frame.header;
	xQueueSendFromISR(s_rx_queue, &received, &higher_priority_task_woken);
	return higher_priority_task_woken == pdTRUE;
}

static void process_frame(const can_rx_frame_t *received) {
#if CONFIG_EPHOROS_CAN_LOG_RECEIVED_FRAMES
	ESP_LOGI(TAG, "RX %s ID=0x%03" PRIx32 " DLC=%u",
			 received->header.ide ? "extended" : "standard", received->header.id,
			 received->header.dlc);
#endif

#if CONFIG_EPHOROS_CAN_VERA_DECODER
	twai_frame_t frame = {
		.header = received->header,
		.buffer = (uint8_t *)received->data,
		.buffer_len = sizeof(received->data),
	};
	vera_decoding_result_t decoded = {0};
	vera_err_t err = vera_decode_espidf_rx_frame(&frame, &decoded);
	if (err != vera_err_ok) {
		ESP_LOGD(TAG, "Vera does not decode CAN ID 0x%03" PRIx32,
				 frame.header.id);
		return;
	}

	for (uint8_t i = 0; i < decoded.n_signals; ++i) {
		const vera_decoded_signal_t *signal = &decoded.decoded_signals[i];
		ESP_LOGI(TAG, "%s = %.2f %s", signal->name, signal->value,
				 signal->unit);
	}
	free(decoded.decoded_signals);
#endif
}

static void can_receive_task(void *arg) {
	(void)arg;
	can_rx_frame_t received;

	for (;;) {
		if (xQueueReceive(s_rx_queue, &received, portMAX_DELAY) == pdTRUE) {
			process_frame(&received);
		}
	}
}

esp_err_t can_start(void) {
	if (s_rx_queue != NULL) {
		return ESP_ERR_INVALID_STATE;
	}

	s_rx_queue = xQueueCreate(CONFIG_EPHOROS_CAN_RX_QUEUE_DEPTH,
						 sizeof(can_rx_frame_t));
	if (s_rx_queue == NULL) {
		return ESP_ERR_NO_MEM;
	}

	twai_onchip_node_config_t config = {
		.io_cfg = {
			.tx = CONFIG_EPHOROS_CAN_TX_GPIO,
			.rx = CONFIG_EPHOROS_CAN_RX_GPIO,
			.quanta_clk_out = GPIO_NUM_NC,
			.bus_off_indicator = GPIO_NUM_NC,
		},
		.bit_timing.bitrate = CONFIG_EPHOROS_CAN_BITRATE,
		.flags.enable_listen_only = CONFIG_EPHOROS_CAN_LISTEN_ONLY,
	};
	twai_node_handle_t node = NULL;
	esp_err_t err = twai_new_node_onchip(&config, &node);
	if (err != ESP_OK) {
		vQueueDelete(s_rx_queue);
		s_rx_queue = NULL;
		return err;
	}

	const twai_event_callbacks_t callbacks = {
		.on_rx_done = can_rx_callback,
	};
	err = twai_node_register_event_callbacks(node, &callbacks, NULL);
	if (err != ESP_OK) {
		twai_node_delete(node);
		vQueueDelete(s_rx_queue);
		s_rx_queue = NULL;
		return err;
	}

	TaskHandle_t task = NULL;
	if (xTaskCreate(can_receive_task, "can_rx",
				CONFIG_EPHOROS_CAN_TASK_STACK_SIZE, NULL,
				CONFIG_EPHOROS_CAN_TASK_PRIORITY, &task) != pdPASS) {
		twai_node_delete(node);
		vQueueDelete(s_rx_queue);
		s_rx_queue = NULL;
		return ESP_ERR_NO_MEM;
	}

	err = twai_node_enable(node);
	if (err != ESP_OK) {
		vTaskDelete(task);
		twai_node_delete(node);
		vQueueDelete(s_rx_queue);
		s_rx_queue = NULL;
		return err;
	}

	ESP_LOGI(TAG, "receiving CAN at %d bit/s on RX GPIO %d%s",
			 CONFIG_EPHOROS_CAN_BITRATE, CONFIG_EPHOROS_CAN_RX_GPIO,
			 CONFIG_EPHOROS_CAN_LISTEN_ONLY ? " (listen-only)" : "");
	return ESP_OK;
}
