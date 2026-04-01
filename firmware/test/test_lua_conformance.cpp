/* Host conformance test for the Lua runtime API.
 * Runs shared Lua test scripts against the compiled lua_bindings. */

#include <cstring>
#include <dirent.h>

#include "test_utils.hpp"
#include "test_fixtures.hpp"

/* Include lua_bindings.cpp directly to access static functions */
#include "script/lua_bindings.cpp"

int main(int argc, char* argv[]) {
    if (argc < 2) {
        fprintf(stderr, "Usage: %s <conformance-dir>\n", argv[0]);
        return 1;
    }

    const char* dir = argv[1];
    char path[512];

    TestFixtures fix;
    fix.reset();

    lua_State* L = create_test_lua();
    if (!L) { fprintf(stderr, "Failed to create Lua state\n"); return 1; }
    lua_set_binding_context(L, &fix.ctx);

    snprintf(path, sizeof(path), "%s/helpers.lua", dir);
    if (luaL_dofile(L, path) != LUA_OK) {
        fprintf(stderr, "Failed to load helpers.lua: %s\n", lua_tostring(L, -1));
        lua_close(L);
        return 1;
    }

    /* Collect and sort test_*.lua files */
    DIR* d = opendir(dir);
    if (!d) { fprintf(stderr, "Cannot open directory: %s\n", dir); lua_close(L); return 1; }

    char test_files[32][256];
    int test_count = 0;
    struct dirent* entry;
    while ((entry = readdir(d)) != nullptr && test_count < 32) {
        if (strncmp(entry->d_name, "test_", 5) == 0 &&
            strcmp(entry->d_name + strlen(entry->d_name) - 4, ".lua") == 0) {
            strncpy(test_files[test_count], entry->d_name, 255);
            test_count++;
        }
    }
    closedir(d);

    for (int i = 0; i < test_count - 1; i++)
        for (int j = i + 1; j < test_count; j++)
            if (strcmp(test_files[i], test_files[j]) > 0) {
                char tmp[256];
                strncpy(tmp, test_files[i], 255);
                strncpy(test_files[i], test_files[j], 255);
                strncpy(test_files[j], tmp, 255);
            }

    printf("Running %d conformance test files...\n", test_count);
    int total_failures = 0;
    for (int i = 0; i < test_count; i++) {
        luaL_dostring(L, "_results = { pass = 0, fail = 0, errors = {} }");
        lua_set_binding_context(L, &fix.ctx);
        snprintf(path, sizeof(path), "%s/%s", dir, test_files[i]);
        if (luaL_dofile(L, path) != LUA_OK) {
            printf("  ERROR: %s — %s\n", test_files[i], lua_tostring(L, -1));
            lua_pop(L, 1);
            total_failures++;
        } else {
            total_failures += check_lua_results(L, test_files[i]);
        }
    }

    lua_close(L);
    printf("\n%s: %d file(s), %d failure(s)\n",
           total_failures > 0 ? "FAILED" : "PASSED", test_count, total_failures);
    return total_failures > 0 ? 1 : 0;
}
