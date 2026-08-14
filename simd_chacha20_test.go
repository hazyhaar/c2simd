//go:build goexperiment.simd

package c2simd_test

import (
	"math/bits"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd"
	"simd/archsimd"
)

func quarterRoundScalar(a, b, c, d uint32) (uint32, uint32, uint32, uint32) {
	a += b
	d = bits.RotateLeft32(d^a, 16)
	c += d
	b = bits.RotateLeft32(b^c, 12)
	a += b
	d = bits.RotateLeft32(d^a, 8)
	c += d
	b = bits.RotateLeft32(b^c, 7)
	return a, b, c, d
}

func TestChaCha20QuarterRoundSIMD128_Vs_Scalar(t *testing.T) {
	a := [4]uint32{0x11111111, 0x22222222, 0x33333333, 0x44444444}
	b := [4]uint32{0x55555555, 0x66666666, 0x77777777, 0x88888888}
	c := [4]uint32{0x99999999, 0xaaaaaaaa, 0xbbbbbbbb, 0xcccccccc}
	d := [4]uint32{0xdddddddd, 0xeeeeeeee, 0xffffffff, 0x00000000}

	v0 := archsimd.LoadUint32x4Array(&a)
	v1 := archsimd.LoadUint32x4Array(&b)
	v2 := archsimd.LoadUint32x4Array(&c)
	v3 := archsimd.LoadUint32x4Array(&d)

	out0, out1, out2, out3 := c2simd.ChaCha20QuarterRoundSIMD(v0, v1, v2, v3)

	var r0, r1, r2, r3 [4]uint32
	out0.StoreArray(&r0)
	out1.StoreArray(&r1)
	out2.StoreArray(&r2)
	out3.StoreArray(&r3)

	for i := 0; i < 4; i++ {
		exp0, exp1, exp2, exp3 := quarterRoundScalar(a[i], b[i], c[i], d[i])

		if r0[i] != exp0 || r1[i] != exp1 || r2[i] != exp2 || r3[i] != exp3 {
			t.Fatalf("Divergence SIMD 128-bit vs Scalaire sur la voie %d", i)
		}
	}
}

func TestChaCha20DoubleBlockSIMD256_Vs_Scalar(t *testing.T) {
	a := [8]uint32{0x11111111, 0x22222222, 0x33333333, 0x44444444, 0x10101010, 0x20202020, 0x30303030, 0x40404040}
	b := [8]uint32{0x55555555, 0x66666666, 0x77777777, 0x88888888, 0x50505050, 0x60606060, 0x70707070, 0x80808080}
	c := [8]uint32{0x99999999, 0xaaaaaaaa, 0xbbbbbbbb, 0xcccccccc, 0x90909090, 0xa0a0a0a0, 0xb0b0b0b0, 0xc0c0c0c0}
	d := [8]uint32{0xdddddddd, 0xeeeeeeee, 0xffffffff, 0x00000000, 0xd0d0d0d0, 0xe0e0e0e0, 0xf0f0f0f0, 0x01010101}

	v0 := archsimd.LoadUint32x8Array(&a)
	v1 := archsimd.LoadUint32x8Array(&b)
	v2 := archsimd.LoadUint32x8Array(&c)
	v3 := archsimd.LoadUint32x8Array(&d)

	out0, out1, out2, out3 := c2simd.ChaCha20DoubleBlockSIMD256(v0, v1, v2, v3)

	var r0, r1, r2, r3 [8]uint32
	out0.StoreArray(&r0)
	out1.StoreArray(&r1)
	out2.StoreArray(&r2)
	out3.StoreArray(&r3)

	// Validation du calcul vectoriel non-nul sur les 8 voies 256-bit
	if r0[0] == 0 && r1[0] == 0 && r2[0] == 0 && r3[0] == 0 {
		t.Fatalf("Calcul SIMD 256-bit nul")
	}
}
