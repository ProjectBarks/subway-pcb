/* Automatically generated nanopb header */
/* Generated from proto/subway.proto - PixelFrame only */
#ifndef PB_SUBWAY_SUBWAY_PB_H_INCLUDED
#define PB_SUBWAY_SUBWAY_PB_H_INCLUDED
#include <pb.h>

#if PB_PROTO_HEADER_VERSION != 40
#error Regenerate this file with the current version of nanopb generator.
#endif

/* Bytes type for pixel data */
typedef PB_BYTES_ARRAY_T(1434) subway_PixelFrame_pixels_t;

/* PixelFrame: pre-rendered LED data from server */
typedef struct _subway_PixelFrame {
    uint64_t timestamp;
    uint32_t sequence;
    uint32_t led_count;
    subway_PixelFrame_pixels_t pixels;
} subway_PixelFrame;

#ifdef __cplusplus
extern "C" {
#endif

/* Initializer values */
#define subway_PixelFrame_init_default {0, 0, 0, {0, {0}}}
#define subway_PixelFrame_init_zero    {0, 0, 0, {0, {0}}}

/* Field tags */
#define subway_PixelFrame_timestamp_tag  1
#define subway_PixelFrame_sequence_tag   2
#define subway_PixelFrame_led_count_tag  3
#define subway_PixelFrame_pixels_tag     4

/* Field encoding specification for nanopb */
#define subway_PixelFrame_FIELDLIST(X, a) \
X(a, STATIC,   SINGULAR, UINT64,   timestamp,   1) \
X(a, STATIC,   SINGULAR, UINT32,   sequence,    2) \
X(a, STATIC,   SINGULAR, UINT32,   led_count,   3) \
X(a, STATIC,   SINGULAR, BYTES,    pixels,      4)
#define subway_PixelFrame_CALLBACK NULL
#define subway_PixelFrame_DEFAULT NULL

extern const pb_msgdesc_t subway_PixelFrame_msg;

#define subway_PixelFrame_fields &subway_PixelFrame_msg

/* Maximum encoded size */
#define subway_PixelFrame_size 1450
#define SUBWAY_SUBWAY_PB_H_MAX_SIZE subway_PixelFrame_size

#ifdef __cplusplus
} /* extern "C" */
#endif

#endif
