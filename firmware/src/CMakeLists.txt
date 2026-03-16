idf_component_register(
    SRCS "main.c" "led_driver.c" "subway_client.c"
         "${CMAKE_SOURCE_DIR}/proto/subway.pb.c"
    INCLUDE_DIRS "." "${CMAKE_SOURCE_DIR}/proto"
    REQUIRES nvs_flash esp_http_client esp_event esp_timer driver esp_wifi esp_netif
             led_strip esp32-wifi-manager
)
