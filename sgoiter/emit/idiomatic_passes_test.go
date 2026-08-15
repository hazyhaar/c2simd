package emit

import (
	"strings"
	"testing"
)

func TestArchCompoundAssigns(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{
			in:   "\tv2 = v2 + v14\n\tv35 = v35 + 1",
			want: "\tv2 += v14\n\tv35++",
		},
		{
			in:   "\tv10 = v10 - 1\n\tv1 = v1 ^ v2",
			want: "\tv10--\n\tv1 ^= v2",
		},
		{
			in:   "\th[i] = h[i] + val",
			want: "\th[i] += val",
		},
		{
			in:   "\tctx.Counter = ctx.Counter + 4",
			want: "\tctx.Counter += 4",
		},
		{
			in:   "\tv = v - a - b",
			want: "\tv = v - a - b",
		},
		{
			in:   "\tv = v >> k | y",
			want: "\tv = v >> k | y",
		},
		{
			in:   "\tv = v - (a - b)",
			want: "\tv -= (a - b)",
		},
		{
			in:   "\tA = A * B + C",
			want: "\tA = A * B + C",
		},
		{
			in:   "\tA = A ^ B + C",
			want: "\tA = A ^ B + C",
		},
		{
			in:   "\tA = A * (B + C)",
			want: "\tA *= (B + C)",
		},
	}
	for _, tc := range tests {
		got := archCompoundAssigns(tc.in)
		if got != tc.want {
			t.Errorf("archCompoundAssigns(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

func TestArchStripIndexIntCasts(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{
			in:   "buf[int(v3)] = 1",
			want: "buf[v3] = 1",
		},
		{
			in:   "buf[int(v1):int(v2)]",
			want: "buf[v1:v2]",
		},
		{
			in:   "buf[int(v1):]",
			want: "buf[v1:]",
		},
		{
			in:   "buf[:int(v2)]",
			want: "buf[:v2]",
		},
	}
	for _, tc := range tests {
		got := archStripIndexIntCasts(tc.in)
		if got != tc.want {
			t.Errorf("archStripIndexIntCasts(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

func TestAstBuiltinMinMax(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"nested", `v10 = func() uint64 { if (func() int { if v1 < v2 { return 1 }; return 0 }()) != 0 { return v1 }; return v2 }()`, `v10 = min(v1, v2)`},
		{"simple", `v10 = func() uint64 { if v1 < v2 { return v1 }; return v2 }()`, `v10 = min(v1, v2)`},
		{"le", `v10 = func() uint64 { if v1 <= v2 { return v1 }; return v2 }()`, `v10 = min(v1, v2)`},
		{"ge", `v10 = func() uint64 { if v1 >= v2 { return v1 }; return v2 }()`, `v10 = max(v1, v2)`},
		{"casts_appariés", `v10 = func() uint64 { if uint64(a) < uint64(b) { return uint64(a) }; return uint64(b) }()`, `v10 = min(uint64(a), uint64(b))`},
		{"cast_litteral", `v10 = func() uint32 { if digest_size <= 64 { return digest_size }; return uint32(64) }()`, `v10 = min(digest_size, uint32(64))`},
		{"cast_litteral_nested", `v10 = func() uint64 { if (func() int { if ctx.Hash_size < 64 { return 1 }; return 0 }()) != 0 { return ctx.Hash_size }; return uint64(64) }()`, `v10 = min(ctx.Hash_size, uint64(64))`},
	}
	for _, tc := range cases {
		if got := astBuiltinMinMax(tc.in); got != tc.want {
			t.Errorf("%s : astBuiltinMinMax(%q) = %q; want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

// Non-régression du défaut démontré par oracle (audit 2026-08-15) : une
// comparaison TRONQUÉE par cast avec retours pleine largeur ne doit JAMAIS
// devenir min/max — la variante regex le faisait (divergence a=1, b=2^32).
func TestAstBuiltinMinMaxRefusesCastMismatch(t *testing.T) {
	cases := []string{
		`v1 = func() uint64 { if int32(a) < int32(b) { return a }; return b }()`,
		`v1 = func() uint64 { if int32(a) < b { return a }; return b }()`,
		`v1 = func() uint64 { if a < b { return uint32(a) }; return b }()`,
	}
	for _, in := range cases {
		if got := astBuiltinMinMax(in); got != in {
			t.Errorf("réécriture interdite : %q -> %q", in, got)
		}
	}
}

func TestArchBalanceAdditionTrees(t *testing.T) {
	in := "\tv105 = int64((((((((((t0) + (t1)) + (t2)) + (t3)) + (t4)) + (t5)) + (t6)) + (t7)) + (t8)) + (t9))"
	got := archBalanceAdditionTrees(in)
	want := "\tv105 = int64(((t0 + t1) + ((t2) + (t3 + t4))) + ((t5 + t6) + ((t7) + (t8 + t9))))"
	if got != want {
		t.Errorf("archBalanceAdditionTrees = %q; want %q", got, want)
	}
}

func TestArchFoldPingPongCasts(t *testing.T) {
	in := "\tval := uint64(uint64(x))\n\tv := int64(int64(y))"
	want := "\tval := uint64(x)\n\tv := int64(y)"
	got := archFoldPingPongCasts(in)
	if got != want {
		t.Errorf("archFoldPingPongCasts = %q; want %q", got, want)
	}
}

func TestArchStripRedundantLiteralCasts(t *testing.T) {
	in := "\tif ctx.C_idx == uint64(16) { ctx.C_idx = 0 }\n\tif v3 != uint32(64) { v4 = 0 }\n\tif v5 < uint64(32) { v5 += uint64(1) }"
	want := "\tif ctx.C_idx == 16 { ctx.C_idx = 0 }\n\tif v3 != 64 { v4 = 0 }\n\tif v5 < 32 { v5 += 1 }"
	got := archStripRedundantLiteralCasts(in)
	if got != want {
		t.Errorf("archStripRedundantLiteralCasts = %q; want %q", got, want)
	}
}

func TestArchPowerOfTwoShifts(t *testing.T) {
	in := "v105 -= (v405 * (1 << 26))"
	want := "v105 -= (v405 << 26)"
	got := archPowerOfTwoShifts(in)
	if got != want {
		t.Errorf("archPowerOfTwoShifts = %q; want %q", got, want)
	}
}

func TestArchFoldRotateLeftConstants(t *testing.T) {
	in := "\tv1 := bits.RotateLeft64(x, 64-24)\n\tv2 := bits.RotateLeft32(y, 32-12)"
	want := "\tv1 := bits.RotateLeft64(x, -24)\n\tv2 := bits.RotateLeft32(y, -12)"
	got := archFoldRotateLeftConstants(in)
	if got != want {
		t.Errorf("archFoldRotateLeftConstants = %q; want %q", got, want)
	}
}

func TestAstSimplifyNegatedComparisons(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"ident", "\tif !(v17 != 0) {\n\t\tv1 = 0\n\t}", "\tif v17 == 0 {\n\t\tv1 = 0\n\t}"},
		{"eq", "\tif !(flag == 0) {\n\t\tv1 = 0\n\t}", "\tif flag != 0 {\n\t\tv1 = 0\n\t}"},
		// La forme sur APPEL que la regex laissait passer (ge.go:33 du vivant).
		{"appel", "\tif !(Invsqrt(h.X[:], h.X[:]) != 0) {\n\t\tv1 = 0\n\t}", "\tif Invsqrt(h.X[:], h.X[:]) == 0 {\n\t\tv1 = 0\n\t}"},
		{"pas_zero", "\tif !(v != 1) {\n\t\tv1 = 0\n\t}", "\tif !(v != 1) {\n\t\tv1 = 0\n\t}"},
	}
	for _, tc := range cases {
		if got := astSimplifyNegatedComparisons(tc.in); got != tc.want {
			t.Errorf("%s : %q -> %q ; want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestAstStripAssignLiteralCasts(t *testing.T) {
	in := "\th[0] = int32(1)\n\tv := int32(1)\n\tx = uint8(0xff)"
	want := "\th[0] = 1\n\tv := int32(1)\n\tx = 0xff"
	if got := astStripAssignLiteralCasts(in); got != want {
		t.Errorf("strip assign casts = %q; want %q", got, want)
	}
}

func TestAstStripDeadTypes(t *testing.T) {
	in := `package p

type S25 struct {
	X int64
}

type Ge struct {
	X [10]int32
}

var g Ge

func F() { _ = g }
`
	got := astStripDeadGlobals(in)
	if strings.Contains(got, "S25") {
		t.Errorf("type mort survivant : %q", got)
	}
	if !strings.Contains(got, "type Ge") {
		t.Errorf("type vivant supprimé : %q", got)
	}
}

func TestAstFoldGapLiteralConstants(t *testing.T) {
	in := "\tgap := Gap(ad_size, uint64(16))\n\tgap2 := Gap(text_size, 16)"
	want := "\tgap := (-ad_size) & 15\n\tgap2 := (-text_size) & 15"
	got := astFoldGapLiteralConstants(in)
	if got != want {
		t.Errorf("astFoldGapLiteralConstants = %q; want %q", got, want)
	}
}

// Non-régression du défaut latent démontré (audit 2026-08-15) : un argument
// composé doit être parenthésé sous la négation ((-(a + b)) & 15, jamais
// (-a + b) & 15), et le remplacement entier est parenthésé quand le contexte
// parent lie plus fort que &.
func TestAstFoldGapLiteralConstantsPrecedence(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"somme", "\tgap := Gap(a + b, 16)", "\tgap := (-(a + b)) & 15"},
		{"comparaison", "\tif Gap(x, 16) > 0 {\n\t\tx++\n\t}", "\tif (-x) & 15 > 0 {\n\t\tx++\n\t}"},
		{"produit", "\tv := 2 * Gap(x, 16)", "\tv := 2 * ((-x) & 15)"},
		{"argument", "\tf(Gap(x, 16))", "\tf((-x) & 15)"},
		{"pas_16", "\tgap := Gap(x, 32)", "\tgap := Gap(x, 32)"},
	}
	for _, tc := range cases {
		if got := astFoldGapLiteralConstants(tc.in); got != tc.want {
			t.Errorf("%s : astFoldGapLiteralConstants(%q) = %q; want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestArchUnrollBlake2bSigma(t *testing.T) {
	in := "\tv91 = 0\n\tfor v91 < 12 {\n\t\tv20 += ctx.Input[blake2b_compress_sigma[0]]\n\t}\n\tctx.Hash[0] ^="
	got := archUnrollBlake2bSigma(in)
	if strings.Contains(got, "blake2b_compress_sigma") || strings.Contains(got, "for v91 < 12") {
		t.Errorf("archUnrollBlake2bSigma still contains dynamic loop/table: %s", got)
	}
	if !strings.Contains(got, "Blake2b Tour 0") || !strings.Contains(got, "Blake2b Tour 11") {
		t.Errorf("archUnrollBlake2bSigma does not contain unrolled rounds: %s", got)
	}
}

func TestArchEmitX25519InverseLm2Loop(t *testing.T) {
	in := "\tv28 = 252\n\tfor v28 >= 0 {\n\t\tMultiply(v27, v6, v6)\n\t}\n\tv50 = 0"
	got := archEmitX25519InverseLm2Loop(in)
	if strings.Contains(got, "v28 = 252") || strings.Contains(got, "for v28 >= 0") {
		t.Errorf("ancienne boucle SSA survivante : %s", got)
	}
	if strings.Count(got, "Multiply(") > 4 {
		t.Errorf("déroulage straight-line encore présent : %s", got)
	}
	if !strings.Contains(got, "var lm2Bytes = [32]byte{") {
		t.Errorf("table lm2Bytes absente du texte émis : %s", got)
	}
	if !strings.Contains(got, "for bit := 252; bit >= 0; bit--") {
		t.Errorf("boucle compacte absente : %s", got)
	}
	if !strings.Contains(got, "le if est admissible") {
		t.Errorf("commentaire de bit public absent : %s", got)
	}
	if !strings.Contains(got, "0xeb, 0xd3, 0xf5, 0x5c") {
		t.Errorf("octets L-2 absents : %s", got)
	}
}


