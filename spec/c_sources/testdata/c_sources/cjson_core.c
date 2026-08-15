#include <stddef.h>
#include <stdint.h>

/* cJSON case-insensitive string comparison and hex4 parser */

static unsigned char to_lower(unsigned char c) {
    if (c >= 'A' && c <= 'Z') {
        return (unsigned char)(c + ('a' - 'A'));
    }
    return c;
}

int cjson_casecmp(const uint8_t *s1, const uint8_t *s2, size_t n) {
    for (size_t i = 0; i < n; i++) {
        unsigned char c1 = to_lower(s1[i]);
        unsigned char c2 = to_lower(s2[i]);
        if (c1 != c2) {
            return (int)c1 - (int)c2;
        }
        if (c1 == 0) {
            return 0;
        }
    }
    return 0;
}

uint32_t cjson_parse_hex4(const uint8_t *s) {
    uint32_t val = 0;
    for (size_t i = 0; i < 4; i++) {
        uint8_t c = s[i];
        uint32_t d = 0;
        if (c >= '0' && c <= '9') {
            d = (uint32_t)(c - '0');
        } else if (c >= 'a' && c <= 'f') {
            d = (uint32_t)(c - 'a' + 10);
        } else if (c >= 'A' && c <= 'F') {
            d = (uint32_t)(c - 'A' + 10);
        } else {
            return 0;
        }
        val = (val << 4) | d;
    }
    return val;
}
