#include <configuration.h>
#include <unity.h>

#include "json/example_valid.h"
#include "json/example_invalid.h"

void test_configuration_initialize() {
	CONFIGURATION_configs* cfg = NULL;
	CONFIGURATION_err err = CONFIGURATION_initialize(
		&cfg,
		example_valid_json,
		example_valid_json_len
	);

	TEST_ASSERT_NOT_NULL(cfg);
	TEST_ASSERT_EQUAL(CONFIGURATION_err_ok, err);
	TEST_ASSERT_EQUAL_STRING("NTC1", cfg->configs[0].id);
	TEST_ASSERT_EQUAL_STRING("Battery/Module-1/NTC-1", cfg->configs[0].topic);
	CONFIGURATION_cleanup(cfg);
}

void test_configuration_initialize_invalid_config() {
	CONFIGURATION_configs* cfg = NULL;
	CONFIGURATION_err err = CONFIGURATION_initialize(
		&cfg,
		example_invalid_json,
		example_invalid_json_len
	);

	TEST_ASSERT_NULL(cfg);
	TEST_ASSERT_EQUAL(CONFIGURATION_err_invalid_config, err);
	CONFIGURATION_cleanup(cfg);
}

void test_configuration_find_topic_by_id() {
	CONFIGURATION_configs* cfg = NULL;
	CONFIGURATION_initialize(
		&cfg,
		example_valid_json,
		example_valid_json_len
	);

	char* topic = CONFIGURATION_get_topic_by_id(cfg, "NTC1");
	TEST_ASSERT_NOT_NULL(topic);
	TEST_ASSERT_EQUAL_STRING("Battery/Module-1/NTC-1", topic);
}

void test_configuration_find_topic_by_id_not_found() {
	CONFIGURATION_configs* cfg = NULL;
	CONFIGURATION_initialize(
		&cfg,
		example_valid_json,
		example_valid_json_len
	);

	char* topic = CONFIGURATION_get_topic_by_id(cfg, "NTC2");
	TEST_ASSERT_NULL(topic);
}

void app_main(void) {
  UNITY_BEGIN();

  RUN_TEST(test_configuration_initialize);
  RUN_TEST(test_configuration_initialize_invalid_config);
  RUN_TEST(test_configuration_find_topic_by_id);
  RUN_TEST(test_configuration_find_topic_by_id_not_found);

  UNITY_END();
}
