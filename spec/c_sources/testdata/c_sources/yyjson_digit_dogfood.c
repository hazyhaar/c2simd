#include <stdint.h>
/* Dogfood sgoiter — scan digits / ws (esprit yyjson), while compound only. */

uint64_t yyjson_count_digits_dogfood(const uint8_t *s, uint64_t len) {
    uint64_t i = 0;
    if (s == 0) return 0;
    while (i < len && s[i] >= (uint8_t)'0' && s[i] <= (uint8_t)'9') {
        i = i + 1;
    }
    return i;
}

uint64_t yyjson_skip_ws_dogfood(const uint8_t *s, uint64_t len) {
    uint64_t i = 0;
    uint8_t c;
    if (s == 0) return 0;
    while (i < len) {
        c = s[i];
        if (c != (uint8_t)' ' && c != (uint8_t)'\t' && c != (uint8_t)'\n' && c != (uint8_t)'\r') {
            return i;
        }
        i = i + 1;
    }
    return i;
}
