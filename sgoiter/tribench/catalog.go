package tribench

import (
	"path/filepath"
)

// Kind classifies the fixture harness to generate.
type Kind string

const (
	KindHash64     Kind = "hash64"      // h(data,len) → u64
	KindHash32     Kind = "hash32"      // h(data,len) → u32
	KindHash32Seed Kind = "hash32_seed" // h(data,len,seed) → u32
	KindSipHash    Kind = "siphash"     // h(data,len,key16) → u64
	KindXor        Kind = "xor"         // xor dst,s1,s2,len → sha256(dst)
	KindBlake2b    Kind = "blake2b"     // compress one block → h[8]
	KindChaChaQR   Kind = "chacha_qr"   // quarter round on words
	KindMD5        Kind = "md5"         // transform block → state[4]
	KindPoly5      Kind = "poly5"       // poly1305_block5
	KindBase64     Kind = "base64"      // encode stream
	KindTweetVer   Kind = "tweet_ver"   // crypto_verify_16
	KindLibInj     Kind = "libinj"      // strlenspn
)

// Lib is one of the 12 dogfood kernels.
type Lib struct {
	ID       string // directory / report key
	Kind     Kind
	CRel     string // under c2simd root, or absolute
	CFunc    string // C symbol
	SgoFunc  string // Go export from sgoiter
	CcgoFunc string // ccgo symbol (usually same as C)
	Notes    string
	// SkipC: multi-file / missing headers — no gcc oracle
	SkipC bool
	// StubHeaders: write empty headers next to kernel for gcc/ccgo
	StubHeaders []string
}

// DefaultLibs returns the 12-kernel catalog.
func DefaultLibs(c2simdRoot string) []Lib {
	cs := filepath.Join(c2simdRoot, "spec/c_sources/testdata/c_sources")
	dog := filepath.Join(c2simdRoot, "spec/dogfood/testdata/cycles/20260811b_sgoiter12")
	return []Lib{
		{ID: "fnv1a_64", Kind: KindHash64, CRel: filepath.Join(cs, "fnv1a_64.c"), CFunc: "fnv1a_64", SgoFunc: "Fnv1a_64", CcgoFunc: "fnv1a_64"},
		{ID: "crc32_ieee", Kind: KindHash32, CRel: filepath.Join(cs, "crc32_ieee.c"), CFunc: "crc32_ieee", SgoFunc: "Crc32_ieee", CcgoFunc: "crc32_ieee"},
		{ID: "fast_xor", Kind: KindXor, CRel: filepath.Join(cs, "fast_xor.c"), CFunc: "fast_xor_bytes", SgoFunc: "Fast_xor_bytes", CcgoFunc: "fast_xor_bytes"},
		{ID: "siphash24", Kind: KindSipHash, CRel: filepath.Join(cs, "siphash24.c"), CFunc: "siphash24", SgoFunc: "Siphash24", CcgoFunc: "siphash24"},
		{ID: "murmur3_x86_32", Kind: KindHash32Seed, CRel: filepath.Join(cs, "murmur3_x86_32.c"), CFunc: "murmur3_x86_32", SgoFunc: "Murmur3_x86_32", CcgoFunc: "murmur3_x86_32"},
		{ID: "blake2b_compress", Kind: KindBlake2b, CRel: filepath.Join(cs, "blake2b_compress.c"), CFunc: "blake2b_compress_block", SgoFunc: "Blake2b_compress_block", CcgoFunc: "blake2b_compress_block"},
		{ID: "chacha20_qr", Kind: KindChaChaQR, CRel: filepath.Join(cs, "chacha20_qr.c"), CFunc: "chacha20_quarter_round", SgoFunc: "Chacha20_quarter_round", CcgoFunc: "chacha20_quarter_round"},
		{ID: "md5_transform", Kind: KindMD5, CRel: filepath.Join(cs, "md5_transform.c"), CFunc: "md5_transform_block", SgoFunc: "Md5_transform_block", CcgoFunc: "md5_transform_block"},
		{ID: "poly1305_block5", Kind: KindPoly5, CRel: filepath.Join(cs, "poly1305_block5.c"), CFunc: "poly1305_block5", SgoFunc: "Poly1305_block5", CcgoFunc: "poly1305_block5"},
		{ID: "base64_simd", Kind: KindBase64, CRel: filepath.Join(cs, "base64_simd.c"), CFunc: "base64_encode_stream", SgoFunc: "Base64_encode_stream", CcgoFunc: "base64_encode_stream"},
		{ID: "tweetnacl_dogfood", Kind: KindTweetVer, CRel: filepath.Join(cs, "tweetnacl_dogfood.c"), CFunc: "crypto_verify_16", SgoFunc: "Crypto_verify_16", CcgoFunc: "crypto_verify_16", Notes: "surface verify_16; stub tweetnacl.h", StubHeaders: []string{"tweetnacl.h"}},
		// cycle src + empty stub headers (full multi-header not harvested)
		// SkipC permanent until front multi-tuile headers (TODO_NIGHT N11).
		// Oracle triangle for strspn-like: use strlenspn_lab instead.
		{ID: "libinjection_sqli", Kind: KindLibInj, CRel: filepath.Join(dog, "libinjection_sqli/src.c"), CFunc: "strlenspn", SgoFunc: "Strlenspn", CcgoFunc: "strlenspn", Notes: "SkipC permanent: multi-header upstream; see strlenspn_lab", SkipC: true, StubHeaders: []string{"libinjection.h", "libinjection_sqli.h", "libinjection_sqli_data.h"}},
		// Minimal strspn oracle for triangle (not full libinjection).
		{ID: "strlenspn_lab", Kind: KindLibInj, CRel: filepath.Join(cs, "strlenspn_lab.c"), CFunc: "strlenspn_lab", SgoFunc: "Strlenspn_lab", CcgoFunc: "strlenspn_lab", Notes: "oracle C for strspn-like"},
		// Full MD5 64-step (optional dogfood; reduced md5_transform stays default).
		{ID: "md5_transform_full", Kind: KindMD5, CRel: filepath.Join(cs, "md5_transform_full.c"), CFunc: "md5_transform_full_block", SgoFunc: "Md5_transform_full_block", CcgoFunc: "md5_transform_full_block", Notes: "full 64-step MD5"},
	}
}
