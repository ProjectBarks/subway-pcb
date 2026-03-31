#pragma once
#include <cstdint>
#include <cstring>

extern "C" {
#include "subway.pb.h"

#include <pb_decode.h>
#include <pb_encode.h>
}

namespace codec {

// Decode DeviceState from buffer. Returns true on success.
bool decode_state(const uint8_t* data, int len, subway_DeviceState& out);

// Decode DeviceBoard from buffer. Returns true on success.
bool decode_board(const uint8_t* data, int len, subway_DeviceBoard& out);

// Decode DeviceScript from buffer. Returns true on success.
bool decode_script(const uint8_t* data, int len, subway_DeviceScript& out);

// Lightweight script info — only the fields firmware needs (~16.5KB vs ~20.6KB).
// Skips config array and plugin_description to reduce heap allocation.
struct ScriptInfo {
    char hash[65]{};
    char lua_source[16384]{};
    char plugin_name[64]{};
};

// Decode only hash, lua_source, plugin_name from a DeviceScript protobuf.
// Uses ~16.5KB vs ~20.6KB for the full struct.
bool decode_script_info(const uint8_t* data, int len, ScriptInfo& out);

// Encode DeviceDiagnostics into buffer. Returns bytes written, or 0 on failure.
int encode_diagnostics(const subway_DeviceDiagnostics& diag, uint8_t* buf, int buf_size);

} // namespace codec
