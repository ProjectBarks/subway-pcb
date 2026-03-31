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

// Helper: read a protobuf length-delimited string into a fixed buffer.
static bool read_string(pb_istream_t* stream, char* buf, size_t buf_size) {
    uint32_t slen;
    if (!pb_decode_varint32(stream, &slen))
        return false;
    if (slen >= buf_size) {
        // Skip oversized string
        while (slen > 0) {
            uint8_t tmp;
            if (!pb_read(stream, &tmp, 1))
                return false;
            slen--;
        }
        return true;
    }
    if (!pb_read(stream, reinterpret_cast<uint8_t*>(buf), slen))
        return false;
    buf[slen] = '\0';
    return true;
}

bool decode_script_info(const uint8_t* data, int len, ScriptInfo& out) {
    out = ScriptInfo{};
    // Manually walk protobuf wire format, extracting only needed fields:
    //   field 1 (string): hash
    //   field 2 (string): lua_source
    //   field 3 (string): plugin_name
    //   field 4+: skip
    pb_istream_t stream = pb_istream_from_buffer(data, len);
    while (stream.bytes_left > 0) {
        pb_wire_type_t wire_type;
        uint32_t tag;
        bool eof;
        if (!pb_decode_tag(&stream, &wire_type, &tag, &eof)) {
            if (eof)
                break;
            DLOG_E(TAG, "ScriptInfo tag: %s", PB_GET_ERROR(&stream));
            return false;
        }
        switch (tag) {
        case 1:
            if (!read_string(&stream, out.hash, sizeof(out.hash)))
                return false;
            break;
        case 2:
            if (!read_string(&stream, out.lua_source, sizeof(out.lua_source)))
                return false;
            break;
        case 3:
            if (!read_string(&stream, out.plugin_name, sizeof(out.plugin_name)))
                return false;
            break;
        default:
            if (!pb_skip_field(&stream, wire_type))
                return false;
            break;
        }
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
