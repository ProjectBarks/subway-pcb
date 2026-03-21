cmake_minimum_required(VERSION 3.16.0)
include($ENV{IDF_PATH}/tools/cmake/project.cmake)
list(APPEND SDKCONFIG_DEFAULTS "sdkconfig.defaults")

list(APPEND EXTRA_COMPONENT_DIRS
    "${CMAKE_SOURCE_DIR}/components/led_strip"
    "${CMAKE_SOURCE_DIR}/components/mdns"
    "${CMAKE_SOURCE_DIR}/components/esp32-wifi-manager"
    "${CMAKE_SOURCE_DIR}/components/esp_ghota"
    "${CMAKE_SOURCE_DIR}/components/lua54"
)

list(APPEND kconfigs "${CMAKE_SOURCE_DIR}/components/esp_ghota/Kconfig")
list(APPEND kconfigs "${CMAKE_SOURCE_DIR}/components/esp32-wifi-manager/Kconfig")

project(subway-pcb)
