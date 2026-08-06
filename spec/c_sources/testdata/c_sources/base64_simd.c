#include <stdint.h>
#include <stddef.h>

static const char b64_table[] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

size_t base64_encode_stream(const uint8_t *src, size_t len, char *dst) {
    size_t i, j = 0;
    for (i = 0; i + 2 < len; i += 3) {
        uint32_t triple = (src[i] << 16) | (src[i+1] << 8) | src[i+2];
        dst[j++] = b64_table[(triple >> 18) & 0x3F];
        dst[j++] = b64_table[(triple >> 12) & 0x3F];
        dst[j++] = b64_table[(triple >> 6) & 0x3F];
        dst[j++] = b64_table[triple & 0x3F];
    }
    if (i < len) {
        uint32_t triple = src[i] << 16;
        if (i + 1 < len) triple |= src[i+1] << 8;
        dst[j++] = b64_table[(triple >> 18) & 0x3F];
        dst[j++] = b64_table[(triple >> 12) & 0x3F];
        if (i + 1 < len) {
            dst[j++] = b64_table[(triple >> 6) & 0x3F];
        } else {
            dst[j++] = '=';
        }
        dst[j++] = '=';
    }
    return j;
}
