#include <stdint.h>

void sort_i64(int64_t *a, int n) {
	int i;
	int j;
	int64_t t;
	for (i = 0; i < n; i = i + 1) {
		for (j = 0; j < n - 1; j = j + 1) {
			if (a[j] > a[j + 1]) {
				t = a[j];
				a[j] = a[j + 1];
				a[j + 1] = t;
			}
		}
	}
}
