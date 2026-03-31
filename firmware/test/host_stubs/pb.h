/* Minimal nanopb stub for host conformance tests.
 * Only provides types and macros used by subway.pb.h and render_context.h. */

#ifndef PB_H_INCLUDED
#define PB_H_INCLUDED

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>
#include <stddef.h>

typedef uint32_t pb_size_t;
typedef uint8_t pb_byte_t;

#define PB_PROTO_HEADER_VERSION 40

#define pb_membersize(st, m) (sizeof ((st*)0)->m)
#define pb_arraysize(st, m) (pb_membersize(st, m) / pb_membersize(st, m[0]))

/* Opaque message descriptor — only declared, never used in test */
typedef struct pb_msgdesc_s pb_msgdesc_t;

#ifdef __cplusplus
}
#endif

#endif /* PB_H_INCLUDED */
