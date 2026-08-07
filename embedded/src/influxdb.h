#pragma once

#include <stdint.h>

#include "esp_err.h"

/**
 * Connection details for an InfluxDB 2.x Write API endpoint.
 *
 * write_url must include the organization, bucket, and precision query
 * parameters. For example:
 * https://influx.example.com/api/v2/write?org=apex&bucket=telemetry&precision=ns
 *
 * token is an InfluxDB API token with write access to the configured bucket.
 * Keep it out of source control; provision it from secure storage in production.
 */
typedef struct {
	const char *write_url;
	const char *token;
	int timeout_ms;
} influxdb_config_t;

typedef struct {
	influxdb_config_t config;
} influxdb_client_t;

/** Build an InfluxDB configuration from the Kconfig CONFIG_EPHOROS_INFLUXDB_* values. */
influxdb_config_t influxdb_config_from_kconfig(void);

/** Initialize an InfluxDB client. Does not perform network I/O. */
esp_err_t influxdb_client_init(
	influxdb_client_t *client,
	const influxdb_config_t *config
);

/**
 * Write one or more newline-separated Influx line-protocol points.
 *
 * A successful InfluxDB write returns HTTP 204. The line protocol is supplied
 * by the caller so decoded CAN data can choose its own measurement, tags, and
 * fields. Example: "can_signal,frame_id=0x123 speed=42.5 1735689600000000000".
 */
esp_err_t influxdb_write_line(
	const influxdb_client_t *client,
	const char *line_protocol
);

/**
 * Convenience wrapper for a single numeric CAN signal.
 *
 * timestamp_ns is Unix time in nanoseconds. Pass 0 to let InfluxDB assign the
 * server receive time. Measurement, tag, and field names must already follow
 * Influx line-protocol escaping rules.
 */
esp_err_t influxdb_write_number(
	const influxdb_client_t *client,
	const char *measurement,
	const char *tag_key,
	const char *tag_value,
	const char *field_key,
	double value,
	int64_t timestamp_ns
);
