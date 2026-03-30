CC      ?= gcc
CFLAGS  := -Wall -Wextra -Wno-unused-parameter -Wno-unused-function \
           -I../components/lua54 -I../src -I../proto -Ihost_stubs \
           -I../.pio/libdeps/esp32dev/Nanopb -DHOST_BUILD
LUA_SRC := $(wildcard ../components/lua54/l*.c)

.PHONY: test clean

test: conformance
	./conformance ../../tests/lua-conformance/

conformance: test_lua_conformance.c stubs.c $(LUA_SRC)
	$(CC) $(CFLAGS) -o $@ $^ -lm

clean:
	rm -f conformance
