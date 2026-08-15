#include <stddef.h>
#include <stdint.h>

/* FastLZ Level-1 streaming decompression kernel */

int fastlz1_decompress(const uint8_t *input, int in_len, uint8_t *output, int maxout) {
    int ip = 0;
    int op = 0;
    while (ip < in_len && op < maxout) {
        uint8_t ctrl = input[ip];
        ip++;
        if (ctrl < 32) {
            int len = (int)ctrl + 1;
            if (ip + len > in_len || op + len > maxout) {
                return 0;
            }
            for (int i = 0; i < len; i++) {
                output[op] = input[ip];
                op++;
                ip++;
            }
        } else {
            int len = (int)(ctrl >> 5);
            int dist = ((int)(ctrl & 31)) << 8;
            if (len == 7) {
                if (ip >= in_len) {
                    return 0;
                }
                len += (int)input[ip];
                ip++;
            }
            len += 2;
            if (ip >= in_len) {
                return 0;
            }
            dist |= (int)input[ip];
            ip++;
            dist += 1;
            if (dist > op || op + len > maxout) {
                return 0;
            }
            int ref = op - dist;
            for (int i = 0; i < len; i++) {
                output[op] = output[ref + i];
                op++;
            }
        }
    }
    return op;
}
