package c2simd

func AEADLock(key, nonce, ad, message []byte) ([]byte, []byte, error) {
	return AEADLockSIMD256_Fused(key, nonce, ad, message)
}

func AEADLockDst(dstBuffer []byte, outMac *[16]byte, key, nonce, ad, message []byte) ([]byte, error) {
	return AEADLockSIMD256_FusedDst(dstBuffer, outMac, key, nonce, ad, message)
}

func AEADUnlock(key, nonce, ad, ciphertext, mac []byte) ([]byte, error) {
	return AEADUnlockSIMD256_Fused(key, nonce, ad, ciphertext, mac)
}

func AEADUnlockDst(dstBuffer []byte, key, nonce, ad, ciphertext, mac []byte) ([]byte, error) {
	return AEADUnlockSIMD256_FusedDst(dstBuffer, key, nonce, ad, ciphertext, mac)
}
