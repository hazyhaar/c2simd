//go:build goexperiment.simd

package main

import (
	"bytes"
	"fmt"
	"math/bits"
	"math/rand"
	"simd/archsimd"
	"time"
	"unsafe"
)

// Algorithme 1 : Scanner SIMD AVX2 (256-bit SIMD Text Scan - Pattern simdjson / AVX2 memchr)
func SIMD_TextScan_AVX2(data []byte, target byte) int {
	targetVec := archsimd.LoadUint8x32Array(&[32]byte{
		target, target, target, target, target, target, target, target,
		target, target, target, target, target, target, target, target,
		target, target, target, target, target, target, target, target,
		target, target, target, target, target, target, target, target,
	})

	count := 0
	off := 0

	for off+32 <= len(data) {
		chunkPtr := (*[32]byte)(unsafe.Pointer(&data[off]))
		vChunk := archsimd.LoadUint8x32Array(chunkPtr)
		vEq := vChunk.Equal(targetVec)

		// Extraction du masque 32 bits et comptage 1-cycle POPCNT
		count += bits.OnesCount32(vEq.ToBits())
		off += 32
	}

	for i := off; i < len(data); i++ {
		if data[i] == target {
			count++
		}
	}
	return count
}

func Scalar_TextScan(data []byte, target byte) int {
	count := 0
	for i := 0; i < len(data); i++ {
		if data[i] == target {
			count++
		}
	}
	return count
}

// Algorithme 2 : Produit Scalaire SIMD (Vector DB Cosine Distance 1024-dim Float32)
// Accumulation 100% en registres YMM (0 StoreArray dans la boucle chaude)
func SIMD_DotProduct_AVX2(a, b []float32) float32 {
	n := len(a)
	i := 0

	// Accumulateurs en registres vectoriels YMM
	var zeroBuf [8]float32
	vSum0 := archsimd.LoadFloat32x8Array(&zeroBuf)
	vSum1 := archsimd.LoadFloat32x8Array(&zeroBuf)

	// Traitement 16 float32 par itération (2x256-bit FMA)
	for i+16 <= n {
		vA0 := archsimd.LoadFloat32x8Array((*[8]float32)(unsafe.Pointer(&a[i])))
		vB0 := archsimd.LoadFloat32x8Array((*[8]float32)(unsafe.Pointer(&b[i])))
		vSum0 = vSum0.Add(vA0.Mul(vB0))

		vA1 := archsimd.LoadFloat32x8Array((*[8]float32)(unsafe.Pointer(&a[i+8])))
		vB1 := archsimd.LoadFloat32x8Array((*[8]float32)(unsafe.Pointer(&b[i+8])))
		vSum1 = vSum1.Add(vA1.Mul(vB1))

		i += 16
	}

	vSumTotal := vSum0.Add(vSum1)

	// StoreArray exécuté UNE SEULE FOIS en sortie de boucle (0 Store-Forwarding stall)
	var buf [8]float32
	vSumTotal.StoreArray(&buf)

	total := buf[0] + buf[1] + buf[2] + buf[3] + buf[4] + buf[5] + buf[6] + buf[7]
	for ; i < n; i++ {
		total += a[i] * b[i]
	}
	return total
}

func Scalar_DotProduct(a, b []float32) float32 {
	var sum float32
	for i := 0; i < len(a); i++ {
		sum += a[i] * b[i]
	}
	return sum
}

func main() {
	fmt.Println("=== Laboratoire d'Exploration SIMD Non-Cryptographique (Go 1.27 archsimd) ===")
	fmt.Println()

	// 1. Benchmark Text Scan (10 Mo de texte)
	textSize := 10 * 1024 * 1024
	textData := make([]byte, textSize)
	for i := 0; i < textSize; i++ {
		textData[i] = byte('a' + rand.Intn(26))
	}
	for i := 0; i < textSize; i += 100 {
		textData[i] = '\n'
	}

	iterations := 100

	t0 := time.Now()
	simdCount := 0
	for it := 0; it < iterations; it++ {
		simdCount = SIMD_TextScan_AVX2(textData, '\n')
	}
	simdDuration := time.Since(t0)
	simdThroughput := float64(textSize*iterations) / (simdDuration.Seconds() * 1024 * 1024)

	t0 = time.Now()
	scalarCount := 0
	for it := 0; it < iterations; it++ {
		scalarCount = Scalar_TextScan(textData, '\n')
	}
	scalarDuration := time.Since(t0)
	scalarThroughput := float64(textSize*iterations) / (scalarDuration.Seconds() * 1024 * 1024)

	t0 = time.Now()
	bytesCount := 0
	for it := 0; it < iterations; it++ {
		bytesCount = bytes.Count(textData, []byte{'\n'})
	}
	bytesDuration := time.Since(t0)
	bytesThroughput := float64(textSize*iterations) / (bytesDuration.Seconds() * 1024 * 1024)

	fmt.Println("--- Domaine N°1 : Balayage de texte SIMD 256 bits (Parsing / Lexer 10 Mo) ---")
	fmt.Printf("  • Scalaire Go classique     : %8.2f Mo/s (%d occurrences)\n", scalarThroughput, scalarCount)
	fmt.Printf("  • `bytes.Count` standard Go : %8.2f Mo/s (%d occurrences)\n", bytesThroughput, bytesCount)
	fmt.Printf("  • Pure Go SIMD AVX2 256 bits: %8.2f Mo/s (%d occurrences) [Acceleration: +%.1f%% vs Scalaire]\n",
		simdThroughput, simdCount, ((simdThroughput/scalarThroughput)-1)*100)
	fmt.Println()

	// 2. Benchmark Produit Scalaire IA (Vector DB Cosine Distance 1024-dim Float32)
	dim := 1024
	numVecs := 100000
	vecA := make([]float32, dim)
	vecB := make([]float32, dim)
	for i := 0; i < dim; i++ {
		vecA[i] = rand.Float32()
		vecB[i] = rand.Float32()
	}

	t0 = time.Now()
	var resSimd float32
	for i := 0; i < numVecs; i++ {
		resSimd = SIMD_DotProduct_AVX2(vecA, vecB)
	}
	durSimdVec := time.Since(t0)
	opsSimd := float64(numVecs) / durSimdVec.Seconds()

	t0 = time.Now()
	var resScal float32
	for i := 0; i < numVecs; i++ {
		resScal = Scalar_DotProduct(vecA, vecB)
	}
	durScalVec := time.Since(t0)
	opsScal := float64(numVecs) / durScalVec.Seconds()

	fmt.Println("--- Domaine N°2 : Produit Scalaire Vector DB / Embeddings IA (1024-dim Float32) ---")
	fmt.Printf("  • Scalaire Go `for` loop    : %8.0f vecs/s (Résultat: %.4f)\n", opsScal, resScal)
	fmt.Printf("  • Pure Go SIMD AVX2 FMA     : %8.0f vecs/s (Résultat: %.4f) [Acceleration: +%.1f%% vs Scalaire]\n",
		opsSimd, resSimd, ((opsSimd/opsScal)-1)*100)
	fmt.Println()
}
