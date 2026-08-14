#include <stdint.h>

/* ChaCha20 quarter round — ROTL32 dense, cœur crypto hors monocypher. */
#define ROTL32(v, n) (((v) << (n)) | ((v) >> (32 - (n))))

void chacha20_quarter_round(uint32_t *a, uint32_t *b, uint32_t *c, uint32_t *d) {
    *a += *b; *d ^= *a; *d = ROTL32(*d, 16);
    *c += *d; *b ^= *c; *b = ROTL32(*b, 12);
    *a += *b; *d ^= *a; *d = ROTL32(*d, 8);
    *c += *d; *b ^= *c; *b = ROTL32(*b, 7);
}

/* Un double-round sur 16 words (in-place), 2 colonnes + 2 diagonales. */
void chacha20_double_round(uint32_t x[16]) {
    chacha20_quarter_round(&x[0], &x[4], &x[8], &x[12]);
    chacha20_quarter_round(&x[1], &x[5], &x[9], &x[13]);
    chacha20_quarter_round(&x[2], &x[6], &x[10], &x[14]);
    chacha20_quarter_round(&x[3], &x[7], &x[11], &x[15]);
    chacha20_quarter_round(&x[0], &x[5], &x[10], &x[15]);
    chacha20_quarter_round(&x[1], &x[6], &x[11], &x[12]);
    chacha20_quarter_round(&x[2], &x[7], &x[8], &x[13]);
    chacha20_quarter_round(&x[3], &x[4], &x[9], &x[14]);
}
