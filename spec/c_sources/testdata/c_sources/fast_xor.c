#include <stdint.h>
#include <stddef.h>

void fast_xor_bytes(uint8_t *dst, const uint8_t *src1, const uint8_t *src2, size_t len) {
    size_t i = 0;
    for (; i + 8 <= len; i += 8) {
        uint64_t w1 = ((uint64_t*)src1)[i/8];
        uint64_t w2 = ((uint64_t*)src2)[i/8];
        ((uint64_t*)dst)[i/8] = w1 ^ w2;
    }
    for (; i < len; i++) {
        dst[i] = src1[i] ^ src2[i];
    }
}
