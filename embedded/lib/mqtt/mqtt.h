#ifndef EPHOROSMQTT_H
#define EPHOROSMQTT_H

#include <stdbool.h>

#include "mqtt_client.h"

typedef enum {
	ephoros_mqtt_err_ok,
	ephoros_mqtt_err_invalid_config,
	ephoros_mqtt_err_allocation,
	ephoros_mqtt_err_start,
	ephoros_mqtt_err_not_connected,
	ephoros_mqtt_err_publish
} ephoros_mqtt_err_t;

typedef struct {
	esp_mqtt_client_handle_t client;
	volatile bool connected;
} ephoros_mqtt_client_t;

typedef struct {
	const char* broker_uri;
	const char* username;
	const char* password;
} ephoros_mqtt_config_t;

typedef struct {
	const char* topic;
	double      value;
} ephoros_mqtt_message_t;

ephoros_mqtt_err_t ephoros_mqtt_start(
	ephoros_mqtt_client_t** client,
	const ephoros_mqtt_config_t* config
);
ephoros_mqtt_err_t ephoros_mqtt_publish(
	ephoros_mqtt_client_t* client,
	const ephoros_mqtt_message_t* message
);
void ephoros_mqtt_stop(ephoros_mqtt_client_t* client);

#endif // EPHOROSMQTT_H
