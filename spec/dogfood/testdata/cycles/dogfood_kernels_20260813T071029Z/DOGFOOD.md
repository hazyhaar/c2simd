# Dogfood kernels fresh emit

- stamp: 20260813T071029Z
- sgoiter: rebuilt 2026-08-13T09:10:33+02:00
- mode: kernel
- count: 21 ok / 0 fail

| kernel | lines |
|--------|------:|
| adv_computed_goto | 32 |
| adv_pointer_alias | 23 |
| adv_stack_buf | 27 |
| adv_tls_depth | 28 |
| base64_simd | 40 |
| blake2b_compress | 142 |
| chacha20_qr | 30 |
| cjson_number_dogfood | 39 |
| crc32_ieee | 22 |
| fast_xor | 49 |
| fnv1a_64 | 26 |
| md5_transform_full | 247 |
| md5_transform | 43 |
| murmur3_x86_32 | 57 |
| poly1305_block5 | 54 |
| siphash24 | 68 |
| stbi_crc_dogfood | 22 |
| strlenspn_lab | 20 |
| tweetnacl_dogfood | 116 |
| utf8_iterate_dogfood | 44 |
| yyjson_digit_dogfood | 34 |

## Monocypher AEAD

Emit naïf `sgoiter -in monocypher_amalg.c` **ne doit pas** remplacer le package versionné :
parity MAC vs ccgo **casse** (tests `TestParityVsCCGO_*` rouges).
Package conservé = HEAD `pkg/secretstream55/internal/monocypher_sgoiter/` (1798 L, tests verts).
`ci_check` reste rouge sur le contrôle « regen mécanique == versionné » — dette connue.
