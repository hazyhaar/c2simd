#include <stdint.h>
#include <stddef.h>

/* QuickJS helper: min/max fold over buffer */
uint64_t quickjs_minmax_fold(const uint8_t *buf, uint64_t len) {
    uint32_t min_v = 0xffffffffu;
    uint32_t max_v = 0u;
    uint64_t i;
    for (i = 0; i < len; i++) {
        uint32_t v = (uint32_t)buf[i];
        if (v < min_v) min_v = v;
        if (v > max_v) max_v = v;
    }
    return ((uint64_t)max_v << 32) | (uint64_t)min_v;
}
