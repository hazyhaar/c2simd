#include <stdint.h>
#include <stddef.h>

/* CRC-32 IEEE 802.3, table-less bit algorithm — exercice shifts/xor. */
uint32_t crc32_ieee(const uint8_t *data, size_t len) {
    uint32_t crc = 0xFFFFFFFFu;
    size_t i;
    int b;
    for (i = 0; i < len; i++) {
        crc ^= data[i];
        for (b = 0; b < 8; b++) {
            uint32_t mask = (uint32_t)-(int32_t)(crc & 1u);
            crc = (crc >> 1) ^ (0xEDB88320u & mask);
        }
    }
    return ~crc;
}
