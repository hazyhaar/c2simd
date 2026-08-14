#include <stdint.h>
#include <stddef.h>
/* Adversarial lab: switch-heavy dispatch (ccgo goto noise). */
uint32_t adv_dispatch(uint32_t op, uint32_t a, uint32_t b) {
    switch (op & 7u) {
    case 0: return a + b;
    case 1: return a - b;
    case 2: return a ^ b;
    case 3: return a | b;
    case 4: return a & b;
    case 5: return (a << (b & 31)) | (a >> (32 - (b & 31))); /* ROL var */
    case 6: return a * (b | 1u);
    default: return a ^ (b * 0x9e3779b1u);
    }
}
