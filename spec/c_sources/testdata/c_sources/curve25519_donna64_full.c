#include <stddef.h>
#include <stdint.h>

#define MASK51 0x7FFFFFFFFFFFFULL

typedef uint64_t fe51[5];

static uint64_t read64_le(const uint8_t *p) {
    return (uint64_t)p[0] |
           ((uint64_t)p[1] << 8) |
           ((uint64_t)p[2] << 16) |
           ((uint64_t)p[3] << 24) |
           ((uint64_t)p[4] << 32) |
           ((uint64_t)p[5] << 40) |
           ((uint64_t)p[6] << 48) |
           ((uint64_t)p[7] << 56);
}

static void write64_le(uint8_t *p, uint64_t v) {
    p[0] = (uint8_t)v;
    p[1] = (uint8_t)(v >> 8);
    p[2] = (uint8_t)(v >> 16);
    p[3] = (uint8_t)(v >> 24);
    p[4] = (uint8_t)(v >> 32);
    p[5] = (uint8_t)(v >> 40);
    p[6] = (uint8_t)(v >> 48);
    p[7] = (uint8_t)(v >> 56);
}

void fe51_frombytes(fe51 out, const uint8_t in[32]) {
    uint64_t w0 = read64_le(in + 0);
    uint64_t w1 = read64_le(in + 8);
    uint64_t w2 = read64_le(in + 16);
    uint64_t w3 = read64_le(in + 24);

    out[0] = w0 & MASK51;
    out[1] = ((w0 >> 51) | (w1 << 13)) & MASK51;
    out[2] = ((w1 >> 38) | (w2 << 26)) & MASK51;
    out[3] = ((w2 >> 25) | (w3 << 39)) & MASK51;
    out[4] = (w3 >> 12) & MASK51;
}

void fe51_carry(fe51 t) {
    uint64_t c;
    c = t[0] >> 51; t[0] &= MASK51; t[1] += c;
    c = t[1] >> 51; t[1] &= MASK51; t[2] += c;
    c = t[2] >> 51; t[2] &= MASK51; t[3] += c;
    c = t[3] >> 51; t[3] &= MASK51; t[4] += c;
    c = t[4] >> 51; t[4] &= MASK51; t[0] += c * 19;
    c = t[0] >> 51; t[0] &= MASK51; t[1] += c;
}

void fe51_tobytes(uint8_t out[32], const fe51 in) {
    fe51 t;
    for (int i = 0; i < 5; i++) {
        t[i] = in[i];
    }
    fe51_carry(t);

    uint64_t f0 = t[0];
    uint64_t f1 = t[1];
    uint64_t f2 = t[2];
    uint64_t f3 = t[3];
    uint64_t f4 = t[4];

    write64_le(out + 0, f0 | (f1 << 51));
    write64_le(out + 8, (f1 >> 13) | (f2 << 38));
    write64_le(out + 16, (f2 >> 26) | (f3 << 25));
    write64_le(out + 24, (f3 >> 39) | (f4 << 12));
}

void fe51_0(fe51 out) {
    for (int i = 0; i < 5; i++) {
        out[i] = 0;
    }
}

void fe51_1(fe51 out) {
    out[0] = 1;
    out[1] = 0;
    out[2] = 0;
    out[3] = 0;
    out[4] = 0;
}

void fe51_copy(fe51 out, const fe51 in) {
    for (int i = 0; i < 5; i++) {
        out[i] = in[i];
    }
}

void fe51_cswap(fe51 a, fe51 b, int swap) {
    uint64_t mask = (uint64_t)(-(int64_t)(swap != 0));
    for (int i = 0; i < 5; i++) {
        uint64_t x = mask & (a[i] ^ b[i]);
        a[i] ^= x;
        b[i] ^= x;
    }
}

void fe51_add(fe51 out, const fe51 a, const fe51 b) {
    out[0] = a[0] + b[0];
    out[1] = a[1] + b[1];
    out[2] = a[2] + b[2];
    out[3] = a[3] + b[3];
    out[4] = a[4] + b[4];
    fe51_carry(out);
}

void fe51_sub(fe51 out, const fe51 a, const fe51 b) {
    uint64_t bias0 = (1ULL << 52) - 38;
    uint64_t bias1 = (1ULL << 52) - 2;
    out[0] = (a[0] + bias0) - b[0];
    out[1] = (a[1] + bias1) - b[1];
    out[2] = (a[2] + bias1) - b[2];
    out[3] = (a[3] + bias1) - b[3];
    out[4] = (a[4] + bias1) - b[4];
    fe51_carry(out);
}

void fe51_mul121666(fe51 out, const fe51 in) {
    uint64_t a0 = in[0] * 121666ULL;
    uint64_t a1 = in[1] * 121666ULL;
    uint64_t a2 = in[2] * 121666ULL;
    uint64_t a3 = in[3] * 121666ULL;
    uint64_t a4 = in[4] * 121666ULL;

    uint64_t c;
    c = a0 >> 51; a0 &= MASK51; a1 += c;
    c = a1 >> 51; a1 &= MASK51; a2 += c;
    c = a2 >> 51; a2 &= MASK51; a3 += c;
    c = a3 >> 51; a3 &= MASK51; a4 += c;
    c = a4 >> 51; a4 &= MASK51; a0 += c * 19;
    c = a0 >> 51; a0 &= MASK51; a1 += c;

    out[0] = a0;
    out[1] = a1;
    out[2] = a2;
    out[3] = a3;
    out[4] = a4;
}

static int scalar_bit(const uint8_t s[32], int i) {
    return (s[i >> 3] >> (i & 7)) & 1;
}

void curve25519_scalarmult51(uint8_t q[32], const uint8_t scalar[32], const uint8_t p[32], int nb_bits) {
    fe51 x1, x2, z2, x3, z3, a, b, c, d, e, aa, bb, da, cb;

    fe51_frombytes(x1, p);
    fe51_1(x2);
    fe51_0(z2);
    fe51_copy(x3, x1);
    fe51_1(z3);

    int swap = 0;
    for (int pos = nb_bits - 1; pos >= 0; pos--) {
        int b_bit = scalar_bit(scalar, pos);
        swap ^= b_bit;
        fe51_cswap(x2, x3, swap);
        fe51_cswap(z2, z3, swap);
        swap = b_bit;

        fe51_add(a, x2, z2);
        fe51_sub(b, x2, z2);
        fe51_add(c, x3, z3);
        fe51_sub(d, x3, z3);

        fe51_add(aa, a, b);
        fe51_sub(bb, a, b);
        fe51_add(da, d, a);
        fe51_sub(cb, c, b);

        fe51_add(x3, da, cb);
        fe51_sub(z3, da, cb);
        fe51_mul121666(e, bb);
        fe51_add(z2, aa, e);
        fe51_copy(x2, aa);
    }
    fe51_cswap(x2, x3, swap);
    fe51_cswap(z2, z3, swap);

    fe51_tobytes(q, x2);
}
