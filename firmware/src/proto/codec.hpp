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

// Encode DeviceDiagnostics into buffer. Returns bytes written, or 0 on failure.
int encode_diagnostics(const subway_DeviceDiagnostics& diag, uint8_t* buf, int buf_size);

} // namespace codec
