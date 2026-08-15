#include <stddef.h>
#include <stdint.h>

/* Poly1305 Donna-32 implementation in 26-bit limbs */

static uint32_t read32_le(const uint8_t *p) {
    return (uint32_t)p[0] |
           ((uint32_t)p[1] << 8) |
           ((uint32_t)p[2] << 16) |
           ((uint32_t)p[3] << 24);
}

void poly1305_donna32_block(uint32_t h[5], const uint32_t r[5], const uint8_t in[16], uint32_t hibit) {
    uint32_t r0 = r[0];
    uint32_t r1 = r[1];
    uint32_t r2 = r[2];
    uint32_t r3 = r[3];
    uint32_t r4 = r[4];

    uint32_t s1 = r1 * 5;
    uint32_t s2 = r2 * 5;
    uint32_t s3 = r3 * 5;
    uint32_t s4 = r4 * 5;

    uint32_t w0 = read32_le(in + 0);
    uint32_t w1 = read32_le(in + 3);
    uint32_t w2 = read32_le(in + 6);
    uint32_t w3 = read32_le(in + 9);
    uint32_t w4 = read32_le(in + 12);

    uint32_t h0 = h[0] + (w0 & 0x3ffffff);
    uint32_t h1 = h[1] + ((w1 >> 2) & 0x3ffffff);
    uint32_t h2 = h[2] + ((w2 >> 4) & 0x3ffffff);
    uint32_t h3 = h[3] + ((w3 >> 6) & 0x3ffffff);
    uint32_t h4 = h[4] + ((w4 >> 8) | hibit);

    uint64_t d0 = (uint64_t)h0 * r0 + (uint64_t)h1 * s4 + (uint64_t)h2 * s3 + (uint64_t)h3 * s2 + (uint64_t)h4 * s1;
    uint64_t d1 = (uint64_t)h0 * r1 + (uint64_t)h1 * r0 + (uint64_t)h2 * s4 + (uint64_t)h3 * s3 + (uint64_t)h4 * s2;
    uint64_t d2 = (uint64_t)h0 * r2 + (uint64_t)h1 * r1 + (uint64_t)h2 * r0 + (uint64_t)h3 * s4 + (uint64_t)h4 * s3;
    uint64_t d3 = (uint64_t)h0 * r3 + (uint64_t)h1 * r2 + (uint64_t)h2 * r1 + (uint64_t)h3 * r0 + (uint64_t)h4 * s4;
    uint64_t d4 = (uint64_t)h0 * r4 + (uint64_t)h1 * r3 + (uint64_t)h2 * r2 + (uint64_t)h3 * r1 + (uint64_t)h4 * r0;

    uint64_t c = d0 >> 26;
    h0 = (uint32_t)d0 & 0x3ffffff;
    d1 += c;

    c = d1 >> 26;
    h1 = (uint32_t)d1 & 0x3ffffff;
    d2 += c;

    c = d2 >> 26;
    h2 = (uint32_t)d2 & 0x3ffffff;
    d3 += c;

    c = d3 >> 26;
    h3 = (uint32_t)d3 & 0x3ffffff;
    d4 += c;

    c = d4 >> 26;
    h4 = (uint32_t)d4 & 0x3ffffff;
    h0 += (uint32_t)(c * 5);

    c = (uint64_t)h0 >> 26;
    h0 &= 0x3ffffff;
    h1 += (uint32_t)c;

    h[0] = h0;
    h[1] = h1;
    h[2] = h2;
    h[3] = h3;
    h[4] = h4;
}
