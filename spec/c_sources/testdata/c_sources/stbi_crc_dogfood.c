#include <stdint.h>
/* Dogfood sgoiter — CRC32 slice style stb_image PNG helper (nothings/stb), table-free bit version. */
uint32_t stbi_crc32_dogfood(const uint8_t *buf, uint64_t len) {
    uint32_t crc = 0xffffffffu;
    uint64_t i, j;
    for (i = 0; i < len; i++) {
        crc ^= (uint32_t)buf[i];
        for (j = 0; j < 8; j++) {
            uint32_t mask = -(crc & 1u);
            crc = (crc >> 1) ^ (0xedb88320u & mask);
        }
    }
    return ~crc;
}
