#include <stdint.h>

/* local + param arrays */
void add_vec(uint32_t h[4], const uint32_t a[4]) {
    int i;
    for (i = 0; i < 4; i++) {
        h[i] = h[i] + a[i];
    }
}

uint32_t sum4(const uint32_t v[4]) {
    uint32_t s;
    s = 0;
    s = s + v[0];
    s = s + v[1];
    s = s + v[2];
    s = s + v[3];
    return s;
}

uint32_t local_arr(uint32_t x) {
    uint32_t t[2];
    t[0] = x;
    t[1] = x + 1;
    return t[0] + t[1];
}
