#include <stdint.h>
#include <stddef.h>

uint32_t miniz_adler32(const uint8_t *ptr, size_t buf_len) {
    uint32_t adler = 1U;
    uint32_t s1 = (uint32_t)(adler & 0xffff), s2 = (uint32_t)(adler >> 16);
    size_t block_len = buf_len % 5552;
    if (!ptr)
        return 1U;
    while (buf_len) {
        size_t i = 0;
        for (i = 0; i + 7 < block_len; i += 8, ptr += 8) {
            s1 += ptr[0], s2 += s1;
            s1 += ptr[1], s2 += s1;
            s1 += ptr[2], s2 += s1;
            s1 += ptr[3], s2 += s1;
            s1 += ptr[4], s2 += s1;
            s1 += ptr[5], s2 += s1;
            s1 += ptr[6], s2 += s1;
            s1 += ptr[7], s2 += s1;
        }
        for (; i < block_len; ++i)
            s1 += *ptr++, s2 += s1;
        s1 %= 65521U, s2 %= 65521U;
        buf_len -= block_len;
        block_len = 5552;
    }
    return (s2 << 16) + s1;
}
