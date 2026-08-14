#include <stdint.h>
/* Adversarial lab: deep tls-only call chain (T0 fixed-point stress). */
static uint32_t leaf(uint32_t x) { return x * 0x9e3779b1u ^ (x >> 16); }
static uint32_t m1(uint32_t x) { return leaf(x + 1); }
static uint32_t m2(uint32_t x) { return m1(x ^ 0x85ebca6bu); }
static uint32_t m3(uint32_t x) { return m2(x + 0xc2b2ae35u); }
static uint32_t m4(uint32_t x) { return m3(x ^ (x << 13)); }
uint32_t adv_tls_depth(uint32_t x) { return m4(x); }
