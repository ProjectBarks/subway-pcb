function render()
    local br = (get_int_config("brightness") or 255) / 255
    local snake_length = get_int_config("snake_length") or 5
    local snake_count = get_int_config("snake_count") or 1
    local speed_ms = get_int_config("speed_ms") or 2000
    local step = math.floor(get_time() * 1000 / speed_ms)
    local strips = get_strip_info()
    local offset = 0
    for strip_idx, strip_size in ipairs(strips) do
        local key = string.format("strip_%d_color", strip_idx)
        local r, g, b = get_rgb_config(key)
        if not r then r, g, b = 255, 255, 255 end
        for sn = 0, snake_count - 1 do
            local sn_off = math.floor(strip_size * sn / snake_count)
            local start = (step + sn_off) % strip_size
            for px = 0, snake_length - 1 do
                set_led(offset + (start + px) % strip_size, r * br, g * br, b * br)
            end
        end
        offset = offset + strip_size
    end
end
