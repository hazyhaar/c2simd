#include <stdint.h>
#include <stddef.h>
#include <string.h>
/* Adversarial lab: large stack buffer + manual walk — resource pattern.
 * Simulates "big stack workspace" style without shellcode. */
uint32_t adv_stack_workspace(const uint8_t *in, size_t n) {
    uint8_t ws[4096];
    size_t i;
    uint32_t h = 0x811c9dc5u;
    if (n > sizeof(ws)) n = sizeof(ws);
    for (i = 0; i < n; i++) ws[i] = in[i] ^ (uint8_t)i;
    for (i = 0; i < n; i++) {
        h ^= ws[i];
        h *= 0x01000193u;
    }
    /* intentional leftover: touch end of buffer */
    h ^= ws[sizeof(ws) - 1];
    return h;
}
