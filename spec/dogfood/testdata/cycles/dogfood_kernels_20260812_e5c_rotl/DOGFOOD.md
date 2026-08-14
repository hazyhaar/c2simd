# Dogfood post-e5c36f5 + Rotl module drop

## Highlights
- base64: string table + `dst[j:j+4]` (tail ~1.38×)
- fnv: int index + `data[i:i+8]`
- crc: one-line bit step
- blake: ~143 L densified loop
- murmur: no dead `func Rotl32` (module-level archInline)
- nacl: L32/R → RotateLeft

## Lines
   20 spec/dogfood/testdata/cycles/dogfood_kernels_20260812_e5c_rotl/kernels/strlenspn_lab.go
   22 spec/dogfood/testdata/cycles/dogfood_kernels_20260812_e5c_rotl/kernels/crc32_ieee.go
   26 spec/dogfood/testdata/cycles/dogfood_kernels_20260812_e5c_rotl/kernels/fnv1a_64.go
   30 spec/dogfood/testdata/cycles/dogfood_kernels_20260812_e5c_rotl/kernels/chacha20_qr.go
   40 spec/dogfood/testdata/cycles/dogfood_kernels_20260812_e5c_rotl/kernels/base64_simd.go
   43 spec/dogfood/testdata/cycles/dogfood_kernels_20260812_e5c_rotl/kernels/md5_transform.go
   49 spec/dogfood/testdata/cycles/dogfood_kernels_20260812_e5c_rotl/kernels/fast_xor.go
   54 spec/dogfood/testdata/cycles/dogfood_kernels_20260812_e5c_rotl/kernels/poly1305_block5.go
   57 spec/dogfood/testdata/cycles/dogfood_kernels_20260812_e5c_rotl/kernels/murmur3_x86_32.go
   68 spec/dogfood/testdata/cycles/dogfood_kernels_20260812_e5c_rotl/kernels/siphash24.go
  116 spec/dogfood/testdata/cycles/dogfood_kernels_20260812_e5c_rotl/kernels/tweetnacl_dogfood.go
  142 spec/dogfood/testdata/cycles/dogfood_kernels_20260812_e5c_rotl/kernels/blake2b_compress.go
  247 spec/dogfood/testdata/cycles/dogfood_kernels_20260812_e5c_rotl/kernels/md5_transform_full.go
  914 total
no Rotl32 defs
