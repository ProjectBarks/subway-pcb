-- helpers.lua — minimal test framework for conformance tests
-- Both C and TS harnesses load this before each test file.

_results = { pass = 0, fail = 0, errors = {} }

function expect(got, expected, label)
    if got == expected then
        _results.pass = _results.pass + 1
    else
        _results.fail = _results.fail + 1
        table.insert(_results.errors,
            label .. ": expected " .. tostring(expected) .. ", got " .. tostring(got))
    end
end

function expect_nil(got, label)
    if got == nil then
        _results.pass = _results.pass + 1
    else
        _results.fail = _results.fail + 1
        table.insert(_results.errors,
            label .. ": expected nil, got " .. tostring(got))
    end
end

function expect_table_eq(got, expected, label)
    -- Accept both "table" (native Lua) and "userdata" (wasmoon JS arrays)
    local gt = type(got)
    if gt ~= "table" and gt ~= "userdata" then
        _results.fail = _results.fail + 1
        table.insert(_results.errors, label .. ": expected table, got " .. gt)
        return
    end
    if #got ~= #expected then
        _results.fail = _results.fail + 1
        table.insert(_results.errors,
            label .. ": length " .. tostring(#got) .. " != " .. tostring(#expected))
        return
    end
    for i = 1, #expected do
        if got[i] ~= expected[i] then
            _results.fail = _results.fail + 1
            table.insert(_results.errors,
                label .. "[" .. i .. "]: expected " .. tostring(expected[i]) .. ", got " .. tostring(got[i]))
            return
        end
    end
    _results.pass = _results.pass + 1
end
