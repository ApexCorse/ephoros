#include <inttypes.h>
#include <stdlib.h>
#include <string.h>

#include "can.h"
#include "esp_log.h"
#include "sdkconfig.h"
#if CONFIG_EPHOROS_CAN_ENABLED
#include "esp_twai.h"
#include "esp_twai_onchip.h"
#include "freertos/FreeRTOS.h"
#include "freertos/queue.h"
#include "freertos/task.h"
#endif

#if CONFIG_EPHOROS_CAN_SIMULATOR
#include "telemetry_simulator.h"
#endif

#if CONFIG_EPHOROS_CAN_VERA_DECODER
#include "vera_espidf.h"
#endif

static const char *TAG = "can";

#if CONFIG_EPHOROS_CAN_ENABLED
typedef struct {
	twai_frame_header_t header;
	uint8_t data[TWAI_FRAME_MAX_LEN];
} can_rx_frame_t;

static QueueHandle_t s_rx_queue;
static QueueHandle_t s_decoded_queue;

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
		can_decoded_signal_t event = {
			.can_id = frame.header.id,
			.value = signal->value,
			.timestamp = received->header.timestamp,
		};
		memcpy(event.name, signal->name, sizeof(event.name));
		memcpy(event.unit, signal->unit, sizeof(event.unit));
		memcpy(event.topic, signal->topic, sizeof(event.topic));
		event.name[sizeof(event.name) - 1] = '\0';
		event.unit[sizeof(event.unit) - 1] = '\0';
		event.topic[sizeof(event.topic) - 1] = '\0';

		if (xQueueSend(s_decoded_queue, &event, 0) != pdTRUE) {
			ESP_LOGD(TAG, "decoded-signal queue full; dropping %s", event.name);
		}
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
#endif

esp_err_t can_start(void) {
#if CONFIG_EPHOROS_CAN_SIMULATOR
	return telemetry_simulator_start();
#elif !CONFIG_EPHOROS_CAN_ENABLED
	ESP_LOGI(TAG, "CAN receiver disabled");
	return ESP_OK;
#else
	if (s_rx_queue != NULL) {
		return ESP_ERR_INVALID_STATE;
	}

	s_rx_queue = xQueueCreate(CONFIG_EPHOROS_CAN_RX_QUEUE_DEPTH,
						 sizeof(can_rx_frame_t));
	if (s_rx_queue == NULL) {
		return ESP_ERR_NO_MEM;
	}
	s_decoded_queue = xQueueCreate(CONFIG_EPHOROS_CAN_DECODED_QUEUE_DEPTH,
							  sizeof(can_decoded_signal_t));
	if (s_decoded_queue == NULL) {
		vQueueDelete(s_rx_queue);
		s_rx_queue = NULL;
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
		.tx_queue_depth = CONFIG_EPHOROS_CAN_LISTEN_ONLY == 0 ? 1 : 0,
	};
	twai_node_handle_t node = NULL;
	esp_err_t err = twai_new_node_onchip(&config, &node);
	if (err != ESP_OK) {
		vQueueDelete(s_decoded_queue);
		s_decoded_queue = NULL;
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
		vQueueDelete(s_decoded_queue);
		s_decoded_queue = NULL;
		vQueueDelete(s_rx_queue);
		s_rx_queue = NULL;
		return err;
	}

	TaskHandle_t task = NULL;
	if (xTaskCreate(can_receive_task, "can_rx",
				CONFIG_EPHOROS_CAN_TASK_STACK_SIZE, NULL,
				CONFIG_EPHOROS_CAN_TASK_PRIORITY, &task) != pdPASS) {
		twai_node_delete(node);
		vQueueDelete(s_decoded_queue);
		s_decoded_queue = NULL;
		vQueueDelete(s_rx_queue);
		s_rx_queue = NULL;
		return ESP_ERR_NO_MEM;
	}

	err = twai_node_enable(node);
	if (err != ESP_OK) {
		vTaskDelete(task);
		twai_node_delete(node);
		vQueueDelete(s_decoded_queue);
		s_decoded_queue = NULL;
		vQueueDelete(s_rx_queue);
		s_rx_queue = NULL;
		return err;
	}

	ESP_LOGI(TAG, "receiving CAN at %d bit/s on RX GPIO %d%s",
			 CONFIG_EPHOROS_CAN_BITRATE, CONFIG_EPHOROS_CAN_RX_GPIO,
			 CONFIG_EPHOROS_CAN_LISTEN_ONLY ? " (listen-only)" : "");
	return ESP_OK;
#endif
}

bool can_receive_decoded_signal(can_decoded_signal_t *signal,
							 TickType_t timeout) {
#if CONFIG_EPHOROS_CAN_SIMULATOR
	return telemetry_simulator_next(signal, timeout);
#elif !CONFIG_EPHOROS_CAN_ENABLED
	(void)signal;
	(void)timeout;
	return false;
#else
	if (signal == NULL || s_decoded_queue == NULL) {
		return false;
	}

	return xQueueReceive(s_decoded_queue, signal, timeout) == pdTRUE;
#endif
}
