#include <cstdio>
#include <cstdarg>
#include <cstring>

#include "log/device_log.hpp"

extern "C" {
#include "lua.h"
#include "lauxlib.h"
}

/* device_log — just printf */
void device_log(LogLevel level, const char* tag, const char* fmt, ...)
{
    (void)level;
    va_list ap;
    va_start(ap, fmt);
    printf("[%s] ", tag);
    vprintf(fmt, ap);
    printf("\n");
    va_end(ap);
}

void device_log_init() {}
void device_log_set_remote(bool enabled) { (void)enabled; }
int device_log_drain(char* buf, int buf_size) { (void)buf; (void)buf_size; return 0; }

/* Lua library openers referenced by linit.c but stripped from this build */
extern "C" {
int luaopen_io(lua_State* L) { (void)L; return 0; }
int luaopen_os(lua_State* L) { (void)L; return 0; }
int luaopen_package(lua_State* L) { (void)L; return 0; }
int luaopen_debug(lua_State* L) { (void)L; return 0; }
}
