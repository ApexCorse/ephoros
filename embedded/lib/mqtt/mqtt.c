#include "mqtt.h"

#include <math.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#include "esp_event.h"

#define MQTT_PAYLOAD_LENGTH 128
#define MQTT_VALID_EPOCH 1704067200LL /* 2024-01-01T00:00:00Z */

static ephoros_mqtt_err_t validate_config(const ephoros_mqtt_config_t *config) {
	if (config == NULL || config->broker_uri == NULL ||
		config->broker_uri[0] == '\0' || config->username == NULL ||
		config->password == NULL ||
		strncmp(config->broker_uri, "mqtt://", strlen("mqtt://")) != 0 ||
		strchr(config->broker_uri, '<') != NULL) {
		return ephoros_mqtt_err_invalid_config;
	}

	return ephoros_mqtt_err_ok;
}

static void mqtt_event_handler(void *handler_args, esp_event_base_t base,
						   int32_t event_id, void *event_data) {
	(void)base;
	(void)event_data;
	ephoros_mqtt_client_t *client = handler_args;
	if (client == NULL) {
		return;
	}

	if (event_id == MQTT_EVENT_CONNECTED) {
		client->connected = true;
	} else if (event_id == MQTT_EVENT_DISCONNECTED) {
		client->connected = false;
	}
}

ephoros_mqtt_err_t ephoros_mqtt_start(
	ephoros_mqtt_client_t **client,
	const ephoros_mqtt_config_t *config
) {
	if (client == NULL) {
		return ephoros_mqtt_err_invalid_config;
	}
	*client = NULL;

	const ephoros_mqtt_err_t config_err = validate_config(config);
	if (config_err != ephoros_mqtt_err_ok) {
		return config_err;
	}

	ephoros_mqtt_client_t *wrapper = calloc(1, sizeof(*wrapper));
	if (wrapper == NULL) {
		return ephoros_mqtt_err_allocation;
	}

	const esp_mqtt_client_config_t mqtt_config = {
		.broker.address.uri = config->broker_uri,
		.credentials.username = config->username,
		.credentials.authentication.password = config->password,
	};
	wrapper->client = esp_mqtt_client_init(&mqtt_config);
	if (wrapper->client == NULL) {
		free(wrapper);
		return ephoros_mqtt_err_allocation;
	}

	if (esp_mqtt_client_register_event(wrapper->client, ESP_EVENT_ANY_ID,
			mqtt_event_handler, wrapper) != ESP_OK) {
		esp_mqtt_client_destroy(wrapper->client);
		free(wrapper);
		return ephoros_mqtt_err_start;
	}
	if (esp_mqtt_client_start(wrapper->client) != ESP_OK) {
		esp_mqtt_client_destroy(wrapper->client);
		free(wrapper);
		return ephoros_mqtt_err_start;
	}

	*client = wrapper;
	return ephoros_mqtt_err_ok;
}

void ephoros_mqtt_stop(ephoros_mqtt_client_t *client) {
	if (client == NULL) {
		return;
	}
	if (client->client != NULL) {
		(void)esp_mqtt_client_stop(client->client);
		esp_mqtt_client_destroy(client->client);
	}
	free(client);
}

ephoros_mqtt_err_t ephoros_mqtt_publish(
	ephoros_mqtt_client_t *client,
	const ephoros_mqtt_message_t *message
) {
	if (client == NULL || client->client == NULL || message == NULL ||
		message->topic == NULL || message->topic[0] == '\0' ||
		!isfinite(message->value)) {
		return ephoros_mqtt_err_invalid_config;
	}
	if (!client->connected) {
		return ephoros_mqtt_err_not_connected;
	}

	char payload[MQTT_PAYLOAD_LENGTH];
	time_t now = time(NULL);
	int written;
	if (now >= MQTT_VALID_EPOCH) {
		struct tm utc;
		char timestamp[sizeof("2024-01-01T00:00:00Z")];
		if (gmtime_r(&now, &utc) == NULL ||
			strftime(timestamp, sizeof(timestamp), "%Y-%m-%dT%H:%M:%SZ", &utc) == 0) {
			return ephoros_mqtt_err_publish;
		}
		written = snprintf(payload, sizeof(payload),
			"{\"value\":%.17g,\"timestamp\":\"%s\"}", message->value, timestamp);
	} else {
		written = snprintf(payload, sizeof(payload), "{\"value\":%.17g}",
			message->value);
	}
	if (written < 0 || written >= (int)sizeof(payload)) {
		return ephoros_mqtt_err_publish;
	}

	return esp_mqtt_client_publish(client->client, message->topic, payload, 0,
			0, 0) < 0 ? ephoros_mqtt_err_publish : ephoros_mqtt_err_ok;
}
