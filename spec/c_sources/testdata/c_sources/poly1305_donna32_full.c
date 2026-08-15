#include <stddef.h>
#include <stdint.h>

static uint32_t read32_le(const uint8_t *p) {
    return (uint32_t)p[0] |
           ((uint32_t)p[1] << 8) |
           ((uint32_t)p[2] << 16) |
           ((uint32_t)p[3] << 24);
}

typedef struct {
    uint32_t r[5];
    uint32_t h[5];
    uint32_t pad[4];
    size_t leftover;
    uint8_t buffer[16];
    uint8_t final;
} poly1305_context;

void poly1305_donna32_blocks(poly1305_context *ctx, const uint8_t *m, size_t bytes, uint32_t hibit) {
    uint32_t r0 = ctx->r[0];
    uint32_t r1 = ctx->r[1];
    uint32_t r2 = ctx->r[2];
    uint32_t r3 = ctx->r[3];
    uint32_t r4 = ctx->r[4];

    uint32_t s1 = r1 * 5;
    uint32_t s2 = r2 * 5;
    uint32_t s3 = r3 * 5;
    uint32_t s4 = r4 * 5;

    uint32_t h0 = ctx->h[0];
    uint32_t h1 = ctx->h[1];
    uint32_t h2 = ctx->h[2];
    uint32_t h3 = ctx->h[3];
    uint32_t h4 = ctx->h[4];

    while (bytes >= 16) {
        uint32_t w0 = read32_le(m + 0);
        uint32_t w1 = read32_le(m + 3);
        uint32_t w2 = read32_le(m + 6);
        uint32_t w3 = read32_le(m + 9);
        uint32_t w4 = read32_le(m + 12);

        h0 += (w0 & 0x3ffffff);
        h1 += ((w1 >> 2) & 0x3ffffff);
        h2 += ((w2 >> 4) & 0x3ffffff);
        h3 += ((w3 >> 6) & 0x3ffffff);
        h4 += ((w4 >> 8) | hibit);

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
        h0 += (uint32_t)c * 5;

        c = h0 >> 26;
        h0 &= 0x3ffffff;
        h1 += (uint32_t)c;

        m += 16;
        bytes -= 16;
    }

    ctx->h[0] = h0;
    ctx->h[1] = h1;
    ctx->h[2] = h2;
    ctx->h[3] = h3;
    ctx->h[4] = h4;
}

void poly1305_donna32_init(poly1305_context *ctx, const uint8_t key[32]) {
    uint32_t t0 = read32_le(key + 0);
    uint32_t t1 = read32_le(key + 4);
    uint32_t t2 = read32_le(key + 8);
    uint32_t t3 = read32_le(key + 12);

    ctx->r[0] = t0 & 0x3ffffff;
    ctx->r[1] = ((t0 >> 26) | (t1 << 6)) & 0x3ffff03;
    ctx->r[2] = ((t1 >> 20) | (t2 << 12)) & 0x3ffc0ff;
    ctx->r[3] = ((t2 >> 14) | (t3 << 18)) & 0x3f03fff;
    ctx->r[4] = (t3 >> 8) & 0x00fffff;

    ctx->h[0] = 0;
    ctx->h[1] = 0;
    ctx->h[2] = 0;
    ctx->h[3] = 0;
    ctx->h[4] = 0;

    ctx->pad[0] = read32_le(key + 16);
    ctx->pad[1] = read32_le(key + 20);
    ctx->pad[2] = read32_le(key + 24);
    ctx->pad[3] = read32_le(key + 28);

    ctx->leftover = 0;
    ctx->final = 0;
}
