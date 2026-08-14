package main

import (
	"bytes"
	"crypto/rand"
	"testing"
	"unsafe"

	opt_blake2b "code.hazyhaar.fr/devhoros/c2simd/spec/c_sources/opt/blake2b_compress"
	opt_xor "code.hazyhaar.fr/devhoros/c2simd/spec/c_sources/opt/fast_xor"
	opt_md5 "code.hazyhaar.fr/devhoros/c2simd/spec/c_sources/opt/md5_transform"
	opt_sip "code.hazyhaar.fr/devhoros/c2simd/spec/c_sources/opt/siphash24"

	raw_blake2b "code.hazyhaar.fr/devhoros/c2simd/spec/c_sources/raw/blake2b_compress"
	raw_xor "code.hazyhaar.fr/devhoros/c2simd/spec/c_sources/raw/fast_xor"
	raw_md5 "code.hazyhaar.fr/devhoros/c2simd/spec/c_sources/raw/md5_transform"
	raw_sip "code.hazyhaar.fr/devhoros/c2simd/spec/c_sources/raw/siphash24"

	"modernc.org/libc"
)

// TestKAT_SipHash24_Equivalence certifie l'équivalence exacte entre C transpilé brut et Go optimisé par l'AST
func TestKAT_SipHash24_Equivalence(t *testing.T) {
	tls := libc.NewTLS()
	defer tls.Close()

	key := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	msg := make([]byte, 1024)
	rand.Read(msg)

	keyPtr := tls.Alloc(16)
	msgPtr := tls.Alloc(1024)
	copy(unsafe.Slice((*byte)(unsafe.Pointer(keyPtr)), 16), key[:])
	copy(unsafe.Slice((*byte)(unsafe.Pointer(msgPtr)), 1024), msg)

	resRaw := raw_sip.Siphash24(tls, msgPtr, 1024, keyPtr)
	resOpt := opt_sip.Siphash24(tls, msgPtr, 1024, keyPtr)

	if resRaw != resOpt {
		t.Fatalf("SipHash24 DIVERGENCE: Raw=%x vs Opt=%x", resRaw, resOpt)
	}
}

// TestKAT_BLAKE2b_Equivalence certifie l'équivalence exacte pour la compression BLAKE2b
func TestKAT_BLAKE2b_Equivalence(t *testing.T) {
	tls := libc.NewTLS()
	defer tls.Close()

	hRaw := [8]uint64{0x6a09e667f3bcc908, 0xbb67ae8584caa73b, 0x3c6ef372fe94f82b, 0xa54ff53a5f1d36f1, 0x510e527fea90715d, 0x9b05688c2b3e6c1f, 0x1f83d9abfb41bd6b, 0x5be0cd19137e2179}
	hOpt := hRaw
	block := [128]byte{1, 2, 3, 4, 5, 6, 7, 8}
	rand.Read(block[:])

	hRawPtr := tls.Alloc(64)
	hOptPtr := tls.Alloc(64)
	blockPtr := tls.Alloc(128)

	copy(unsafe.Slice((*byte)(unsafe.Pointer(hRawPtr)), 64), unsafe.Slice((*byte)(unsafe.Pointer(&hRaw[0])), 64))
	copy(unsafe.Slice((*byte)(unsafe.Pointer(hOptPtr)), 64), unsafe.Slice((*byte)(unsafe.Pointer(&hOpt[0])), 64))
	copy(unsafe.Slice((*byte)(unsafe.Pointer(blockPtr)), 128), block[:])

	raw_blake2b.Blake2b_compress_block(tls, hRawPtr, blockPtr, 128, 0, 0, 0)
	opt_blake2b.Blake2b_compress_block(tls, hOptPtr, blockPtr, 128, 0, 0, 0)

	outRaw := unsafe.Slice((*byte)(unsafe.Pointer(hRawPtr)), 64)
	outOpt := unsafe.Slice((*byte)(unsafe.Pointer(hOptPtr)), 64)

	if !bytes.Equal(outRaw, outOpt) {
		t.Fatalf("BLAKE2b DIVERGENCE:\n Raw=%x\n Opt=%x", outRaw, outOpt)
	}
}

// TestKAT_MD5_Equivalence certifie l'équivalence exacte pour la transformation MD5
func TestKAT_MD5_Equivalence(t *testing.T) {
	tls := libc.NewTLS()
	defer tls.Close()

	stRaw := [4]uint32{0x67452301, 0xefcdab89, 0x98badcfe, 0x10325476}
	stOpt := stRaw
	block := [64]byte{1, 2, 3, 4}
	rand.Read(block[:])

	stRawPtr := tls.Alloc(16)
	stOptPtr := tls.Alloc(16)
	blockPtr := tls.Alloc(64)

	copy(unsafe.Slice((*byte)(unsafe.Pointer(stRawPtr)), 16), unsafe.Slice((*byte)(unsafe.Pointer(&stRaw[0])), 16))
	copy(unsafe.Slice((*byte)(unsafe.Pointer(stOptPtr)), 16), unsafe.Slice((*byte)(unsafe.Pointer(&stOpt[0])), 16))
	copy(unsafe.Slice((*byte)(unsafe.Pointer(blockPtr)), 64), block[:])

	raw_md5.Md5_transform_block(tls, stRawPtr, blockPtr)
	opt_md5.Md5_transform_block(tls, stOptPtr, blockPtr)

	outRaw := unsafe.Slice((*byte)(unsafe.Pointer(stRawPtr)), 16)
	outOpt := unsafe.Slice((*byte)(unsafe.Pointer(stOptPtr)), 16)

	if !bytes.Equal(outRaw, outOpt) {
		t.Fatalf("MD5 DIVERGENCE:\n Raw=%x\n Opt=%x", outRaw, outOpt)
	}
}

// TestKAT_FastXOR_Equivalence certifie l'équivalence exacte pour le masquage XOR
func TestKAT_FastXOR_Equivalence(t *testing.T) {
	tls := libc.NewTLS()
	defer tls.Close()

	src1 := make([]byte, 4096)
	src2 := make([]byte, 4096)
	rand.Read(src1)
	rand.Read(src2)

	dstRawPtr := tls.Alloc(4096)
	dstOptPtr := tls.Alloc(4096)
	s1Ptr := tls.Alloc(4096)
	s2Ptr := tls.Alloc(4096)

	copy(unsafe.Slice((*byte)(unsafe.Pointer(s1Ptr)), 4096), src1)
	copy(unsafe.Slice((*byte)(unsafe.Pointer(s2Ptr)), 4096), src2)

	raw_xor.Fast_xor_bytes(tls, dstRawPtr, s1Ptr, s2Ptr, 4096)
	opt_xor.Fast_xor_bytes(tls, dstOptPtr, s1Ptr, s2Ptr, 4096)

	outRaw := unsafe.Slice((*byte)(unsafe.Pointer(dstRawPtr)), 4096)
	outOpt := unsafe.Slice((*byte)(unsafe.Pointer(dstOptPtr)), 4096)

	if !bytes.Equal(outRaw, outOpt) {
		t.Fatalf("FastXOR DIVERGENCE: Raw and Opt output mismatch")
	}
}
