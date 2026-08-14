package tribench

import (
	"fmt"
	"strings"
)

// GenHarnessC produces a C main that links against the kernel .c and prints fixture digests.
func GenHarnessC(lib Lib) string {
	var b strings.Builder
	b.WriteString(`#include <stdint.h>
#include <stddef.h>
#include <stdio.h>
#include <string.h>
#include <stdlib.h>

/* kernel symbols */
`)
	switch lib.Kind {
	case KindHash64:
		fmt.Fprintf(&b, "uint64_t %s(const uint8_t *data, size_t len);\n", lib.CFunc)
	case KindHash32:
		fmt.Fprintf(&b, "uint32_t %s(const uint8_t *data, size_t len);\n", lib.CFunc)
	case KindHash32Seed:
		fmt.Fprintf(&b, "uint32_t %s(const uint8_t *key, size_t len, uint32_t seed);\n", lib.CFunc)
	case KindSipHash:
		fmt.Fprintf(&b, "uint64_t %s(const uint8_t *in, size_t inlen, const uint8_t *k);\n", lib.CFunc)
	case KindXor:
		fmt.Fprintf(&b, "void %s(uint8_t *dst, const uint8_t *src1, const uint8_t *src2, size_t len);\n", lib.CFunc)
	case KindBlake2b:
		fmt.Fprintf(&b, "void %s(uint64_t h[8], const uint8_t block[128], uint64_t t0, uint64_t t1, uint64_t f0, uint64_t f1);\n", lib.CFunc)
	case KindChaChaQR:
		fmt.Fprintf(&b, "void %s(uint32_t *a, uint32_t *b, uint32_t *c, uint32_t *d);\n", lib.CFunc)
	case KindMD5:
		fmt.Fprintf(&b, "void %s(uint32_t state[4], const uint8_t block[64]);\n", lib.CFunc)
	case KindPoly5:
		fmt.Fprintf(&b, "void %s(uint32_t h[5], const uint32_t r[5], const uint32_t m[4]);\n", lib.CFunc)
	case KindBase64:
		fmt.Fprintf(&b, "size_t %s(const uint8_t *src, size_t len, char *dst);\n", lib.CFunc)
	case KindTweetVer:
		fmt.Fprintf(&b, "int %s(const uint8_t *x, const uint8_t *y);\n", lib.CFunc)
	case KindLibInj:
		fmt.Fprintf(&b, "size_t %s(const uint8_t *s, size_t len);\n", lib.CFunc)
	}

	b.WriteString(`
static void hex_u64(uint64_t v) { printf("%016llx", (unsigned long long)v); }
static void hex_u32(uint32_t v) { printf("%08x", v); }
static void hex_buf(const uint8_t *p, size_t n) {
  size_t i; for (i = 0; i < n; i++) printf("%02x", p[i]);
}
`)
	if lib.Kind == KindXor {
		b.WriteString("/* minimal sha256 for xor digests */\n")
		b.WriteString(sha256CSnippet)
	}
	b.WriteString("\nint main(void) {\n")

	// embed fixtures as C arrays
	fixs := FixturesFor(lib.Kind)
	usesData := lib.Kind != KindChaChaQR && lib.Kind != KindPoly5
	for i, f := range fixs {
		fmt.Fprintf(&b, "  /* fixture %s */\n", f.Name)
		if usesData {
			emitCBytes(&b, fmt.Sprintf("d%d", i), f.Data)
		}
		if len(f.Data2) > 0 {
			emitCBytes(&b, fmt.Sprintf("e%d", i), f.Data2)
		}
	}

	for i, f := range fixs {
		fmt.Fprintf(&b, "  printf(\"%s \");\n", f.Name)
		switch lib.Kind {
		case KindHash64:
			fmt.Fprintf(&b, "  { uint64_t h = %s(d%d, %d); hex_u64(h); printf(\"\\n\"); }\n", lib.CFunc, i, len(f.Data))
		case KindHash32:
			fmt.Fprintf(&b, "  { uint32_t h = %s(d%d, %d); hex_u32(h); printf(\"\\n\"); }\n", lib.CFunc, i, len(f.Data))
		case KindHash32Seed:
			fmt.Fprintf(&b, "  { uint32_t h = %s(d%d, %d, %dU); hex_u32(h); printf(\"\\n\"); }\n", lib.CFunc, i, len(f.Data), f.Seed)
		case KindSipHash:
			fmt.Fprintf(&b, "  { uint64_t h = %s(d%d, %d, e%d); hex_u64(h); printf(\"\\n\"); }\n", lib.CFunc, i, len(f.Data), i)
		case KindXor:
			fmt.Fprintf(&b, `  { uint8_t *dst = (uint8_t*)malloc(%d); %s(dst, d%d, e%d, %d);
    uint8_t dig[32]; sha256_buf(dst, %d, dig); hex_buf(dig, 32); printf("\n"); free(dst); }
`, len(f.Data), lib.CFunc, i, i, len(f.Data), len(f.Data))
		case KindBlake2b:
			fmt.Fprintf(&b, `  { uint64_t h[8] = {0}; uint8_t block[128]; memset(block, 0, 128);
    size_t n = %d; if (n > 128) n = 128; if (n) memcpy(block, d%d, n);
    %s(h, block, 0, 0, 0xffffffffffffffffULL, 0xffffffffffffffffULL);
    int j; for (j=0;j<8;j++) hex_u64(h[j]); printf("\n"); }
`, len(f.Data), i, lib.CFunc)
		case KindChaChaQR:
			fmt.Fprintf(&b, `  { uint32_t a=0x11111111,b=0x22222222,c=0x33333333,d=0x44444444;
    %s(&a,&b,&c,&d); hex_u32(a); hex_u32(b); hex_u32(c); hex_u32(d); printf("\n"); }
`, lib.CFunc)
		case KindMD5:
			fmt.Fprintf(&b, `  { uint32_t st[4] = {0x67452301,0xefcdab89,0x98badcfe,0x10325476};
    uint8_t block[64]; memset(block,0,64); size_t n=%d; if(n>64)n=64; if(n)memcpy(block,d%d,n);
    %s(st, block); int j; for(j=0;j<4;j++) hex_u32(st[j]); printf("\n"); }
`, len(f.Data), i, lib.CFunc)
		case KindPoly5:
			fmt.Fprintf(&b, `  { uint32_t h[5]={0}, r[5]={1,2,3,4,5}, m[4]={9,8,7,6};
    %s(h,r,m); int j; for(j=0;j<5;j++) hex_u32(h[j]); printf("\n"); }
`, lib.CFunc)
		case KindBase64:
			// dst must hold 4/3 * len + pad
			dstN := (len(f.Data)+2)/3*4 + 8
			if dstN < 64 {
				dstN = 64
			}
			fmt.Fprintf(&b, "  { char dst[%d]; size_t n=%s(d%d,%d,dst); printf(\"%%.*s\\n\", (int)n, dst); }\n", dstN, lib.CFunc, i, len(f.Data))
		case KindTweetVer:
			fmt.Fprintf(&b, "  { int r = %s(d%d, e%d); printf(\"%%d\\n\", r); }\n", lib.CFunc, i, i)
		case KindLibInj:
			// 2-arg form (accept internal); size_t / uint64_t return
			fmt.Fprintf(&b, "  { uint64_t r = (uint64_t)%s(d%d, %d); hex_u64(r); printf(\"\\n\"); }\n", lib.CFunc, i, len(f.Data))
		}
	}
	b.WriteString("  return 0;\n}\n")
	return b.String()
}

func emitCBytes(b *strings.Builder, name string, data []byte) {
	if len(data) == 0 {
		fmt.Fprintf(b, "  static const uint8_t %s[1] = {0};\n", name)
		return
	}
	fmt.Fprintf(b, "  static const uint8_t %s[%d] = {", name, len(data))
	for i, v := range data {
		if i > 0 {
			b.WriteByte(',')
		}
		if i%16 == 0 {
			b.WriteString("\n    ")
		}
		fmt.Fprintf(b, "0x%02x", v)
	}
	b.WriteString("};\n")
}

// compact sha256 used only for xor digest in C harness
const sha256CSnippet = `
typedef struct { uint32_t s[8]; uint64_t bits; uint8_t buf[64]; size_t n; } sha256_ctx;
static uint32_t rotr(uint32_t x,int n){return (x>>n)|(x<<(32-n));}
static void sha256_transform(sha256_ctx *c,const uint8_t *p){
  static const uint32_t K[64]={
    0x428a2f98,0x71374491,0xb5c0fbcf,0xe9b5dba5,0x3956c25b,0x59f111f1,0x923f82a4,0xab1c5ed5,
    0xd807aa98,0x12835b01,0x243185be,0x550c7dc3,0x72be5d74,0x80deb1fe,0x9bdc06a7,0xc19bf174,
    0xe49b69c1,0xefbe4786,0x0fc19dc6,0x240ca1cc,0x2de92c6f,0x4a7484aa,0x5cb0a9dc,0x76f988da,
    0x983e5152,0xa831c66d,0xb00327c8,0xbf597fc7,0xc6e00bf3,0xd5a79147,0x06ca6351,0x14292967,
    0x27b70a85,0x2e1b2138,0x4d2c6dfc,0x53380d13,0x650a7354,0x766a0abb,0x81c2c92e,0x92722c85,
    0xa2bfe8a1,0xa81a664b,0xc24b8b70,0xc76c51a3,0xd192e819,0xd6990624,0xf40e3585,0x106aa070,
    0x19a4c116,0x1e376c08,0x2748774c,0x34b0bcb5,0x391c0cb3,0x4ed8aa4a,0x5b9cca4f,0x682e6ff3,
    0x748f82ee,0x78a5636f,0x84c87814,0x8cc70208,0x90befffa,0xa4506ceb,0xbef9a3f7,0xc67178f2};
  uint32_t w[64],a,b,c2,d,e,f,g,h,t1,t2; int i;
  for(i=0;i<16;i++){ w[i]=(p[i*4]<<24)|(p[i*4+1]<<16)|(p[i*4+2]<<8)|p[i*4+3]; }
  for(;i<64;i++){ uint32_t s0=rotr(w[i-15],7)^rotr(w[i-15],18)^(w[i-15]>>3);
    uint32_t s1=rotr(w[i-2],17)^rotr(w[i-2],19)^(w[i-2]>>10); w[i]=w[i-16]+s0+w[i-7]+s1; }
  a=c->s[0];b=c->s[1];c2=c->s[2];d=c->s[3];e=c->s[4];f=c->s[5];g=c->s[6];h=c->s[7];
  for(i=0;i<64;i++){ uint32_t S1=rotr(e,6)^rotr(e,11)^rotr(e,25); uint32_t ch=(e&f)^((~e)&g);
    t1=h+S1+ch+K[i]+w[i]; uint32_t S0=rotr(a,2)^rotr(a,13)^rotr(a,22); uint32_t maj=(a&b)^(a&c2)^(b&c2);
    t2=S0+maj; h=g;g=f;f=e;e=d+t1;d=c2;c2=b;b=a;a=t1+t2; }
  c->s[0]+=a;c->s[1]+=b;c->s[2]+=c2;c->s[3]+=d;c->s[4]+=e;c->s[5]+=f;c->s[6]+=g;c->s[7]+=h;
}
static void sha256_init(sha256_ctx *c){ c->s[0]=0x6a09e667;c->s[1]=0xbb67ae85;c->s[2]=0x3c6ef372;c->s[3]=0xa54ff53a;
  c->s[4]=0x510e527f;c->s[5]=0x9b05688c;c->s[6]=0x1f83d9ab;c->s[7]=0x5be0cd19; c->bits=0; c->n=0; }
static void sha256_update(sha256_ctx *c,const uint8_t *p,size_t n){
  c->bits += (uint64_t)n*8; while(n){ size_t k=64-c->n; if(k>n)k=n; memcpy(c->buf+c->n,p,k); c->n+=k; p+=k; n-=k;
    if(c->n==64){ sha256_transform(c,c->buf); c->n=0; } } }
static void sha256_final(sha256_ctx *c,uint8_t out[32]){ size_t i; c->buf[c->n++]=0x80;
  if(c->n>56){ while(c->n<64)c->buf[c->n++]=0; sha256_transform(c,c->buf); c->n=0; }
  while(c->n<56)c->buf[c->n++]=0;
  for(i=0;i<8;i++) c->buf[63-i]=(uint8_t)(c->bits>>(8*i));
  sha256_transform(c,c->buf);
  for(i=0;i<8;i++){ out[i*4]=(uint8_t)(c->s[i]>>24); out[i*4+1]=(uint8_t)(c->s[i]>>16);
    out[i*4+2]=(uint8_t)(c->s[i]>>8); out[i*4+3]=(uint8_t)c->s[i]; } }
static void sha256_buf(const uint8_t *p,size_t n,uint8_t out[32]){ sha256_ctx c; sha256_init(&c); sha256_update(&c,p,n); sha256_final(&c,out); }
`
