cmake_minimum_required(VERSION 3.16.0)
include($ENV{IDF_PATH}/tools/cmake/project.cmake)
list(APPEND SDKCONFIG_DEFAULTS "sdkconfig.defaults")

list(APPEND kconfigs "${CMAKE_SOURCE_DIR}/lib/esp_ghota/Kconfig")

project(subway-pcb)