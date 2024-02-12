cmake_minimum_required(VERSION 3.16.0)
include($ENV{IDF_PATH}/tools/cmake/project.cmake)
list(APPEND SDKCONFIG_DEFAULTS "sdkconfig.defaults")

get_filename_component(configName "${CMAKE_BINARY_DIR}" NAME)
list(APPEND kconfigs "${CMAKE_SOURCE_DIR}/.pio/libdeps/${configName}/esp_ghota/Kconfig")

project(subway-pcb)