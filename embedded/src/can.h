#pragma once

#include <stdbool.h>
#include <stdint.h>

#include "esp_err.h"
#include "freertos/FreeRTOS.h"

#define CAN_DECODED_SIGNAL_NAME_MAX_LEN 32
#define CAN_DECODED_SIGNAL_UNIT_MAX_LEN 32
#define CAN_DECODED_SIGNAL_TOPIC_MAX_LEN 32

/** A decoded CAN signal ready for transport by an application task. */
typedef struct {
	uint32_t can_id;
	char name[CAN_DECODED_SIGNAL_NAME_MAX_LEN];
	char unit[CAN_DECODED_SIGNAL_UNIT_MAX_LEN];
	char topic[CAN_DECODED_SIGNAL_TOPIC_MAX_LEN];
	float value;
	uint64_t timestamp;
} can_decoded_signal_t;

/**
 * Start the ESP32 TWAI receiver and its frame-processing task.
 *
 * GPIO pins, bitrate, and receive mode are configured through menuconfig.
 */
esp_err_t can_start(void);

/**
 * Wait for a decoded CAN signal.
 *
 * Ownership stays with the CAN module: the returned value contains no Vera
 * pointers and remains valid after this function returns.
 */
bool can_receive_decoded_signal(can_decoded_signal_t *signal,
							 TickType_t timeout);
