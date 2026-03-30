function render()
    local br = (get_int_config("brightness") or 255) / 255
    for i = 0, led_count() - 1 do
        if has_status(i, STOPPED_AT) then
            local route = get_route(i)
            if route then
                local r, g, b = get_rgb_config(route)
                if r then
                    set_led(i, math.floor(r * br), math.floor(g * br), math.floor(b * br))
                end
            end
        end
    end
end
