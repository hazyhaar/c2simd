#include <stddef.h>
#include <stdint.h>

static uint64_t rotl64(uint64_t x, int8_t r) {
    return (x << r) | (x >> (64 - r));
}

static uint64_t fmix64(uint64_t k) {
    k ^= k >> 33;
    k *= 0xff51afd7ed558ccdULL;
    k ^= k >> 33;
    k *= 0xc4ceb9fe1a85ec53ULL;
    k ^= k >> 33;
    return k;
}

void murmur3_x64_128(const uint8_t *key, size_t len, uint32_t seed, uint64_t out[2]) {
    size_t nblocks = len / 16;
    uint64_t h1 = (uint64_t)seed;
    uint64_t h2 = (uint64_t)seed;

    uint64_t c1 = 0x87c37b91114253d5ULL;
    uint64_t c2 = 0x4cf5ad432745937fULL;

    for (size_t i = 0; i < nblocks; i++) {
        size_t off = i * 16;
        uint64_t k1 = (uint64_t)key[off] | ((uint64_t)key[off+1] << 8) | ((uint64_t)key[off+2] << 16) | ((uint64_t)key[off+3] << 24) |
                      ((uint64_t)key[off+4] << 32) | ((uint64_t)key[off+5] << 40) | ((uint64_t)key[off+6] << 48) | ((uint64_t)key[off+7] << 56);
        uint64_t k2 = (uint64_t)key[off+8] | ((uint64_t)key[off+9] << 8) | ((uint64_t)key[off+10] << 16) | ((uint64_t)key[off+11] << 24) |
                      ((uint64_t)key[off+12] << 32) | ((uint64_t)key[off+13] << 40) | ((uint64_t)key[off+14] << 48) | ((uint64_t)key[off+15] << 56);

        k1 *= c1; k1 = rotl64(k1, 31); k1 *= c2; h1 ^= k1;
        h1 = rotl64(h1, 27); h1 += h2; h1 = h1 * 5 + 0x52dce729ULL;

        k2 *= c2; k2 = rotl64(k2, 33); k2 *= c1; h2 ^= k2;
        h2 = rotl64(h2, 31); h2 += h1; h2 = h2 * 5 + 0x38495ab5ULL;
    }

    size_t tail_off = nblocks * 16;
    uint64_t k1 = 0;
    uint64_t k2 = 0;
    size_t rem = len & 15;

    if (rem >= 15) k2 ^= (uint64_t)key[tail_off + 14] << 48;
    if (rem >= 14) k2 ^= (uint64_t)key[tail_off + 13] << 40;
    if (rem >= 13) k2 ^= (uint64_t)key[tail_off + 12] << 32;
    if (rem >= 12) k2 ^= (uint64_t)key[tail_off + 11] << 24;
    if (rem >= 11) k2 ^= (uint64_t)key[tail_off + 10] << 16;
    if (rem >= 10) k2 ^= (uint64_t)key[tail_off + 9] << 8;
    if (rem >= 9) {
        k2 ^= (uint64_t)key[tail_off + 8];
        k2 *= c2;
        k2 = rotl64(k2, 33);
        k2 *= c1;
        h2 ^= k2;
    }

    if (rem >= 8) k1 ^= (uint64_t)key[tail_off + 7] << 56;
    if (rem >= 7) k1 ^= (uint64_t)key[tail_off + 6] << 48;
    if (rem >= 6) k1 ^= (uint64_t)key[tail_off + 5] << 40;
    if (rem >= 5) k1 ^= (uint64_t)key[tail_off + 4] << 32;
    if (rem >= 4) k1 ^= (uint64_t)key[tail_off + 3] << 24;
    if (rem >= 3) k1 ^= (uint64_t)key[tail_off + 2] << 16;
    if (rem >= 2) k1 ^= (uint64_t)key[tail_off + 1] << 8;
    if (rem >= 1) {
        k1 ^= (uint64_t)key[tail_off];
        k1 *= c1;
        k1 = rotl64(k1, 31);
        k1 *= c2;
        h1 ^= k1;
    }

    h1 ^= (uint64_t)len;
    h2 ^= (uint64_t)len;

    h1 += h2;
    h2 += h1;

    h1 = fmix64(h1);
    h2 = fmix64(h2);

    h1 += h2;
    h2 += h1;

    out[0] = h1;
    out[1] = h2;
}
