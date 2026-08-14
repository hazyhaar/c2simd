#include <stdint.h>
#define HASH_LOG 13
#define HASH_SIZE (1 << HASH_LOG)
#define HASH_MASK (HASH_SIZE - 1)

static uint16_t flz_hash(uint32_t v) {
  uint32_t h = (v * 2654435769LL) >> (32 - HASH_LOG);
  return h & HASH_MASK;
}

static uint32_t flz_readu32(const void* ptr) {
  return *(const uint32_t*)ptr;
}

static uint32_t flz_readu32_idx(const uint8_t* p, int off) {
  return *(const uint32_t*)(p + off);
}
