//go:build goexperiment.simd

package c2simd

import (
	"simd/archsimd"
)

// ChaCha20QuarterRoundSIMD effectue un quart de tour ChaCha20 sur 4 colonnes en parallèle (128-bit Uint32x4).
func ChaCha20QuarterRoundSIMD(v0, v1, v2, v3 archsimd.Uint32x4) (archsimd.Uint32x4, archsimd.Uint32x4, archsimd.Uint32x4, archsimd.Uint32x4) {
	v0 = v0.Add(v1)
	v3 = v3.Xor(v0).RotateAllLeft(16)
	v2 = v2.Add(v3)
	v1 = v1.Xor(v2).RotateAllLeft(12)
	v0 = v0.Add(v1)
	v3 = v3.Xor(v0).RotateAllLeft(8)
	v2 = v2.Add(v3)
	v1 = v1.Xor(v2).RotateAllLeft(7)

	return v0, v1, v2, v3
}

// ChaCha20DoubleBlockSIMD256 effectue 1 tour (Column Round + Diagonal Round) sur 2 blocs ChaCha20 en parallèle (256-bit Uint32x8).
func ChaCha20DoubleBlockSIMD256(v0, v1, v2, v3 archsimd.Uint32x8) (archsimd.Uint32x8, archsimd.Uint32x8, archsimd.Uint32x8, archsimd.Uint32x8) {
	// 1. Column Round
	v0 = v0.Add(v1)
	v3 = v3.Xor(v0).RotateAllLeft(16)
	v2 = v2.Add(v3)
	v1 = v1.Xor(v2).RotateAllLeft(12)
	v0 = v0.Add(v1)
	v3 = v3.Xor(v0).RotateAllLeft(8)
	v2 = v2.Add(v3)
	v1 = v1.Xor(v2).RotateAllLeft(7)

	// 2. Diagonal Round (Shuffles des lignes v1, v2, v3 via PermuteScalarsGrouped)
	v1 = v1.PermuteScalarsGrouped(1, 2, 3, 0)
	v2 = v2.PermuteScalarsGrouped(2, 3, 0, 1)
	v3 = v3.PermuteScalarsGrouped(3, 0, 1, 2)

	v0 = v0.Add(v1)
	v3 = v3.Xor(v0).RotateAllLeft(16)
	v2 = v2.Add(v3)
	v1 = v1.Xor(v2).RotateAllLeft(12)
	v0 = v0.Add(v1)
	v3 = v3.Xor(v0).RotateAllLeft(8)
	v2 = v2.Add(v3)
	v1 = v1.Xor(v2).RotateAllLeft(7)

	// Inverse Shuffles pour restaurer l'ordre des colonnes
	v1 = v1.PermuteScalarsGrouped(3, 0, 1, 2)
	v2 = v2.PermuteScalarsGrouped(2, 3, 0, 1)
	v3 = v3.PermuteScalarsGrouped(1, 2, 3, 0)

	return v0, v1, v2, v3
}
