#include <stddef.h>
#include <stdint.h>

static uint32_t L32(uint32_t x, int c) {
    return (x << c) | (x >> (32 - c));
}

static uint32_t ld32(const uint8_t *x) {
    uint32_t u = (uint32_t)x[3];
    u = (u << 8) | (uint32_t)x[2];
    u = (u << 8) | (uint32_t)x[1];
    return (u << 8) | (uint32_t)x[0];
}

static void st32(uint8_t *x, uint32_t u) {
    for (int i = 0; i < 4; i++) {
        x[i] = (uint8_t)u;
        u >>= 8;
    }
}

static void core(uint8_t *out, const uint8_t *in, const uint8_t *k, const uint8_t *c, int h) {
    uint32_t w[16];
    uint32_t x[16];
    uint32_t y[16];
    uint32_t t[4];

    for (int i = 0; i < 4; i++) {
        x[5 * i] = ld32(c + 4 * i);
        x[1 + i] = ld32(k + 4 * i);
        x[6 + i] = ld32(in + 4 * i);
        x[11 + i] = ld32(k + 16 + 4 * i);
    }

    for (int i = 0; i < 16; i++) {
        y[i] = x[i];
    }

    for (int i = 0; i < 20; i++) {
        for (int j = 0; j < 4; j++) {
            for (int m = 0; m < 4; m++) {
                t[m] = x[(5 * j + 4 * m) % 16];
            }
            t[1] ^= L32(t[0] + t[3], 7);
            t[2] ^= L32(t[1] + t[0], 9);
            t[3] ^= L32(t[2] + t[1], 13);
            t[0] ^= L32(t[3] + t[2], 18);
            for (int m = 0; m < 4; m++) {
                w[4 * j + (j + m) % 4] = t[m];
            }
        }
        for (int m = 0; m < 16; m++) {
            x[m] = w[m];
        }
    }

    if (h) {
        for (int i = 0; i < 16; i++) {
            x[i] += y[i];
        }
        for (int i = 0; i < 4; i++) {
            x[5 * i] -= ld32(c + 4 * i);
            x[6 + i] -= ld32(in + 4 * i);
        }
        for (int i = 0; i < 4; i++) {
            st32(out + 4 * i, x[5 * i]);
            st32(out + 16 + 4 * i, x[6 + i]);
        }
    } else {
        for (int i = 0; i < 16; i++) {
            st32(out + 4 * i, x[i] + y[i]);
        }
    }
}

int crypto_core_hsalsa20(uint8_t *out, const uint8_t *in, const uint8_t *k, const uint8_t *c) {
    core(out, in, k, c, 1);
    return 0;
}
