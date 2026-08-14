# Bench comparatif C O2 / sgoiter / ccgo — 2026-08-11

Stamp: 2026-08-11T20:47:07Z · linux/amd64 · go1.26.5

| lib | C O2 ns/op | sgoiter ns/op | ccgo ns/op | sgo/C | sgo≡C | ccgo≡C | ccgo st |
|-----|------------|---------------|------------|-------|-------|--------|---------|
| fnv1a_64 | 502504 | 1142802 | 1681057 | 2.27x | True | True | ok |
| crc32_ieee | 1029980 | 4751870 | 1926878 | 4.61x | True | True | ok |
| fast_xor | 739476 | 1360130 | 1579350 | 1.84x | True | True | ok |
| siphash24 | 440965 | 1086582 | 1493188 | 2.46x | True | True | ok |
| murmur3_x86_32 | 397925 | 1086273 | 1412043 | 2.73x | True | True | ok |
| blake2b_compress | 402429 | 990417 | 1341348 | 2.46x | True | True | ok |
| chacha20_qr | 439254 | 1063685 | 1257060 | 2.42x | True | True | ok |
| md5_transform | 374251 | 1036768 | 1332784 | 2.77x | True | True | ok |
| poly1305_block5 | 406835 | 1017611 | 1294001 | 2.50x | True | True | ok |
| base64_simd | 396741 | 1150731 | 2824698 | 2.90x | True | True | ok |
| tweetnacl_dogfood | 383488 | 1071758 | 1339449 | 2.79x | True | True | ok |
| libinjection_sqli | 0 | 1154758 | 0 | — | False | False | skip! |

Summary JSON: `{'libs_total': 12, 'libs_all_match': 11, 'backend_ok': 45, 'backend_fail': 0, 'backend_skip': 3, 'sgoiter_match_oracle': 11, 'ccgo_match_oracle': 11, 'libs_compared': 11, 'libs_no_oracle': 1}`

Méthode: `attachBench` = boucle de process (ns/op grossier, comparable entre backends).
ccgo: modernc.org/libc via go get avant transpile.
libinjection: pas d'oracle C (SkipC).
