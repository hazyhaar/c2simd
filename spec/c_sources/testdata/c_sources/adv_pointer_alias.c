#include <stdint.h>
#include <stddef.h>
/* Adversarial lab: strict-aliasing games + type punning — transpile stress.
 * NOT an exploit. Documents AVOID patterns post-ccgo. */
uint32_t adv_alias_load32(const uint8_t *p) {
    /* classic unaligned pun — UB in C strict, common in codecs */
    return *(const uint32_t *)p;
}
void adv_alias_store32(uint8_t *p, uint32_t v) {
    *(uint32_t *)p = v;
}
/* overlapping memcpy-like with pointer arithmetic */
void adv_overlap_xor(uint8_t *dst, const uint8_t *src, size_t n) {
    size_t i;
    for (i = 0; i < n; i++)
        dst[i] ^= src[i];
}
