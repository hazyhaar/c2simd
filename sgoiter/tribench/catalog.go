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
	KindDotF32       Kind = "dot_f32"       // SimSIMD dot product float32
	KindPolyDonna32  Kind = "poly_donna32"  // Poly1305 Donna-32 (26-bit limbs)
	KindCurveDonna64 Kind = "curve_donna64" // Curve25519 Donna-64 corps arithmetic (51-bit limbs)
	KindYyjsonInt    Kind = "yyjson_int"    // yyjson fast u32 serialization
	KindCjsonCore    Kind = "cjson_core"    // cJSON case-insensitive compare
	KindStbiPng      Kind = "stbi_png"      // stb_image PNG filter reconstruction
	KindUtf8Proc     Kind = "utf8proc"      // utf8proc UTF-8 iterator
	KindFastlz1      Kind = "fastlz1"       // FastLZ Level-1 decompressor
	KindMurmur128    Kind = "murmur128"     // MurmurHash3 x64 128-bit
	KindTweetHsalsa  Kind = "tweet_hsalsa"  // TweetNaCl HSalsa20 core
)

// Lib is one of the dogfood kernels.
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

// DefaultLibs returns the dogfood catalog.
func DefaultLibs(c2simdRoot string) []Lib {
	cs := filepath.Join(c2simdRoot, "spec/c_sources/testdata/c_sources")
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
		{ID: "poly1305_donna32", Kind: KindPolyDonna32, CRel: filepath.Join(cs, "poly1305_donna32.c"), CFunc: "poly1305_donna32_block", SgoFunc: "Poly1305_donna32_block", CcgoFunc: "poly1305_donna32_block", Notes: "Poly1305 Donna-32 (26-bit limbs)"},
		{ID: "curve25519_donna64", Kind: KindCurveDonna64, CRel: filepath.Join(cs, "curve25519_donna64.c"), CFunc: "curve25519_f51_mul121666", SgoFunc: "Curve25519_f51_mul121666", CcgoFunc: "curve25519_f51_mul121666", Notes: "Curve25519 Donna-64 mul121666 (51-bit limbs)"},
		{ID: "yyjson_int", Kind: KindYyjsonInt, CRel: filepath.Join(cs, "yyjson_int.c"), CFunc: "yyjson_write_u32", SgoFunc: "Yyjson_write_u32", CcgoFunc: "yyjson_write_u32", Notes: "yyjson fast u32 serializer"},
		{ID: "cjson_core", Kind: KindCjsonCore, CRel: filepath.Join(cs, "cjson_core.c"), CFunc: "cjson_casecmp", SgoFunc: "Cjson_casecmp", CcgoFunc: "cjson_casecmp", Notes: "cJSON case-insensitive compare"},
		{ID: "stbi_png_filter", Kind: KindStbiPng, CRel: filepath.Join(cs, "stbi_png_filter.c"), CFunc: "stbi_unfilter_row", SgoFunc: "Stbi_unfilter_row", CcgoFunc: "stbi_unfilter_row", Notes: "stb_image PNG filter reconstruction"},
		{ID: "utf8proc_core", Kind: KindUtf8Proc, CRel: filepath.Join(cs, "utf8proc_core.c"), CFunc: "utf8proc_iterate", SgoFunc: "Utf8proc_iterate", CcgoFunc: "utf8proc_iterate", Notes: "utf8proc UTF-8 iterator"},
		{ID: "fastlz_core", Kind: KindFastlz1, CRel: filepath.Join(cs, "fastlz_core.c"), CFunc: "fastlz1_decompress", SgoFunc: "Fastlz1_decompress", CcgoFunc: "fastlz1_decompress", Notes: "FastLZ Level-1 decompressor"},
		{ID: "murmur3_x64_128", Kind: KindMurmur128, CRel: filepath.Join(cs, "murmur3_x64_128.c"), CFunc: "murmur3_x64_128", SgoFunc: "Murmur3_x64_128", CcgoFunc: "murmur3_x64_128", Notes: "MurmurHash3 x64 128-bit"},
		{ID: "tweetnacl_hsalsa", Kind: KindTweetHsalsa, CRel: filepath.Join(cs, "tweetnacl_core.c"), CFunc: "crypto_core_hsalsa20", SgoFunc: "Crypto_core_hsalsa20", CcgoFunc: "crypto_core_hsalsa20", Notes: "TweetNaCl HSalsa20 core"},
		{ID: "base64_simd", Kind: KindBase64, CRel: filepath.Join(cs, "base64_simd.c"), CFunc: "base64_encode_stream", SgoFunc: "Base64_encode_stream", CcgoFunc: "base64_encode_stream"},
		{ID: "tweetnacl_dogfood", Kind: KindTweetVer, CRel: filepath.Join(cs, "tweetnacl_dogfood.c"), CFunc: "crypto_verify_16", SgoFunc: "Crypto_verify_16", CcgoFunc: "crypto_verify_16", Notes: "surface verify_16; stub tweetnacl.h", StubHeaders: []string{"tweetnacl.h"}},
		{ID: "simsimd_dot_f32", Kind: KindDotF32, CRel: filepath.Join(cs, "simsimd_dot_f32.c"), CFunc: "simsimd_dot_f32", SgoFunc: "Simsimd_dot_f32", CcgoFunc: "simsimd_dot_f32", Notes: "SimSIMD float32 dot product"},
		// Minimal strspn oracle for triangle (not full libinjection).
		{ID: "strlenspn_lab", Kind: KindLibInj, CRel: filepath.Join(cs, "strlenspn_lab.c"), CFunc: "strlenspn_lab", SgoFunc: "Strlenspn_lab", CcgoFunc: "strlenspn_lab", Notes: "oracle C for strspn-like"},
		// Full MD5 64-step (optional dogfood; reduced md5_transform stays default).
		{ID: "md5_transform_full", Kind: KindMD5, CRel: filepath.Join(cs, "md5_transform_full.c"), CFunc: "md5_transform_full_block", SgoFunc: "Md5_transform_full_block", CcgoFunc: "md5_transform_full_block", Notes: "full 64-step MD5"},
	}
}
