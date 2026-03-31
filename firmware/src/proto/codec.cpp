#include "proto/codec.hpp"
#include "log/device_log.hpp"

static const char* TAG = "codec";

namespace codec {

bool decode_state(const uint8_t* data, int len, subway_DeviceState& out) {
    std::memset(&out, 0, sizeof(out));
    pb_istream_t stream = pb_istream_from_buffer(data, len);
    if (!pb_decode(&stream, subway_DeviceState_fields, &out)) {
        DLOG_E(TAG, "DeviceState decode failed (len=%d): %s", len, PB_GET_ERROR(&stream));
        return false;
    }
    return true;
}

bool decode_board(const uint8_t* data, int len, subway_DeviceBoard& out) {
    std::memset(&out, 0, sizeof(out));
    pb_istream_t stream = pb_istream_from_buffer(data, len);
    if (!pb_decode(&stream, subway_DeviceBoard_fields, &out)) {
        DLOG_E(TAG, "DeviceBoard decode failed (len=%d): %s", len, PB_GET_ERROR(&stream));
        return false;
    }
    return true;
}

bool decode_script(const uint8_t* data, int len, subway_DeviceScript& out) {
    std::memset(&out, 0, sizeof(out));
    pb_istream_t stream = pb_istream_from_buffer(data, len);
    if (!pb_decode(&stream, subway_DeviceScript_fields, &out)) {
        DLOG_E(TAG, "DeviceScript decode failed (len=%d): %s", len, PB_GET_ERROR(&stream));
        return false;
    }
    return true;
}

int encode_diagnostics(const subway_DeviceDiagnostics& diag, uint8_t* buf, int buf_size) {
    pb_ostream_t ostream = pb_ostream_from_buffer(buf, buf_size);
    if (!pb_encode(&ostream, subway_DeviceDiagnostics_fields, &diag)) {
        DLOG_E(TAG, "Diagnostics encode failed");
        return 0;
    }
    return static_cast<int>(ostream.bytes_written);
}

} // namespace codec
