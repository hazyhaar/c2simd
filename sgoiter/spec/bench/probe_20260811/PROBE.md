# Probebench stratifié — CPU / RAM / disques

Stamp: 2026-08-11T21:04:14Z

## Disques hôtes (piège classique)

Les micro-bench **ne doivent pas** écrire artefacts sur `/data` (volume bulk).

- /tmp → / (/dev/nvme0n1p2, ext4, SSD/NVMe)
- /devhoros → /devhoros (/dev/nvme1n1p1, ext4, SSD/NVMe)
- /data → /data (/dev/sda, ext4, HDD/rotational)

**Workdir CPU probes:** /tmp/sgoiter_probebench → / (/dev/nvme0n1p2, ext4, SSD/NVMe)

## I/O séquentiel 64 MiB (par montage)

| id | mount | type | write MB/s | read MB/s | err |
|----|-------|------|------------|-----------|-----|
| io_seq_64m_root | / | ext4 | 1791.1 | 7140.6 |  |
| io_seq_64m_devhoros | /devhoros | ext4 | 1734.8 | 8689.5 |  |
| io_seq_64m_data | /data | ext4 | 188.7 | 16646.1 |  |

## Probes kernel (in-process)

| lib | stratum | phase | backend | ns/op | MB/s | allocs | RSS KiB | minflt |
|-----|---------|-------|---------|-------|------|--------|---------|--------|
| fnv1a_64 | ov_empty | overhead | c_gcc_O2 | 1.3 | 0.0 | 0 | 9680 | 4 |
| fnv1a_64 | ov_empty | overhead | sgoiter | 0.5 | 0.0 | 0 | 0 | 0 |
| fnv1a_64 | tiny_16 | setup | c_gcc_O2 | 9.1 | 1680.0 | 0 | 9680 | 6 |
| fnv1a_64 | tiny_16 | setup | sgoiter | 15.5 | 984.1 | 0 | 0 | 0 |
| fnv1a_64 | l1_1k | hot_l1 | c_gcc_O2 | 941.3 | 1037.4 | 0 | 9680 | 6 |
| fnv1a_64 | l1_1k | hot_l1 | sgoiter | 1188.9 | 821.4 | 0 | 0 | 0 |
| fnv1a_64 | l1_4k | hot_l1 | c_gcc_O2 | 3255.1 | 1200.1 | 0 | 9680 | 7 |
| fnv1a_64 | l1_4k | hot_l1 | sgoiter | 3690.9 | 1058.3 | 0 | 0 | 0 |
| fnv1a_64 | l2_64k | hot_l2 | c_gcc_O2 | 60196.5 | 1038.3 | 0 | 9680 | 22 |
| fnv1a_64 | l2_64k | hot_l2 | sgoiter | 56939.3 | 1097.7 | 0 | 0 | 0 |
| fnv1a_64 | bulk_1m | bulk | c_gcc_O2 | 1097603.0 | 911.1 | 0 | 9680 | 263 |
| fnv1a_64 | bulk_1m | bulk | sgoiter | 924974.3 | 1081.1 | 1 | 0 | 0 |
| crc32_ieee | ov_empty | overhead | c_gcc_O2 | 1.0 | 0.0 | 0 | 9680 | 4 |
| crc32_ieee | ov_empty | overhead | sgoiter | 0.3 | 0.0 | 0 | 0 | 0 |
| crc32_ieee | tiny_16 | setup | c_gcc_O2 | 96.5 | 158.2 | 0 | 9680 | 6 |
| crc32_ieee | tiny_16 | setup | sgoiter | 102.2 | 149.3 | 0 | 0 | 0 |
| crc32_ieee | l1_1k | hot_l1 | c_gcc_O2 | 7535.1 | 129.6 | 0 | 9680 | 6 |
| crc32_ieee | l1_1k | hot_l1 | sgoiter | 8248.9 | 118.4 | 0 | 0 | 0 |
| crc32_ieee | l1_4k | hot_l1 | c_gcc_O2 | 31252.4 | 125.0 | 0 | 9680 | 7 |
| crc32_ieee | l1_4k | hot_l1 | sgoiter | 30198.9 | 129.4 | 0 | 0 | 0 |
| crc32_ieee | l2_64k | hot_l2 | c_gcc_O2 | 485205.7 | 128.8 | 0 | 9680 | 22 |
| crc32_ieee | l2_64k | hot_l2 | sgoiter | 515579.8 | 121.2 | 0 | 0 | 0 |
| crc32_ieee | bulk_1m | bulk | c_gcc_O2 | 7648280.6 | 130.7 | 0 | 9680 | 263 |
| crc32_ieee | bulk_1m | bulk | sgoiter | 7498436.8 | 133.4 | 1 | 0 | 0 |
| fast_xor | tail_17 | tail | c_gcc_O2 | 2.4 | 6795.2 | 0 | 9680 | 6 |
| fast_xor | tail_17 | tail | sgoiter | 4.7 | 3434.9 | 0 | 0 | 0 |
| fast_xor | align_64 | hot_l1 | c_gcc_O2 | 5.3 | 11488.3 | 0 | 9680 | 6 |
| fast_xor | align_64 | hot_l1 | sgoiter | 9.8 | 6200.8 | 0 | 0 | 0 |
| fast_xor | l1_1k | hot_l1 | c_gcc_O2 | 54.7 | 17858.2 | 0 | 9680 | 6 |
| fast_xor | l1_1k | hot_l1 | sgoiter | 131.5 | 7428.0 | 0 | 0 | 0 |
| fast_xor | l2_64k | hot_l2 | c_gcc_O2 | 2761.0 | 22636.5 | 0 | 9680 | 54 |
| fast_xor | l2_64k | hot_l2 | sgoiter | 7454.8 | 8383.8 | 0 | 0 | 0 |
| fast_xor | bulk_1m | bulk | c_gcc_O2 | 74983.9 | 13336.2 | 0 | 9680 | 777 |
| fast_xor | bulk_1m | bulk | sgoiter | 97357.5 | 10271.4 | 3 | 0 | 0 |
| siphash24 | ov_empty | overhead | c_gcc_O2 | 10.7 | 0.0 | 0 | 9680 | 4 |
| siphash24 | ov_empty | overhead | sgoiter | 7.7 | 0.0 | 0 | 0 | 0 |
| siphash24 | tiny_16 | setup | c_gcc_O2 | 17.8 | 858.5 | 0 | 9680 | 6 |
| siphash24 | tiny_16 | setup | sgoiter | 13.3 | 1151.4 | 0 | 0 | 0 |
| siphash24 | l1_1k | hot_l1 | c_gcc_O2 | 699.6 | 1395.9 | 0 | 9680 | 6 |
| siphash24 | l1_1k | hot_l1 | sgoiter | 452.8 | 2156.9 | 0 | 0 | 0 |
| siphash24 | l1_4k | hot_l1 | c_gcc_O2 | 1711.2 | 2282.7 | 0 | 9680 | 7 |
| siphash24 | l1_4k | hot_l1 | sgoiter | 1603.8 | 2435.6 | 0 | 0 | 0 |
| siphash24 | l2_64k | hot_l2 | c_gcc_O2 | 20355.9 | 3070.4 | 0 | 9680 | 22 |
| siphash24 | l2_64k | hot_l2 | sgoiter | 23742.9 | 2632.4 | 0 | 0 | 0 |
| siphash24 | bulk_1m | bulk | c_gcc_O2 | 557519.0 | 1793.7 | 0 | 9680 | 263 |
| siphash24 | bulk_1m | bulk | sgoiter | 328249.8 | 3046.5 | 1 | 0 | 0 |
| murmur3_x86_32 | ov_empty | overhead | c_gcc_O2 | 2.1 | 0.0 | 0 | 9680 | 4 |
| murmur3_x86_32 | ov_empty | overhead | sgoiter | 3.2 | 0.0 | 0 | 0 | 0 |
| murmur3_x86_32 | tiny_16 | setup | c_gcc_O2 | 6.6 | 2320.6 | 0 | 9680 | 6 |
| murmur3_x86_32 | tiny_16 | setup | sgoiter | 5.3 | 2898.1 | 0 | 0 | 0 |
| murmur3_x86_32 | l1_1k | hot_l1 | c_gcc_O2 | 223.0 | 4378.6 | 0 | 9680 | 6 |
| murmur3_x86_32 | l1_1k | hot_l1 | sgoiter | 474.7 | 2057.0 | 0 | 0 | 0 |
| murmur3_x86_32 | l1_4k | hot_l1 | c_gcc_O2 | 1253.1 | 3117.3 | 0 | 9680 | 7 |
| murmur3_x86_32 | l1_4k | hot_l1 | sgoiter | 1261.2 | 3097.2 | 0 | 0 | 0 |
| murmur3_x86_32 | l2_64k | hot_l2 | c_gcc_O2 | 16195.4 | 3859.1 | 0 | 9680 | 22 |
| murmur3_x86_32 | l2_64k | hot_l2 | sgoiter | 20551.5 | 3041.1 | 0 | 0 | 0 |
| murmur3_x86_32 | bulk_1m | bulk | c_gcc_O2 | 243568.7 | 4105.6 | 0 | 9680 | 263 |
| murmur3_x86_32 | bulk_1m | bulk | sgoiter | 455879.1 | 2193.6 | 1 | 0 | 0 |
| blake2b_compress | block_1 | block | c_gcc_O2 | 150.7 | 810.1 | 0 | 9680 | 4 |
| blake2b_compress | block_1 | block | sgoiter | 212.9 | 573.4 | 0 | 0 | 0 |
| blake2b_compress | block_1k | hot_l1 | c_gcc_O2 | 111.4 | 1095.3 | 0 | 9680 | 4 |
| blake2b_compress | block_1k | hot_l1 | sgoiter | 179.7 | 679.2 | 0 | 0 | 0 |
| blake2b_compress | block_64k | bulk | c_gcc_O2 | 177.5 | 687.9 | 0 | 9680 | 4 |
| blake2b_compress | block_64k | bulk | sgoiter | 225.6 | 541.0 | 0 | 0 | 0 |
| chacha20_qr | qr_1 | block | c_gcc_O2 | 3.1 | 4917.9 | 0 | 9680 | 4 |
| chacha20_qr | qr_1 | block | sgoiter | 3.2 | 4830.3 | 0 | 0 | 0 |
| chacha20_qr | qr_1m | hot_l1 | c_gcc_O2 | 3.2 | 4721.7 | 0 | 9680 | 4 |
| chacha20_qr | qr_1m | hot_l1 | sgoiter | 3.3 | 4670.5 | 0 | 0 | 0 |
| md5_transform | block_1 | block | c_gcc_O2 | 9.4 | 6467.5 | 0 | 9680 | 4 |
| md5_transform | block_1 | block | sgoiter | 21.7 | 2812.3 | 0 | 0 | 0 |
| md5_transform | block_1k | hot_l1 | c_gcc_O2 | 8.6 | 7062.0 | 0 | 9680 | 4 |
| md5_transform | block_1k | hot_l1 | sgoiter | 18.1 | 3363.9 | 0 | 0 | 0 |
| md5_transform | block_64k | bulk | c_gcc_O2 | 8.6 | 7115.7 | 0 | 9680 | 4 |
| md5_transform | block_64k | bulk | sgoiter | 18.5 | 3291.8 | 0 | 0 | 0 |
| poly1305_block5 | poly_1 | block | c_gcc_O2 | 9.5 | 1606.6 | 0 | 9680 | 4 |
| poly1305_block5 | poly_1 | block | sgoiter | 10.0 | 1532.5 | 0 | 0 | 0 |
| poly1305_block5 | poly_1m | hot_l1 | c_gcc_O2 | 10.2 | 1491.7 | 0 | 9680 | 4 |
| poly1305_block5 | poly_1m | hot_l1 | sgoiter | 9.4 | 1615.4 | 0 | 0 | 0 |
| base64_simd | tail_17 | tail | c_gcc_O2 | 11.0 | 1472.8 | 0 | 9680 | 6 |
| base64_simd | tail_17 | tail | sgoiter | 14.7 | 1103.7 | 0 | 0 | 0 |
| base64_simd | align_64 | hot_l1 | c_gcc_O2 | 38.2 | 1596.4 | 0 | 9680 | 6 |
| base64_simd | align_64 | hot_l1 | sgoiter | 43.3 | 1408.9 | 1 | 0 | 0 |
| base64_simd | l1_1k | hot_l1 | c_gcc_O2 | 409.1 | 2386.9 | 0 | 9680 | 6 |
| base64_simd | l1_1k | hot_l1 | sgoiter | 820.2 | 1190.6 | 1 | 0 | 0 |
| base64_simd | l2_64k | hot_l2 | c_gcc_O2 | 29448.1 | 2122.4 | 0 | 9680 | 43 |
| base64_simd | l2_64k | hot_l2 | sgoiter | 42342.2 | 1476.1 | 1 | 0 | 0 |
| base64_simd | bulk_1m | bulk | c_gcc_O2 | 490772.1 | 2037.6 | 0 | 9680 | 605 |
| base64_simd | bulk_1m | bulk | sgoiter | 514399.3 | 1944.0 | 2 | 0 | 0 |
| tweetnacl_dogfood | ver_eq | block | c_gcc_O2 | 2.8 | 5500.4 | 0 | 9680 | 4 |
| tweetnacl_dogfood | ver_eq | block | sgoiter | 14.0 | 1087.3 | 0 | 0 | 0 |
| tweetnacl_dogfood | ver_neq | block | c_gcc_O2 | 2.6 | 5808.0 | 0 | 9680 | 4 |
| tweetnacl_dogfood | ver_neq | block | sgoiter | 13.2 | 1160.1 | 0 | 0 | 0 |

## Goulets heuristiques

- **fnv1a_64/bulk_1m** [sgoiter/alloc]: 1 allocs / 1048576 B heap delta on bulk-ish stratum — escape/slice churn
- **crc32_ieee/bulk_1m** [sgoiter/alloc]: 1 allocs / 1048576 B heap delta on bulk-ish stratum — escape/slice churn
- **tweetnacl_dogfood/ver_eq** [sgoiter/cpu]: sgoiter 5.1x slower than C on ver_eq (14 vs 3 ns/op) — hot path candidate
- **siphash24/bulk_1m** [sgoiter/alloc]: 1 allocs / 1048576 B heap delta on bulk-ish stratum — escape/slice churn
- **murmur3_x86_32/bulk_1m** [sgoiter/alloc]: 1 allocs / 1048576 B heap delta on bulk-ish stratum — escape/slice churn
- **base64_simd/l2_64k** [sgoiter/alloc]: 1 allocs / 90112 B heap delta on bulk-ish stratum — escape/slice churn
- **tweetnacl_dogfood/ver_neq** [sgoiter/cpu]: sgoiter 5.0x slower than C on ver_neq (13 vs 3 ns/op) — hot path candidate
- **fast_xor/bulk_1m** [sgoiter/alloc]: 3 allocs / 3145728 B heap delta on bulk-ish stratum — escape/slice churn
- **base64_simd/bulk_1m** [sgoiter/alloc]: 2 allocs / 2449408 B heap delta on bulk-ish stratum — escape/slice churn
- **base64_simd/l1_1k** [sgoiter/alloc]: 1 allocs / 1408 B heap delta on bulk-ish stratum — escape/slice churn

## Micro-opt — ordre suggéré

1. Strates `hot_l1` / `block` avec ratio sgo/C ≥ 3.5× (CPU kernel).
2. Strates `bulk` avec allocs > 0 (échapper slices, pools).
3. Ne pas optimiser sur workdir `/data` ni mélanger IO disque et CPU.


JSON: sgoiter/spec/bench/probe_20260811/probe_report.json
