// Test: assert firmware dispatch table matches serial-commands.json spec
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <vector>
#include <string>

// Minimal test framework (no Lua/proto dependencies)
struct TestResults {
    int pass = 0;
    int fail = 0;
};

static inline void check(TestResults& r, bool cond, const char* label) {
    if (cond) {
        r.pass++;
    } else {
        r.fail++;
        printf("  FAIL: %s\n", label);
    }
}

static inline const char* parse_junit_arg(int argc, char* argv[]) {
    for (int i = 1; i < argc - 1; i++) {
        if (strcmp(argv[i], "--junit") == 0) return argv[i + 1];
    }
    return nullptr;
}

static inline void write_junit_xml(const char* path, const char* suite,
                                   const char* const* names, const int* fails, int count) {
    FILE* f = fopen(path, "w");
    if (!f) return;
    int total_fail = 0;
    for (int i = 0; i < count; i++) { if (fails[i]) total_fail++; }
    fprintf(f, "<?xml version=\"1.0\"?>\n<testsuites>\n");
    fprintf(f, "<testsuite name=\"%s\" tests=\"%d\" failures=\"%d\">\n",
            suite, count, total_fail);
    for (int i = 0; i < count; i++) {
        fprintf(f, "  <testcase name=\"%s\" classname=\"%s\"", names[i], suite);
        if (fails[i]) {
            fprintf(f, ">\n    <failure message=\"%d assertion(s) failed\"/>\n  </testcase>\n", fails[i]);
        } else {
            fprintf(f, "/>\n");
        }
    }
    fprintf(f, "</testsuite>\n</testsuites>\n");
    fclose(f);
}

static std::string read_file_str(const char* path) {
    FILE* f = fopen(path, "rb");
    if (!f) {
        fprintf(stderr, "ERROR: cannot open %s\n", path);
        return "";
    }
    fseek(f, 0, SEEK_END);
    long len = ftell(f);
    fseek(f, 0, SEEK_SET);
    std::string content(len, '\0');
    fread(&content[0], 1, len, f);
    fclose(f);
    return content;
}

// Minimal JSON parsing for the spec file (no cJSON on host)
// We just need to extract "cmd" strings from the spec

struct SpecCommand {
    std::string cmd;
};

static std::vector<SpecCommand> parse_spec(const char* path) {
    std::vector<SpecCommand> commands;
    std::string content = read_file_str(path);
    if (content.empty()) return commands;

    // Simple extraction: find all "cmd": "..." patterns
    size_t pos = 0;
    while ((pos = content.find("\"cmd\"", pos)) != std::string::npos) {
        size_t colon = content.find(':', pos);
        size_t quote1 = content.find('"', colon + 1);
        size_t quote2 = content.find('"', quote1 + 1);
        if (quote1 != std::string::npos && quote2 != std::string::npos) {
            commands.push_back({content.substr(quote1 + 1, quote2 - quote1 - 1)});
        }
        pos = quote2 + 1;
    }
    return commands;
}

// Mirror of the firmware dispatch table command strings
// This must be kept in sync with serial_handler.cpp
static const char* kDispatchCommands[] = {
    "PING",
    "GET INFO",
    "GET WIFI",
    "GET DIAG",
    "SET WIFI_SSID",
    "SET WIFI_PASS",
    "SET SERVER_URL",
    "DO WIFI_SCAN",
    "DO WIFI_APPLY",
    "DO LED_TEST",
    "DO REBOOT",
    "DO FACTORY_RESET",
    "DO SCRIPT_BEGIN",
    "DO SCRIPT_CLEAR",
    "SCRIPT_END",
};
static const int kDispatchCount = sizeof(kDispatchCommands) / sizeof(kDispatchCommands[0]);

int main(int argc, char* argv[]) {
    const char* junit_path = parse_junit_arg(argc, argv);
    TestResults results;

    // Parse spec (path relative to firmware/test/)
    auto spec = parse_spec("../../proto/serial-commands.json");

    check(results, !spec.empty(), "Spec file loaded with commands");

    // Test: every spec command exists in dispatch table
    for (const auto& sc : spec) {
        bool found = false;
        for (int i = 0; i < kDispatchCount; i++) {
            if (sc.cmd == kDispatchCommands[i]) {
                found = true;
                break;
            }
        }
        char msg[128];
        snprintf(msg, sizeof(msg), "Spec command '%s' exists in firmware dispatch table", sc.cmd.c_str());
        check(results, found, msg);
    }

    // Test: every dispatch command exists in spec
    for (int i = 0; i < kDispatchCount; i++) {
        bool found = false;
        for (const auto& sc : spec) {
            if (sc.cmd == kDispatchCommands[i]) {
                found = true;
                break;
            }
        }
        char msg[128];
        snprintf(msg, sizeof(msg), "Firmware command '%s' exists in spec", kDispatchCommands[i]);
        check(results, found, msg);
    }

    // Test: counts match
    check(results, (int)spec.size() == kDispatchCount, "Spec and dispatch table have same command count");

    printf("\n%d passed, %d failed out of %d tests\n",
           results.pass, results.fail, results.pass + results.fail);

    // JUnit output
    if (junit_path) {
        std::vector<const char*> names;
        std::vector<int> fails;
        // For simplicity, emit a single test case for the suite
        names.push_back("serial_spec_parity");
        fails.push_back(results.fail);
        write_junit_xml(junit_path, "serial_spec", names.data(), fails.data(), (int)names.size());
    }

    return results.fail > 0 ? 1 : 0;
}
