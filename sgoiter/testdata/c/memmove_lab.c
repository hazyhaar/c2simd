#include <stdint.h>

/* classic byte copy loop — dogfood *p++ / do-while */
void fastlz_memmove(uint8_t* dest, const uint8_t* src, uint32_t count) {
  do {
    *dest++ = *src++;
  } while (--count);
}

void fastlz_memcpy(uint8_t* dest, const uint8_t* src, uint32_t count) {
  fastlz_memmove(dest, src, count);
}
