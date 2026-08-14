package sgoiterbench

import (
	"path/filepath"
)

// Kind classifies the kernel interface shape and fixture set.
type Kind string

const (
	KindHash64   Kind = "hash64"
	KindHash32   Kind = "hash32"
	KindXor      Kind = "fast_xor"
	KindBase64   Kind = "base64"
	KindCrypto   Kind = "crypto"
	KindSqli     Kind = "sqli"
)

// Spec describes how a dogfood C kernel maps to CGO/sgoiter/ccgo.
type Spec struct {
	ID        string
	Kind      Kind
	RelCPath  string
	FuncName  string
	Notes     string
	CgoWrap   string
}

// Catalog holds all 12 dogfood kernels.
func Catalog(c2root string) []Spec {
	u := func(rel string) string {
		return filepath.Join(c2root, "spec/c_sources/testdata/c_sources", rel)
	}
	_ = u
	return []Spec{
		{
			ID:       "fnv1a_64",
			Kind:     KindHash64,
			RelCPath: "fnv1a_64.c",
			FuncName: "fnv1a_64",
			Notes:    "FNV-1a 64-bit hash over byte slice",
		},
		{
			ID:       "crc32_ieee",
			Kind:     KindHash32,
			RelCPath: "crc32_ieee.c",
			FuncName: "crc32_ieee",
			Notes:    "CRC32 IEEE 802.3 over byte slice",
		},
		{
			ID:       "fast_xor",
			Kind:     KindXor,
			RelCPath: "fast_xor.c",
			FuncName: "fast_xor_bytes",
			Notes:    "In-place/out-of-place XOR 8-byte unrolled",
		},
		{
			ID:       "siphash24",
			Kind:     KindHash64,
			RelCPath: "siphash24.c",
			FuncName: "siphash24",
			Notes:    "SipHash-2-4 64-bit keyed hash",
		},
		{
			ID:       "murmur3_x86_32",
			Kind:     KindHash32,
			RelCPath: "murmur3_x86_32.c",
			FuncName: "murmur3_x86_32",
			Notes:    "MurmurHash3 32-bit (x86 variant)",
		},
		{
			ID:       "md5_transform",
			Kind:     KindCrypto,
			RelCPath: "md5_transform.c",
			FuncName: "md5_transform_block",
			Notes:    "MD5 64-byte block transform",
		},
		{
			ID:       "poly1305_block5",
			Kind:     KindCrypto,
			RelCPath: "poly1305_block5.c",
			FuncName: "poly1305_block5",
			Notes:    "Poly1305 16-byte block accumulator",
		},
		{
			ID:       "chacha20_qr",
			Kind:     KindCrypto,
			RelCPath: "chacha20_qr.c",
			FuncName: "chacha20_quarter_round",
			Notes:    "ChaCha20 quarter-round (4x uint32)",
		},
		{
			ID:       "base64_simd",
			Kind:     KindBase64,
			RelCPath: "base64_simd.c",
			FuncName: "base64_encode_stream",
			Notes:    "Base64 encoder stream93",
		},
		{
			ID:       "blake2b_compress",
			Kind:     KindCrypto,
			RelCPath: "blake2b_compress.c",
			FuncName: "blake2b_compress_block",
			Notes:    "BLAKE2b 128-byte block compress",
		},
		{
			ID:       "tweetnacl_dogfood",
			Kind:     KindCrypto,
			RelCPath: "tweetnacl_dogfood.c",
			FuncName: "crypto_stream_salsa20",
			Notes:    "Salsa20 stream04 from TweetNaCl",
		},
		{
			ID:       "libinjection_sqli",
			Kind:     KindSqli,
			RelCPath: "libinjection_sqli.c",
			FuncName: "libinjection_is_sqli",
			Notes:    "Libinjection SQLi 3-pass01",
		},
	}
}

// ExtraTier classifies how far an extra lib is in the sgoiter pipeline.
type ExtraTier string

const (
	// ExtraTierUpstream = full upstream .c (front-smoke only; often fail/stub).
	ExtraTierUpstream ExtraTier = "upstream_full"
	// ExtraTierDogfood = thin subset kernel under testdata/c_sources (emit OK).
	ExtraTierDogfood ExtraTier = "dogfood_kernel"
)

// ExtraSpec is a C→Go candidate outside the core 12 tribench kernels.
type ExtraSpec struct {
	ID         string
	Tier       ExtraTier
	RelCPath   string // relative to c2root
	HPM55Name  string
	Upstream   string // github URL
	Notes      string
	FrontExpect string // ok|fail_asm|fail_empty|stub — documented first-pass
}

// CatalogExtra — libs HPM55 sgoiter-{stb-image,cjson,yyjson,utf8proc} + dogfood slices.
func CatalogExtra(c2root string) []ExtraSpec {
	_ = c2root
	return []ExtraSpec{
		// --- upstream full (qualification only) ---
		{
			ID:          "upstream_cjson",
			Tier:        ExtraTierUpstream,
			RelCPath:    "spec/c_sources/upstream/cjson/cJSON.c",
			HPM55Name:   "sgoiter-cjson",
			Upstream:    "https://github.com/DaveGamble/cJSON",
			Notes:       "full cJSON.c — front OK but near-empty emit (structs + stubs)",
			FrontExpect: "stub",
		},
		{
			ID:          "upstream_yyjson",
			Tier:        ExtraTierUpstream,
			RelCPath:    "spec/c_sources/upstream/yyjson/yyjson.c",
			HPM55Name:   "sgoiter-yyjson",
			Upstream:    "https://github.com/ibireme/yyjson",
			Notes:       "full yyjson.c — asm barriers stripped (no-op); emit partial stub-heavy",
			FrontExpect: "stub",
		},
		{
			ID:          "upstream_utf8proc",
			Tier:        ExtraTierUpstream,
			RelCPath:    "spec/c_sources/upstream/utf8proc/utf8proc.c",
			HPM55Name:   "sgoiter-utf8proc",
			Upstream:    "https://github.com/JuliaStrings/utf8proc",
			Notes:       "needs utf8proc_data.c; front OK stub emit",
			FrontExpect: "stub",
		},
		{
			ID:          "upstream_stb_image",
			Tier:        ExtraTierUpstream,
			RelCPath:    "spec/c_sources/upstream/stb/stb_image_impl.c",
			HPM55Name:   "sgoiter-stb-image",
			Upstream:    "https://github.com/nothings/stb",
			Notes:       "STB_IMAGE_IMPLEMENTATION — err_empty/include (amalgamation / header)",
			FrontExpect: "fail_include",
		},
		// --- dogfood kernels (subset, emit green) ---
		{
			ID:          "cjson_number_dogfood",
			Tier:        ExtraTierDogfood,
			RelCPath:    "spec/c_sources/testdata/c_sources/cjson_number_dogfood.c",
			HPM55Name:   "sgoiter-cjson",
			Upstream:    "https://github.com/DaveGamble/cJSON",
			Notes:       "parse u64 + null literal",
			FrontExpect: "ok",
		},
		{
			ID:          "yyjson_digit_dogfood",
			Tier:        ExtraTierDogfood,
			RelCPath:    "spec/c_sources/testdata/c_sources/yyjson_digit_dogfood.c",
			HPM55Name:   "sgoiter-yyjson",
			Upstream:    "https://github.com/ibireme/yyjson",
			Notes:       "count digits + skip ws",
			FrontExpect: "ok",
		},
		{
			ID:          "utf8_iterate_dogfood",
			Tier:        ExtraTierDogfood,
			RelCPath:    "spec/c_sources/testdata/c_sources/utf8_iterate_dogfood.c",
			HPM55Name:   "sgoiter-utf8proc",
			Upstream:    "https://github.com/JuliaStrings/utf8proc",
			Notes:       "utf8 class + 1–2 byte iterate",
			FrontExpect: "ok",
		},
		{
			ID:          "stbi_crc_dogfood",
			Tier:        ExtraTierDogfood,
			RelCPath:    "spec/c_sources/testdata/c_sources/stbi_crc_dogfood.c",
			HPM55Name:   "sgoiter-stb-image",
			Upstream:    "https://github.com/nothings/stb",
			Notes:       "CRC32 bit-by-bit (PNG-style poly)",
			FrontExpect: "ok",
		},
	}
}

// CCGOPkgEntry is one HPM55 child under parent ccgo-pkg (quota 16 = 12 core + 4 extra).
// Spec: sgoiter/spec/ccgo_pkg/ROSTER.md · roster.json
type CCGOPkgEntry struct {
	N          int
	HPM55      string
	Family     string // core | extra
	TribenchID string // empty if extra-only
	Status     string
}

// CatalogCCGOPkg returns the formal 16-subproject roster (stable order).
func CatalogCCGOPkg() []CCGOPkgEntry {
	return []CCGOPkgEntry{
		{1, "sgoiter-fnv1a", "core", "fnv1a_64", "banc_vert"},
		{2, "sgoiter-crc32", "core", "crc32_ieee", "banc_vert"},
		{3, "sgoiter-fast-xor", "core", "fast_xor", "banc_vert"},
		{4, "sgoiter-siphash", "core", "siphash24", "banc_vert"},
		{5, "sgoiter-murmur3", "core", "murmur3_x86_32", "banc_vert"},
		{6, "sgoiter-blake2b", "core", "blake2b_compress", "banc_vert"},
		{7, "sgoiter-chacha20", "core", "chacha20_qr", "banc_vert"},
		{8, "sgoiter-md5", "core", "md5_transform", "banc_vert"},
		{9, "sgoiter-poly1305", "core", "poly1305_block5", "banc_vert"},
		{10, "sgoiter-base64", "core", "base64_simd", "banc_vert"},
		{11, "sgoiter-tweetnacl", "core", "tweetnacl_dogfood", "banc_vert_dette_p0"},
		{12, "sgoiter-libinjection", "core", "libinjection_sqli", "emit_ok"},
		{13, "sgoiter-cjson", "extra", "", "dogfood_ok_full_stub"},
		{14, "sgoiter-yyjson", "extra", "", "dogfood_ok_full_stub"},
		{15, "sgoiter-utf8proc", "extra", "", "dogfood_ok_full_stub"},
		{16, "sgoiter-stb-image", "extra", "", "dogfood_ok_full_fail_include"},
	}
}
