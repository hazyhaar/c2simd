#!/usr/bin/env bash
set -euo pipefail
# Décret 2026-08-14 : toolchain Go 1.27 + GOEXPERIMENT=simd (voir c2simd/CLAUDE.md)
export GOTOOLCHAIN=go1.27rc3
export GOEXPERIMENT=simd
MOD="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$MOD"
OUT="spec/dogfood/testdata/cycles/20260810k_monocypher/sgoiter_out"
AMALG="spec/c_sources/upstream/monocypher/4.0.2/monocypher_amalg.c"
mkdir -p "$OUT"
EXCLUDES="Slide_init,Slide_step,Remove_l,Mod_l,Invsqrt,Lookup_add,Crypto_argon2,Crypto_elligator_key_pair,Crypto_chacha20_djb,Crypto_x25519_dirty_small,Poly_blocks,Ge_scalarmult_base,Crypto_eddsa_check_equation,Crypto_aead_write,Slide_ctx"
./bin/sgoiter -in "$AMALG" -out "$OUT/monocypher_aead_sgoiter.go" -exclude "$EXCLUDES"
sed -i 's/^package .*/package monocypher_amalg/' "$OUT/monocypher_aead_sgoiter.go"
python3 sgoiter/scripts/postprocess_monocypher_go.py "$OUT/monocypher_aead_sgoiter.go"
printf 'module monocypher_amalg\n\ngo 1.27\n' >"$OUT/go.mod"
DST="/devhoros/pkg/monocypher55"
if [[ -d "$DST" ]]; then
  sed 's/^package monocypher_amalg/package monocypher55/' \
    "$OUT/monocypher_aead_sgoiter.go" >"$DST/monocypher_aead_sgoiter.go"
  echo "  synced $DST/monocypher_aead_sgoiter.go"
  # mechanical comb tables from C (bit-exact)
  python3 sgoiter/scripts/extract_comb_tables.py \
    spec/c_sources/upstream/monocypher/4.0.2/monocypher.c \
    "$DST" monocypher55

  # hand overrides live only under DST; copy into OUT so amalg package compiles
  for h in hand_invsqrt.go hand_mod_l.go hand_slide.go hand_argon2.go hand_elligator_key_pair.go \
           hand_chacha_simd.go chacha20_simd_amd64.go chacha20_simd_fallback.go \
           hand_poly1305_simd.go \
           hand_aead_fused.go \
           hand_x25519_dirty_small.go \
           ge_scalarmult_base.go eddsa_check_equation.go \
           b_comb_low_table.go b_comb_high_table.go b_window_table.go; do
    if [[ -f "$DST/$h" ]]; then
      sed 's/^package monocypher55/package monocypher_amalg/' "$DST/$h" >"$OUT/$h"
    fi
  done
fi
echo "OK $OUT/monocypher_aead_sgoiter.go ($(wc -l <"$OUT/monocypher_aead_sgoiter.go") L)"
(cd "$OUT" && GOWORK=off go test -count=1 .)
echo "regen_monocypher_dogfood OK"
