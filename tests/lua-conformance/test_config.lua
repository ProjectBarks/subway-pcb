-- test_config.lua — get_string_config, get_int_config, get_rgb_config

expect(get_string_config("brightness"), "200", "string_config existing key")
expect(get_string_config("name"), "test", "string_config name key")
expect(get_string_config("color"), "#FF8800", "string_config color key")
expect_nil(get_string_config("nonexistent"), "string_config missing key")
expect(get_string_config("empty"), "", "string_config empty value")

expect(get_int_config("brightness"), 200, "int_config numeric value")
expect_nil(get_int_config("nonexistent"), "int_config missing key")
expect(get_int_config("name"), 0, "int_config non-numeric returns 0")

local r, g, b = get_rgb_config("color")
expect(r, 255, "rgb_config red")
expect(g, 136, "rgb_config green")
expect(b, 0, "rgb_config blue")

expect_nil(get_rgb_config("nonexistent"), "rgb_config missing key")
expect_nil(get_rgb_config("name"), "rgb_config non-hex value")
expect_nil(get_rgb_config("empty"), "rgb_config empty value")
