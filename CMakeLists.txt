cmake_minimum_required(VERSION 3.16.0)
include($ENV{IDF_PATH}/tools/cmake/project.cmake)
list(APPEND SDKCONFIG_DEFAULTS "sdkconfig.defaults")

list(APPEND kconfigs "${CMAKE_SOURCE_DIR}/lib/esp_ghota/Kconfig")
list(APPEND kconfigs "${CMAKE_SOURCE_DIR}/lib/esp32-wifi-manager/Kconfig")

project(subway-pcb)