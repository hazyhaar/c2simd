#include <stddef.h>
#include <stdint.h>

static const char digits[201] =
    "00010203040506070809"
    "10111213141516171819"
    "20212223242526272829"
    "30313233343536373839"
    "40414243444546474849"
    "50515253545556575859"
    "60616263646566676869"
    "70717273747576777879"
    "80818283848586878889"
    "90919293949596979899";

size_t yyjson_write_u32(char *buf, uint32_t val) {
    char tmp[16];
    size_t i = 16;
    if (val == 0) {
        buf[0] = '0';
        return 1;
    }
    while (val >= 100) {
        uint32_t q = val / 100;
        uint32_t r = val - q * 100;
        val = q;
        i -= 2;
        tmp[i] = digits[r * 2];
        tmp[i + 1] = digits[r * 2 + 1];
    }
    if (val >= 10) {
        i -= 2;
        tmp[i] = digits[val * 2];
        tmp[i + 1] = digits[val * 2 + 1];
    } else {
        i -= 1;
        tmp[i] = (char)('0' + val);
    }
    size_t len = 16 - i;
    for (size_t k = 0; k < len; k++) {
        buf[k] = tmp[i + k];
    }
    return len;
}
