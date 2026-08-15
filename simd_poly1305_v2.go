//go:build goexperiment.simd

package c2simd

import (
	"encoding/binary"
	"math/bits"
	"simd/archsimd"
	"unsafe"
)

const (
	p26     = uint64(1)<<26 - 1
	hibit26 = uint64(1) << 24
)

type Poly1305V2 struct {
	acc        [5][4]uint64
	r4u        [5][8]uint32
	s4u        [4][8]uint32
	r1         [5]uint32
	s1         [5]uint32
	rPow       [4][5]uint32
	pad0, pad1 uint64
	buf        [64]byte
	bufLen     int
}

func NewPoly1305V2(key []byte) Poly1305V2 {
	k0 := binary.LittleEndian.Uint64(key[0:8]) & 0x0ffffffc0fffffff
	k1 := binary.LittleEndian.Uint64(key[8:16]) & 0x0ffffffc0ffffffc

	r0 := uint32(k0 & 0x3ffffff)
	r1 := uint32((k0 >> 26) & 0x3ffffff)
	r2 := uint32(((k0 >> 52) | (k1 << 12)) & 0x3ffffff)
	r3 := uint32((k1 >> 14) & 0x3ffffff)
	r4 := uint32((k1 >> 40) & 0x3ffffff)

	var st Poly1305V2
	st.r1 = [5]uint32{r0, r1, r2, r3, r4}
	st.s1 = [5]uint32{r0, r1 * 5, r2 * 5, r3 * 5, r4 * 5}
	st.pad0 = binary.LittleEndian.Uint64(key[16:24])
	st.pad1 = binary.LittleEndian.Uint64(key[24:32])

	st.rPow[0] = st.r1
	st.rPow[1] = mul26u32(st.rPow[0], st.rPow[0])
	st.rPow[2] = mul26u32(st.rPow[1], st.rPow[0])
	st.rPow[3] = mul26u32(st.rPow[1], st.rPow[1])

	for i := 0; i < 5; i++ {
		x := st.rPow[3][i]
		st.r4u[i] = [8]uint32{x, 0, x, 0, x, 0, x, 0}
	}
	for i := 1; i <= 4; i++ {
		x := st.rPow[3][i] * 5
		st.s4u[i-1] = [8]uint32{x, 0, x, 0, x, 0, x, 0}
	}
	return st
}

func mul26u32(a, b [5]uint32) [5]uint32 {
	a0, a1, a2, a3, a4 := uint64(a[0]), uint64(a[1]), uint64(a[2]), uint64(a[3]), uint64(a[4])
	b0, b1, b2, b3, b4 := uint64(b[0]), uint64(b[1]), uint64(b[2]), uint64(b[3]), uint64(b[4])
	s1, s2, s3, s4 := b1*5, b2*5, b3*5, b4*5
	d0 := a0*b0 + a1*s4 + a2*s3 + a3*s2 + a4*s1
	d1 := a0*b1 + a1*b0 + a2*s4 + a3*s3 + a4*s2
	d2 := a0*b2 + a1*b1 + a2*b0 + a3*s4 + a4*s3
	d3 := a0*b3 + a1*b2 + a2*b1 + a3*b0 + a4*s4
	d4 := a0*b4 + a1*b3 + a2*b2 + a3*b1 + a4*b0
	h0, h1, h2, h3, h4 := carry26(d0, d1, d2, d3, d4)
	return [5]uint32{uint32(h0), uint32(h1), uint32(h2), uint32(h3), uint32(h4)}
}

func mul26uu(a0, a1, a2, a3, a4 uint64, r [5]uint32) (uint64, uint64, uint64, uint64, uint64) {
	b0, b1, b2, b3, b4 := uint64(r[0]), uint64(r[1]), uint64(r[2]), uint64(r[3]), uint64(r[4])
	s1, s2, s3, s4 := b1*5, b2*5, b3*5, b4*5
	return carry26(
		a0*b0+a1*s4+a2*s3+a3*s2+a4*s1,
		a0*b1+a1*b0+a2*s4+a3*s3+a4*s2,
		a0*b2+a1*b1+a2*b0+a3*s4+a4*s3,
		a0*b3+a1*b2+a2*b1+a3*b0+a4*s4,
		a0*b4+a1*b3+a2*b2+a3*b1+a4*b0,
	)
}

func carry26(d0, d1, d2, d3, d4 uint64) (h0, h1, h2, h3, h4 uint64) {
	c := d0 >> 26
	h0 = d0 & p26
	d1 += c
	c = d1 >> 26
	h1 = d1 & p26
	d2 += c
	c = d2 >> 26
	h2 = d2 & p26
	d3 += c
	c = d3 >> 26
	h3 = d3 & p26
	d4 += c
	c = d4 >> 26
	h4 = d4 & p26
	h0 += c * 5
	c = h0 >> 26
	h0 &= p26
	h1 += c
	return
}

var (
	mask26splat = [4]uint64{p26, p26, p26, p26}
	hibitSplat  = [4]uint64{hibit26, hibit26, hibit26, hibit26}
)

func (st *Poly1305V2) Update(msg []byte) {
	st.UpdateStream(msg)
}

func (st *Poly1305V2) processBlock64(blk []byte) {
	mask26 := archsimd.LoadUint64x4Array(&mask26splat)
	hibit := archsimd.LoadUint64x4Array(&hibitSplat)

	r4_0 := archsimd.LoadUint32x8Array(&st.r4u[0])
	r4_1 := archsimd.LoadUint32x8Array(&st.r4u[1])
	r4_2 := archsimd.LoadUint32x8Array(&st.r4u[2])
	r4_3 := archsimd.LoadUint32x8Array(&st.r4u[3])
	r4_4 := archsimd.LoadUint32x8Array(&st.r4u[4])

	s4_1 := archsimd.LoadUint32x8Array(&st.s4u[0])
	s4_2 := archsimd.LoadUint32x8Array(&st.s4u[1])
	s4_3 := archsimd.LoadUint32x8Array(&st.s4u[2])
	s4_4 := archsimd.LoadUint32x8Array(&st.s4u[3])

	h0 := archsimd.LoadUint64x4Array(&st.acc[0])
	h1 := archsimd.LoadUint64x4Array(&st.acc[1])
	h2 := archsimd.LoadUint64x4Array(&st.acc[2])
	h3 := archsimd.LoadUint64x4Array(&st.acc[3])
	h4 := archsimd.LoadUint64x4Array(&st.acc[4])

	raw := unsafe.Slice((*uint64)(unsafe.Pointer(&blk[0])), 8)
	v0 := archsimd.LoadUint64x4(raw[0:4])
	v1 := archsimd.LoadUint64x4(raw[4:8])
	lo0 := v0.GetLo()
	hi0 := v0.GetHi()
	lo1 := v1.GetLo()
	hi1 := v1.GetHi()
	var t0, t1 archsimd.Uint64x4
	t0 = t0.SetLo(lo0.InterleaveLo(hi0)).SetHi(lo1.InterleaveLo(hi1))
	t1 = t1.SetLo(lo0.InterleaveHi(hi0)).SetHi(lo1.InterleaveHi(hi1))

	m0 := t0.And(mask26)
	m1 := t0.ShiftAllRight(26).And(mask26)
	m2 := t0.ShiftAllRight(52).Or(t1.ShiftAllLeft(12)).And(mask26)
	m3 := t1.ShiftAllRight(14).And(mask26)
	m4 := t1.ShiftAllRight(40).And(mask26).Or(hibit)

	u0 := h0.ReshapeToUint32s()
	u1 := h1.ReshapeToUint32s()
	u2 := h2.ReshapeToUint32s()
	u3 := h3.ReshapeToUint32s()
	u4 := h4.ReshapeToUint32s()

	d0 := u0.MulWidenEven(r4_0).Add(u1.MulWidenEven(s4_4)).Add(u2.MulWidenEven(s4_3)).Add(u3.MulWidenEven(s4_2)).Add(u4.MulWidenEven(s4_1)).Add(m0)
	d1 := u0.MulWidenEven(r4_1).Add(u1.MulWidenEven(r4_0)).Add(u2.MulWidenEven(s4_4)).Add(u3.MulWidenEven(s4_3)).Add(u4.MulWidenEven(s4_2)).Add(m1)
	d2 := u0.MulWidenEven(r4_2).Add(u1.MulWidenEven(r4_1)).Add(u2.MulWidenEven(r4_0)).Add(u3.MulWidenEven(s4_4)).Add(u4.MulWidenEven(s4_3)).Add(m2)
	d3 := u0.MulWidenEven(r4_3).Add(u1.MulWidenEven(r4_2)).Add(u2.MulWidenEven(r4_1)).Add(u3.MulWidenEven(r4_0)).Add(u4.MulWidenEven(s4_4)).Add(m3)
	d4 := u0.MulWidenEven(r4_4).Add(u1.MulWidenEven(r4_3)).Add(u2.MulWidenEven(r4_2)).Add(u3.MulWidenEven(r4_1)).Add(u4.MulWidenEven(r4_0)).Add(m4)

	c0 := d0.ShiftAllRight(26)
	h0 = d0.And(mask26)
	d1 = d1.Add(c0)

	c1 := d1.ShiftAllRight(26)
	h1 = d1.And(mask26)
	d2 = d2.Add(c1)

	c2 := d2.ShiftAllRight(26)
	h2 = d2.And(mask26)
	d3 = d3.Add(c2)

	c3 := d3.ShiftAllRight(26)
	h3 = d3.And(mask26)
	d4 = d4.Add(c3)

	c4 := d4.ShiftAllRight(26)
	h4 = d4.And(mask26)

	c4_5 := c4.Add(c4.ShiftAllLeft(2))
	h0 = h0.Add(c4_5)
	c0 = h0.ShiftAllRight(26)
	h0 = h0.And(mask26)
	h1 = h1.Add(c0)

	h0.StoreArray(&st.acc[0])
	h1.StoreArray(&st.acc[1])
	h2.StoreArray(&st.acc[2])
	h3.StoreArray(&st.acc[3])
	h4.StoreArray(&st.acc[4])
}

// UpdateBlocks64 traite directement un slice de blocs de 64 octets dans une seule boucle
// sans appel de fonction intermédiaire, maintenant tout le live-set dans les registres YMM.
func (st *Poly1305V2) UpdateBlocks64(msg []byte) {
	if len(msg) < 64 {
		return
	}

	mask26 := archsimd.LoadUint64x4Array(&mask26splat)
	hibit := archsimd.LoadUint64x4Array(&hibitSplat)

	h0 := archsimd.LoadUint64x4Array(&st.acc[0])
	h1 := archsimd.LoadUint64x4Array(&st.acc[1])
	h2 := archsimd.LoadUint64x4Array(&st.acc[2])
	h3 := archsimd.LoadUint64x4Array(&st.acc[3])
	h4 := archsimd.LoadUint64x4Array(&st.acc[4])

	for len(msg) >= 64 {
		raw := unsafe.Slice((*uint64)(unsafe.Pointer(&msg[0])), 8)
		v0 := archsimd.LoadUint64x4(raw[0:4])
		v1 := archsimd.LoadUint64x4(raw[4:8])
		lo0 := v0.GetLo()
		hi0 := v0.GetHi()
		lo1 := v1.GetLo()
		hi1 := v1.GetHi()
		var t0, t1 archsimd.Uint64x4
		t0 = t0.SetLo(lo0.InterleaveLo(hi0)).SetHi(lo1.InterleaveLo(hi1))
		t1 = t1.SetLo(lo0.InterleaveHi(hi0)).SetHi(lo1.InterleaveHi(hi1))

		m0 := t0.And(mask26)
		m1 := t0.ShiftAllRight(26).And(mask26)
		m2 := t0.ShiftAllRight(52).Or(t1.ShiftAllLeft(12)).And(mask26)
		m3 := t1.ShiftAllRight(14).And(mask26)
		m4 := t1.ShiftAllRight(40).And(mask26).Or(hibit)

		u0 := h0.ReshapeToUint32s()
		u1 := h1.ReshapeToUint32s()
		u2 := h2.ReshapeToUint32s()
		u3 := h3.ReshapeToUint32s()
		u4 := h4.ReshapeToUint32s()

		d0 := u0.MulWidenEven(archsimd.LoadUint32x8Array(&st.r4u[0])).
			Add(u1.MulWidenEven(archsimd.LoadUint32x8Array(&st.s4u[3]))).
			Add(u2.MulWidenEven(archsimd.LoadUint32x8Array(&st.s4u[2]))).
			Add(u3.MulWidenEven(archsimd.LoadUint32x8Array(&st.s4u[1]))).
			Add(u4.MulWidenEven(archsimd.LoadUint32x8Array(&st.s4u[0]))).
			Add(m0)

		d1 := u0.MulWidenEven(archsimd.LoadUint32x8Array(&st.r4u[1])).
			Add(u1.MulWidenEven(archsimd.LoadUint32x8Array(&st.r4u[0]))).
			Add(u2.MulWidenEven(archsimd.LoadUint32x8Array(&st.s4u[3]))).
			Add(u3.MulWidenEven(archsimd.LoadUint32x8Array(&st.s4u[2]))).
			Add(u4.MulWidenEven(archsimd.LoadUint32x8Array(&st.s4u[1]))).
			Add(m1)

		d2 := u0.MulWidenEven(archsimd.LoadUint32x8Array(&st.r4u[2])).
			Add(u1.MulWidenEven(archsimd.LoadUint32x8Array(&st.r4u[1]))).
			Add(u2.MulWidenEven(archsimd.LoadUint32x8Array(&st.r4u[0]))).
			Add(u3.MulWidenEven(archsimd.LoadUint32x8Array(&st.s4u[3]))).
			Add(u4.MulWidenEven(archsimd.LoadUint32x8Array(&st.s4u[2]))).
			Add(m2)

		d3 := u0.MulWidenEven(archsimd.LoadUint32x8Array(&st.r4u[3])).
			Add(u1.MulWidenEven(archsimd.LoadUint32x8Array(&st.r4u[2]))).
			Add(u2.MulWidenEven(archsimd.LoadUint32x8Array(&st.r4u[1]))).
			Add(u3.MulWidenEven(archsimd.LoadUint32x8Array(&st.r4u[0]))).
			Add(u4.MulWidenEven(archsimd.LoadUint32x8Array(&st.s4u[3]))).
			Add(m3)

		d4 := u0.MulWidenEven(archsimd.LoadUint32x8Array(&st.r4u[4])).
			Add(u1.MulWidenEven(archsimd.LoadUint32x8Array(&st.r4u[3]))).
			Add(u2.MulWidenEven(archsimd.LoadUint32x8Array(&st.r4u[2]))).
			Add(u3.MulWidenEven(archsimd.LoadUint32x8Array(&st.r4u[1]))).
			Add(u4.MulWidenEven(archsimd.LoadUint32x8Array(&st.r4u[0]))).
			Add(m4)

		c0 := d0.ShiftAllRight(26)
		h0 = d0.And(mask26)
		d1 = d1.Add(c0)

		c1 := d1.ShiftAllRight(26)
		h1 = d1.And(mask26)
		d2 = d2.Add(c1)

		c2 := d2.ShiftAllRight(26)
		h2 = d2.And(mask26)
		d3 = d3.Add(c2)

		c3 := d3.ShiftAllRight(26)
		h3 = d3.And(mask26)
		d4 = d4.Add(c3)

		c4 := d4.ShiftAllRight(26)
		h4 = d4.And(mask26)

		c4_5 := c4.Add(c4.ShiftAllLeft(2))
		h0 = h0.Add(c4_5)
		c0 = h0.ShiftAllRight(26)
		h0 = h0.And(mask26)
		h1 = h1.Add(c0)

		msg = msg[64:]
	}

	h0.StoreArray(&st.acc[0])
	h1.StoreArray(&st.acc[1])
	h2.StoreArray(&st.acc[2])
	h3.StoreArray(&st.acc[3])
	h4.StoreArray(&st.acc[4])
}

func (st *Poly1305V2) UpdateStream(msg []byte) {
	if st.bufLen > 0 {
		n := copy(st.buf[st.bufLen:], msg)
		st.bufLen += n
		msg = msg[n:]
		if st.bufLen < 64 {
			return
		}
		st.processBlock64(st.buf[:])
		st.bufLen = 0
	}

	if len(msg) >= 64 {
		bulkLen := len(msg) &^ 63
		st.UpdateBlocks64(msg[:bulkLen])
		msg = msg[bulkLen:]
	}

	if len(msg) > 0 {
		st.bufLen = copy(st.buf[:], msg)
	}
}

func (st *Poly1305V2) Finish(out *[16]byte) {
	var a0, a1, a2, a3, a4 [4]uint64
	a0, a1, a2, a3, a4 = st.acc[0], st.acc[1], st.acc[2], st.acc[3], st.acc[4]

	// Repli horizontal des 4 voies vectorielles avec pondération r^3, r^2, r, 1
	p0_0, p0_1, p0_2, p0_3, p0_4 := mul26uu(a0[0], a1[0], a2[0], a3[0], a4[0], st.rPow[3])
	p1_0, p1_1, p1_2, p1_3, p1_4 := mul26uu(a0[1], a1[1], a2[1], a3[1], a4[1], st.rPow[2])
	p2_0, p2_1, p2_2, p2_3, p2_4 := mul26uu(a0[2], a1[2], a2[2], a3[2], a4[2], st.rPow[1])
	p3_0, p3_1, p3_2, p3_3, p3_4 := mul26uu(a0[3], a1[3], a2[3], a3[3], a4[3], st.rPow[0])

	h0, h1, h2, h3, h4 := carry26(
		p0_0+p1_0+p2_0+p3_0,
		p0_1+p1_1+p2_1+p3_1,
		p0_2+p1_2+p2_2+p3_2,
		p0_3+p1_3+p2_3+p3_3,
		p0_4+p1_4+p2_4+p3_4,
	)

	rem := st.buf[:st.bufLen]
	for len(rem) >= 16 {
		t0 := binary.LittleEndian.Uint64(rem[0:8])
		t1 := binary.LittleEndian.Uint64(rem[8:16])
		h0 += t0 & 0x3ffffff
		h1 += (t0 >> 26) & 0x3ffffff
		h2 += ((t0 >> 52) | (t1 << 12)) & 0x3ffffff
		h3 += (t1 >> 14) & 0x3ffffff
		h4 += ((t1 >> 40) & 0x3ffffff) | hibit26
		h0, h1, h2, h3, h4 = mul26uu(h0, h1, h2, h3, h4, st.r1)
		rem = rem[16:]
	}
	if len(rem) > 0 {
		var tmp [16]byte
		copy(tmp[:], rem)
		tmp[len(rem)] = 1
		t0 := binary.LittleEndian.Uint64(tmp[0:8])
		t1 := binary.LittleEndian.Uint64(tmp[8:16])
		h0 += t0 & 0x3ffffff
		h1 += (t0 >> 26) & 0x3ffffff
		h2 += ((t0 >> 52) | (t1 << 12)) & 0x3ffffff
		h3 += (t1 >> 14) & 0x3ffffff
		h4 += (t1 >> 40) & 0x3ffffff
		h0, h1, h2, h3, h4 = mul26uu(h0, h1, h2, h3, h4, st.r1)
	}
	st.bufLen = 0

	h0, h1, h2, h3, h4 = carry26(h0, h1, h2, h3, h4)

	g0 := h0 + 5
	c := g0 >> 26
	g0 &= p26
	g1 := h1 + c
	c = g1 >> 26
	g1 &= p26
	g2 := h2 + c
	c = g2 >> 26
	g2 &= p26
	g3 := h3 + c
	c = g3 >> 26
	g3 &= p26
	g4 := h4 + c
	mask := uint64(0) - (g4 >> 26)

	h0 = (h0 &^ mask) | (g0 & mask)
	h1 = (h1 &^ mask) | (g1 & mask)
	h2 = (h2 &^ mask) | (g2 & mask)
	h3 = (h3 &^ mask) | (g3 & mask)
	h4 = (h4 &^ mask) | ((g4 & p26) & mask)

	f0 := h0 | (h1 << 26) | ((h2 & 0xfff) << 52)
	f1 := (h2 >> 12) | (h3 << 14) | (h4 << 40)
	f0, carry := bits.Add64(f0, st.pad0, 0)
	f1, _ = bits.Add64(f1, st.pad1, carry)
	binary.LittleEndian.PutUint64(out[0:8], f0)
	binary.LittleEndian.PutUint64(out[8:16], f1)
}

// polyMAC est l'interface commune des deux moteurs Poly1305 du chemin AEAD
// fusionné (phases séquentielles par tranches de 4 Ko — pas d'entrelacement,
// donc pas de conflit de registres avec ChaCha).
type polyMAC interface {
	Update(msg []byte)
	Finish(out *[16]byte)
}

// newPolyMAC choisit le moteur par la taille dominante à authentifier :
// la voie vectorielle v3 gagne dès ~1 Ko (mesuré 2026-08-15 : 3076 vs
// 2254 MB/s au croisement), le scalaire QuadChain reste devant en dessous
// (coût fixe des puissances r⁴ et de la réduction horizontale finale).
const polyV2Threshold = 1024

func newPolyMAC(key []byte, total int) polyMAC {
	if total >= polyV2Threshold {
		st := NewPoly1305V2(key)
		return &st
	}
	st := NewPoly1305QuadChain(key)
	return &st
}

func Poly1305SumV2(out *[16]byte, msg, key []byte) {
	st := NewPoly1305V2(key)
	st.UpdateStream(msg)
	st.Finish(out)
}
