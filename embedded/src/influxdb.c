#include "influxdb.h"

#include <inttypes.h>
#include <math.h>
#include <stdio.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>

#include "can.h"
#include "esp_crt_bundle.h"
#include "esp_http_client.h"
#include "esp_log.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "sdkconfig.h"

#define INFLUXDB_DEFAULT_TIMEOUT_MS 10000
#define INFLUXDB_LINE_BUFFER_SIZE 512
#define INFLUXDB_TASK_STACK_SIZE 4096
#define INFLUXDB_TASK_PRIORITY 4

static const char *TAG = "influxdb";
static influxdb_client_t s_client;
static TaskHandle_t s_task;

static bool valid_config(const influxdb_config_t *config) {
	return config != NULL && config->write_url != NULL &&
		config->write_url[0] != '\0' && config->token != NULL &&
		config->token[0] != '\0';
}

influxdb_config_t influxdb_config_from_kconfig(void) {
	return (influxdb_config_t){
		.write_url = CONFIG_EPHOROS_INFLUXDB_WRITE_URL,
		.token = CONFIG_EPHOROS_INFLUXDB_TOKEN,
		.timeout_ms = CONFIG_EPHOROS_INFLUXDB_TIMEOUT_MS,
	};
}

static void influxdb_task(void *arg) {
	const influxdb_client_t *client = arg;
	can_decoded_signal_t signal;

	for (;;) {
		if (!can_receive_decoded_signal(&signal, portMAX_DELAY)) {
			continue;
		}

		/* TWAI timestamps are not Unix timestamps, so let InfluxDB assign it. */
		esp_err_t err = influxdb_write_can_signal(client, &signal);
		if (err != ESP_OK) {
			ESP_LOGW(TAG, "failed to write CAN signal %s: %s", signal.name,
					 esp_err_to_name(err));
		}
	}
}

esp_err_t influxdb_start(void) {
	if (s_task != NULL) {
		return ESP_ERR_INVALID_STATE;
	}

	influxdb_config_t config = influxdb_config_from_kconfig();
	esp_err_t err = influxdb_client_init(&s_client, &config);
	if (err != ESP_OK) {
		return err;
	}

	if (xTaskCreate(influxdb_task, "influxdb", INFLUXDB_TASK_STACK_SIZE,
					&s_client, INFLUXDB_TASK_PRIORITY, &s_task) != pdPASS) {
		return ESP_ERR_NO_MEM;
	}

	ESP_LOGI(TAG, "InfluxDB telemetry task started");
	return ESP_OK;
}

esp_err_t influxdb_client_init(
	influxdb_client_t *client,
	const influxdb_config_t *config
) {
	if (client == NULL || !valid_config(config)) {
		return ESP_ERR_INVALID_ARG;
	}

	client->config = *config;
	if (client->config.timeout_ms <= 0) {
		client->config.timeout_ms = INFLUXDB_DEFAULT_TIMEOUT_MS;
	}

	return ESP_OK;
}

esp_err_t influxdb_write_line(
	const influxdb_client_t *client,
	const char *line_protocol
) {
	if (client == NULL || !valid_config(&client->config) ||
		line_protocol == NULL || line_protocol[0] == '\0') {
		return ESP_ERR_INVALID_ARG;
	}

	const size_t authorization_size = strlen(client->config.token) +
		strlen("Token ") + 1;
	char *authorization = malloc(authorization_size);
	if (authorization == NULL) {
		return ESP_ERR_NO_MEM;
	}
	snprintf(authorization, authorization_size, "Token %s", client->config.token);

	esp_http_client_config_t http_config = {
		.url = client->config.write_url,
		.timeout_ms = client->config.timeout_ms,
		.crt_bundle_attach = esp_crt_bundle_attach,
	};
	esp_http_client_handle_t http = esp_http_client_init(&http_config);
	if (http == NULL) {
		free(authorization);
		return ESP_ERR_NO_MEM;
	}

	esp_err_t err = esp_http_client_set_method(http, HTTP_METHOD_POST);
	if (err == ESP_OK) {
		err = esp_http_client_set_header(http, "Authorization", authorization);
	}
	if (err == ESP_OK) {
		err = esp_http_client_set_header(http, "Content-Type", "text/plain; charset=utf-8");
	}
	if (err == ESP_OK) {
		err = esp_http_client_set_post_field(http, line_protocol, strlen(line_protocol));
	}
	if (err == ESP_OK) {
		err = esp_http_client_perform(http);
	}
	if (err == ESP_OK && esp_http_client_get_status_code(http) != 204) {
		err = ESP_FAIL;
	}

	esp_http_client_cleanup(http);
	free(authorization);
	return err;
}

esp_err_t influxdb_write_number(
	const influxdb_client_t *client,
	const char *measurement,
	const char *tag_key,
	const char *tag_value,
	const char *field_key,
	double value,
	int64_t timestamp_ns
) {
	if (measurement == NULL || tag_key == NULL || tag_value == NULL ||
		field_key == NULL || !isfinite(value)) {
		return ESP_ERR_INVALID_ARG;
	}

	char line_protocol[INFLUXDB_LINE_BUFFER_SIZE];
	int written;
	if (timestamp_ns == 0) {
		written = snprintf(line_protocol, sizeof(line_protocol), "%s,%s=%s %s=%.17g",
			measurement, tag_key, tag_value, field_key, value);
	} else {
		written = snprintf(line_protocol, sizeof(line_protocol),
			"%s,%s=%s %s=%.17g %" PRId64,
			measurement, tag_key, tag_value, field_key, value, timestamp_ns);
	}

	if (written < 0 || written >= (int)sizeof(line_protocol)) {
		return ESP_ERR_INVALID_SIZE;
	}

	return influxdb_write_line(client, line_protocol);
}

static bool escape_tag_value(char *destination, size_t destination_size,
					 const char *value) {
	if (destination == NULL || destination_size == 0 || value == NULL) {
		return false;
	}

	size_t written = 0;
	for (const char *current = value; *current != '\0'; ++current) {
		if (*current == ',' || *current == '=' || *current == ' ') {
			if (written + 1 >= destination_size) {
				return false;
			}
			destination[written++] = '\\';
		}
		if (written + 1 >= destination_size) {
			return false;
		}
		destination[written++] = *current;
	}
	destination[written] = '\0';
	return true;
}

esp_err_t influxdb_write_can_signal(
	const influxdb_client_t *client,
	const can_decoded_signal_t *signal
) {
	if (signal == NULL || !isfinite(signal->value)) {
		return ESP_ERR_INVALID_ARG;
	}

	char topic[CAN_DECODED_SIGNAL_TOPIC_MAX_LEN * 2];
	char name[CAN_DECODED_SIGNAL_NAME_MAX_LEN * 2];
	char unit[CAN_DECODED_SIGNAL_UNIT_MAX_LEN * 2];
	if (!escape_tag_value(topic, sizeof(topic), signal->topic) ||
		!escape_tag_value(name, sizeof(name), signal->name) ||
		!escape_tag_value(unit, sizeof(unit), signal->unit)) {
		return ESP_ERR_INVALID_SIZE;
	}

	char line_protocol[INFLUXDB_LINE_BUFFER_SIZE];
	int written = snprintf(line_protocol, sizeof(line_protocol),
		"can_signal,topic=%s,name=%s,unit=%s value=%.17g", topic, name, unit,
		signal->value);
	if (written < 0 || written >= (int)sizeof(line_protocol)) {
		return ESP_ERR_INVALID_SIZE;
	}

	return influxdb_write_line(client, line_protocol);
}
