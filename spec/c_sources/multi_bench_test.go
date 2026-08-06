//go:build goexperiment.simd

package main

import (
	"crypto/md5"
	"testing"
	"unsafe"

	opt_blake2b "github.com/hazyhaar/c2simd/spec/c_sources/opt/blake2b_compress"
	opt_xor "github.com/hazyhaar/c2simd/spec/c_sources/opt/fast_xor"
	opt_md5 "github.com/hazyhaar/c2simd/spec/c_sources/opt/md5_transform"
	opt_sip "github.com/hazyhaar/c2simd/spec/c_sources/opt/siphash24"

	raw_blake2b "github.com/hazyhaar/c2simd/spec/c_sources/raw/blake2b_compress"
	raw_xor "github.com/hazyhaar/c2simd/spec/c_sources/raw/fast_xor"
	raw_md5 "github.com/hazyhaar/c2simd/spec/c_sources/raw/md5_transform"
	raw_sip "github.com/hazyhaar/c2simd/spec/c_sources/raw/siphash24"

	"modernc.org/libc"
)

// Benchmarks 1 : SipHash 2-4 (Chiffrement / Paquets Stream 1024 bytes)
func BenchmarkSipHash24_Raw_Transpiled(b *testing.B) {
	tls := libc.NewTLS()
	defer tls.Close()

	key := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	msg := make([]byte, 1024)
	for i := range msg {
		msg[i] = byte(i)
	}

	keyPtr := tls.Alloc(16)
	msgPtr := tls.Alloc(1024)
	copy(unsafe.Slice((*byte)(unsafe.Pointer(keyPtr)), 16), key[:])
	copy(unsafe.Slice((*byte)(unsafe.Pointer(msgPtr)), 1024), msg)

	b.SetBytes(1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		raw_sip.Siphash24(tls, msgPtr, 1024, keyPtr)
	}
}

func BenchmarkSipHash24_AST_Optimized(b *testing.B) {
	tls := libc.NewTLS()
	defer tls.Close()

	key := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	msg := make([]byte, 1024)
	for i := range msg {
		msg[i] = byte(i)
	}

	keyPtr := tls.Alloc(16)
	msgPtr := tls.Alloc(1024)
	copy(unsafe.Slice((*byte)(unsafe.Pointer(keyPtr)), 16), key[:])
	copy(unsafe.Slice((*byte)(unsafe.Pointer(msgPtr)), 1024), msg)

	b.SetBytes(1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		opt_sip.Siphash24(tls, msgPtr, 1024, keyPtr)
	}
}

// Benchmarks 2 : BLAKE2b Round Compress (Hachage d'arbre 128 bytes)
func BenchmarkBLAKE2b_Raw_Transpiled(b *testing.B) {
	tls := libc.NewTLS()
	defer tls.Close()

	h := [8]uint64{0x6a09e667f3bcc908, 0xbb67ae8584caa73b, 0x3c6ef372fe94f82b, 0xa54ff53a5f1d36f1, 0x510e527fea90715d, 0x9b05688c2b3e6c1f, 0x1f83d9abfb41bd6b, 0x5be0cd19137e2179}
	block := [128]byte{1, 2, 3, 4, 5, 6, 7, 8}

	hPtr := tls.Alloc(64)
	blockPtr := tls.Alloc(128)
	copy(unsafe.Slice((*byte)(unsafe.Pointer(hPtr)), 64), unsafe.Slice((*byte)(unsafe.Pointer(&h[0])), 64))
	copy(unsafe.Slice((*byte)(unsafe.Pointer(blockPtr)), 128), block[:])

	b.SetBytes(128)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		raw_blake2b.Blake2b_compress_block(tls, hPtr, blockPtr, 128, 0, 0, 0)
	}
}

func BenchmarkBLAKE2b_AST_Optimized(b *testing.B) {
	tls := libc.NewTLS()
	defer tls.Close()

	h := [8]uint64{0x6a09e667f3bcc908, 0xbb67ae8584caa73b, 0x3c6ef372fe94f82b, 0xa54ff53a5f1d36f1, 0x510e527fea90715d, 0x9b05688c2b3e6c1f, 0x1f83d9abfb41bd6b, 0x5be0cd19137e2179}
	block := [128]byte{1, 2, 3, 4, 5, 6, 7, 8}

	hPtr := tls.Alloc(64)
	blockPtr := tls.Alloc(128)
	copy(unsafe.Slice((*byte)(unsafe.Pointer(hPtr)), 64), unsafe.Slice((*byte)(unsafe.Pointer(&h[0])), 64))
	copy(unsafe.Slice((*byte)(unsafe.Pointer(blockPtr)), 128), block[:])

	b.SetBytes(128)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		opt_blake2b.Blake2b_compress_block(tls, hPtr, blockPtr, 128, 0, 0, 0)
	}
}

// Benchmarks 3 : MD5 Transform Block (Formatage legacy stream)
func BenchmarkMD5_Raw_Transpiled(b *testing.B) {
	tls := libc.NewTLS()
	defer tls.Close()

	state := [4]uint32{0x67452301, 0xefcdab89, 0x98badcfe, 0x10325476}
	block := [64]byte{1, 2, 3, 4}

	statePtr := tls.Alloc(16)
	blockPtr := tls.Alloc(64)
	copy(unsafe.Slice((*byte)(unsafe.Pointer(statePtr)), 16), unsafe.Slice((*byte)(unsafe.Pointer(&state[0])), 16))
	copy(unsafe.Slice((*byte)(unsafe.Pointer(blockPtr)), 64), block[:])

	b.SetBytes(64)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		raw_md5.Md5_transform_block(tls, statePtr, blockPtr)
	}
}

func BenchmarkMD5_AST_Optimized(b *testing.B) {
	tls := libc.NewTLS()
	defer tls.Close()

	state := [4]uint32{0x67452301, 0xefcdab89, 0x98badcfe, 0x10325476}
	block := [64]byte{1, 2, 3, 4}

	statePtr := tls.Alloc(16)
	blockPtr := tls.Alloc(64)
	copy(unsafe.Slice((*byte)(unsafe.Pointer(statePtr)), 16), unsafe.Slice((*byte)(unsafe.Pointer(&state[0])), 16))
	copy(unsafe.Slice((*byte)(unsafe.Pointer(blockPtr)), 64), block[:])

	b.SetBytes(64)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		opt_md5.Md5_transform_block(tls, statePtr, blockPtr)
	}
}

func BenchmarkMD5_NativeGo_Stdlib(b *testing.B) {
	block := [64]byte{1, 2, 3, 4}
	b.SetBytes(64)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = md5.Sum(block[:])
	}
}

// Benchmarks 4 : Fast XOR Stream (4096 bytes)
func BenchmarkFastXOR_Raw_Transpiled(b *testing.B) {
	tls := libc.NewTLS()
	defer tls.Close()

	src1 := make([]byte, 4096)
	src2 := make([]byte, 4096)

	dstPtr := tls.Alloc(4096)
	s1Ptr := tls.Alloc(4096)
	s2Ptr := tls.Alloc(4096)

	copy(unsafe.Slice((*byte)(unsafe.Pointer(s1Ptr)), 4096), src1)
	copy(unsafe.Slice((*byte)(unsafe.Pointer(s2Ptr)), 4096), src2)

	b.SetBytes(4096)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		raw_xor.Fast_xor_bytes(tls, dstPtr, s1Ptr, s2Ptr, 4096)
	}
}

func BenchmarkFastXOR_AST_Optimized(b *testing.B) {
	tls := libc.NewTLS()
	defer tls.Close()

	src1 := make([]byte, 4096)
	src2 := make([]byte, 4096)

	dstPtr := tls.Alloc(4096)
	s1Ptr := tls.Alloc(4096)
	s2Ptr := tls.Alloc(4096)

	copy(unsafe.Slice((*byte)(unsafe.Pointer(s1Ptr)), 4096), src1)
	copy(unsafe.Slice((*byte)(unsafe.Pointer(s2Ptr)), 4096), src2)

	b.SetBytes(4096)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		opt_xor.Fast_xor_bytes(tls, dstPtr, s1Ptr, s2Ptr, 4096)
	}
}
