#include <stdint.h>
/* Dogfood sgoiter — utf8 1–2 byte iterate (esprit utf8proc). */

uint64_t utf8_cl_dogfood(uint8_t c) {
    if (c < 0x80u) return 1;
    if ((c & 0xe0u) == 0xc0u) return 2;
    if ((c & 0xf0u) == 0xe0u) return 3;
    if ((c & 0xf8u) == 0xf0u) return 4;
    return 0;
}

/* ASCII / 2-byte only for first dogfood slice; out via return high bits? 
 * returns: low 32 = codepoint, high 32 = nbytes (0 = err as nbytes 0, cp=0xffffffff in low) */
uint64_t utf8_iterate2_dogfood(const uint8_t *str, uint64_t len) {
    uint32_t uc;
    uint64_t cl;
    if (str == 0 || len == 0) {
        return 0xffffffffull;
    }
    cl = utf8_cl_dogfood(str[0]);
    if (cl == 0 || cl > len) {
        return 0xffffffffull;
    }
    if (cl == 1) {
        return ((uint64_t)1 << 32) | (uint64_t)str[0];
    }
    if (cl == 2) {
        if ((str[1] & 0xc0u) != 0x80u) {
            return 0xffffffffull;
        }
        uc = ((uint32_t)(str[0] & 0x1fu) << 6) | (uint32_t)(str[1] & 0x3fu);
        if (uc < 0x80u) {
            return 0xffffffffull;
        }
        return ((uint64_t)2 << 32) | (uint64_t)uc;
    }
    /* 3–4 byte: signal unsupported in this slice via err */
    return 0xffffffffull;
}
