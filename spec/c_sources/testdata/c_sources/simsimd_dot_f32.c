#include <stddef.h>

void simsimd_dot_f32(const float *a, const float *b, size_t n, double *result) {
    double sum = 0.0;
    for (size_t i = 0; i < n; ++i) {
        sum += (double)a[i] * (double)b[i];
    }
    *result = sum;
}

void simsimd_l2_sq_f32(const float *a, const float *b, size_t n, double *result) {
    double sum = 0.0;
    for (size_t i = 0; i < n; ++i) {
        double diff = (double)a[i] - (double)b[i];
        sum += diff * diff;
    }
    *result = sum;
}
