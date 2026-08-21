#include <stdint.h>
#include <stddef.h>

static uint32_t xxh32_round(uint32_t seed, uint32_t input) {
    seed += input * 0x85EBCA77U;
    seed = (seed << 13) | (seed >> (32 - 13));
    seed *= 0x9E3779B1U;
    return seed;
}

static uint64_t xxh64_round(uint64_t acc, uint64_t input) {
    acc += input * 0xC2B2AE3D27D4EB4FULL;
    acc = (acc << 31) | (acc >> (64 - 31));
    acc *= 0x9E3779B185EBCA87ULL;
    return acc;
}

static uint64_t xxh64_merge_round(uint64_t acc, uint64_t val) {
    val = xxh64_round(0, val);
    acc ^= val;
    acc = acc * 0x9E3779B185EBCA87ULL + 0x85EBCA77C2B2AE63ULL;
    return acc;
}

uint64_t xxhash64_core(const uint8_t* input, size_t len, uint64_t seed) {
    const uint8_t* p = input;
    const uint8_t* bEnd = p + len;
    uint64_t h64;

    if (len >= 32) {
        const uint8_t* limit = bEnd - 32;
        uint64_t v1 = seed + 0x9E3779B185EBCA87ULL + 0xC2B2AE3D27D4EB4FULL;
        uint64_t v2 = seed + 0xC2B2AE3D27D4EB4FULL;
        uint64_t v3 = seed + 0;
        uint64_t v4 = seed - 0x9E3779B185EBCA87ULL;

        do {
            uint64_t k1 = ((uint64_t)p[0]) | (((uint64_t)p[1])<<8) | (((uint64_t)p[2])<<16) | (((uint64_t)p[3])<<24) |
                          (((uint64_t)p[4])<<32) | (((uint64_t)p[5])<<40) | (((uint64_t)p[6])<<48) | (((uint64_t)p[7])<<56);
            uint64_t k2 = ((uint64_t)p[8]) | (((uint64_t)p[9])<<8) | (((uint64_t)p[10])<<16) | (((uint64_t)p[11])<<24) |
                          (((uint64_t)p[12])<<32) | (((uint64_t)p[13])<<40) | (((uint64_t)p[14])<<48) | (((uint64_t)p[15])<<56);
            uint64_t k3 = ((uint64_t)p[16]) | (((uint64_t)p[17])<<8) | (((uint64_t)p[18])<<16) | (((uint64_t)p[19])<<24) |
                          (((uint64_t)p[20])<<32) | (((uint64_t)p[21])<<40) | (((uint64_t)p[22])<<48) | (((uint64_t)p[23])<<56);
            uint64_t k4 = ((uint64_t)p[24]) | (((uint64_t)p[25])<<8) | (((uint64_t)p[26])<<16) | (((uint64_t)p[27])<<24) |
                          (((uint64_t)p[28])<<32) | (((uint64_t)p[29])<<40) | (((uint64_t)p[30])<<48) | (((uint64_t)p[31])<<56);
            v1 = xxh64_round(v1, k1);
            v2 = xxh64_round(v2, k2);
            v3 = xxh64_round(v3, k3);
            v4 = xxh64_round(v4, k4);
            p += 32;
        } while (p <= limit);

        h64 = ((v1 << 1) | (v1 >> (64 - 1))) +
              ((v2 << 7) | (v2 >> (64 - 7))) +
              ((v3 << 12) | (v3 >> (64 - 12))) +
              ((v4 << 18) | (v4 >> (64 - 18)));
        h64 = xxh64_merge_round(h64, v1);
        h64 = xxh64_merge_round(h64, v2);
        h64 = xxh64_merge_round(h64, v3);
        h64 = xxh64_merge_round(h64, v4);
    } else {
        h64 = seed + 0x27D4EB2F165667C5ULL;
    }

    h64 += (uint64_t)len;

    while (p + 8 <= bEnd) {
        uint64_t k1 = ((uint64_t)p[0]) | (((uint64_t)p[1])<<8) | (((uint64_t)p[2])<<16) | (((uint64_t)p[3])<<24) |
                      (((uint64_t)p[4])<<32) | (((uint64_t)p[5])<<40) | (((uint64_t)p[6])<<48) | (((uint64_t)p[7])<<56);
        k1 = xxh64_round(0, k1);
        h64 ^= k1;
        h64 = ((h64 << 27) | (h64 >> (64 - 27))) * 0x9E3779B185EBCA87ULL + 0x85EBCA77C2B2AE63ULL;
        p += 8;
    }

    if (p + 4 <= bEnd) {
        uint32_t k1 = ((uint32_t)p[0]) | (((uint32_t)p[1])<<8) | (((uint32_t)p[2])<<16) | (((uint32_t)p[3])<<24);
        h64 ^= ((uint64_t)k1) * 0x9E3779B185EBCA87ULL;
        h64 = ((h64 << 23) | (h64 >> (64 - 23))) * 0xC2B2AE3D27D4EB4FULL + 0x165667B19E3779F9ULL;
        p += 4;
    }

    while (p < bEnd) {
        h64 ^= ((uint64_t)(*p)) * 0x27D4EB2F165667C5ULL;
        h64 = ((h64 << 11) | (h64 >> (64 - 11))) * 0x9E3779B185EBCA87ULL;
        p++;
    }

    h64 ^= h64 >> 33;
    h64 *= 0xC2B2AE3D27D4EB4FULL;
    h64 ^= h64 >> 29;
    h64 *= 0x165667B19E3779F9ULL;
    h64 ^= h64 >> 32;

    return h64;
}
