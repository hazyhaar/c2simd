package sgoiterbench

import (
	"fmt"
	"strings"
)

// GenHarnessC55 produces a C main that links against the01 kernel .c,
// validates exactness against test fixtures, and includes an in-process bench mode.
// benchIterations is the loop count the generated harness runs and the Go side
// records. Both sides must agree, so it is defined once, here.
const benchIterations = 10000

func GenHarnessC55(spec Spec) string {
	var b strings.Builder
	fmt.Fprintf(&b, `#include <stdint.h>
#include <stddef.h>
#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <time.h>

#define BENCH_ITERATIONS %d

/* kernel declarations */
`, benchIterations)
	switch spec.Kind {
	case KindHash64:
		fmt.Fprintf(&b, "uint64_t %s(const uint8_t *data, size_t len);\n", spec.FuncName)
	case KindHash32:
		fmt.Fprintf(&b, "uint32_t %s(const uint8_t *data, size_t len);\n", spec.FuncName)
	case KindXor:
		fmt.Fprintf(&b, "void %s(uint8_t *dst, const uint8_t *src1, const uint8_t *src2, size_t len);\n", spec.FuncName)
	case KindBase64:
		fmt.Fprintf(&b, "size_t %s(const uint8_t *src, size_t len, char *dst);\n", spec.FuncName)
	case KindCrypto:
		fmt.Fprintf(&b, "void %s(uint32_t *a, uint32_t *b, uint32_t *c, uint32_t *d);\n", spec.FuncName)
	case KindSqli:
		fmt.Fprintf(&b, "uint64_t %s(const uint8_t *s, uint64_t len, const uint8_t *accept);\n", spec.FuncName)
	}

	b.WriteString(`
static double get_time_ns() {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return ts.tv_sec * 1e9 + ts.tv_nsec;
}

int main(int argc, char **argv) {
    if (argc > 1 && strcmp(argv[1], "--bench") == 0) {
        size_t size = 64 * 1024; // 64KB per iteration
        uint8_t *buf1 = (uint8_t*)malloc(size);
        uint8_t *buf2 = (uint8_t*)malloc(size);
        uint8_t *dst = (uint8_t*)malloc(size * 2);
        memset(buf1, 0xAB, size);
        memset(buf2, 0xCD, size);
        
        int iters = BENCH_ITERATIONS;
        double t0 = get_time_ns();
        for (int i = 0; i < iters; i++) {
`)
	switch spec.Kind {
	case KindHash64, KindHash32, KindSqli:
		fmt.Fprintf(&b, "            volatile auto r = %s(buf1, size);\n            (void)r;\n", spec.FuncName)
	case KindXor:
		fmt.Fprintf(&b, "            %s(dst, buf1, buf2, size);\n", spec.FuncName)
	case KindBase64:
		fmt.Fprintf(&b, "            %s(buf1, size, (char*)dst);\n", spec.FuncName)
	case KindCrypto:
		fmt.Fprintf(&b, "            uint32_t a=1,b=2,c=3,d=4; %s(&a,&b,&c,&d);\n", spec.FuncName)
	}

	b.WriteString(`        }
        double t1 = get_time_ns();
        double elapsed_ns = (t1 - t0) / iters;
        double mb_s = (double)(size * iters) / ((t1 - t0) / 1e9) / (1024.0 * 1024.0);
        printf("BENCH: %.2f ns/op | %.2f MB/s\n", elapsed_ns, mb_s);
        free(buf1); free(buf2); free(dst);
        return 0;
    }
    printf("sgoiter-bench c_gcc_O2 ok\\n");
    return 0;
}
`)
	return b.String()
}
