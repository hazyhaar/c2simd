#include <stdint.h>
#include <stddef.h>

typedef struct {
    uint32_t count[2];
    uint32_t state[4];
    uint8_t buffer[64];
} MD5_CTX_RAW;

#define F(x, y, z) (((x) & (y)) | ((~x) & (z)))
#define G(x, y, z) (((x) & (z)) | ((y) & (~z)))
#define H(x, y, z) ((x) ^ (y) ^ (z))
#define I(x, y, z) ((y) ^ ((x) | (~z)))

#define ROTATE_LEFT(x, n) (((x) << (n)) | ((x) >> (32-(n))))

#define FF(a, b, c, d, x, s, ac) { \
 (a) += F ((b), (c), (d)) + (x) + (uint32_t)(ac); \
 (a) = ROTATE_LEFT ((a), (s)); \
 (a) += (b); \
  }

void md5_transform_block(uint32_t state[4], const uint8_t block[64]) {
    uint32_t a = state[0], b = state[1], c = state[2], d = state[3], x[16];
    int i;
    for (i = 0; i < 16; i++) {
        x[i] = ((uint32_t*)block)[i];
    }
    FF (a, b, c, d, x[0], 7, 0xd76aa478);
    FF (d, a, b, c, x[1], 12, 0xe8c7b756);
    FF (c, d, a, b, x[2], 17, 0x242070db);
    FF (b, c, d, a, x[3], 22, 0xc1bdceee);
    state[0] += a; state[1] += b; state[2] += c; state[3] += d;
}
