#ifndef EPHOROSMQTT_H
#define EPHOROSMQTT_H

#include "mqtt_client.h"

typedef enum {
	ephoros_mqtt_err_ok,
	ephoros_mqtt_err_invalid_config,
	ephoros_mqtt_err_allocation,
	ephoros_mqtt_err_start,
	ephoros_mqtt_err_publish
} ephoros_mqtt_err_t;

typedef struct {
	esp_mqtt_client_handle_t client;
} ephoros_mqtt_client_t;

typedef struct {
	const char* broker_uri;
	const char* username;
	const char* password;
} ephoros_mqtt_config_t;

typedef struct {
	float 		value;
	uint64_t 	timestamp;
} ephoros_mqtt_record_t;

typedef struct {
	const char* 						topic;
	ephoros_mqtt_record_t* 	record;
} ephoros_mqtt_message_t;

ephoros_mqtt_err_t ephoros_mqtt_start(
	ephoros_mqtt_client_t** client,
	ephoros_mqtt_config_t* 	config
);
ephoros_mqtt_err_t ephoros_mqtt_publish(
	ephoros_mqtt_client_t* client,
	ephoros_mqtt_message_t* message
);

#endif // EPHOROSMQTT_H

