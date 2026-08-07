# This file was automatically generated for projects
# without default 'CMakeLists.txt' file.

FILE(GLOB_RECURSE app_sources ${CMAKE_SOURCE_DIR}/src/*.*)
set(app_include_dirs)

if(CONFIG_EPHOROS_CAN_VERA_DECODER)
    FILE(GLOB_RECURSE vera_sources ${CMAKE_SOURCE_DIR}/generated/*.c)
    list(APPEND app_sources ${vera_sources})
    list(APPEND app_include_dirs ${CMAKE_SOURCE_DIR}/generated)
endif()

idf_component_register(
	SRCS ${app_sources}
	INCLUDE_DIRS ${app_include_dirs}
	REQUIRES esp_http_client esp_modem esp_driver_twai mbedtls
)
