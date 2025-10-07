#ifndef CONFIGURATION_H
#define CONFIGURATION_H

#include <tiny-json.h>

typedef enum {
  CONFIGURATION_err_ok,
  CONFIGURATION_err_not_found,
  CONFIGURATION_err_allocation,
  CONFIGURATION_err_parsing,
  CONFIGURATION_err_invalid_config
} CONFIGURATION_err;

typedef struct {
  char* id;
  char* topic; 
} CONFIGURATION_sensor_config;

typedef struct {
  CONFIGURATION_sensor_config* configs;
  int n;
} CONFIGURATION_configs;

CONFIGURATION_err CONFIGURATION_initialize(
  CONFIGURATION_configs** configsPtr, 
  char bytes[], 
  unsigned int nBytes
);
char* CONFIGURATION_get_topic_by_id(
  CONFIGURATION_configs* configs,
  const char* id
);
void CONFIGURATION_cleanup(CONFIGURATION_configs* configs);

#endif // CONFIGURATION_H
