# Lua 5.4.7 for ESP-IDF

This component wraps the Lua 5.4.7 interpreter for use on ESP32 via ESP-IDF.

## Setup

1. Download the Lua 5.4.7 source archive:

       curl -O https://www.lua.org/ftp/lua-5.4.7.tar.gz

2. Extract and copy the source files into this directory:

       tar xzf lua-5.4.7.tar.gz
       cp lua-5.4.7/src/*.c lua-5.4.7/src/*.h .

3. Remove files not needed on ESP32 (no filesystem I/O, no OS calls, no
   dynamic loading, no standalone interpreter/compiler):

       rm -f liolib.c loslib.c ldblib.c loadlib.c lua.c luac.c

4. Build with `idf.py build` as usual. The CMakeLists.txt in this directory
   registers all remaining `.c` files and exposes the headers.

## Files included

### Core VM
lapi.c lcode.c lctype.c ldebug.c ldo.c ldump.c lfunc.c lgc.c llex.c
lmem.c lobject.c lopcodes.c lparser.c lstate.c lstring.c ltable.c
ltm.c lundump.c lvm.c lzio.c

### Standard libraries (safe subset)
lauxlib.c lbaselib.c lcorolib.c lmathlib.c lstrlib.c ltablib.c
lutf8lib.c linit.c

### Headers
lua.h luaconf.h lualib.h lauxlib.h (plus internal headers)

## Memory

The firmware uses a custom allocator to cap Lua memory at 80 KB. See
`lua_runtime.c` for details.
