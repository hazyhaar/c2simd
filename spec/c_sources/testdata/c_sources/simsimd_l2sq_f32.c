#include <stdint.h>
#include <stddef.h>

/* SimSIMD L2 squared distance float32 kernel */
void simsimd_l2sq_f32(const float *a, const float *b, uint64_t n, double *out) {
    double sum = 0.0;
    uint64_t i;
    for (i = 0; i < n; i++) {
        double diff = (double)a[i] - (double)b[i];
        sum += diff * diff;
    }
    *out = sum;
}
