package emit

import (
	"strings"
	"testing"
)

func TestAstForwardAdjacentStoreLoad(t *testing.T) {
	in := "\tb.A[v1782] = v2960\n\tv2967 := b.A[v1782]\n\tb.A[v1549] = v2972\n\tv2979 := b.A[v1549]"
	want := "\tb.A[v1782] = v2960\n\tv2967 := v2960\n\tb.A[v1549] = v2972\n\tv2979 := v2972"
	if got := astForwardAdjacentStoreLoad(in); got != want {
		t.Errorf("forward = %q; want %q", got, want)
	}
	// Non-adjacent ou indice différent : intouché.
	keep := "\tb.A[i] = v1\n\tx := b.A[j]"
	if got := astForwardAdjacentStoreLoad(keep); got != keep {
		t.Errorf("indice différent réécrit : %q", got)
	}
	sep := "\tb.A[i] = v1\n\tcall()\n\tx := b.A[i]"
	if got := astForwardAdjacentStoreLoad(sep); got != sep {
		t.Errorf("non-adjacent réécrit : %q", got)
	}
}

func TestAstDropIdentityMasks(t *testing.T) {
	in := "\tv1 := ^L[v4] & uint32(0xffffffff)\n\tv2 := uint64(0xffffffffffffffff) & x\n\tv3 := y & uint32(0xfffffff)"
	want := "\tv1 := ^L[v4]\n\tv2 := x\n\tv3 := y & uint32(0xfffffff)"
	if got := astDropIdentityMasks(in); got != want {
		t.Errorf("masks = %q; want %q", got, want)
	}
}

func TestAstSimplifyIndexCasts(t *testing.T) {
	in := "\th[int(uint64(1)+v3)] = 0\n\tctx.C[int(ctx.C_idx)] = m\n\tdst[int(v3*4)] = 0"
	want := "\th[v3+1] = 0\n\tctx.C[ctx.C_idx] = m\n\tdst[int(v3*4)] = 0"
	if got := astSimplifyIndexCasts(in); got != want {
		t.Errorf("index casts = %q; want %q", got, want)
	}
}

func TestAstUnrollConstTripLoops(t *testing.T) {
	in := "\tv20 = 0\n\tfor v20 < 1 {\n\t\tctx.R[v20] &= 0xfffffff\n\t\tv20++\n\t}"
	got := astUnrollConstTripLoops(in)
	if !strings.Contains(got, "ctx.R[0] &= 0xfffffff") || strings.Contains(got, "for ") {
		t.Errorf("trip 1 = %q", got)
	}
	// Trip 3 : trois copies aux indices 0..2.
	in3 := "\tv30 = 0\n\tfor v30 < 3 {\n\t\th[v30] = f[v30] + g[v30]\n\t\tv30++\n\t}"
	got3 := astUnrollConstTripLoops(in3)
	for _, wantLine := range []string{"h[0] = f[0] + g[0]", "h[1] = f[1] + g[1]", "h[2] = f[2] + g[2]"} {
		if !strings.Contains(got3, wantLine) {
			t.Errorf("trip 3 : %q absent de %q", wantLine, got3)
		}
	}
	if strings.Contains(got3, "for ") {
		t.Errorf("trip 3 : boucle survivante %q", got3)
	}
	// SANS le `v = 0` adjacent : refus (valeur d'entrée non garantie).
	noInit := "\tx = y\n\tfor v40 < 2 {\n\t\th[v40] = 0\n\t\tv40++\n\t}"
	if got := astUnrollConstTripLoops(noInit); got != noInit {
		t.Errorf("déroulé sans garantie d'entrée : %q", got)
	}
	// Compteur utilisé APRÈS la boucle : refus (v vaudrait N).
	usedAfter := "\tv50 = 0\n\tfor v50 < 2 {\n\t\th[v50] = 0\n\t\tv50++\n\t}\n\tz = v50"
	if got := astUnrollConstTripLoops(usedAfter); got != usedAfter {
		t.Errorf("déroulé avec compteur vivant après : %q", got)
	}
	// Trip au-delà du plafond : refus.
	big := "\tv60 = 0\n\tfor v60 < 11 {\n\t\th[v60] = 0\n\t\tv60++\n\t}"
	if got := astUnrollConstTripLoops(big); got != big {
		t.Errorf("trip 11 déroulé : %q", got)
	}
}

func TestAstTransformShiftedClearLoops(t *testing.T) {
	in := "\th[0] = 1\n\tv3 = 0\n\tfor v3 < 9 {\n\t\th[v3+1] = 0\n\t\tv3++\n\t}"
	got := astTransformShiftedClearLoops(in)
	if !strings.Contains(got, "clear(h[1:10])") || strings.Contains(got, "for ") {
		t.Errorf("clear décalé = %q", got)
	}
	// K=0 : forme [:N].
	in0 := "\tv4 = 0\n\tfor v4 < 8 {\n\t\tbuf[v4] = uint8(0)\n\t\tv4++\n\t}"
	got0 := astTransformShiftedClearLoops(in0)
	if !strings.Contains(got0, "clear(buf[:8])") {
		t.Errorf("clear K=0 = %q", got0)
	}
	// Membre droit non nul : refus.
	keep := "\tv5 = 0\n\tfor v5 < 8 {\n\t\tbuf[v5] = 1\n\t\tv5++\n\t}"
	if got := astTransformShiftedClearLoops(keep); got != keep {
		t.Errorf("clear sur valeur non nulle : %q", got)
	}
}

func TestAstInsertBoundsHints(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("func Rounds(out []uint32, in []uint32) {\n")
	for i := 0; i < 16; i++ {
		sb.WriteString("\tout[" + itoa(i) + "] = in[" + itoa(i) + "]\n")
	}
	sb.WriteString("}\n")
	got := astInsertBoundsHints(sb.String())
	if !strings.Contains(got, "_ = out[15]") || !strings.Contains(got, "_ = in[15]") {
		t.Errorf("hints absents : %q", got)
	}
	// Moins de 8 indices constants : pas de hint.
	small := "func F(a []byte) {\n\ta[0] = 1\n\ta[1] = 2\n}\n"
	if got := astInsertBoundsHints(small); strings.Contains(got, "_ = a[") {
		t.Errorf("hint injecté à tort : %q", got)
	}
}

func itoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}

func TestAstStripDeadGlobals(t *testing.T) {
	in := `package p

var sigma = [4]byte{1, 2, 3, 4}
var used = [2]byte{1, 2}
var zero_arr [8]byte
var zero = zero_arr[:]
var kept [4]byte

func F() byte { return used[0] }
`
	got := astStripDeadGlobals(in, "kept")
	for _, dead := range []string{"var sigma", "var zero_arr", "var zero "} {
		if strings.Contains(got, dead) {
			t.Errorf("globale morte survivante %q dans %q", dead, got)
		}
	}
	for _, live := range []string{"var used", "var kept"} {
		if !strings.Contains(got, live) {
			t.Errorf("globale vivante supprimée %q dans %q", live, got)
		}
	}
}
