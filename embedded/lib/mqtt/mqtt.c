#include <time.h>
#include <mqtt.h>

#define MAX_PAYLOAD_LENGTH 512
#define PAYLOAD_TEMPLATE "{"\
	                       "  \"value\": %.4f,"\
	                       "  \"timestamp\": %s"\
	                       "}"

ephoros_mqtt_err_t _validate_config(ephoros_mqtt_config_t* config);
char* _get_current_time();

ephoros_mqtt_err_t ephoros_mqtt_start(
	ephoros_mqtt_client_t** client,
	ephoros_mqtt_config_t*  config
) {
	const ephoros_mqtt_err_t config_err = _validate_config(config);
	if (config_err != ephoros_mqtt_err_ok) return config_err;

	const esp_mqtt_client_config_t mqtt_config = {
		.broker = {
			.address.uri = strdup(config->broker_uri)
		}
	};
	esp_mqtt_client_handle_t mqtt_client = esp_mqtt_client_init(&mqtt_config);

	const esp_err_t err = esp_mqtt_client_start(mqtt_client);
	if (err != ESP_OK) return ephoros_mqtt_err_start;

	*client = (ephoros_mqtt_client_t*)malloc(sizeof(ephoros_mqtt_client_t));
	if (!*client) return ephoros_mqtt_err_allocation;
	(*client)->client = mqtt_client;

	return ephoros_mqtt_err_ok;
}

ephoros_mqtt_err_t ephoros_mqtt_publish(
	ephoros_mqtt_client_t*  client,
	ephoros_mqtt_message_t* message
) {
	char* payload = (char*)malloc(sizeof(char)*(MAX_PAYLOAD_LENGTH+1));
	if (!payload) return ephoros_mqtt_err_allocation;

	char* now = _get_current_time();
	if (!now) return ephoros_mqtt_err_allocation;

	sprintf(payload, PAYLOAD_TEMPLATE, message->value, now);

	int id = esp_mqtt_client_publish(
		client->client,
		message->topic,
		payload,
		0, // length to be calculated
		0, // QoS
		0  // retain flag
	);
	if (id < 0) return ephoros_mqtt_err_publish;

	return ephoros_mqtt_err_ok;
}

ephoros_mqtt_err_t _validate_config(ephoros_mqtt_config_t* config) {
	if (!config->broker_uri || !config->username || !config->password)
		return ephoros_mqtt_err_invalid_config;

	return ephoros_mqtt_err_ok;
}

char* _get_current_time() {
	time_t now = time(NULL);
	struct tm now_info;

	gmtime_r(&now, &now_info);

	char* iso8691_buf = (char*)malloc(sizeof(char)*30);
	if (!iso8691_buf) return NULL;

	strftime(iso8691_buf, sizeof(iso8691_buf), "%Y-%m-%dT%H:%M:%SZ", &now_info);
	return iso8691_buf;
}
