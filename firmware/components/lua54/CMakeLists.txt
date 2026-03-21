# Lua 5.4.7 ESP-IDF Component
#
# To set up:
# 1. Download Lua 5.4.7 source from https://www.lua.org/ftp/lua-5.4.7.tar.gz
# 2. Extract and copy these files to this directory:
#    lapi.c lcode.c lctype.c ldebug.c ldo.c ldump.c lfunc.c lgc.c llex.c
#    lmem.c lobject.c lopcodes.c lparser.c lstate.c lstring.c ltable.c
#    ltm.c lundump.c lvm.c lzio.c lauxlib.c lbaselib.c lcorolib.c
#    lmathlib.c lstrlib.c ltablib.c lutf8lib.c linit.c
#    And headers: lua.h luaconf.h lualib.h lauxlib.h (plus internal headers)
#
# Excluded (not needed on ESP32):
#    liolib.c loslib.c ldblib.c loadlib.c lua.c luac.c

file(GLOB LUA_SOURCES "${CMAKE_CURRENT_SOURCE_DIR}/*.c")

if(LUA_SOURCES)
    idf_component_register(
        SRC_DIRS "."
        INCLUDE_DIRS "."
        REQUIRES ""
    )
    # Suppress warnings in third-party Lua code
    target_compile_options(${COMPONENT_LIB} PRIVATE -w)
else()
    # No Lua sources downloaded yet — register as header-only stub.
    # See README.md for setup instructions.
    idf_component_register(
        INCLUDE_DIRS "."
        REQUIRES ""
    )
endif()
