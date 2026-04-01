CXX      ?= c++
CXXFLAGS := -std=c++20 -Wall -Wextra -Wno-unused-parameter -Wno-unused-function \
            -Ihost_stubs -Inanopb -I../components/lua54 -I../src -I../proto \
            -DHOST_BUILD
CC       ?= cc
CFLAGS   := -Wall -Wextra -Wno-unused-parameter \
            -Ihost_stubs -Inanopb -I../components/lua54 -I../src -I../proto \
            -DHOST_BUILD

LUA_SRC  := $(wildcard ../components/lua54/l*.c)
LUA_OBJ  := $(patsubst ../components/lua54/%.c,build/lua/%.o,$(LUA_SRC))
NANOPB_OBJ := build/nanopb/pb_common.o build/nanopb/pb_decode.o build/nanopb/pb_encode.o
PROTO_OBJ  := build/proto/subway.pb.o

.PHONY: test clean

test: conformance e2e
	./conformance ../../tests/lua-conformance/
	./e2e

# --- Lua objects ---
build/lua:
	mkdir -p build/lua

build/lua/%.o: ../components/lua54/%.c | build/lua
	$(CC) $(CFLAGS) -c -o $@ $<

# --- Nanopb objects ---
build/nanopb:
	mkdir -p build/nanopb

build/nanopb/%.o: nanopb/%.c | build/nanopb
	$(CC) $(CFLAGS) -c -o $@ $<

# --- Proto objects ---
build/proto:
	mkdir -p build/proto

build/proto/subway.pb.o: ../proto/subway.pb.c | build/proto
	$(CC) $(CFLAGS) -c -o $@ $<

# --- Board snapshot (C++ src) ---
build/board_snapshot.o: ../src/data/board_snapshot.cpp | build
	$(CXX) $(CXXFLAGS) -c -o $@ $<

build:
	mkdir -p build

# --- Conformance test (Lua API) ---
conformance: test_lua_conformance.cpp stubs.cpp $(LUA_OBJ)
	$(CXX) $(CXXFLAGS) -o $@ test_lua_conformance.cpp stubs.cpp $(LUA_OBJ) -lm -lpthread

# --- E2E test (protobuf → snapshot → Lua → pixels) ---
e2e: test_e2e.cpp stubs.cpp $(LUA_OBJ) $(NANOPB_OBJ) $(PROTO_OBJ) build/board_snapshot.o
	$(CXX) $(CXXFLAGS) -o $@ test_e2e.cpp stubs.cpp \
		../src/proto/codec.cpp ../src/script/lua_bindings.cpp \
		$(LUA_OBJ) $(NANOPB_OBJ) $(PROTO_OBJ) build/board_snapshot.o -lm -lpthread

clean:
	rm -rf conformance e2e build
