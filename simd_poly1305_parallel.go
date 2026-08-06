//go:build goexperiment.simd

package c2simd

import (
	"encoding/binary"
	"math/bits"
)

type poly44Key struct {
	r0, r1, r2 uint64
	s1, s2     uint64
}

func make44Key(r0, r1, r2 uint64) poly44Key {
	return poly44Key{
		r0: r0,
		r1: r1,
		r2: r2,
		s1: r1 * 20,
		s2: r2 * 20,
	}
}

func mul44(h0, h1, h2 uint64, k poly44Key) (u0, u1, u2 uint64) {
	// 9 multiplications 64x64 -> 128 bit (1 cycle matériel via MULQ)
	hi0_0, lo0_0 := bits.Mul64(h0, k.r0)
	hi0_1, lo0_1 := bits.Mul64(h1, k.s2)
	hi0_2, lo0_2 := bits.Mul64(h2, k.s1)

	hi1_0, lo1_0 := bits.Mul64(h0, k.r1)
	hi1_1, lo1_1 := bits.Mul64(h1, k.r0)
	hi1_2, lo1_2 := bits.Mul64(h2, k.s2)

	hi2_0, lo2_0 := bits.Mul64(h0, k.r2)
	hi2_1, lo2_1 := bits.Mul64(h1, k.r1)
	hi2_2, lo2_2 := bits.Mul64(h2, k.r0)

	// Accumulation 128 bits
	d0_lo, c0 := bits.Add64(lo0_0, lo0_1, 0)
	d0_hi, _ := bits.Add64(hi0_0, hi0_1, c0)
	d0_lo, c0 = bits.Add64(d0_lo, lo0_2, 0)
	d0_hi, _ = bits.Add64(d0_hi, hi0_2, c0)

	d1_lo, c1 := bits.Add64(lo1_0, lo1_1, 0)
	d1_hi, _ := bits.Add64(hi1_0, hi1_1, c1)
	d1_lo, c1 = bits.Add64(d1_lo, lo1_2, 0)
	d1_hi, _ = bits.Add64(d1_hi, hi1_2, c1)

	d2_lo, c2 := bits.Add64(lo2_0, lo2_1, 0)
	d2_hi, _ := bits.Add64(hi2_0, hi2_1, c2)
	d2_lo, c2 = bits.Add64(d2_lo, lo2_2, 0)
	d2_hi, _ = bits.Add64(d2_hi, hi2_2, c2)

	// Propagations de retenues sur 44 bits (d0 -> u0, d1 -> u1)
	c := (d0_lo >> 44) | (d0_hi << 20)
	u0 = d0_lo & 0xfffffffffff

	d1_lo, c1 = bits.Add64(d1_lo, c, 0)
	d1_hi, _ = bits.Add64(d1_hi, 0, c1)

	c = (d1_lo >> 44) | (d1_hi << 20)
	u1 = d1_lo & 0xfffffffffff

	d2_lo, c2 = bits.Add64(d2_lo, c, 0)
	d2_hi, _ = bits.Add64(d2_hi, 0, c2)

	// Coupure à 42 bits pour d2 (poids 2^130)
	c = (d2_lo >> 42) | (d2_hi << 22)
	u2 = d2_lo & 0x3ffffffffff

	// Réduction modulo 2^130-5 (c * 5 car c est à 2^130)
	c5 := c * 5
	u0 += c5
	c = u0 >> 44
	u0 &= 0xfffffffffff
	u1 += c

	return u0, u1, u2
}

// Poly1305QuadChain (Moteur Poly1305-Donna64 3x44-bit Limbs ILP HexadecaChain 16-Way)
type Poly1305QuadChain struct {
	h0, h1, h2                                 uint64
	r1Key, r2Key, r3Key, r4Key                 poly44Key
	r5Key, r6Key, r7Key, r8Key                 poly44Key
	r9Key, r10Key, r11Key, r12Key              poly44Key
	r13Key, r14Key, r15Key, r16Key             poly44Key
	pad0, pad1, pad2, pad3                     uint32
	buf                                        [16]byte
	bufLen                                     int
}

func NewPoly1305QuadChain(key []byte) Poly1305QuadChain {
	k0 := binary.LittleEndian.Uint64(key[0:8]) & 0x0ffffffc0fffffff
	k1 := binary.LittleEndian.Uint64(key[8:16]) & 0x0ffffffc0ffffffc

	r0 := k0 & 0xfffffffffff
	r1 := ((k0 >> 44) | (k1 << 20)) & 0xfffffffffff
	r2 := (k1 >> 24) & 0x3ffffffffff

	r1Key := make44Key(r0, r1, r2)

	r2_0, r2_1, r2_2 := mul44(r0, r1, r2, r1Key)
	r2Key := make44Key(r2_0, r2_1, r2_2)

	r3_0, r3_1, r3_2 := mul44(r2_0, r2_1, r2_2, r1Key)
	r3Key := make44Key(r3_0, r3_1, r3_2)

	r4_0, r4_1, r4_2 := mul44(r2_0, r2_1, r2_2, r2Key)
	r4Key := make44Key(r4_0, r4_1, r4_2)

	r5_0, r5_1, r5_2 := mul44(r4_0, r4_1, r4_2, r1Key)
	r5Key := make44Key(r5_0, r5_1, r5_2)

	r6_0, r6_1, r6_2 := mul44(r4_0, r4_1, r4_2, r2Key)
	r6Key := make44Key(r6_0, r6_1, r6_2)

	r7_0, r7_1, r7_2 := mul44(r4_0, r4_1, r4_2, r3Key)
	r7Key := make44Key(r7_0, r7_1, r7_2)

	r8_0, r8_1, r8_2 := mul44(r4_0, r4_1, r4_2, r4Key)
	r8Key := make44Key(r8_0, r8_1, r8_2)

	r9_0, r9_1, r9_2 := mul44(r8_0, r8_1, r8_2, r1Key)
	r9Key := make44Key(r9_0, r9_1, r9_2)

	r10_0, r10_1, r10_2 := mul44(r8_0, r8_1, r8_2, r2Key)
	r10Key := make44Key(r10_0, r10_1, r10_2)

	r11_0, r11_1, r11_2 := mul44(r8_0, r8_1, r8_2, r3Key)
	r11Key := make44Key(r11_0, r11_1, r11_2)

	r12_0, r12_1, r12_2 := mul44(r8_0, r8_1, r8_2, r4Key)
	r12Key := make44Key(r12_0, r12_1, r12_2)

	r13_0, r13_1, r13_2 := mul44(r12_0, r12_1, r12_2, r1Key)
	r13Key := make44Key(r13_0, r13_1, r13_2)

	r14_0, r14_1, r14_2 := mul44(r12_0, r12_1, r12_2, r2Key)
	r14Key := make44Key(r14_0, r14_1, r14_2)

	r15_0, r15_1, r15_2 := mul44(r12_0, r12_1, r12_2, r3Key)
	r15Key := make44Key(r15_0, r15_1, r15_2)

	r16_0, r16_1, r16_2 := mul44(r12_0, r12_1, r12_2, r4Key)
	r16Key := make44Key(r16_0, r16_1, r16_2)

	return Poly1305QuadChain{
		r1Key:  r1Key,
		r2Key:  r2Key,
		r3Key:  r3Key,
		r4Key:  r4Key,
		r5Key:  r5Key,
		r6Key:  r6Key,
		r7Key:  r7Key,
		r8Key:  r8Key,
		r9Key:  r9Key,
		r10Key: r10Key,
		r11Key: r11Key,
		r12Key: r12Key,
		r13Key: r13Key,
		r14Key: r14Key,
		r15Key: r15Key,
		r16Key: r16Key,
		pad0:   binary.LittleEndian.Uint32(key[16:20]),
		pad1:   binary.LittleEndian.Uint32(key[20:24]),
		pad2:   binary.LittleEndian.Uint32(key[24:28]),
		pad3:   binary.LittleEndian.Uint32(key[28:32]),
	}
}

func (st *Poly1305QuadChain) Update(msg []byte) {
	if st.bufLen > 0 {
		n := copy(st.buf[st.bufLen:], msg)
		st.bufLen += n
		msg = msg[n:]
		if st.bufLen == 16 {
			st.updateBlock(st.buf[:], 1)
			st.bufLen = 0
		}
	}

	// Traitement 16 blocs par 16 blocs (256 octets) en parallèle HexadecaChain ILP
	for len(msg) >= 256 {
		m0_lo := binary.LittleEndian.Uint64(msg[0:8])
		m0_hi := binary.LittleEndian.Uint64(msg[8:16])
		m1_lo := binary.LittleEndian.Uint64(msg[16:24])
		m1_hi := binary.LittleEndian.Uint64(msg[24:32])
		m2_lo := binary.LittleEndian.Uint64(msg[32:40])
		m2_hi := binary.LittleEndian.Uint64(msg[40:48])
		m3_lo := binary.LittleEndian.Uint64(msg[48:56])
		m3_hi := binary.LittleEndian.Uint64(msg[56:64])

		m4_lo := binary.LittleEndian.Uint64(msg[64:72])
		m4_hi := binary.LittleEndian.Uint64(msg[72:80])
		m5_lo := binary.LittleEndian.Uint64(msg[80:88])
		m5_hi := binary.LittleEndian.Uint64(msg[88:96])
		m6_lo := binary.LittleEndian.Uint64(msg[96:104])
		m6_hi := binary.LittleEndian.Uint64(msg[104:112])
		m7_lo := binary.LittleEndian.Uint64(msg[112:120])
		m7_hi := binary.LittleEndian.Uint64(msg[120:128])

		m8_lo := binary.LittleEndian.Uint64(msg[128:136])
		m8_hi := binary.LittleEndian.Uint64(msg[136:144])
		m9_lo := binary.LittleEndian.Uint64(msg[144:152])
		m9_hi := binary.LittleEndian.Uint64(msg[152:160])
		m10_lo := binary.LittleEndian.Uint64(msg[160:168])
		m10_hi := binary.LittleEndian.Uint64(msg[168:176])
		m11_lo := binary.LittleEndian.Uint64(msg[176:184])
		m11_hi := binary.LittleEndian.Uint64(msg[184:192])

		m12_lo := binary.LittleEndian.Uint64(msg[192:200])
		m12_hi := binary.LittleEndian.Uint64(msg[200:208])
		m13_lo := binary.LittleEndian.Uint64(msg[208:216])
		m13_hi := binary.LittleEndian.Uint64(msg[216:224])
		m14_lo := binary.LittleEndian.Uint64(msg[224:232])
		m14_hi := binary.LittleEndian.Uint64(msg[232:240])
		m15_lo := binary.LittleEndian.Uint64(msg[240:248])
		m15_hi := binary.LittleEndian.Uint64(msg[248:256])

		b0_0 := m0_lo & 0xfffffffffff
		b0_1 := ((m0_lo >> 44) | (m0_hi << 20)) & 0xfffffffffff
		b0_2 := (m0_hi >> 24) | (1 << 40)

		b1_0 := m1_lo & 0xfffffffffff
		b1_1 := ((m1_lo >> 44) | (m1_hi << 20)) & 0xfffffffffff
		b1_2 := (m1_hi >> 24) | (1 << 40)

		b2_0 := m2_lo & 0xfffffffffff
		b2_1 := ((m2_lo >> 44) | (m2_hi << 20)) & 0xfffffffffff
		b2_2 := (m2_hi >> 24) | (1 << 40)

		b3_0 := m3_lo & 0xfffffffffff
		b3_1 := ((m3_lo >> 44) | (m3_hi << 20)) & 0xfffffffffff
		b3_2 := (m3_hi >> 24) | (1 << 40)

		b4_0 := m4_lo & 0xfffffffffff
		b4_1 := ((m4_lo >> 44) | (m4_hi << 20)) & 0xfffffffffff
		b4_2 := (m4_hi >> 24) | (1 << 40)

		b5_0 := m5_lo & 0xfffffffffff
		b5_1 := ((m5_lo >> 44) | (m5_hi << 20)) & 0xfffffffffff
		b5_2 := (m5_hi >> 24) | (1 << 40)

		b6_0 := m6_lo & 0xfffffffffff
		b6_1 := ((m6_lo >> 44) | (m6_hi << 20)) & 0xfffffffffff
		b6_2 := (m6_hi >> 24) | (1 << 40)

		b7_0 := m7_lo & 0xfffffffffff
		b7_1 := ((m7_lo >> 44) | (m7_hi << 20)) & 0xfffffffffff
		b7_2 := (m7_hi >> 24) | (1 << 40)

		b8_0 := m8_lo & 0xfffffffffff
		b8_1 := ((m8_lo >> 44) | (m8_hi << 20)) & 0xfffffffffff
		b8_2 := (m8_hi >> 24) | (1 << 40)

		b9_0 := m9_lo & 0xfffffffffff
		b9_1 := ((m9_lo >> 44) | (m9_hi << 20)) & 0xfffffffffff
		b9_2 := (m9_hi >> 24) | (1 << 40)

		b10_0 := m10_lo & 0xfffffffffff
		b10_1 := ((m10_lo >> 44) | (m10_hi << 20)) & 0xfffffffffff
		b10_2 := (m10_hi >> 24) | (1 << 40)

		b11_0 := m11_lo & 0xfffffffffff
		b11_1 := ((m11_lo >> 44) | (m11_hi << 20)) & 0xfffffffffff
		b11_2 := (m11_hi >> 24) | (1 << 40)

		b12_0 := m12_lo & 0xfffffffffff
		b12_1 := ((m12_lo >> 44) | (m12_hi << 20)) & 0xfffffffffff
		b12_2 := (m12_hi >> 24) | (1 << 40)

		b13_0 := m13_lo & 0xfffffffffff
		b13_1 := ((m13_lo >> 44) | (m13_hi << 20)) & 0xfffffffffff
		b13_2 := (m13_hi >> 24) | (1 << 40)

		b14_0 := m14_lo & 0xfffffffffff
		b14_1 := ((m14_lo >> 44) | (m14_hi << 20)) & 0xfffffffffff
		b14_2 := (m14_hi >> 24) | (1 << 40)

		b15_0 := m15_lo & 0xfffffffffff
		b15_1 := ((m15_lo >> 44) | (m15_hi << 20)) & 0xfffffffffff
		b15_2 := (m15_hi >> 24) | (1 << 40)

		pOld0, pOld1, pOld2 := mul44(st.h0, st.h1, st.h2, st.r16Key)
		p0_0, p0_1, p0_2 := mul44(b0_0, b0_1, b0_2, st.r16Key)
		p1_0, p1_1, p1_2 := mul44(b1_0, b1_1, b1_2, st.r15Key)
		p2_0, p2_1, p2_2 := mul44(b2_0, b2_1, b2_2, st.r14Key)
		p3_0, p3_1, p3_2 := mul44(b3_0, b3_1, b3_2, st.r13Key)
		p4_0, p4_1, p4_2 := mul44(b4_0, b4_1, b4_2, st.r12Key)
		p5_0, p5_1, p5_2 := mul44(b5_0, b5_1, b5_2, st.r11Key)
		p6_0, p6_1, p6_2 := mul44(b6_0, b6_1, b6_2, st.r10Key)
		p7_0, p7_1, p7_2 := mul44(b7_0, b7_1, b7_2, st.r9Key)
		p8_0, p8_1, p8_2 := mul44(b8_0, b8_1, b8_2, st.r8Key)
		p9_0, p9_1, p9_2 := mul44(b9_0, b9_1, b9_2, st.r7Key)
		p10_0, p10_1, p10_2 := mul44(b10_0, b10_1, b10_2, st.r6Key)
		p11_0, p11_1, p11_2 := mul44(b11_0, b11_1, b11_2, st.r5Key)
		p12_0, p12_1, p12_2 := mul44(b12_0, b12_1, b12_2, st.r4Key)
		p13_0, p13_1, p13_2 := mul44(b13_0, b13_1, b13_2, st.r3Key)
		p14_0, p14_1, p14_2 := mul44(b14_0, b14_1, b14_2, st.r2Key)
		p15_0, p15_1, p15_2 := mul44(b15_0, b15_1, b15_2, st.r1Key)

		h0 := pOld0 + p0_0 + p1_0 + p2_0 + p3_0 + p4_0 + p5_0 + p6_0 + p7_0 + p8_0 + p9_0 + p10_0 + p11_0 + p12_0 + p13_0 + p14_0 + p15_0
		h1 := pOld1 + p0_1 + p1_1 + p2_1 + p3_1 + p4_1 + p5_1 + p6_1 + p7_1 + p8_1 + p9_1 + p10_1 + p11_1 + p12_1 + p13_1 + p14_1 + p15_1
		h2 := pOld2 + p0_2 + p1_2 + p2_2 + p3_2 + p4_2 + p5_2 + p6_2 + p7_2 + p8_2 + p9_2 + p10_2 + p11_2 + p12_2 + p13_2 + p14_2 + p15_2

		c := h0 >> 44
		st.h0 = h0 & 0xfffffffffff
		h1 += c
		c = h1 >> 44
		st.h1 = h1 & 0xfffffffffff
		h2 += c
		c = h2 >> 42
		st.h2 = h2 & 0x3ffffffffff

		st.h0 += c * 5
		c = st.h0 >> 44
		st.h0 &= 0xfffffffffff
		st.h1 += c

		msg = msg[256:]
	}

	// Reliquats 8 blocs par 8 blocs (128 octets)
	for len(msg) >= 128 {
		m0_lo := binary.LittleEndian.Uint64(msg[0:8])
		m0_hi := binary.LittleEndian.Uint64(msg[8:16])
		m1_lo := binary.LittleEndian.Uint64(msg[16:24])
		m1_hi := binary.LittleEndian.Uint64(msg[24:32])
		m2_lo := binary.LittleEndian.Uint64(msg[32:40])
		m2_hi := binary.LittleEndian.Uint64(msg[40:48])
		m3_lo := binary.LittleEndian.Uint64(msg[48:56])
		m3_hi := binary.LittleEndian.Uint64(msg[56:64])

		m4_lo := binary.LittleEndian.Uint64(msg[64:72])
		m4_hi := binary.LittleEndian.Uint64(msg[72:80])
		m5_lo := binary.LittleEndian.Uint64(msg[80:88])
		m5_hi := binary.LittleEndian.Uint64(msg[88:96])
		m6_lo := binary.LittleEndian.Uint64(msg[96:104])
		m6_hi := binary.LittleEndian.Uint64(msg[104:112])
		m7_lo := binary.LittleEndian.Uint64(msg[112:120])
		m7_hi := binary.LittleEndian.Uint64(msg[120:128])

		b0_0 := m0_lo & 0xfffffffffff
		b0_1 := ((m0_lo >> 44) | (m0_hi << 20)) & 0xfffffffffff
		b0_2 := (m0_hi >> 24) | (1 << 40)

		b1_0 := m1_lo & 0xfffffffffff
		b1_1 := ((m1_lo >> 44) | (m1_hi << 20)) & 0xfffffffffff
		b1_2 := (m1_hi >> 24) | (1 << 40)

		b2_0 := m2_lo & 0xfffffffffff
		b2_1 := ((m2_lo >> 44) | (m2_hi << 20)) & 0xfffffffffff
		b2_2 := (m2_hi >> 24) | (1 << 40)

		b3_0 := m3_lo & 0xfffffffffff
		b3_1 := ((m3_lo >> 44) | (m3_hi << 20)) & 0xfffffffffff
		b3_2 := (m3_hi >> 24) | (1 << 40)

		b4_0 := m4_lo & 0xfffffffffff
		b4_1 := ((m4_lo >> 44) | (m4_hi << 20)) & 0xfffffffffff
		b4_2 := (m4_hi >> 24) | (1 << 40)

		b5_0 := m5_lo & 0xfffffffffff
		b5_1 := ((m5_lo >> 44) | (m5_hi << 20)) & 0xfffffffffff
		b5_2 := (m5_hi >> 24) | (1 << 40)

		b6_0 := m6_lo & 0xfffffffffff
		b6_1 := ((m6_lo >> 44) | (m6_hi << 20)) & 0xfffffffffff
		b6_2 := (m6_hi >> 24) | (1 << 40)

		b7_0 := m7_lo & 0xfffffffffff
		b7_1 := ((m7_lo >> 44) | (m7_hi << 20)) & 0xfffffffffff
		b7_2 := (m7_hi >> 24) | (1 << 40)

		pOld0, pOld1, pOld2 := mul44(st.h0, st.h1, st.h2, st.r8Key)
		p0_0, p0_1, p0_2 := mul44(b0_0, b0_1, b0_2, st.r8Key)
		p1_0, p1_1, p1_2 := mul44(b1_0, b1_1, b1_2, st.r7Key)
		p2_0, p2_1, p2_2 := mul44(b2_0, b2_1, b2_2, st.r6Key)
		p3_0, p3_1, p3_2 := mul44(b3_0, b3_1, b3_2, st.r5Key)
		p4_0, p4_1, p4_2 := mul44(b4_0, b4_1, b4_2, st.r4Key)
		p5_0, p5_1, p5_2 := mul44(b5_0, b5_1, b5_2, st.r3Key)
		p6_0, p6_1, p6_2 := mul44(b6_0, b6_1, b6_2, st.r2Key)
		p7_0, p7_1, p7_2 := mul44(b7_0, b7_1, b7_2, st.r1Key)

		h0 := pOld0 + p0_0 + p1_0 + p2_0 + p3_0 + p4_0 + p5_0 + p6_0 + p7_0
		h1 := pOld1 + p0_1 + p1_1 + p2_1 + p3_1 + p4_1 + p5_1 + p6_1 + p7_1
		h2 := pOld2 + p0_2 + p1_2 + p2_2 + p3_2 + p4_2 + p5_2 + p6_2 + p7_2

		c := h0 >> 44
		st.h0 = h0 & 0xfffffffffff
		h1 += c
		c = h1 >> 44
		st.h1 = h1 & 0xfffffffffff
		h2 += c
		c = h2 >> 42
		st.h2 = h2 & 0x3ffffffffff

		st.h0 += c * 5
		c = st.h0 >> 44
		st.h0 &= 0xfffffffffff
		st.h1 += c

		msg = msg[128:]
	}

	// Reliquats 4 blocs par 4 blocs
	for len(msg) >= 64 {
		m0_lo := binary.LittleEndian.Uint64(msg[0:8])
		m0_hi := binary.LittleEndian.Uint64(msg[8:16])
		m1_lo := binary.LittleEndian.Uint64(msg[16:24])
		m1_hi := binary.LittleEndian.Uint64(msg[24:32])
		m2_lo := binary.LittleEndian.Uint64(msg[32:40])
		m2_hi := binary.LittleEndian.Uint64(msg[40:48])
		m3_lo := binary.LittleEndian.Uint64(msg[48:56])
		m3_hi := binary.LittleEndian.Uint64(msg[56:64])

		b0_0 := m0_lo & 0xfffffffffff
		b0_1 := ((m0_lo >> 44) | (m0_hi << 20)) & 0xfffffffffff
		b0_2 := (m0_hi >> 24) | (1 << 40)

		b1_0 := m1_lo & 0xfffffffffff
		b1_1 := ((m1_lo >> 44) | (m1_hi << 20)) & 0xfffffffffff
		b1_2 := (m1_hi >> 24) | (1 << 40)

		b2_0 := m2_lo & 0xfffffffffff
		b2_1 := ((m2_lo >> 44) | (m2_hi << 20)) & 0xfffffffffff
		b2_2 := (m2_hi >> 24) | (1 << 40)

		b3_0 := m3_lo & 0xfffffffffff
		b3_1 := ((m3_lo >> 44) | (m3_hi << 20)) & 0xfffffffffff
		b3_2 := (m3_hi >> 24) | (1 << 40)

		pOld0, pOld1, pOld2 := mul44(st.h0, st.h1, st.h2, st.r4Key)
		p0_0, p0_1, p0_2 := mul44(b0_0, b0_1, b0_2, st.r4Key)
		p1_0, p1_1, p1_2 := mul44(b1_0, b1_1, b1_2, st.r3Key)
		p2_0, p2_1, p2_2 := mul44(b2_0, b2_1, b2_2, st.r2Key)
		p3_0, p3_1, p3_2 := mul44(b3_0, b3_1, b3_2, st.r1Key)

		h0 := pOld0 + p0_0 + p1_0 + p2_0 + p3_0
		h1 := pOld1 + p0_1 + p1_1 + p2_1 + p3_1
		h2 := pOld2 + p0_2 + p1_2 + p2_2 + p3_2

		c := h0 >> 44
		st.h0 = h0 & 0xfffffffffff
		h1 += c
		c = h1 >> 44
		st.h1 = h1 & 0xfffffffffff
		h2 += c
		c = h2 >> 42
		st.h2 = h2 & 0x3ffffffffff

		st.h0 += c * 5
		c = st.h0 >> 44
		st.h0 &= 0xfffffffffff
		st.h1 += c

		msg = msg[64:]
	}

	// Reliquats bloc par bloc
	for len(msg) >= 16 {
		st.updateBlock(msg[:16], 1)
		msg = msg[16:]
	}

	if len(msg) > 0 {
		st.bufLen = copy(st.buf[:], msg)
	}
}

func (st *Poly1305QuadChain) updateBlock(msg []byte, hibit uint64) {
	m_lo := binary.LittleEndian.Uint64(msg[0:8])
	m_hi := binary.LittleEndian.Uint64(msg[8:16])

	st.h0 += m_lo & 0xfffffffffff
	st.h1 += ((m_lo >> 44) | (m_hi << 20)) & 0xfffffffffff
	st.h2 += (m_hi >> 24) | (hibit << 40)

	st.h0, st.h1, st.h2 = mul44(st.h0, st.h1, st.h2, st.r1Key)
}

func (st *Poly1305QuadChain) Finish(out *[16]byte) {
	if st.bufLen > 0 {
		var block [16]byte
		copy(block[:], st.buf[:st.bufLen])
		block[st.bufLen] = 1
		st.updateBlock(block[:], 0)
	}

	// Réduction modulaire finale 3x44 bits
	c := st.h0 >> 44
	st.h0 &= 0xfffffffffff
	st.h1 += c
	c = st.h1 >> 44
	st.h1 &= 0xfffffffffff
	st.h2 += c
	c = st.h2 >> 42
	st.h2 &= 0x3ffffffffff

	st.h0 += c * 5
	c = st.h0 >> 44
	st.h0 &= 0xfffffffffff
	st.h1 += c

	// Soustraction conditionnelle p = 2^130 - 5
	g0 := st.h0 + 5
	c = g0 >> 44
	g0 &= 0xfffffffffff
	g1 := st.h1 + c
	c = g1 >> 44
	g1 &= 0xfffffffffff
	g2 := st.h2 + c - (1 << 42)

	mask := uint64(g2 >> 63) - 1

	st.h0 = (st.h0 &^ mask) | (g0 & mask)
	st.h1 = (st.h1 &^ mask) | (g1 & mask)
	st.h2 = (st.h2 &^ mask) | (g2 & mask)

	u0 := st.h0 | (st.h1 << 44)
	u1 := (st.h1 >> 20) | (st.h2 << 24)

	u0, carry := bits.Add64(u0, uint64(st.pad0)|(uint64(st.pad1)<<32), 0)
	u1, _ = bits.Add64(u1, uint64(st.pad2)|(uint64(st.pad3)<<32), carry)

	binary.LittleEndian.PutUint64(out[0:8], u0)
	binary.LittleEndian.PutUint64(out[8:16], u1)
}
