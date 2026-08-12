#include "telemetry.h"

#include "can.h"
#include "esp_check.h"
#include "esp_log.h"
#include "freertos/FreeRTOS.h"
#include "freertos/queue.h"
#include "freertos/task.h"
#include "influxdb.h"
#include "sdkconfig.h"

#if CONFIG_EPHOROS_MQTT_ENABLED
#include "mqtt.h"
#endif

#define TELEMETRY_TASK_STACK_SIZE 4096
#define TELEMETRY_TASK_PRIORITY 4
#define TELEMETRY_MQTT_RETRY_DELAY_MS 1000
#define TELEMETRY_DROP_LOG_INTERVAL 100

static const char *TAG = "telemetry";
static QueueHandle_t s_influx_queue;
static TaskHandle_t s_dispatch_task;

#if CONFIG_EPHOROS_MQTT_ENABLED
static QueueHandle_t s_mqtt_queue;
static ephoros_mqtt_client_t *s_mqtt_client;
#endif

static void log_drop(const char *sink, unsigned long drops) {
	if (drops == 1 || drops % TELEMETRY_DROP_LOG_INTERVAL == 0) {
		ESP_LOGW(TAG, "%s queue full; dropped %lu signal%s", sink, drops,
				drops == 1 ? "" : "s");
	}
}

static void influx_task(void *arg) {
	(void)arg;
	can_decoded_signal_t signal;
	for (;;) {
		if (xQueueReceive(s_influx_queue, &signal, portMAX_DELAY) != pdTRUE) {
			continue;
		}
		esp_err_t err = influxdb_write_signal(&signal);
		if (err != ESP_OK) {
			ESP_LOGW(TAG, "failed to write InfluxDB signal %s: %s", signal.name,
					esp_err_to_name(err));
		}
	}
}

#if CONFIG_EPHOROS_MQTT_ENABLED
static void mqtt_task(void *arg) {
	(void)arg;
	can_decoded_signal_t signal;
	bool pending = false;

	for (;;) {
		if (!pending && xQueueReceive(s_mqtt_queue, &signal, portMAX_DELAY) != pdTRUE) {
			continue;
		}
		pending = true;

		const ephoros_mqtt_message_t message = {
			.topic = signal.topic,
			.value = signal.value,
		};
		ephoros_mqtt_err_t err = ephoros_mqtt_publish(s_mqtt_client, &message);
		if (err == ephoros_mqtt_err_ok) {
			pending = false;
			continue;
		}
		if (err != ephoros_mqtt_err_not_connected) {
			ESP_LOGW(TAG, "failed to publish MQTT signal %s: %d", signal.name, err);
		}
		vTaskDelay(pdMS_TO_TICKS(TELEMETRY_MQTT_RETRY_DELAY_MS));
	}
}
#endif

static void telemetry_dispatch_task(void *arg) {
	(void)arg;
	can_decoded_signal_t signal;
	unsigned long influx_drops = 0;
#if CONFIG_EPHOROS_MQTT_ENABLED
	unsigned long mqtt_drops = 0;
#endif

	for (;;) {
		if (!can_receive_decoded_signal(&signal, portMAX_DELAY)) {
			continue;
		}
		if (xQueueSend(s_influx_queue, &signal, 0) != pdTRUE) {
			log_drop("InfluxDB", ++influx_drops);
		}
#if CONFIG_EPHOROS_MQTT_ENABLED
		if (s_mqtt_queue != NULL && xQueueSend(s_mqtt_queue, &signal, 0) != pdTRUE) {
			log_drop("MQTT", ++mqtt_drops);
		}
#endif
	}
}

esp_err_t telemetry_start(void) {
	if (s_dispatch_task != NULL) {
		return ESP_ERR_INVALID_STATE;
	}
	ESP_RETURN_ON_ERROR(influxdb_start(), TAG, "initialize InfluxDB");

	s_influx_queue = xQueueCreate(CONFIG_EPHOROS_INFLUXDB_QUEUE_DEPTH,
		sizeof(can_decoded_signal_t));
	if (s_influx_queue == NULL) {
		return ESP_ERR_NO_MEM;
	}
	if (xTaskCreate(influx_task, "influxdb", TELEMETRY_TASK_STACK_SIZE, NULL,
			TELEMETRY_TASK_PRIORITY, NULL) != pdPASS) {
		vQueueDelete(s_influx_queue);
		s_influx_queue = NULL;
		return ESP_ERR_NO_MEM;
	}

#if CONFIG_EPHOROS_MQTT_ENABLED
	const ephoros_mqtt_config_t mqtt_config = {
		.broker_uri = CONFIG_EPHOROS_MQTT_BROKER_URI,
		.username = CONFIG_EPHOROS_MQTT_USERNAME,
		.password = CONFIG_EPHOROS_MQTT_PASSWORD,
	};
	s_mqtt_queue = xQueueCreate(CONFIG_EPHOROS_MQTT_QUEUE_DEPTH,
		sizeof(can_decoded_signal_t));
	if (s_mqtt_queue == NULL) {
		ESP_LOGW(TAG, "MQTT disabled: could not allocate its telemetry queue");
	} else {
		ephoros_mqtt_err_t mqtt_err = ephoros_mqtt_start(&s_mqtt_client, &mqtt_config);
		if (mqtt_err != ephoros_mqtt_err_ok) {
			ESP_LOGW(TAG, "MQTT disabled after startup failure: %d", mqtt_err);
			vQueueDelete(s_mqtt_queue);
			s_mqtt_queue = NULL;
		} else if (xTaskCreate(mqtt_task, "mqtt", TELEMETRY_TASK_STACK_SIZE,
				NULL, TELEMETRY_TASK_PRIORITY, NULL) != pdPASS) {
			ESP_LOGW(TAG, "MQTT disabled: could not allocate its telemetry worker");
			ephoros_mqtt_stop(s_mqtt_client);
			s_mqtt_client = NULL;
			vQueueDelete(s_mqtt_queue);
			s_mqtt_queue = NULL;
		}
	}
#endif

	if (xTaskCreate(telemetry_dispatch_task, "telemetry", TELEMETRY_TASK_STACK_SIZE,
			NULL, TELEMETRY_TASK_PRIORITY, &s_dispatch_task) != pdPASS) {
		return ESP_ERR_NO_MEM;
	}
	ESP_LOGI(TAG, "telemetry fan-out started");
	return ESP_OK;
}
