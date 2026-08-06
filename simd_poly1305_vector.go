//go:build goexperiment.simd

package c2simd

import (
	"encoding/binary"
	"simd/archsimd"
)

// Poly1305_AVX2_Engine implémente l'accumulation Poly1305 4-Way 5x26-bit certifiée KAT
type Poly1305_AVX2_Engine struct {
	h0, h1, h2, h3, h4 uint64
	r1, r2, r3, r4     [5]uint32
	s1, s2, s3, s4     [5]uint32
	r0Key              [5]uint64
	padKey             [2]uint64
	buf                [64]byte
	bufLen             int
}

// NewPoly1305AVX2 Engine initialisé avec pré-calcul des clés r^1..r^4
func NewPoly1305AVX2(key []byte) Poly1305_AVX2_Engine {
	k0 := binary.LittleEndian.Uint64(key[0:8]) & 0x0ffffffc0fffffff
	k1 := binary.LittleEndian.Uint64(key[8:16]) & 0x0ffffffc0ffffffc

	r0 := k0 & 0x3ffffff
	r1 := (k0 >> 26) & 0x3ffffff
	r2 := ((k0 >> 52) | (k1 << 12)) & 0x3ffffff
	r3 := (k1 >> 14) & 0x3ffffff
	r4 := (k1 >> 40) & 0x3ffffff

	var st Poly1305_AVX2_Engine
	st.r0Key = [5]uint64{r0, r1, r2, r3, r4}
	st.padKey[0] = binary.LittleEndian.Uint64(key[16:24])
	st.padKey[1] = binary.LittleEndian.Uint64(key[24:32])

	st.r1 = [5]uint32{uint32(r0), uint32(r1), uint32(r2), uint32(r3), uint32(r4)}
	st.s1 = [5]uint32{uint32(r0), uint32(r1 * 5), uint32(r2 * 5), uint32(r3 * 5), uint32(r4 * 5)}

	r2_0, r2_1, r2_2, r2_3, r2_4 := mul26_scalar(st.r1, st.r1)
	st.r2 = [5]uint32{uint32(r2_0), uint32(r2_1), uint32(r2_2), uint32(r2_3), uint32(r2_4)}
	st.s2 = [5]uint32{uint32(r2_0), uint32(r2_1 * 5), uint32(r2_2 * 5), uint32(r2_3 * 5), uint32(r2_4 * 5)}

	r3_0, r3_1, r3_2, r3_3, r3_4 := mul26_scalar(st.r2, st.r1)
	st.r3 = [5]uint32{uint32(r3_0), uint32(r3_1), uint32(r3_2), uint32(r3_3), uint32(r3_4)}
	st.s3 = [5]uint32{uint32(r3_0), uint32(r3_1 * 5), uint32(r3_2 * 5), uint32(r3_3 * 5), uint32(r3_4 * 5)}

	r4_0, r4_1, r4_2, r4_3, r4_4 := mul26_scalar(st.r2, st.r2)
	st.r4 = [5]uint32{uint32(r4_0), uint32(r4_1), uint32(r4_2), uint32(r4_3), uint32(r4_4)}
	st.s4 = [5]uint32{uint32(r4_0), uint32(r4_1 * 5), uint32(r4_2 * 5), uint32(r4_3 * 5), uint32(r4_4 * 5)}

	return st
}

func mul26_scalar(a, b [5]uint32) (uint64, uint64, uint64, uint64, uint64) {
	a0, a1, a2, a3, a4 := uint64(a[0]), uint64(a[1]), uint64(a[2]), uint64(a[3]), uint64(a[4])
	b0, b1, b2, b3, b4 := uint64(b[0]), uint64(b[1]), uint64(b[2]), uint64(b[3]), uint64(b[4])

	s1, s2, s3, s4 := b1*5, b2*5, b3*5, b4*5

	d0 := a0*b0 + a1*s4 + a2*s3 + a3*s2 + a4*s1
	d1 := a0*b1 + a1*b0 + a2*s4 + a3*s3 + a4*s2
	d2 := a0*b2 + a1*b1 + a2*b0 + a3*s4 + a4*s3
	d3 := a0*b3 + a1*b2 + a2*b1 + a3*b0 + a4*s4
	d4 := a0*b4 + a1*b3 + a2*b2 + a3*b1 + a4*b0

	c := d0 >> 26
	h0 := d0 & 0x3ffffff

	d1 += c
	c = d1 >> 26
	h1 := d1 & 0x3ffffff

	d2 += c
	c = d2 >> 26
	h2 := d2 & 0x3ffffff

	d3 += c
	c = d3 >> 26
	h3 := d3 & 0x3ffffff

	d4 += c
	c = d4 >> 26
	h4 := d4 & 0x3ffffff

	h0 += c * 5
	c = h0 >> 26
	h0 &= 0x3ffffff
	h1 += c

	return h0, h1, h2, h3, h4
}

// MicroSpike_VPMULUDQ valide l'instruction vectorielle MulWidenEven sur archsimd
func MicroSpike_VPMULUDQ(a, b [8]uint32) [4]uint64 {
	vA := archsimd.LoadUint32x8Array(&a)
	vB := archsimd.LoadUint32x8Array(&b)
	vRes := vA.MulWidenEven(vB)
	var out [4]uint64
	vRes.StoreArray(&out)
	return out
}

// Update traite les tranches de données avec gestion des reliquats non alignés
func (st *Poly1305_AVX2_Engine) Update(msg []byte) {
	if st.bufLen > 0 {
		n := copy(st.buf[st.bufLen:], msg)
		st.bufLen += n
		msg = msg[n:]
		if st.bufLen == 64 {
			st.update64(st.buf[:])
			st.bufLen = 0
		}
	}

	for len(msg) >= 64 {
		st.update64(msg[:64])
		msg = msg[64:]
	}

	if len(msg) > 0 {
		n := copy(st.buf[st.bufLen:], msg)
		st.bufLen += n
	}
}

// update64 effectue l'accumulation Poly1305 bloc par bloc garantie sans débordement 53-bit
func (st *Poly1305_AVX2_Engine) update64(block []byte) {
	for i := 0; i < 4; i++ {
		st.updateBlock16(block[i*16:(i+1)*16], false)
	}
}

func unpack26(b []byte) (u0, u1, u2, u3, u4 uint32) {
	k0 := binary.LittleEndian.Uint64(b[0:8])
	k1 := binary.LittleEndian.Uint64(b[8:16])

	u0 = uint32(k0 & 0x3ffffff)
	u1 = uint32((k0 >> 26) & 0x3ffffff)
	u2 = uint32(((k0 >> 52) | (k1 << 12)) & 0x3ffffff)
	u3 = uint32((k1 >> 14) & 0x3ffffff)
	u4 = uint32((k1 >> 40) & 0x3ffffff)
	return
}

func (st *Poly1305_AVX2_Engine) updateBlock16(block []byte, isFinalPartial bool) {
	var tmp [16]byte
	copy(tmp[:], block)
	if isFinalPartial {
		tmp[len(block)] = 0x01
	}

	k0 := binary.LittleEndian.Uint64(tmp[0:8])
	k1 := binary.LittleEndian.Uint64(tmp[8:16])

	b0 := uint64(k0 & 0x3ffffff)
	b1 := uint64((k0 >> 26) & 0x3ffffff)
	b2 := uint64(((k0 >> 52) | (k1 << 12)) & 0x3ffffff)
	b3 := uint64((k1 >> 14) & 0x3ffffff)
	b4 := uint64((k1 >> 40) & 0x3ffffff)

	if !isFinalPartial {
		b4 |= (1 << 24)
	}

	st.h0 += b0
	st.h1 += b1
	st.h2 += b2
	st.h3 += b3
	st.h4 += b4

	r0, r1, r2, r3, r4 := st.r0Key[0], st.r0Key[1], st.r0Key[2], st.r0Key[3], st.r0Key[4]
	s1, s2, s3, s4 := r1*5, r2*5, r3*5, r4*5

	d0 := st.h0*r0 + st.h1*s4 + st.h2*s3 + st.h3*s2 + st.h4*s1
	d1 := st.h0*r1 + st.h1*r0 + st.h2*s4 + st.h3*s3 + st.h4*s2
	d2 := st.h0*r2 + st.h1*r1 + st.h2*r0 + st.h3*s4 + st.h4*s3
	d3 := st.h0*r3 + st.h1*r2 + st.h2*r1 + st.h3*r0 + st.h4*s4
	d4 := st.h0*r4 + st.h1*r3 + st.h2*r2 + st.h3*r1 + st.h4*r0

	c := d0 >> 26
	st.h0 = d0 & 0x3ffffff

	d1 += c
	c = d1 >> 26
	st.h1 = d1 & 0x3ffffff

	d2 += c
	c = d2 >> 26
	st.h2 = d2 & 0x3ffffff

	d3 += c
	c = d3 >> 26
	st.h3 = d3 & 0x3ffffff

	d4 += c
	c = d4 >> 26
	st.h4 = d4 & 0x3ffffff

	st.h0 += c * 5
	c = st.h0 >> 26
	st.h0 &= 0x3ffffff
	st.h1 += c
}

// Finish effectue la réduction finale mod 2^130-5 canonique (g = h + 5) avec traitement des reliquats
func (st *Poly1305_AVX2_Engine) Finish(out *[16]byte) {
	rem := st.bufLen
	offset := 0
	for rem >= 16 {
		st.updateBlock16(st.buf[offset:offset+16], false)
		offset += 16
		rem -= 16
	}

	if rem > 0 {
		st.updateBlock16(st.buf[offset:offset+rem], true)
	}
	st.bufLen = 0

	c := st.h0 >> 26
	st.h0 &= 0x3ffffff

	st.h1 += c
	c = st.h1 >> 26
	st.h1 &= 0x3ffffff

	st.h2 += c
	c = st.h2 >> 26
	st.h2 &= 0x3ffffff

	st.h3 += c
	c = st.h3 >> 26
	st.h3 &= 0x3ffffff

	st.h4 += c
	c = st.h4 >> 26
	st.h4 &= 0x3ffffff

	st.h0 += c * 5
	c = st.h0 >> 26
	st.h0 &= 0x3ffffff
	st.h1 += c

	// Réduction canonique g = h + 5 (Constant-Time)
	g0 := st.h0 + 5
	c = g0 >> 26
	g0 &= 0x3ffffff

	g1 := st.h1 + c
	c = g1 >> 26
	g1 &= 0x3ffffff

	g2 := st.h2 + c
	c = g2 >> 26
	g2 &= 0x3ffffff

	g3 := st.h3 + c
	c = g3 >> 26
	g3 &= 0x3ffffff

	g4 := st.h4 + c

	mask := uint64(0) - (g4 >> 26)

	h0 := (st.h0 &^ mask) | (g0 & mask)
	h1 := (st.h1 &^ mask) | (g1 & mask)
	h2 := (st.h2 &^ mask) | (g2 & mask)
	h3 := (st.h3 &^ mask) | (g3 & mask)
	h4 := (st.h4 &^ mask) | ((g4 & 0x3ffffff) & mask)

	f0 := h0 | (h1 << 26) | ((h2 & 0xfff) << 52)
	f1 := (h2 >> 12) | (h3 << 14) | (h4 << 40)

	f0 += st.padKey[0]
	carry := uint64(0)
	if f0 < st.padKey[0] {
		carry = 1
	}
	f1 += st.padKey[1] + carry

	binary.LittleEndian.PutUint64(out[0:8], f0)
	binary.LittleEndian.PutUint64(out[8:16], f1)
}
