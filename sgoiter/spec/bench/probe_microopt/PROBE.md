# Probebench stratifié — CPU / RAM / disques

Stamp: 2026-08-12T07:17:47Z

## Disques hôtes (piège classique)

Les micro-bench **ne doivent pas** écrire artefacts sur `/data` (volume bulk).

- /tmp → / (/dev/nvme0n1p2, ext4, SSD/NVMe)
- /devhoros → /devhoros (/dev/nvme1n1p1, ext4, SSD/NVMe)
- /data → /data (/dev/sda, ext4, HDD/rotational)

**Workdir CPU probes:** /tmp/sgoiter_probebench → / (/dev/nvme0n1p2, ext4, SSD/NVMe)

## Probes kernel (in-process) — triangle C / sgoiter / ccgo

| lib | stratum | phase | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | sgo MB/s |
|-----|---------|-------|------|--------|---------|-------|--------|----------|----------|
| fnv1a_64 | ov_empty | overhead | 0.8 | 2.1 | 0.5 | 2.56× | 0.62× | 4.12× | 0.0 |
| fnv1a_64 | tiny_16 | setup | 8.2 | 7.5 | 13.0 | 0.91× | 1.59× | 0.58× | 2038.2 |
| fnv1a_64 | l1_1k | hot_l1 | 978.2 | 1017.4 | 974.7 | 1.04× | 1.00× | 1.04× | 959.9 |
| fnv1a_64 | l1_4k | hot_l1 | 3461.8 | 3961.1 | 3493.5 | 1.14× | 1.01× | 1.13× | 986.2 |
| fnv1a_64 | l2_64k | hot_l2 | 63526.0 | 73458.6 | 53075.2 | 1.16× | 0.84× | 1.38× | 850.8 |
| fnv1a_64 | bulk_1m | bulk | 976468.8 | 923262.5 | 1192620.7 | 0.95× | 1.22× | 0.77× | 1083.1 |
| crc32_ieee | ov_empty | overhead | 0.9 | 2.1 | 2.0 | 2.34× | 2.24× | 1.05× | 0.0 |
| crc32_ieee | tiny_16 | setup | 98.6 | 105.1 | 115.2 | 1.07× | 1.17× | 0.91× | 145.2 |
| crc32_ieee | l1_1k | hot_l1 | 6847.0 | 7235.5 | 7163.7 | 1.06× | 1.05× | 1.01× | 135.0 |
| crc32_ieee | l1_4k | hot_l1 | 28575.6 | 29406.7 | 28455.1 | 1.03× | 1.00× | 1.03× | 132.8 |
| crc32_ieee | l2_64k | hot_l2 | 455073.7 | 450831.3 | 462523.6 | 0.99× | 1.02× | 0.97× | 138.6 |
| crc32_ieee | bulk_1m | bulk | 7342837.1 | 7490604.1 | 7461067.7 | 1.02× | 1.02× | 1.00× | 133.5 |
| fast_xor | tail_17 | tail | 2.6 | 3.2 | 4.5 | 1.23× | 1.73× | 0.71× | 5106.3 |
| fast_xor | align_64 | hot_l1 | 5.1 | 5.2 | 6.9 | 1.02× | 1.34× | 0.77× | 11640.4 |
| fast_xor | l1_1k | hot_l1 | 64.9 | 33.5 | 79.5 | 0.52× | 1.22× | 0.42× | 29183.5 |
| fast_xor | l2_64k | hot_l2 | 4329.0 | 2947.3 | 3581.5 | 0.68× | 0.83× | 0.82× | 21206.0 |
| fast_xor | bulk_1m | bulk | 66468.1 | 50168.7 | 68481.5 | 0.75× | 1.03× | 0.73× | 19932.8 |
| siphash24 | ov_empty | overhead | 9.9 | 8.0 | 8.9 | 0.80× | 0.90× | 0.89× | 0.0 |
| siphash24 | tiny_16 | setup | 13.8 | 15.5 | 15.9 | 1.12× | 1.15× | 0.98× | 984.8 |
| siphash24 | l1_1k | hot_l1 | 385.6 | 322.6 | 358.3 | 0.84× | 0.93× | 0.90× | 3027.5 |
| siphash24 | l1_4k | hot_l1 | 1264.9 | 1371.0 | 1531.1 | 1.08× | 1.21× | 0.90× | 2849.3 |
| siphash24 | l2_64k | hot_l2 | 21581.2 | 21985.4 | 24946.3 | 1.02× | 1.16× | 0.88× | 2842.8 |
| siphash24 | bulk_1m | bulk | 340186.6 | 369216.8 | 421483.2 | 1.09× | 1.24× | 0.88× | 2708.4 |
| murmur3_x86_32 | ov_empty | overhead | 2.3 | 3.1 | 2.4 | 1.34× | 1.02× | 1.31× | 0.0 |
| murmur3_x86_32 | tiny_16 | setup | 6.6 | 7.2 | 7.8 | 1.09× | 1.18× | 0.92× | 2132.2 |
| murmur3_x86_32 | l1_1k | hot_l1 | 238.2 | 314.2 | 347.7 | 1.32× | 1.46× | 0.90× | 3107.9 |
| murmur3_x86_32 | l1_4k | hot_l1 | 1131.9 | 1256.4 | 1170.9 | 1.11× | 1.03× | 1.07× | 3109.0 |
| murmur3_x86_32 | l2_64k | hot_l2 | 14803.9 | 18118.4 | 18554.6 | 1.22× | 1.25× | 0.98× | 3449.5 |
| murmur3_x86_32 | bulk_1m | bulk | 231772.2 | 350272.8 | 312903.7 | 1.51× | 1.35× | 1.12× | 2854.9 |
| blake2b_compress | block_1 | block | 117.6 | 145.5 | 219.7 | 1.24× | 1.87× | 0.66× | 838.9 |
| blake2b_compress | block_1k | hot_l1 | 164.5 | 202.4 | 230.5 | 1.23× | 1.40× | 0.88× | 603.2 |
| blake2b_compress | block_64k | bulk | 166.5 | 204.5 | 274.2 | 1.23× | 1.65× | 0.75× | 596.8 |
| chacha20_qr | qr_1 | block | 2.8 | 3.0 | 7.8 | 1.07× | 2.76× | 0.39× | 5028.1 |
| chacha20_qr | qr_1m | hot_l1 | 2.9 | 2.9 | 6.8 | 1.02× | 2.39× | 0.43× | 5250.5 |
| md5_transform | block_1 | block | 9.2 | 11.3 | 12.2 | 1.23× | 1.32× | 0.93× | 5396.3 |
| md5_transform | block_1k | hot_l1 | 9.4 | 12.8 | 12.1 | 1.36× | 1.28× | 1.06× | 4784.6 |
| md5_transform | block_64k | bulk | 8.6 | 12.8 | 12.5 | 1.49× | 1.46× | 1.02× | 4776.8 |
| poly1305_block5 | poly_1 | block | 10.2 | 8.2 | 15.5 | 0.81× | 1.52× | 0.53× | 1851.1 |
| poly1305_block5 | poly_1m | hot_l1 | 10.7 | 10.6 | 19.5 | 0.99× | 1.82× | 0.54× | 1441.1 |
| base64_simd | tail_17 | tail | 10.4 | 18.8 | 13.5 | 1.81× | 1.30× | 1.39× | 863.1 |
| base64_simd | align_64 | hot_l1 | 37.9 | 49.9 | 41.4 | 1.32× | 1.09× | 1.20× | 1222.4 |
| base64_simd | l1_1k | hot_l1 | 568.5 | 765.7 | 650.6 | 1.35× | 1.14× | 1.18× | 1275.4 |
| base64_simd | l2_64k | hot_l2 | 32779.8 | 38495.8 | 34136.3 | 1.17× | 1.04× | 1.13× | 1623.6 |
| base64_simd | bulk_1m | bulk | 488406.3 | 655788.0 | 531330.7 | 1.34× | 1.09× | 1.23× | 1524.9 |
| tweetnacl_dogfood | ver_eq | block | 2.8 | 2.3 | 11.5 | 0.82× | 4.06× | 0.20× | 6584.3 |
| tweetnacl_dogfood | ver_neq | block | 2.6 | 2.3 | 10.8 | 0.89× | 4.14× | 0.22× | 6552.3 |
| strlenspn_lab | ov_empty | overhead | 1.2 | 0.3 | 1.7 | 0.25× | 1.49× | 0.17× | 0.0 |
| strlenspn_lab | tiny_16 | setup | 4.2 | 0.9 | 5.6 | 0.22× | 1.34× | 0.17× | 16356.4 |
| strlenspn_lab | l1_1k | hot_l1 | 4.3 | 1.0 | 5.3 | 0.24× | 1.21× | 0.20× | 951129.3 |
| strlenspn_lab | l1_4k | hot_l1 | 4.0 | 0.9 | 5.3 | 0.24× | 1.33× | 0.18× | 4169835.7 |
| strlenspn_lab | l2_64k | hot_l2 | 4.0 | 1.0 | 5.2 | 0.25× | 1.30× | 0.19× | 63759245.1 |
| strlenspn_lab | bulk_1m | bulk | 3.5 | 1.5 | 6.2 | 0.43× | 1.79× | 0.24× | 666666666.7 |
| md5_transform_full | block_1 | block | 68.9 | 113.8 | 115.2 | 1.65× | 1.67× | 0.99× | 536.1 |
| md5_transform_full | block_1k | hot_l1 | 80.4 | 117.3 | 115.1 | 1.46× | 1.43× | 1.02× | 520.4 |
| md5_transform_full | block_64k | bulk | 80.2 | 111.1 | 131.3 | 1.39× | 1.64× | 0.85× | 549.3 |

## Goulets heuristiques

- **base64_simd/tail_17** [sgoiter/cpu]: goulet modéré: sgoiter 1.81x C on tail_17 (18.8 vs 10.4 ns/op)

## Micro-opt — ordre suggéré

1. Strates `hot_l1` / `block` avec ratio sgo/C ≥ 3.5× (CPU kernel).
2. Strates `bulk` avec allocs > 0 (échapper slices, pools).
3. Ne pas optimiser sur workdir `/data` ni mélanger IO disque et CPU.
