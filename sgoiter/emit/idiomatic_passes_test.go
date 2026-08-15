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

func TestArchBuiltinMinMax(t *testing.T) {
	in := `v10 = func() uint64 { if (func() int { if v1 < v2 { return 1 }; return 0 }()) != 0 { return v1 }; return v2 }()`
	want := `v10 = min(v1, v2)`
	got := archBuiltinMinMax(in)
	if got != want {
		t.Errorf("archBuiltinMinMax = %q; want %q", got, want)
	}

	inSimple := `v10 = func() uint64 { if v1 < v2 { return v1 }; return v2 }()`
	wantSimple := `v10 = min(v1, v2)`
	gotSimple := archBuiltinMinMax(inSimple)
	if gotSimple != wantSimple {
		t.Errorf("archBuiltinMinMax simple = %q; want %q", gotSimple, wantSimple)
	}

	inLE := `v10 = func() uint64 { if v1 <= v2 { return v1 }; return v2 }()`
	wantLE := `v10 = min(v1, v2)`
	if gotLE := archBuiltinMinMax(inLE); gotLE != wantLE {
		t.Errorf("archBuiltinMinMax <= = %q; want %q", gotLE, wantLE)
	}

	inGE := `v10 = func() uint64 { if v1 >= v2 { return v1 }; return v2 }()`
	wantGE := `v10 = max(v1, v2)`
	if gotGE := archBuiltinMinMax(inGE); gotGE != wantGE {
		t.Errorf("archBuiltinMinMax >= = %q; want %q", gotGE, wantGE)
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

func TestArchSimplifyDoubleNegations(t *testing.T) {
	in := "\tif !(v17 != 0) { return 0 }\n\tif !(flag == 0) { break }"
	want := "\tif v17 == 0 { return 0 }\n\tif flag != 0 { break }"
	got := archSimplifyDoubleNegations(in)
	if got != want {
		t.Errorf("archSimplifyDoubleNegations = %q; want %q", got, want)
	}
}

func TestArchFoldGapLiteralConstants(t *testing.T) {
	in := "\tgap := Gap(ad_size, uint64(16))\n\tgap2 := Gap(text_size, 16)"
	want := "\tgap := (-ad_size) & 15\n\tgap2 := (-text_size) & 15"
	got := archFoldGapLiteralConstants(in)
	if got != want {
		t.Errorf("archFoldGapLiteralConstants = %q; want %q", got, want)
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

func TestArchUnrollX25519InverseChain(t *testing.T) {
	in := "\tv28 = 252\n\tfor v28 >= 0 {\n\t\tMultiply(v27, v6, v6)\n\t}\n\tv50 = 0"
	got := archUnrollX25519InverseChain(in)
	if strings.Contains(got, "v28 = 252") || strings.Contains(got, "for v28 >= 0") {
		t.Errorf("archUnrollX25519InverseChain still contains dynamic loop: %s", got)
	}
	if !strings.Contains(got, "Chaîne d'addition ARCHTIME constant-time pour L-2") {
		t.Errorf("archUnrollX25519InverseChain does not contain addition chain: %s", got)
	}
}


