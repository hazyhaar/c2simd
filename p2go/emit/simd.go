// Strate SIMD de l'emit — helpers DUALS : cœur scalaire dans main.go
// (p2go*Scalar), délégation sous !goexperiment.simd, variante simd/archsimd
// (AVX2, garde runtime, queue scalaire) sous goexperiment.simd. Doctrine pôle
// /devhoros/c2simd/CLAUDE.md : parité bit-exacte, fallback systématique.
package emit

type simdHelper struct {
	core string // défini dans main.go, sans build tag
	off  string // fichier !goexperiment.simd
	on   string // fichier goexperiment.simd (archsimd)
}

var simdHelpers = map[string]simdHelper{
	"sum": {
		core: `func p2goSumI64Scalar(a []int64) int64 {
	var s int64
	for _, v := range a {
		s += v
	}
	return s
}
`,
		off: `func p2goSumI64(a []int64) int64 { return p2goSumI64Scalar(a) }
`,
		on: `func p2goSumI64(a []int64) int64 {
	if !archsimd.X86.AVX2() {
		return p2goSumI64Scalar(a)
	}
	acc := archsimd.BroadcastInt64x4(0)
	i := 0
	for ; i+4 <= len(a); i += 4 {
		acc = acc.Add(archsimd.LoadInt64x4(a[i : i+4]))
	}
	var lanes [4]int64
	acc.StoreArray(&lanes)
	s := lanes[0] + lanes[1] + lanes[2] + lanes[3]
	for ; i < len(a); i++ {
		s += a[i]
	}
	return s
}
`,
	},
	"dot": {
		core: `func p2goDotI64Scalar(a, b []int64) int64 {
	var s int64
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}
`,
		off: `func p2goDotI64(a, b []int64) int64 { return p2goDotI64Scalar(a, b) }
`,
		// Multiply 64-bit bas émulé AVX2 : VPMULLQ est AVX-512, on compose
		// lo64(x·y) = uduq(lo,lo) + ((uduq(hi,lo)+uduq(lo,hi)) << 32) via
		// MulWidenEven (VPMULUDQ) — wraparound mod 2⁶⁴ identique au scalaire.
		on: `func p2goDotI64(a, b []int64) int64 {
	if !archsimd.X86.AVX2() {
		return p2goDotI64Scalar(a, b)
	}
	acc := archsimd.BroadcastUint64x4(0)
	i := 0
	for ; i+4 <= len(a); i += 4 {
		x := archsimd.LoadInt64x4(a[i : i+4]).AsUint64x4()
		y := archsimd.LoadInt64x4(b[i : i+4]).AsUint64x4()
		xl := x.AsUint32x8()
		yl := y.AsUint32x8()
		xh := x.ShiftAllRight(32).AsUint32x8()
		yh := y.ShiftAllRight(32).AsUint32x8()
		ll := xl.MulWidenEven(yl)
		cross := xh.MulWidenEven(yl).Add(xl.MulWidenEven(yh))
		acc = acc.Add(ll.Add(cross.ShiftAllLeft(32)))
	}
	var lanes [4]int64
	acc.AsInt64x4().StoreArray(&lanes)
	s := lanes[0] + lanes[1] + lanes[2] + lanes[3]
	for ; i < len(a); i++ {
		s += a[i] * b[i]
	}
	return s
}
`,
	},
	"max": {
		core: `func p2goMaxI64Scalar(cur int64, a []int64) int64 {
	for _, v := range a {
		if v > cur {
			cur = v
		}
	}
	return cur
}
`,
		off: `func p2goMaxI64(cur int64, a []int64) int64 { return p2goMaxI64Scalar(cur, a) }
`,
		// VPMAXSQ est AVX-512 : max émulé Greater (VPCMPGTQ) + IfElse (blend).
		on: `func p2goMaxI64(cur int64, a []int64) int64 {
	if !archsimd.X86.AVX2() || len(a) < 4 {
		return p2goMaxI64Scalar(cur, a)
	}
	acc := archsimd.LoadInt64x4(a[0:4])
	i := 4
	for ; i+4 <= len(a); i += 4 {
		v := archsimd.LoadInt64x4(a[i : i+4])
		acc = acc.IfElse(acc.Greater(v), v)
	}
	var lanes [4]int64
	acc.StoreArray(&lanes)
	for _, v := range lanes {
		if v > cur {
			cur = v
		}
	}
	for ; i < len(a); i++ {
		if a[i] > cur {
			cur = a[i]
		}
	}
	return cur
}
`,
	},
	"min": {
		core: `func p2goMinI64Scalar(cur int64, a []int64) int64 {
	for _, v := range a {
		if v < cur {
			cur = v
		}
	}
	return cur
}
`,
		off: `func p2goMinI64(cur int64, a []int64) int64 { return p2goMinI64Scalar(cur, a) }
`,
		on: `func p2goMinI64(cur int64, a []int64) int64 {
	if !archsimd.X86.AVX2() || len(a) < 4 {
		return p2goMinI64Scalar(cur, a)
	}
	acc := archsimd.LoadInt64x4(a[0:4])
	i := 4
	for ; i+4 <= len(a); i += 4 {
		v := archsimd.LoadInt64x4(a[i : i+4])
		acc = acc.IfElse(v.Greater(acc), v)
	}
	var lanes [4]int64
	acc.StoreArray(&lanes)
	for _, v := range lanes {
		if v < cur {
			cur = v
		}
	}
	for ; i < len(a); i++ {
		if a[i] < cur {
			cur = a[i]
		}
	}
	return cur
}
`,
	},
	"upper": {
		core: `// strtoupper PHP : ASCII octet à octet, jamais unicode.
func p2goToUpperScalar(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'a' && b[i] <= 'z' {
			b[i] -= 32
		}
	}
	return string(b)
}
`,
		off: `func p2goToUpper(s string) string { return p2goToUpperScalar(s) }
`,
		// F-p2go-simd-ascii-case : bande a..z détectée en comparaison SIGNÉE
		// (VPCMPGTB) — les octets UTF-8 ≥ 0x80 sont négatifs donc hors bande,
		// intacts ; bascule de casse par soustraction de 32 sous masque.
		on: `func p2goToUpper(s string) string {
	if !archsimd.X86.AVX2() || len(s) < 32 {
		return p2goToUpperScalar(s)
	}
	b := []byte(s)
	lo := archsimd.BroadcastInt8x32('a' - 1)
	hi := archsimd.BroadcastInt8x32('z' + 1)
	delta := archsimd.BroadcastUint8x32(32)
	i := 0
	// F-p2go-upper-native-php-gap : 2×32 octets par itération.
	for ; i+64 <= len(b); i += 64 {
		v0 := archsimd.LoadUint8x32(b[i : i+32])
		v1 := archsimd.LoadUint8x32(b[i+32 : i+64])
		s0 := v0.AsInt8x32()
		s1 := v1.AsInt8x32()
		m0 := s0.Greater(lo).And(hi.Greater(s0))
		m1 := s1.Greater(lo).And(hi.Greater(s1))
		v0.Sub(delta).IfElse(m0, v0).Store(b[i : i+32])
		v1.Sub(delta).IfElse(m1, v1).Store(b[i+32 : i+64])
	}
	for ; i+32 <= len(b); i += 32 {
		v := archsimd.LoadUint8x32(b[i : i+32])
		sv := v.AsInt8x32()
		mask := sv.Greater(lo).And(hi.Greater(sv))
		v.Sub(delta).IfElse(mask, v).Store(b[i : i+32])
	}
	for ; i < len(b); i++ {
		if b[i] >= 'a' && b[i] <= 'z' {
			b[i] -= 32
		}
	}
	return string(b)
}
`,
	},
	"lower": {
		core: `// strtolower PHP : ASCII octet à octet, jamais unicode.
func p2goToLowerScalar(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 32
		}
	}
	return string(b)
}
`,
		off: `func p2goToLower(s string) string { return p2goToLowerScalar(s) }
`,
		on: `func p2goToLower(s string) string {
	if !archsimd.X86.AVX2() || len(s) < 32 {
		return p2goToLowerScalar(s)
	}
	b := []byte(s)
	lo := archsimd.BroadcastInt8x32('A' - 1)
	hi := archsimd.BroadcastInt8x32('Z' + 1)
	delta := archsimd.BroadcastUint8x32(32)
	i := 0
	for ; i+64 <= len(b); i += 64 {
		v0 := archsimd.LoadUint8x32(b[i : i+32])
		v1 := archsimd.LoadUint8x32(b[i+32 : i+64])
		s0 := v0.AsInt8x32()
		s1 := v1.AsInt8x32()
		m0 := s0.Greater(lo).And(hi.Greater(s0))
		m1 := s1.Greater(lo).And(hi.Greater(s1))
		v0.Add(delta).IfElse(m0, v0).Store(b[i : i+32])
		v1.Add(delta).IfElse(m1, v1).Store(b[i+32 : i+64])
	}
	for ; i+32 <= len(b); i += 32 {
		v := archsimd.LoadUint8x32(b[i : i+32])
		sv := v.AsInt8x32()
		mask := sv.Greater(lo).And(hi.Greater(sv))
		v.Add(delta).IfElse(mask, v).Store(b[i : i+32])
	}
	for ; i < len(b); i++ {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 32
		}
	}
	return string(b)
}
`,
	},
}

const simdFileHeader = `// Code generated by p2go v0.4.0. DO NOT EDIT.
//go:build %s

package main
`
