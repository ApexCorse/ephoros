# This file was automatically generated for projects
# without default 'CMakeLists.txt' file.

FILE(GLOB_RECURSE app_sources ${CMAKE_SOURCE_DIR}/src/*.*)
list(APPEND app_sources ${CMAKE_SOURCE_DIR}/lib/mqtt/mqtt.c)
set(app_include_dirs ${CMAKE_SOURCE_DIR}/lib/mqtt)

if(CONFIG_EPHOROS_CAN_SIMULATOR)
    set(dbc_file ${CMAKE_SOURCE_DIR}/../config.dbc)
    set(simulator_dir ${CMAKE_SOURCE_DIR}/../simulator)
    set(simulator_generated_dir ${CMAKE_BINARY_DIR}/generated)
    set(simulator_catalog ${simulator_generated_dir}/telemetry_catalog.h)

    if(NOT EXISTS ${dbc_file})
        message(FATAL_ERROR "CAN simulator requires ${dbc_file}")
    endif()
    find_program(GO_EXECUTABLE go)
    if(NOT GO_EXECUTABLE)
        message(FATAL_ERROR "CAN simulator requires the Go toolchain to generate its DBC topic catalog")
    endif()
    file(MAKE_DIRECTORY ${simulator_generated_dir})
    execute_process(
        COMMAND ${GO_EXECUTABLE} run . --dbc-file ${dbc_file}
                --catalog-output ${simulator_catalog}
        WORKING_DIRECTORY ${simulator_dir}
        RESULT_VARIABLE simulator_catalog_result
    )
    if(NOT simulator_catalog_result EQUAL 0)
        message(FATAL_ERROR "Could not generate CAN simulator topic catalog")
    endif()
    set_property(DIRECTORY APPEND PROPERTY CMAKE_CONFIGURE_DEPENDS ${dbc_file})
    list(APPEND app_include_dirs ${simulator_generated_dir})
endif()

if(CONFIG_EPHOROS_CAN_VERA_DECODER)
    FILE(GLOB_RECURSE vera_sources ${CMAKE_SOURCE_DIR}/generated/*.c)
    list(APPEND app_sources ${vera_sources})
    list(APPEND app_include_dirs ${CMAKE_SOURCE_DIR}/generated)
endif()

idf_component_register(
	SRCS ${app_sources}
	INCLUDE_DIRS ${app_include_dirs}
	REQUIRES esp_http_client esp_modem esp_driver_twai mbedtls mqtt
)
