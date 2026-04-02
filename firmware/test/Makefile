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

ALLURE_RESULTS ?=
ifdef ALLURE_RESULTS
JUNIT_CONFORMANCE   := --junit $(ALLURE_RESULTS)/fw-conformance.xml
JUNIT_E2E           := --junit $(ALLURE_RESULTS)/fw-e2e.xml
JUNIT_STRESS        := --junit $(ALLURE_RESULTS)/fw-stress.xml
JUNIT_SERIAL_SPEC   := --junit $(ALLURE_RESULTS)/fw-serial-spec.xml
else
JUNIT_CONFORMANCE   :=
JUNIT_E2E           :=
JUNIT_STRESS        :=
JUNIT_SERIAL_SPEC   :=
endif

.PHONY: test clean

test: conformance e2e stress serial_spec
ifdef ALLURE_RESULTS
	mkdir -p $(ALLURE_RESULTS)
endif
	./conformance ../../tests/lua-conformance/ $(JUNIT_CONFORMANCE)
	./e2e $(JUNIT_E2E)
	./stress $(JUNIT_STRESS)
	./serial_spec $(JUNIT_SERIAL_SPEC)

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

# --- Stress tests (includes lua_bindings.cpp directly for allocator access) ---
stress: test_stress.cpp stubs.cpp $(LUA_OBJ) $(NANOPB_OBJ) $(PROTO_OBJ) build/board_snapshot.o
	$(CXX) $(CXXFLAGS) -o $@ test_stress.cpp stubs.cpp \
		../src/proto/codec.cpp \
		$(LUA_OBJ) $(NANOPB_OBJ) $(PROTO_OBJ) build/board_snapshot.o -lm -lpthread

# --- Serial spec parity test (no firmware deps, just JSON parsing) ---
serial_spec: test_serial_commands.cpp
	$(CXX) -std=c++20 -Wall -Wextra -o $@ test_serial_commands.cpp

clean:
	rm -rf conformance e2e stress serial_spec build
