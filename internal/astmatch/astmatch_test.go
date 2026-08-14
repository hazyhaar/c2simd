package astmatch_test

import (
	"bytes"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/internal/astmatch"
)

func formatGo(t *testing.T, src string) string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v\n%s", err, src)
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		t.Fatalf("format: %v", err)
	}
	return buf.String()
}

func TestAppliedRulesKind(t *testing.T) {
	applied := astmatch.AppliedRules()
	if len(applied) == 0 {
		t.Fatal("AppliedRules vide")
	}
	for _, r := range applied {
		if r.Kind != astmatch.KindRewrite {
			t.Errorf("%s: Kind=%s want rewrite", r.Symbol, r.Kind)
		}
	}
	hw := astmatch.HandwritePointers()
	if len(hw) < 2 {
		t.Fatalf("HandwritePointers: got %d want ≥2 (poly, chacha)", len(hw))
	}
	for _, r := range hw {
		if r.Kind != astmatch.KindHandwritePointer {
			t.Errorf("%s: Kind=%s", r.Symbol, r.Kind)
		}
		if r.DeadCode {
			t.Errorf("%s handwrite_pointer ne doit pas être DeadCode", r.Symbol)
		}
	}
	// poly_blocks / chacha20_rounds ne doivent PAS être dans AppliedRules
	for _, r := range applied {
		if r.Symbol == "poly_blocks" || r.Symbol == "chacha20_rounds" {
			t.Errorf("%s ne doit pas être Kind=rewrite", r.Symbol)
		}
	}
}

func TestTransformAST_table(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantSub []string // sous-chaînes obligatoires après transform
		denySub []string // sous-chaînes interdites
	}{
		{
			name: "rotl32_call_to_bits",
			in: `package sample

import "modernc.org/libc"

func chacha20_step(tls *libc.TLS, x uint32) uint32 {
	return rotl32(tls, x, uint32(16))
}
`,
			wantSub: []string{
				`bits.RotateLeft32(x, int(16))`,
				`"math/bits"`,
			},
			denySub: []string{
				`rotl32(`,
			},
		},
		{
			name: "tweetnacl_L32_call",
			in: `package sample

import "modernc.org/libc"

type u32 = uint64

func L32(tls *libc.TLS, x u32, c int32) u32 {
	return x<<c | x>>(32-c)
}

func core(tls *libc.TLS, x u32) u32 {
	return L32(tls, x, int32(7))
}
`,
			wantSub: []string{
				`u32(bits.RotateLeft32(uint32(x), int(7)))`,
			},
			denySub: []string{
				`func L32`,
				`L32(tls,`,
			},
		},
		{
			name: "rotl32_def_deadcode_purged",
			in: `package sample

import "modernc.org/libc"

func rotl32(tls *libc.TLS, x, n uint32) uint32 {
	return x<<n | x>>(32-n)
}

func use(tls *libc.TLS, x uint32) uint32 {
	return rotl32(tls, x, 16)
}
`,
			wantSub: []string{
				`bits.RotateLeft32`,
				`func use`,
			},
			denySub: []string{
				`func rotl32`,
			},
		},
		{
			name: "load32_le_body_unsafe",
			in: `package sample

import "modernc.org/libc"

func load32_le(tls *libc.TLS, s uintptr) uint32 {
	return uint32(0)
}
`,
			wantSub: []string{
				`*(*uint32)`,
				`"unsafe"`,
				`func load32_le`,
			},
			denySub: []string{
				`return uint32(0)`,
			},
		},
		{
			name: "store32_le_body_unsafe",
			in: `package sample

import "modernc.org/libc"

func store32_le(tls *libc.TLS, out uintptr, in uint32) {
	_ = out
	_ = in
}
`,
			wantSub: []string{
				`unsafe.Pointer`,
				`"unsafe"`,
			},
		},
		{
			name: "tls_t0_elision_unexported_def_and_call",
			in: `package sample

import "modernc.org/libc"

func pure_add(tls *libc.TLS, a, b uint32) uint32 {
	return a + b
}

func caller(tls *libc.TLS, a, b uint32) uint32 {
	return pure_add(tls, a, b)
}
`,
			wantSub: []string{
				`func pure_add(a, b uint32) uint32`,
				`return pure_add(a, b)`,
			},
			denySub: []string{
				`func pure_add(tls *libc.TLS`,
				`pure_add(tls,`,
			},
		},
		{
			name: "tls_t0_fixed_point_caller",
			// Caller ne fait que transmettre tls à une T0 → doit finir T0 aussi.
			in: `package sample

import "modernc.org/libc"

func leaf(tls *libc.TLS, x uint32) uint32 {
	return x + 1
}

func mid(tls *libc.TLS, x uint32) uint32 {
	return leaf(tls, x)
}
`,
			wantSub: []string{
				`func leaf(x uint32) uint32`,
				`func mid(x uint32) uint32`,
				`return leaf(x)`,
			},
			denySub: []string{
				`func mid(tls *libc.TLS`,
				`leaf(tls,`,
				`modernc.org/libc`,
			},
		},
		{
			name: "tls_t0_exported_keeps_abi",
			in: `package sample

import "modernc.org/libc"

func PureAdd(tls *libc.TLS, a, b uint32) uint32 {
	return a + b
}
`,
			wantSub: []string{
				`func PureAdd(tls *libc.TLS`,
				`modernc.org/libc`,
			},
			denySub: []string{
				`func PureAdd(a, b uint32)`,
			},
		},
		{
			name: "tls_t2_kept_when_used",
			in: `package sample

import "modernc.org/libc"

func uses_tls(tls *libc.TLS, a uint32) uint32 {
	_ = tls
	return a
}
`,
			wantSub: []string{
				`func uses_tls(tls *libc.TLS`,
			},
		},
		{
			name: "rotate_pattern_or_shifts",
			in: `package sample

func rot_inline(x uint32) uint32 {
	return x<<16 | x>>(32-16)
}
`,
			wantSub: []string{
				`bits.RotateLeft32`,
				`"math/bits"`,
			},
		},
		{
			name: "rotr64_xor_blake2b_shape",
			// Forme ccgo de ROTR64(x,32) = (x>>32) ^ (x<<(64-32))
			in: `package sample

func rotr(x uint64) uint64 {
	return x>>32 ^ x<<(64-32)
}
`,
			wantSub: []string{
				`bits.RotateLeft64(x, int(32))`,
				`"math/bits"`,
			},
			denySub: []string{
				`x >> 32`,
			},
		},
		{
			name: "rotr64_with_xor_operand",
			in: `package sample

func g(d, a uint64) uint64 {
	return (d ^ a)>>24 ^ (d ^ a)<<(64-24)
}
`,
			wantSub: []string{
				`bits.RotateLeft64`,
				`int(40)`, // RotateRight 24 = RotateLeft 40
			},
		},
		{
			name: "rotr32_or",
			in: `package sample

func ror32(x uint32) uint32 {
	return x>>7 | x<<(32-7)
}
`,
			wantSub: []string{
				`bits.RotateLeft32(x, int(25))`,
			},
		},
		{
			name: "drop_unused_libc_after_tls_t0",
			in: `package sample

import "modernc.org/libc"

func pure_add(tls *libc.TLS, a, b uint32) uint32 {
	return a + b
}
`,
			wantSub: []string{
				`func pure_add(a, b uint32) uint32`,
			},
			denySub: []string{
				`modernc.org/libc`,
				`libc.TLS`,
			},
		},
		{
			name: "uintptr_neg_offset_to_sub",
			in: `package sample

func prev(str uintptr) uintptr {
	return str + uintptr(-int32(1))
}
`,
			wantSub: []string{
				`str - uintptr(1)`,
			},
			denySub: []string{
				`uintptr(-`,
			},
		},
		{
			name: "uintptr_neg_libc_fromint32",
			in: `package sample

func prev(str uintptr) uintptr {
	return str + uintptr(-libc.Int32FromInt32(1))
}
`,
			wantSub: []string{
				`str - uintptr(1)`,
			},
			denySub: []string{
				`uintptr(-`,
			},
		},
		{
			name: "uintptr_neg_inside_unsafe_pointer",
			in: `package sample

import "unsafe"

func prev(str uintptr) int32 {
	return int32(*(*int8)(unsafe.Pointer(str + uintptr(-int32(1)))))
}
`,
			wantSub: []string{
				`str - uintptr(1)`,
			},
			denySub: []string{
				`uintptr(-`,
			},
		},
		{
			name: "iqlibc_builtin_memmove",
			in: `package sample

import "modernc.org/libc"

func mv(tls *libc.TLS, dst, src uintptr, n uint64) {
	iqlibc.__builtin_memmove(tls, dst, src, n)
}
`,
			wantSub: []string{
				`Xmemmove`,
				`libc.`,
			},
			denySub: []string{
				`iqlibc`,
				`__builtin_memmove`,
			},
		},
		{
			name: "no_generic_loop_simd_injection",
			// Garde F-20260810-q2 : une boucle scalaire reste scalaire.
			in: `package sample

func sum(xs []uint32) uint32 {
	var s uint32
	for i := 0; i < len(xs); i++ {
		s += xs[i]
	}
	return s
}
`,
			wantSub: []string{
				`for i := 0; i < len(xs); i++`,
			},
			denySub: []string{
				`simd`,
				`Int32x8`,
				`Float32x8`,
				`archsimd`,
			},
		},
		{
			name: "ccgo_up_scalar_load",
			in: `package sample

import "unsafe"

type uint64_t = uint64

func load(k uintptr) uint64_t {
	return **(**uint64_t)(__ccgo_up(k))
}

func __ccgo_up(n uintptr) unsafe.Pointer {
	return unsafe.Pointer(&n)
}
`,
			wantSub: []string{
				`*(*uint64_t)(unsafe.Pointer(k))`,
			},
			denySub: []string{
				`__ccgo_up`,
				`**(**uint64_t)`,
			},
		},
		{
			name: "ccgo_up_scalar_store",
			in: `package sample

import "unsafe"

type uint8_t = uint8

func store(p uintptr, v uint8_t) {
	**(**uint8_t)(__ccgo_up(p)) = v
}

func __ccgo_up(n uintptr) unsafe.Pointer {
	return unsafe.Pointer(&n)
}
`,
			wantSub: []string{
				`*(*uint8_t)(unsafe.Pointer(p)) = v`,
			},
			denySub: []string{
				`__ccgo_up`,
			},
		},
		{
			name: "ccgo_up_array_index",
			in: `package sample

import "unsafe"

type u32 = uint32

func at(bp uintptr, i int) u32 {
	return (**(**[16]u32)(__ccgo_up(bp)))[i]
}

func __ccgo_up(n uintptr) unsafe.Pointer {
	return unsafe.Pointer(&n)
}
`,
			// **(**[N]T) → (*[N]T)(unsafe.Pointer)[i] après polish index.
			wantSub: []string{
				`(*[16]u32)(unsafe.Pointer(bp))[i]`,
			},
			denySub: []string{
				`__ccgo_up`,
				`**(**[16]u32)`,
				`(*(*[16]u32)`,
			},
		},
		{
			name: "ccgo_up_offset_expr",
			in: `package sample

import "unsafe"

type uint64_t = uint64

func load1(k uintptr) uint64_t {
	return **(**uint64_t)(__ccgo_up(k + 1*8))
}

func __ccgo_up(n uintptr) unsafe.Pointer {
	return unsafe.Pointer(&n)
}
`,
			wantSub: []string{
				`*(*uint64_t)(unsafe.Pointer(k + 1*8))`,
			},
			denySub: []string{
				`__ccgo_up`,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := astmatch.TransformAST([]byte(tc.in))
			if err != nil {
				t.Fatalf("TransformAST: %v", err)
			}
			got := string(out)
			// re-format pour stabilité
			got = formatGo(t, got)
			for _, sub := range tc.wantSub {
				if !strings.Contains(got, sub) {
					t.Errorf("manque %q\n--- got ---\n%s", sub, got)
				}
			}
			for _, sub := range tc.denySub {
				if strings.Contains(got, sub) {
					t.Errorf("interdit présent %q\n--- got ---\n%s", sub, got)
				}
			}
		})
	}
}

// Compat : l'alias public reste branché sur TransformAST.
func TestTransformRotations_alias(t *testing.T) {
	in := []byte(`package sample
func f(x uint32) uint32 { return rotl32(x, uint32(8)) }
`)
	a, err := astmatch.TransformAST(in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := astmatch.TransformRotations(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatal("TransformRotations ≠ TransformAST")
	}
}
