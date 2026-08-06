package c2simd

import "modernc.org/libc"

func chacha20_rounds_sample(tls *libc.TLS, x uint32) uint32 {
	return rotl32(tls, x, uint32(16))
}
