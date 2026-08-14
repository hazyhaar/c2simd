# Probebench stratifié — CPU / RAM / disques

Stamp: 2026-08-13T07:11:24Z

## Disques hôtes (piège classique)

Les micro-bench **ne doivent pas** écrire artefacts sur `/data` (volume bulk).

- /tmp → / (/dev/nvme0n1p2, ext4, SSD/NVMe)
- /devhoros → /devhoros (/dev/nvme1n1p1, ext4, SSD/NVMe)
- /data → /data (/dev/sda, ext4, HDD/rotational)

**Workdir CPU probes:** /tmp/sgoiter_probebench → / (/dev/nvme0n1p2, ext4, SSD/NVMe)

## Probes kernel (in-process) — triangle C / sgoiter / ccgo

| lib | stratum | phase | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | sgo MB/s |
|-----|---------|-------|------|--------|---------|-------|--------|----------|----------|
| fnv1a_64 | ov_empty | overhead | 0.9 | 2.0 | 0.5 | 2.34× | 0.52× | 4.48× | 0.0 |
| fnv1a_64 | tiny_16 | setup | 5.3 | 7.0 | 8.5 | 1.33× | 1.59× | 0.83× | 2164.7 |
| fnv1a_64 | l1_1k | hot_l1 | 730.0 | 726.6 | 741.5 | 1.00× | 1.02× | 0.98× | 1343.9 |
| fnv1a_64 | l1_4k | hot_l1 | 2887.7 | 3017.7 | 2963.1 | 1.05× | 1.03× | 1.02× | 1294.4 |
| fnv1a_64 | l2_64k | hot_l2 | 46470.4 | 46865.4 | 47204.3 | 1.01× | 1.02× | 0.99× | 1333.6 |
| fnv1a_64 | bulk_1m | bulk | 743904.6 | 754803.7 | 756477.2 | 1.01× | 1.02× | 1.00× | 1324.8 |
| crc32_ieee | ov_empty | overhead | 0.9 | 0.3 | 1.8 | 0.31× | 2.00× | 0.16× | 0.0 |
| crc32_ieee | tiny_16 | setup | 91.3 | 91.5 | 92.8 | 1.00× | 1.02× | 0.99× | 166.8 |
| crc32_ieee | l1_1k | hot_l1 | 6000.9 | 6158.9 | 6019.6 | 1.03× | 1.00× | 1.02× | 158.6 |
| crc32_ieee | l1_4k | hot_l1 | 24123.0 | 24454.0 | 24225.4 | 1.01× | 1.00× | 1.01× | 159.7 |
| crc32_ieee | l2_64k | hot_l2 | 385499.0 | 397080.2 | 389680.2 | 1.03× | 1.01× | 1.02× | 157.4 |
| crc32_ieee | bulk_1m | bulk | 6149115.5 | 6287015.6 | 6173029.1 | 1.02× | 1.00× | 1.02× | 159.1 |
| fast_xor | tail_17 | tail | 2.3 | 3.4 | 5.2 | 1.50× | 2.30× | 0.65× | 4739.9 |
| fast_xor | align_64 | hot_l1 | 5.0 | 5.0 | 6.2 | 1.00× | 1.23× | 0.82× | 12143.8 |
| fast_xor | l1_1k | hot_l1 | 64.1 | 32.7 | 77.4 | 0.51× | 1.21× | 0.42× | 29890.0 |
| fast_xor | l2_64k | hot_l2 | 2512.9 | 2302.4 | 3406.7 | 0.92× | 1.36× | 0.68× | 27146.1 |
| fast_xor | bulk_1m | bulk | 51440.7 | 43953.0 | 63646.4 | 0.85× | 1.24× | 0.69× | 22751.6 |
| siphash24 | ov_empty | overhead | 6.5 | 7.3 | 7.5 | 1.13× | 1.16× | 0.97× | 0.0 |
| siphash24 | tiny_16 | setup | 14.2 | 12.1 | 12.2 | 0.85× | 0.86× | 0.99× | 1264.7 |
| siphash24 | l1_1k | hot_l1 | 276.4 | 299.3 | 317.3 | 1.08× | 1.15× | 0.94× | 3262.5 |
| siphash24 | l1_4k | hot_l1 | 1070.4 | 1179.0 | 1222.5 | 1.10× | 1.14× | 0.96× | 3313.2 |
| siphash24 | l2_64k | hot_l2 | 16438.1 | 18143.7 | 19285.9 | 1.10× | 1.17× | 0.94× | 3444.7 |
| siphash24 | bulk_1m | bulk | 268963.6 | 292737.7 | 308191.2 | 1.09× | 1.15× | 0.95× | 3416.0 |
| murmur3_x86_32 | ov_empty | overhead | 1.2 | 3.0 | 2.5 | 2.39× | 2.00× | 1.20× | 0.0 |
| murmur3_x86_32 | tiny_16 | setup | 4.3 | 4.7 | 5.6 | 1.08× | 1.30× | 0.83× | 3277.4 |
| murmur3_x86_32 | l1_1k | hot_l1 | 199.3 | 257.1 | 258.2 | 1.29× | 1.30× | 1.00× | 3799.0 |
| murmur3_x86_32 | l1_4k | hot_l1 | 806.0 | 1028.2 | 1003.2 | 1.28× | 1.24× | 1.02× | 3799.1 |
| murmur3_x86_32 | l2_64k | hot_l2 | 12825.1 | 16409.9 | 16595.3 | 1.28× | 1.29× | 0.99× | 3808.7 |
| murmur3_x86_32 | bulk_1m | bulk | 209602.4 | 261199.7 | 257075.9 | 1.25× | 1.23× | 1.02× | 3828.5 |
| blake2b_compress | block_1 | block | 98.6 | 140.9 | 168.2 | 1.43× | 1.70× | 0.84× | 866.3 |
| blake2b_compress | block_1k | hot_l1 | 108.0 | 148.3 | 176.1 | 1.37× | 1.63× | 0.84× | 823.4 |
| blake2b_compress | block_64k | bulk | 161.5 | 199.3 | 244.5 | 1.23× | 1.51× | 0.82× | 612.4 |
| chacha20_qr | qr_1 | block | 2.3 | 2.4 | 5.4 | 1.02× | 2.30× | 0.44× | 6363.6 |
| chacha20_qr | qr_1m | hot_l1 | 2.7 | 2.5 | 5.9 | 0.93× | 2.18× | 0.43× | 6062.2 |
| md5_transform | block_1 | block | 8.4 | 11.0 | 10.7 | 1.32× | 1.28× | 1.03× | 5539.7 |
| md5_transform | block_1k | hot_l1 | 8.3 | 11.2 | 11.8 | 1.34× | 1.42× | 0.95× | 5442.6 |
| md5_transform | block_64k | bulk | 8.4 | 11.3 | 11.9 | 1.35× | 1.42× | 0.95× | 5386.3 |
| poly1305_block5 | poly_1 | block | 8.3 | 9.1 | 14.5 | 1.09× | 1.74× | 0.63× | 1682.4 |
| poly1305_block5 | poly_1m | hot_l1 | 10.1 | 9.1 | 16.1 | 0.91× | 1.60× | 0.57× | 1670.2 |
| base64_simd | tail_17 | tail | 7.7 | 11.0 | 10.1 | 1.43× | 1.30× | 1.10× | 1467.7 |
| base64_simd | align_64 | hot_l1 | 26.8 | 31.1 | 34.9 | 1.16× | 1.30× | 0.89× | 1963.5 |
| base64_simd | l1_1k | hot_l1 | 377.6 | 456.7 | 487.3 | 1.21× | 1.29× | 0.94× | 2138.3 |
| base64_simd | l2_64k | hot_l2 | 24341.5 | 27408.1 | 30483.2 | 1.13× | 1.25× | 0.90× | 2280.3 |
| base64_simd | bulk_1m | bulk | 436366.3 | 461893.9 | 495091.9 | 1.06× | 1.13× | 0.93× | 2165.0 |
| tweetnacl_dogfood | ver_eq | block | 2.5 | 1.9 | 7.8 | 0.77× | 3.13× | 0.25× | 7868.8 |
| tweetnacl_dogfood | ver_neq | block | 2.2 | 2.2 | 8.0 | 0.98× | 3.56× | 0.28× | 6936.7 |
| strlenspn_lab | ov_empty | overhead | 1.1 | 0.3 | 1.6 | 0.24× | 1.40× | 0.17× | 0.0 |
| strlenspn_lab | tiny_16 | setup | 3.6 | 0.9 | 3.1 | 0.25× | 0.85× | 0.30× | 16657.1 |
| strlenspn_lab | l1_1k | hot_l1 | 3.8 | 1.0 | 5.1 | 0.27× | 1.36× | 0.20× | 975416.4 |
| strlenspn_lab | l1_4k | hot_l1 | 3.8 | 0.9 | 5.4 | 0.24× | 1.42× | 0.17× | 4276369.8 |
| strlenspn_lab | l2_64k | hot_l2 | 3.0 | 1.0 | 5.0 | 0.33× | 1.69× | 0.19× | 64532782.7 |
| strlenspn_lab | bulk_1m | bulk | 3.3 | 1.5 | 5.8 | 0.44× | 1.74× | 0.25× | 684931506.8 |
| md5_transform_full | block_1 | block | 62.8 | 91.2 | 89.0 | 1.45× | 1.42× | 1.02× | 669.5 |
| md5_transform_full | block_1k | hot_l1 | 78.5 | 112.0 | 112.2 | 1.43× | 1.43× | 1.00× | 545.0 |
| md5_transform_full | block_64k | bulk | 78.4 | 109.4 | 112.0 | 1.40× | 1.43× | 0.98× | 557.7 |

## Micro-opt — ordre suggéré

1. Strates `hot_l1` / `block` avec ratio sgo/C ≥ 3.5× (CPU kernel).
2. Strates `bulk` avec allocs > 0 (échapper slices, pools).
3. Ne pas optimiser sur workdir `/data` ni mélanger IO disque et CPU.
