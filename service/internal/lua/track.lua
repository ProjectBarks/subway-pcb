function render()
    for i = 0, led_count() - 1 do
        if has_status(i, STOPPED_AT) then
            local route = get_route(i)
            if route then
                local r, g, b = get_rgb_config(route)
                if r then
                    set_led(i, r, g, b)
                end
            end
        end
    end
end
