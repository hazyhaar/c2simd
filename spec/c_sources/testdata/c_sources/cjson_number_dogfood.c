#include <stdint.h>
/* Dogfood sgoiter — extrait esprit cJSON parse_number (DaveGamble/cJSON), borné ASCII. */
/* parse unsigned decimal integer; returns value, sets *nconsumed */
uint64_t cjson_parse_u64_dogfood(const uint8_t *s, uint64_t len, uint64_t *nconsumed) {
    uint64_t i = 0, v = 0;
    if (nconsumed) *nconsumed = 0;
    if (s == 0 || len == 0) return 0;
    while (i < len && s[i] >= (uint8_t)'0' && s[i] <= (uint8_t)'9') {
        v = v * 10u + (uint64_t)(s[i] - (uint8_t)'0');
        i++;
    }
    if (nconsumed) *nconsumed = i;
    return v;
}

/* true if buffer is JSON literal null */
uint64_t cjson_is_null_lit_dogfood(const uint8_t *s, uint64_t len) {
    if (s == 0 || len < 4) return 0;
    if (s[0]=='n' && s[1]=='u' && s[2]=='l' && s[3]=='l') return 1;
    return 0;
}
