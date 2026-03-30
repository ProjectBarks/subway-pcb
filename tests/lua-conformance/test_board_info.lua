-- test_board_info.lua — get_strip_info, led_to_strip, constants, get_time, log

expect_table_eq(get_strip_info(), {3, 2}, "get_strip_info sizes")

local strip, pixel = led_to_strip(0)
expect(strip, 1, "led_to_strip(0) strip")
expect(pixel, 0, "led_to_strip(0) pixel")

strip, pixel = led_to_strip(2)
expect(strip, 1, "led_to_strip(2) strip")
expect(pixel, 2, "led_to_strip(2) pixel")

strip, pixel = led_to_strip(3)
expect(strip, 2, "led_to_strip(3) strip")
expect(pixel, 0, "led_to_strip(3) pixel")

strip, pixel = led_to_strip(4)
expect(strip, 2, "led_to_strip(4) strip")
expect(pixel, 1, "led_to_strip(4) pixel")

strip, pixel = led_to_strip(5)
expect_nil(strip, "led_to_strip(5) strip nil")
expect_nil(pixel, "led_to_strip(5) pixel nil")

strip, pixel = led_to_strip(99)
expect_nil(strip, "led_to_strip(99) strip nil")
expect_nil(pixel, "led_to_strip(99) pixel nil")

expect(STOPPED_AT, "STOPPED_AT", "STOPPED_AT constant")
expect(INCOMING_AT, "INCOMING_AT", "INCOMING_AT constant")
expect(IN_TRANSIT_TO, "IN_TRANSIT_TO", "IN_TRANSIT_TO constant")

expect(type(get_time()), "number", "get_time returns number")

log("conformance test log message")
_results.pass = _results.pass + 1
