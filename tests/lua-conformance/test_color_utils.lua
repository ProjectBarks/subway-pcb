-- test_color_utils.lua — hsv_to_rgb, hex_to_rgb

local r, g, b = hsv_to_rgb(0, 1, 1)
expect(r, 255, "hsv red r")
expect(g, 0, "hsv red g")
expect(b, 0, "hsv red b")

r, g, b = hsv_to_rgb(1/3, 1, 1)
expect(r, 0, "hsv green r")
expect(g, 255, "hsv green g")
expect(b, 0, "hsv green b")

r, g, b = hsv_to_rgb(2/3, 1, 1)
expect(r, 0, "hsv blue r")
expect(g, 0, "hsv blue g")
expect(b, 255, "hsv blue b")

r, g, b = hsv_to_rgb(0, 0, 1)
expect(r, 255, "hsv white r")
expect(g, 255, "hsv white g")
expect(b, 255, "hsv white b")

r, g, b = hsv_to_rgb(0, 1, 0)
expect(r, 0, "hsv black r")
expect(g, 0, "hsv black g")
expect(b, 0, "hsv black b")

r, g, b = hsv_to_rgb(0, 1, 0.5)
expect(r, 128, "hsv half-red r")
expect(g, 0, "hsv half-red g")
expect(b, 0, "hsv half-red b")

r, g, b = hex_to_rgb("#FF8800")
expect(r, 255, "hex FF8800 r")
expect(g, 136, "hex FF8800 g")
expect(b, 0, "hex FF8800 b")

r, g, b = hex_to_rgb("#000000")
expect(r, 0, "hex 000000 r")
expect(g, 0, "hex 000000 g")
expect(b, 0, "hex 000000 b")

r, g, b = hex_to_rgb("#FFFFFF")
expect(r, 255, "hex FFFFFF r")
expect(g, 255, "hex FFFFFF g")
expect(b, 255, "hex FFFFFF b")

r, g, b = hex_to_rgb("#ff0000")
expect(r, 255, "hex lowercase r")
expect(g, 0, "hex lowercase g")
expect(b, 0, "hex lowercase b")

expect_nil(hex_to_rgb("FF0000"), "hex no hash")
expect_nil(hex_to_rgb("#FFF"), "hex too short")
