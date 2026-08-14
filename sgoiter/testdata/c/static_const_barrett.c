#include <stdint.h>
// Mini repro: function-local static const must become package init, not zero local.
static void mod_like(uint8_t out[4], const uint32_t x[4]) {
	static const uint32_t r[2] = {0x0a2c131b, 0xed9ce5a3};
	uint32_t acc = 0;
	for (int i = 0; i < 2; i++) acc += r[i] + x[i];
	out[0] = (uint8_t)acc;
	out[1] = (uint8_t)(acc >> 8);
	out[2] = (uint8_t)(acc >> 16);
	out[3] = (uint8_t)(acc >> 24);
}
void entry(uint8_t out[4], const uint32_t x[4]) { mod_like(out, x); }
