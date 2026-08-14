#include <stdint.h>
#include <stddef.h>
/* Minimal strspn-like — subset-C friendly (no break/string lit). accept=hel */
static const uint8_t accept_hel[4] = {104, 101, 108, 0};

size_t strlenspn_lab(const uint8_t *s, size_t n) {
    size_t i;
    size_t j;
    int ok;
    int done;
    done = 0;
    i = 0;
    while (i < n && done == 0) {
        ok = 0;
        j = 0;
        while (j < 4 && accept_hel[j] != 0 && ok == 0) {
            if (accept_hel[j] == s[i]) {
                ok = 1;
            }
            j = j + 1;
        }
        if (ok == 0) {
            done = 1;
        } else {
            i = i + 1;
        }
    }
    return i;
}
