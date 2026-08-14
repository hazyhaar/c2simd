package probebench

import (
	"fmt"
	"strings"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/tribench"
)

// GenProbeC generates a C main probing one stratum (clock_gettime + rusage).
func GenProbeC(lib tribench.Lib, st Stratum) string {
	var b strings.Builder
	b.WriteString(`#define _POSIX_C_SOURCE 200809L
#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <sys/resource.h>

static double now_s(void) {
	struct timespec ts;
	clock_gettime(CLOCK_MONOTONIC, &ts);
	return (double)ts.tv_sec + (double)ts.tv_nsec * 1e-9;
}

`)
	switch lib.Kind {
	case tribench.KindHash64:
		fmt.Fprintf(&b, "uint64_t %s(const uint8_t *data, size_t len);\n", lib.CFunc)
	case tribench.KindHash32:
		fmt.Fprintf(&b, "uint32_t %s(const uint8_t *data, size_t len);\n", lib.CFunc)
	case tribench.KindHash32Seed:
		fmt.Fprintf(&b, "uint32_t %s(const uint8_t *data, size_t len, uint32_t seed);\n", lib.CFunc)
	case tribench.KindSipHash:
		fmt.Fprintf(&b, "uint64_t %s(const uint8_t *in, size_t inlen, const uint8_t *k);\n", lib.CFunc)
	case tribench.KindXor:
		fmt.Fprintf(&b, "void %s(uint8_t *dst, const uint8_t *s1, const uint8_t *s2, size_t len);\n", lib.CFunc)
	case tribench.KindBase64:
		fmt.Fprintf(&b, "uint64_t %s(const uint8_t *src, size_t len, uint8_t *dst);\n", lib.CFunc)
	case tribench.KindBlake2b:
		fmt.Fprintf(&b, "void %s(uint64_t h[8], const uint8_t block[128], uint64_t t0, uint64_t t1, uint64_t f0, uint64_t f1);\n", lib.CFunc)
	case tribench.KindMD5:
		fmt.Fprintf(&b, "void %s(uint32_t state[4], const uint8_t block[64]);\n", lib.CFunc)
	case tribench.KindChaChaQR:
		fmt.Fprintf(&b, "void %s(uint32_t *a, uint32_t *b, uint32_t *c, uint32_t *d);\n", lib.CFunc)
	case tribench.KindPoly5:
		fmt.Fprintf(&b, "void %s(uint32_t h[5], const uint32_t r[5], const uint32_t m[4]);\n", lib.CFunc)
	case tribench.KindTweetVer:
		fmt.Fprintf(&b, "int %s(const uint8_t *x, const uint8_t *y);\n", lib.CFunc)
	case tribench.KindLibInj:
		fmt.Fprintf(&b, "size_t %s(const uint8_t *s, size_t n);\n", lib.CFunc)
	}

	b.WriteString("\nint main(void) {\n")
	b.WriteString("\tstruct rusage ru0, ru1;\n\tgetrusage(RUSAGE_SELF, &ru0);\n")
	fmt.Fprintf(&b, "\tint iters = %d;\n\tint payload = %d;\n\tuint64_t sink = 0;\n\tdouble elapsed = 0;\n", st.Iters, st.Bytes)

	switch lib.Kind {
	case tribench.KindHash64, tribench.KindHash32, tribench.KindHash32Seed, tribench.KindLibInj:
		b.WriteString("\tuint8_t *data = payload ? (uint8_t*)malloc((size_t)payload) : NULL;\n")
		b.WriteString("\tfor (int i = 0; i < payload; i++) data[i] = (uint8_t)(i * 3 + 7);\n")
		b.WriteString("\tdouble t0 = now_s();\n\tfor (int i = 0; i < iters; i++) {\n")
		switch lib.Kind {
		case tribench.KindHash64:
			fmt.Fprintf(&b, "\t\tsink ^= %s(data, (size_t)payload);\n", lib.CFunc)
		case tribench.KindHash32:
			fmt.Fprintf(&b, "\t\tsink ^= %s(data, (size_t)payload);\n", lib.CFunc)
		case tribench.KindHash32Seed:
			fmt.Fprintf(&b, "\t\tsink ^= %s(data, (size_t)payload, 42u);\n", lib.CFunc)
		case tribench.KindLibInj:
			fmt.Fprintf(&b, "\t\tsink ^= (uint64_t)%s(data, (size_t)payload);\n", lib.CFunc)
		}
		b.WriteString("\t}\n\telapsed = now_s() - t0;\n")

	case tribench.KindSipHash:
		b.WriteString("\tuint8_t *data = payload ? (uint8_t*)malloc((size_t)payload) : NULL;\n")
		b.WriteString("\tuint8_t key[16] = {0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15};\n")
		b.WriteString("\tfor (int i = 0; i < payload; i++) data[i] = (uint8_t)i;\n")
		b.WriteString("\tdouble t0 = now_s();\n\tfor (int i = 0; i < iters; i++) {\n")
		fmt.Fprintf(&b, "\t\tsink ^= %s(data, (size_t)payload, key);\n", lib.CFunc)
		b.WriteString("\t}\n\telapsed = now_s() - t0;\n")

	case tribench.KindXor:
		b.WriteString("\tint n = payload;\n\tuint8_t *a = (uint8_t*)malloc((size_t)n), *b2 = (uint8_t*)malloc((size_t)n), *dst = (uint8_t*)malloc((size_t)n);\n")
		b.WriteString("\tfor (int i = 0; i < n; i++) { a[i]=(uint8_t)i; b2[i]=(uint8_t)(i*5); }\n")
		b.WriteString("\tdouble t0 = now_s();\n\tfor (int i = 0; i < iters; i++) {\n")
		fmt.Fprintf(&b, "\t\t%s(dst, a, b2, (size_t)n);\n", lib.CFunc)
		b.WriteString("\t}\n\telapsed = now_s() - t0;\n\tsink = dst[0];\n")

	case tribench.KindBase64:
		b.WriteString("\tint n = payload; size_t dn = (size_t)((n+2)/3*4+8);\n")
		b.WriteString("\tuint8_t *src = (uint8_t*)malloc((size_t)n), *dst = (uint8_t*)malloc(dn);\n")
		b.WriteString("\tfor (int i = 0; i < n; i++) src[i]=(uint8_t)i;\n")
		b.WriteString("\tdouble t0 = now_s();\n\tfor (int i = 0; i < iters; i++) {\n")
		fmt.Fprintf(&b, "\t\tsink += %s(src, (size_t)n, dst);\n", lib.CFunc)
		b.WriteString("\t}\n\telapsed = now_s() - t0;\n")

	case tribench.KindBlake2b:
		b.WriteString("\tuint64_t h[8]={0}; uint8_t block[128]; for(int i=0;i<128;i++) block[i]=(uint8_t)i;\n")
		b.WriteString("\tdouble t0 = now_s();\n\tfor (int i = 0; i < iters; i++) {\n")
		fmt.Fprintf(&b, "\t\t%s(h, block, 0, 0, ~0ULL, ~0ULL);\n", lib.CFunc)
		b.WriteString("\t}\n\telapsed = now_s() - t0;\n\tsink = h[0]; payload = 128;\n")

	case tribench.KindMD5:
		b.WriteString("\tuint32_t st[4]={0}; uint8_t block[64]; for(int i=0;i<64;i++) block[i]=(uint8_t)i;\n")
		b.WriteString("\tdouble t0 = now_s();\n\tfor (int i = 0; i < iters; i++) {\n")
		fmt.Fprintf(&b, "\t\t%s(st, block);\n", lib.CFunc)
		b.WriteString("\t}\n\telapsed = now_s() - t0;\n\tsink = st[0]; payload = 64;\n")

	case tribench.KindChaChaQR:
		b.WriteString("\tuint32_t a=1,b=2,c=3,d=4;\n\tdouble t0 = now_s();\n\tfor (int i = 0; i < iters; i++) {\n")
		fmt.Fprintf(&b, "\t\t%s(&a,&b,&c,&d);\n", lib.CFunc)
		b.WriteString("\t}\n\telapsed = now_s() - t0;\n\tsink = a^d; payload = 16;\n")

	case tribench.KindPoly5:
		b.WriteString("\tuint32_t h[5], r[5], m[4]; for(int i=0;i<5;i++){h[i]=i;r[i]=i+3;} for(int i=0;i<4;i++) m[i]=i*7;\n")
		b.WriteString("\tdouble t0 = now_s();\n\tfor (int i = 0; i < iters; i++) {\n")
		fmt.Fprintf(&b, "\t\t%s(h,r,m);\n", lib.CFunc)
		b.WriteString("\t}\n\telapsed = now_s() - t0;\n\tsink = h[0]; payload = 16;\n")

	case tribench.KindTweetVer:
		b.WriteString("\tuint8_t x[16], y[16];\n")
		if st.ID == "ver_eq" {
			b.WriteString("\tfor(int i=0;i<16;i++){x[i]=1;y[i]=1;}\n")
		} else {
			b.WriteString("\tfor(int i=0;i<16;i++){x[i]=1;y[i]=2;}\n")
		}
		b.WriteString("\tdouble t0 = now_s();\n\tfor (int i = 0; i < iters; i++) {\n")
		fmt.Fprintf(&b, "\t\tsink += (uint64_t)%s(x,y);\n", lib.CFunc)
		b.WriteString("\t}\n\telapsed = now_s() - t0;\n\tpayload = 16;\n")
	}

	fmt.Fprintf(&b, `
	getrusage(RUSAGE_SELF, &ru1);
	double ns = (elapsed * 1e9) / (double)(iters > 0 ? iters : 1);
	double mbs = 0;
	if (elapsed > 0) mbs = ((double)payload * (double)iters) / elapsed / (1024.0*1024.0);
	long maxrss = ru1.ru_maxrss;
	long minflt = ru1.ru_minflt - ru0.ru_minflt;
	long majflt = ru1.ru_majflt - ru0.ru_majflt;
	printf("PROBE{\"lib\":\"%s\",\"stratum\":\"%s\",\"phase\":\"%s\",\"backend\":\"c_gcc_O2\",\"payload_bytes\":%%d,\"iters\":%%d,\"ns_per_op\":%%.3f,\"mb_s\":%%.3f,\"allocs\":0,\"alloc_bytes\":0,\"max_rss_kib\":%%ld,\"minflt\":%%ld,\"majflt\":%%ld,\"sink\":%%llu}\n",
		payload, iters, ns, mbs, maxrss, minflt, majflt, (unsigned long long)sink);
	(void)sink;
	return 0;
}
`, lib.ID, st.ID, st.Phase)
	return b.String()
}
