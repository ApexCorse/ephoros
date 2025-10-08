#ifndef CONFIGURATION_H
#define CONFIGURATION_H

#include <tiny-json.h>

/**
 * @brief Possible errors returned from CONFIGURATION functions.
 */
typedef enum {
  CONFIGURATION_err_ok,							/// Everything went fine.
  CONFIGURATION_err_allocation, 		/// Error during allocation.
  CONFIGURATION_err_invalid_config 	/// Invalid configuration file.
} CONFIGURATION_err;

/**
 * @brief Contains info about a single sensor.
 * @struct CONFIGURATION_sensor_config.
 * @var CONFIGURATION_sensor_config::id ID of the sensor assigned by the CAN network.
 * @var CONFIGURATION_sensor_config::topic Correspondent topic in the MQTT network.
 */
typedef struct {
  char* id;     /// ID of the sensor assigned by the CAN network.
  char* topic;  /// Correspondent topic in the MQTT network.
} CONFIGURATION_sensor_config;

/**
 * @brief Incapsulates multiple sensor configs.
 * @struct CONFIGURATION_configs.
 * @var CONFIGURATION_configs::configs Array of sensor configs.
 * @var CONFIGURATION_configs::n Number of sensor configs inside `configs`.
 */
typedef struct {
  CONFIGURATION_sensor_config* configs; /// Array of sensor configs.
  int n;                                /// Number of sensor configs inside `configs`.
} CONFIGURATION_configs;

/**
 * @brief Intializes the configuration.
 *
 * @param configsPtr Pointer to a pointer of `CONFIGURATION_configs`. Must be NULL.
 * @param bytes String containing the JSON.
 * @param nBytes Length of `bytes`.
 * @return Error or `CONFIGURATION_err_ok`.
 *
 * @note Function makes a copy of `bytes`, it doesn't get modified.
 */
CONFIGURATION_err CONFIGURATION_initialize(
  CONFIGURATION_configs** configsPtr, 
  char bytes[], 
  unsigned int nBytes
);

/**
 * @brief Finds the topic for the correspondent sensor ID.
 *
 * @param configs Pointer to the configuration.
 * @param id ID of the sensor.
 * @return Topic of the sensor. NULL if not found.
 */
char* CONFIGURATION_get_topic_by_id(
  CONFIGURATION_configs* configs,
  const char* id
);

/**
 * @brief Deallocates memory for the configuration struct.
 *
 * @param configs Pointer to the configuration.
 *
 * @warning `configs` is not set to NULL after deallocating.
 */
void CONFIGURATION_cleanup(CONFIGURATION_configs* configs);

#endif // CONFIGURATION_H
