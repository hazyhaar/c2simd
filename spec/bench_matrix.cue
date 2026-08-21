package spec

#BenchMetric: {
	name:           string
	target_mbps:    number
	max_allocs:     int
	max_latency_ns: number
}

#BenchMatrix: {
	timestamp: string
	cpu_model: string
	go_version: string
	metrics: [string]: #BenchMetric
}

// Spécification déterministe CUE des seuils de régression c2simd
baseline_thresholds: #BenchMatrix & {
	timestamp:  "2026-08-06T12:37:00Z"
	cpu_model:  "Intel(R) Core(TM) i9-14900K"
	go_version: "go1.27rc1"
	metrics: {
		"chacha20_simd256_pure": {
			name:           "ChaCha20 SIMD256 Pure Stream"
			target_mbps:    1350.0
			max_allocs:     0
			max_latency_ns: 780000
		}
		"poly1305_ymm4_parallel": {
			name:           "Poly1305 YMM4 Parallel Engine"
			target_mbps:    2800.0
			max_allocs:     0
			max_latency_ns: 360000
		}
		"c2simd_aead_fused": {
			name:           "c2simd AEAD Fused SIMD256"
			target_mbps:    1200.0
			max_allocs:     2
			max_latency_ns: 860000
		}
		"c2tuidiff_simd256_diffing": {
			name:           "c2tuidiff 2D Grid SIMD256"
			target_mbps:    3000.0
			max_allocs:     0
			max_latency_ns: 5000
		}
	}
}
