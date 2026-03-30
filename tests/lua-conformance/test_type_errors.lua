-- test_type_errors.lua — verify argument type checking matches luaL_check* behavior

local function expect_error(fn, label)
    local ok, _ = pcall(fn)
    if not ok then
        _results.pass = _results.pass + 1
    else
        _results.fail = _results.fail + 1
        table.insert(_results.errors, label .. ": expected error, got none")
    end
end

local function expect_no_error(fn, label)
    local ok, err = pcall(fn)
    if ok then
        _results.pass = _results.pass + 1
    else
        _results.fail = _results.fail + 1
        table.insert(_results.errors, label .. ": unexpected error: " .. tostring(err))
    end
end

-- luaL_checkinteger: rejects nil, bool, non-numeric strings
expect_error(function() has_train(nil) end, "int rejects nil")
expect_error(function() has_train(true) end, "int rejects bool")
expect_error(function() has_train("hello") end, "int rejects non-numeric string")

-- luaL_checkinteger: accepts integers and integer-valued floats
expect_no_error(function() has_train(0) end, "int accepts integer")
expect_no_error(function() has_train(2.0) end, "int accepts integer-valued float")

-- luaL_checkinteger: rejects non-integer floats
expect_error(function() has_train(1.5) end, "int rejects non-integer float")

-- luaL_checkinteger: coerces numeric strings
expect_no_error(function() has_train("0") end, "int coerces integer string")

-- luaL_checkstring: rejects nil, bool
expect_error(function() get_string_config(nil) end, "str rejects nil")
expect_error(function() get_string_config(true) end, "str rejects bool")

-- luaL_checkstring: coerces numbers to strings
expect_no_error(function() get_string_config(123) end, "str coerces number")

-- luaL_checknumber: rejects nil, bool, non-numeric strings
expect_error(function() hsv_to_rgb(nil, 1, 1) end, "num rejects nil")
expect_error(function() hsv_to_rgb(true, 1, 1) end, "num rejects bool")
expect_error(function() hsv_to_rgb("hello", 1, 1) end, "num rejects non-numeric string")

-- luaL_checknumber: accepts all numbers
expect_no_error(function() hsv_to_rgb(0, 1, 1) end, "num accepts integer")
expect_no_error(function() hsv_to_rgb(0.5, 1, 1) end, "num accepts float")

-- luaL_checknumber: coerces numeric strings
expect_no_error(function() hsv_to_rgb("0.5", 1, 1) end, "num coerces numeric string")

-- set_led: all four args are checkinteger
expect_error(function() set_led("a", 0, 0, 0) end, "set_led rejects string index")
expect_error(function() set_led(0, nil, 0, 0) end, "set_led rejects nil r")
expect_error(function() set_led(0, 0, true, 0) end, "set_led rejects bool g")
expect_error(function() set_led(0, 0, 0, "x") end, "set_led rejects string b")

-- has_status: (checkinteger, checkstring)
expect_error(function() has_status("a", STOPPED_AT) end, "has_status rejects string index")
expect_error(function() has_status(0, nil) end, "has_status rejects nil status")

-- hex_to_rgb: checkstring
expect_error(function() hex_to_rgb(nil) end, "hex_to_rgb rejects nil")
expect_error(function() hex_to_rgb(true) end, "hex_to_rgb rejects bool")

-- led_to_strip: checkinteger
expect_error(function() led_to_strip("a") end, "led_to_strip rejects string")
expect_error(function() led_to_strip(nil) end, "led_to_strip rejects nil")

-- log: checkstring
expect_error(function() log(nil) end, "log rejects nil")
expect_error(function() log(true) end, "log rejects bool")
expect_no_error(function() log(123) end, "log coerces number")
