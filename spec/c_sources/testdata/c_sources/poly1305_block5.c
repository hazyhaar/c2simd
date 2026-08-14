#include <stdint.h>
#include <stddef.h>

/*
 * Un pas d'accumulation Poly1305 style 5x26 (Donna-like simplifié).
 * Exercice : mul 64, and/masks, pas de rotate — levier mul/ILP futur.
 * r[5], h[5] limbs 26-bit ; m[4] = 16 bytes LE en 4x32.
 */
void poly1305_block5(uint32_t h[5], const uint32_t r[5], const uint32_t m[4]) {
    uint64_t d0, d1, d2, d3, d4;
    uint64_t c;

    d0 = ((uint64_t)h[0] + m[0]) * r[0] +
         ((uint64_t)h[1] + 0) * (5 * r[4]) +
         ((uint64_t)h[2] + 0) * (5 * r[3]) +
         ((uint64_t)h[3] + 0) * (5 * r[2]) +
         ((uint64_t)h[4] + 0) * (5 * r[1]);
    d1 = ((uint64_t)h[0] + m[0]) * r[1] +
         ((uint64_t)h[1] + 0) * r[0] +
         ((uint64_t)h[2] + 0) * (5 * r[4]) +
         ((uint64_t)h[3] + 0) * (5 * r[3]) +
         ((uint64_t)h[4] + 0) * (5 * r[2]);
    d2 = ((uint64_t)h[0] + m[1]) * r[2] +
         ((uint64_t)h[1] + 0) * r[1] +
         ((uint64_t)h[2] + 0) * r[0] +
         ((uint64_t)h[3] + 0) * (5 * r[4]) +
         ((uint64_t)h[4] + 0) * (5 * r[3]);
    d3 = ((uint64_t)h[0] + m[2]) * r[3] +
         ((uint64_t)h[1] + 0) * r[2] +
         ((uint64_t)h[2] + 0) * r[1] +
         ((uint64_t)h[3] + 0) * r[0] +
         ((uint64_t)h[4] + 0) * (5 * r[4]);
    d4 = ((uint64_t)h[0] + m[3]) * r[4] +
         ((uint64_t)h[1] + 0) * r[3] +
         ((uint64_t)h[2] + 0) * r[2] +
         ((uint64_t)h[3] + 0) * r[1] +
         ((uint64_t)h[4] + 0) * r[0];

    c = d0 >> 26; h[0] = (uint32_t)d0 & 0x3ffffff; d1 += c;
    c = d1 >> 26; h[1] = (uint32_t)d1 & 0x3ffffff; d2 += c;
    c = d2 >> 26; h[2] = (uint32_t)d2 & 0x3ffffff; d3 += c;
    c = d3 >> 26; h[3] = (uint32_t)d3 & 0x3ffffff; d4 += c;
    c = d4 >> 26; h[4] = (uint32_t)d4 & 0x3ffffff;
    h[0] += (uint32_t)c * 5;
    c = h[0] >> 26; h[0] &= 0x3ffffff; h[1] += (uint32_t)c;
}
