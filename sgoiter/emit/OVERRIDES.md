# Emit kernel overrides

| name | skip IR | Status |
|------|---------|--------|
| `vn` | prefix | **active** (CT n=16/32) |
| `fnv1a_64` | — | **DROPPED** — IR + `maybePostUnrollHashes` |
| `blake2b_compress_block` | — | **DROPPED** — IR G-expand + const-trip unroll 12 + rotate load-alias |
| `fast_xor_bytes` | yes | **active** — 16B+8B+switch tail |
| `murmur3_x86_32` | — | **DROPPED** — IR + `maybeRewriteMurmurLoop` |
| `md5_transform_block` | yes | active (4×FF fixture) |
| `base64_encode_stream` | yes | active |
| `siphash24` | yes | active |
| `strlenspn_lab` | yes | active |
| `chacha20_simd_projections` | — | **landed** (hand) — F-sgoiter-simd-direct-projections (0 stack spill) |
| `aead_fused_streaming` | — | **landed** (hand) — F-sgoiter-aead-fused-streaming-256b (1-pass chunk 256B) |
| `poly1305_radix64` | — | **landed** (hand) — F-sgoiter-poly1305-radix64-reduction (6 mults/bloc) |

## Post-passes / rules

- `unroll_const_trip_load` + **general const-trip unroll** (N≤16, nested do-while OK)
- rotate peephole: `sameRotSrc` via Mov + xor-pair + **load-alias** (ptr reload)
- `stripDeadAssigns`: multi-assign LHS is not a use (unrolled temps)
- `maybePostUnrollHashes` · `maybeRewriteMurmurLoop` · `rewriteKernelLELoads`
- *SIMD pipeline candidate AST passes (proposed) :* `emit_simd_direct_projections` · `ast_aead_fused_streaming` · `emit_poly1305_radix64`
