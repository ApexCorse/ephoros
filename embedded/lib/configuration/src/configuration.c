#include <configuration.h>
#include <stdlib.h>
#include <string.h>

#define CONFIGURATION_MAX_JSON_OBJECTS 2048
#define CONFIGURATION_MAX_SENSOR_CONFIGS 1024

CONFIGURATION_err _init_sensor(const json_t* json, CONFIGURATION_sensor_config* cfgPtr);
CONFIGURATION_err _resize(CONFIGURATION_sensor_config** configs, size_t nConfigs);
CONFIGURATION_err _copy_values_sensor_config(
  CONFIGURATION_sensor_config* cfg,
  const char* id,
  const char* topic
);

CONFIGURATION_err CONFIGURATION_initialize(
  CONFIGURATION_configs** configsPtr, 
  char bytes[], 
  unsigned int nBytes
) {
  json_t* array = malloc(sizeof(json_t)*CONFIGURATION_MAX_JSON_OBJECTS); 
  if (!array) return CONFIGURATION_err_allocation;

  char* bytesCopy = strdup(bytes);
  if (!bytesCopy) {
    free(array);
    return CONFIGURATION_err_allocation;
  }

  const json_t* json = json_create(bytesCopy, array, nBytes);
  if (!json) {
    free(array);
    free(bytesCopy);
    return CONFIGURATION_err_invalid_config;
  }

  const json_t* sensors = json_getProperty(json, "sensors");
  if (!sensors || json_getType(sensors) != JSON_ARRAY) {
    free(array);
    free(bytesCopy);
    return CONFIGURATION_err_invalid_config;
  }

  CONFIGURATION_sensor_config* configs_array = malloc(sizeof(CONFIGURATION_sensor_config)*CONFIGURATION_MAX_SENSOR_CONFIGS);
  if (!configs_array) {
    free(array);
    free(bytesCopy);
    return CONFIGURATION_err_allocation;
  }
  
  const json_t* curr_sensor = json_getChild(sensors);
  CONFIGURATION_err err = CONFIGURATION_err_ok;
  size_t n = 0;
  while (n < CONFIGURATION_MAX_SENSOR_CONFIGS && curr_sensor) {
    if ((err = _init_sensor(curr_sensor, &(configs_array[n++]))) != CONFIGURATION_err_ok) {
      free(array);
      free(configs_array);
      free(bytesCopy);
      return err;
    }
    
    curr_sensor = json_getSibling(curr_sensor);
  }
  free(array);
  free(bytesCopy);

  if ((err = _resize(&configs_array, n)) != CONFIGURATION_err_ok) {
    free(configs_array);
    return err;
  }

  CONFIGURATION_configs* configs = malloc(sizeof(CONFIGURATION_configs));
  if (!configs) {
    free(configs_array);
    return CONFIGURATION_err_allocation;
  }
  configs->configs = configs_array;
  configs->n = n;

  *configsPtr = configs;

  return CONFIGURATION_err_ok;
}

char* CONFIGURATION_get_topic_by_id(
  CONFIGURATION_configs* configs, 
  const char* id 
) {
  for (int i = 0; i < configs->n; i++) {
    if (strcmp(configs->configs[i].id, id) == 0) {
      return strdup(configs->configs[i].topic);
    }
  }

  return NULL;
}

void CONFIGURATION_cleanup(CONFIGURATION_configs* configs) {
  if (!configs) return;
  if (!(configs->configs)) return;

  for (int i = 0; i < configs->n; i++) {
    free(configs->configs[i].topic);
    free(configs->configs[i].id);
  }
  free(configs->configs);
  free(configs);
}

CONFIGURATION_err _init_sensor(const json_t* json, CONFIGURATION_sensor_config* cfgPtr) {
  if (!json) return CONFIGURATION_err_invalid_config;
  if (json_getType(json) != JSON_OBJ) return CONFIGURATION_err_invalid_config;

  const char* id = json_getPropertyValue(json, "id");
  if (!id) return CONFIGURATION_err_invalid_config;

  const char* topic = json_getPropertyValue(json, "topic");
  if (!topic) return CONFIGURATION_err_invalid_config;

  const CONFIGURATION_err err = _copy_values_sensor_config(cfgPtr, id, topic);
  if (err != CONFIGURATION_err_ok) return err;

  return CONFIGURATION_err_ok;
}

CONFIGURATION_err _resize(CONFIGURATION_sensor_config** configs, size_t n_configs) {
  CONFIGURATION_sensor_config* newConfigs = malloc(sizeof(CONFIGURATION_sensor_config)*n_configs);
  if (!newConfigs) return CONFIGURATION_err_allocation;

  memcpy(newConfigs, *configs, sizeof(CONFIGURATION_sensor_config)*n_configs);

  free(*configs);
  *configs = newConfigs;

  return CONFIGURATION_err_ok;
}

CONFIGURATION_err _copy_values_sensor_config(
  CONFIGURATION_sensor_config* cfg,
  const char* id,
  const char* topic
) {
  cfg->id = strdup(id);
  if (!(cfg->id)) return CONFIGURATION_err_allocation;

  cfg->topic = strdup(topic);
  if (!(cfg->topic)) return CONFIGURATION_err_allocation;

  return CONFIGURATION_err_ok;
}
