#include <stddef.h>
#include <stdint.h>

#define MASK51 0x7FFFFFFFFFFFFULL

void curve25519_f51_carry(uint64_t t[5]) {
    uint64_t c;
    c = t[0] >> 51; t[0] &= MASK51; t[1] += c;
    c = t[1] >> 51; t[1] &= MASK51; t[2] += c;
    c = t[2] >> 51; t[2] &= MASK51; t[3] += c;
    c = t[3] >> 51; t[3] &= MASK51; t[4] += c;
    c = t[4] >> 51; t[4] &= MASK51; t[0] += c * 19;
    c = t[0] >> 51; t[0] &= MASK51; t[1] += c;
}

void curve25519_f51_add(uint64_t out[5], const uint64_t a[5], const uint64_t b[5]) {
    out[0] = a[0] + b[0];
    out[1] = a[1] + b[1];
    out[2] = a[2] + b[2];
    out[3] = a[3] + b[3];
    out[4] = a[4] + b[4];
    curve25519_f51_carry(out);
}

void curve25519_f51_sub(uint64_t out[5], const uint64_t a[5], const uint64_t b[5]) {
    uint64_t bias0 = (1ULL << 52) - 38;
    uint64_t bias1 = (1ULL << 52) - 2;
    out[0] = (a[0] + bias0) - b[0];
    out[1] = (a[1] + bias1) - b[1];
    out[2] = (a[2] + bias1) - b[2];
    out[3] = (a[3] + bias1) - b[3];
    out[4] = (a[4] + bias1) - b[4];
    curve25519_f51_carry(out);
}

void curve25519_f51_mul121666(uint64_t out[5], const uint64_t in[5]) {
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
