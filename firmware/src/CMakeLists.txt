idf_component_register(
    SRCS "main.cpp"
         "config/board_config.cpp"
         "log/device_log.cpp"
         "proto/codec.cpp"
         "hal/led_driver.cpp"
         "net/http_client.cpp"
         "net/state_client.cpp"
         "script/lua_bindings.cpp"
         "script/lua_runtime.cpp"
         "data/board_snapshot.cpp"
         "${CMAKE_SOURCE_DIR}/proto/subway.pb.c"
    INCLUDE_DIRS "." "${CMAKE_SOURCE_DIR}/proto"
    REQUIRES nvs_flash esp_http_client esp_event esp_timer driver esp_wifi esp_netif
             led_strip esp32-wifi-manager esp_ghota lua54
)

# C++20 flags applied only to C++ sources in this component (not to subway.pb.c or framework)
target_compile_options(${COMPONENT_LIB} PRIVATE
    $<$<COMPILE_LANGUAGE:CXX>:-std=gnu++20 -fno-exceptions -fno-rtti>
)
