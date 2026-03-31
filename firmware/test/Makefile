CXX     ?= c++
CXXFLAGS := -std=c++20 -Wall -Wextra -Wno-unused-parameter -Wno-unused-function \
            -Ihost_stubs -I../components/lua54 -I../src -I../proto \
            -DHOST_BUILD
CC       ?= cc
CFLAGS   := -Wall -Wextra -Wno-unused-parameter \
            -Ihost_stubs -I../components/lua54 -I../src -I../proto \
            -DHOST_BUILD
LUA_SRC  := $(wildcard ../components/lua54/l*.c)
LUA_OBJ  := $(patsubst ../components/lua54/%.c,build/%.o,$(LUA_SRC))

.PHONY: test clean

test: conformance
	./conformance ../../tests/lua-conformance/

build:
	mkdir -p build

build/%.o: ../components/lua54/%.c | build
	$(CC) $(CFLAGS) -c -o $@ $<

conformance: test_lua_conformance.cpp stubs.cpp $(LUA_OBJ)
	$(CXX) $(CXXFLAGS) -o $@ test_lua_conformance.cpp stubs.cpp $(LUA_OBJ) -lm

clean:
	rm -rf conformance build
