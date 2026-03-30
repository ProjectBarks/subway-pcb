-- test_led_control.lua — set_led, clear_leds, led_count

expect(led_count(), 5, "led_count returns fixture count")

set_led(0, 255, 0, 0)
set_led(4, 0, 255, 0)
set_led(-1, 255, 255, 255)
set_led(5, 255, 255, 255)
set_led(999, 255, 255, 255)
_results.pass = _results.pass + 1 -- survived out-of-bounds without error

clear_leds()
_results.pass = _results.pass + 1 -- survived without error

set_led(0, -10, 300, 128)
_results.pass = _results.pass + 1 -- survived clamping without error
