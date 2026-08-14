#include <stdint.h>
#include <stddef.h>

/* FNV-1a 64-bit — boucle simple, pas de rotate, contrôle négatif gen. */
uint64_t fnv1a_64(const uint8_t *data, size_t len) {
    uint64_t h = 14695981039346656037ULL;
    size_t i;
    for (i = 0; i < len; i++) {
        h ^= (uint64_t)data[i];
        h *= 1099511628211ULL;
    }
    return h;
}
