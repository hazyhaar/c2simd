#include <stddef.h>
#include <stdint.h>

static uint8_t stbi_paeth(int a, int b, int c) {
    int p = a + b - c;
    int pa = p - a; if (pa < 0) pa = -pa;
    int pb = p - b; if (pb < 0) pb = -pb;
    int pc = p - c; if (pc < 0) pc = -pc;
    if (pa <= pb && pa <= pc) return (uint8_t)a;
    if (pb <= pc) return (uint8_t)b;
    return (uint8_t)c;
}

void stbi_unfilter_row(uint8_t *recon, const uint8_t *scanline, const uint8_t *prev, size_t len, int bpp, int filter_type) {
    if (filter_type == 0) {
        for (size_t i = 0; i < len; i++) {
            recon[i] = scanline[i];
        }
    } else if (filter_type == 1) {
        for (size_t i = 0; i < (size_t)bpp && i < len; i++) {
            recon[i] = scanline[i];
        }
        for (size_t i = bpp; i < len; i++) {
            recon[i] = (uint8_t)(scanline[i] + recon[i - bpp]);
        }
    } else if (filter_type == 2) {
        for (size_t i = 0; i < len; i++) {
            recon[i] = (uint8_t)(scanline[i] + prev[i]);
        }
    } else if (filter_type == 3) {
        for (size_t i = 0; i < (size_t)bpp && i < len; i++) {
            recon[i] = (uint8_t)(scanline[i] + (prev[i] >> 1));
        }
        for (size_t i = bpp; i < len; i++) {
            recon[i] = (uint8_t)(scanline[i] + ((recon[i - bpp] + prev[i]) >> 1));
        }
    } else if (filter_type == 4) {
        for (size_t i = 0; i < (size_t)bpp && i < len; i++) {
            recon[i] = (uint8_t)(scanline[i] + stbi_paeth(0, prev[i], 0));
        }
        for (size_t i = bpp; i < len; i++) {
            recon[i] = (uint8_t)(scanline[i] + stbi_paeth(recon[i - bpp], prev[i], prev[i - bpp]));
        }
    }
}
