#include <stddef.h>
#include <stdint.h>

#define UTF8PROC_ERROR_INVALIDUTF8 -1

static int utf_cont(uint8_t ch) {
    return (ch & 0xc0) == 0x80;
}

int64_t utf8proc_iterate(const uint8_t *str, size_t strlen, int32_t *dst) {
    int32_t uc;
    *dst = -1;
    if (strlen == 0) {
        return 0;
    }
    uc = str[0];
    if (uc < 0x80) {
        *dst = uc;
        return 1;
    }
    if ((uint32_t)(uc - 0xc2) > (0xf4 - 0xc2)) {
        return UTF8PROC_ERROR_INVALIDUTF8;
    }
    if (uc < 0xe0) {
        if (strlen < 2 || !utf_cont(str[1])) {
            return UTF8PROC_ERROR_INVALIDUTF8;
        }
        *dst = ((uc & 0x1f) << 6) | (str[1] & 0x3f);
        return 2;
    }
    if (uc < 0xf0) {
        if (strlen < 3 || !utf_cont(str[1]) || !utf_cont(str[2])) {
            return UTF8PROC_ERROR_INVALIDUTF8;
        }
        if (uc == 0xed && str[1] > 0x9f) {
            return UTF8PROC_ERROR_INVALIDUTF8;
        }
        uc = ((uc & 0xf) << 12) | ((str[1] & 0x3f) << 6) | (str[2] & 0x3f);
        if (uc < 0x800) {
            return UTF8PROC_ERROR_INVALIDUTF8;
        }
        *dst = uc;
        return 3;
    }
    if (strlen < 4 || !utf_cont(str[1]) || !utf_cont(str[2]) || !utf_cont(str[3])) {
        return UTF8PROC_ERROR_INVALIDUTF8;
    }
    if (uc == 0xf0) {
        if (str[1] < 0x90) return UTF8PROC_ERROR_INVALIDUTF8;
    } else if (uc == 0xf4) {
        if (str[1] > 0x8f) return UTF8PROC_ERROR_INVALIDUTF8;
    }
    *dst = ((uc & 7) << 18) | ((str[1] & 0x3f) << 12) | ((str[2] & 0x3f) << 6) | (str[3] & 0x3f);
    return 4;
}

int64_t utf8proc_encode_char(int32_t uc, uint8_t *dst) {
    if (uc < 0) {
        return 0;
    } else if (uc < 0x80) {
        dst[0] = (uint8_t)uc;
        return 1;
    } else if (uc < 0x800) {
        dst[0] = (uint8_t)(0xC0 + (uc >> 6));
        dst[1] = (uint8_t)(0x80 + (uc & 0x3F));
        return 2;
    } else if (uc < 0x10000) {
        dst[0] = (uint8_t)(0xE0 + (uc >> 12));
        dst[1] = (uint8_t)(0x80 + ((uc >> 6) & 0x3F));
        dst[2] = (uint8_t)(0x80 + (uc & 0x3F));
        return 3;
    } else if (uc < 0x110000) {
        dst[0] = (uint8_t)(0xF0 + (uc >> 18));
        dst[1] = (uint8_t)(0x80 + ((uc >> 12) & 0x3F));
        dst[2] = (uint8_t)(0x80 + ((uc >> 6) & 0x3F));
        dst[3] = (uint8_t)(0x80 + (uc & 0x3F));
        return 4;
    }
    return 0;
}
