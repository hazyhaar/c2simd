# Dogfood C fixtures (tribench / probebench)

| File | Notes |
|------|--------|
| `md5_transform.c` | **4× FF only** (not full 64-step MD5) — intentional reduced kernel |
| `md5_transform_full.c` | Full 64-step MD5 transform (catalog `md5_transform_full`) |
| `blake2b_compress.c` | Full 12-round compress |
| `siphash24.c` | v1 IV = `0x646f72616d617461` (fixture-specific, not classic dorandom) |
| `crc32_ieee.c` | Bit-wise poly, no table |
| `tweetnacl_dogfood.c` | Surface `crypto_verify_16` → `vn` |
| `strlenspn_lab.c` | Minimal strspn-like, accept=hel, subset-C (triangle oracle) |

Do not interpret reduced `md5_transform` ratios as full-MD5 performance.
