#!/usr/bin/env bash
# ci_check.sh — gate sgoiter G1–G3 locaux
set -euo pipefail
# Décret 2026-08-14 : toolchain Go 1.27 + GOEXPERIMENT=simd (voir c2simd/CLAUDE.md)
export GOTOOLCHAIN=go1.27rc3
export GOEXPERIMENT=simd
export SGOITER_REQUIRE_SOURCES=1
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MOD="$(cd "$ROOT/.." && pwd)"
cd "$MOD"

echo "== cue vet sgoiter findings =="
cue vet ./sgoiter/spec/findings/

echo "== non-fork : pas d'import ccgo sous sgoiter =="
if grep -RIn 'modernc.org/ccgo' sgoiter --include='*.go' --include='go.mod' 2>/dev/null; then
  echo "FAIL: ccgo import under sgoiter" >&2
  exit 1
fi

echo "== go test ./sgoiter/... =="
GOWORK=off go test ./sgoiter/... -count=1

echo "== dogfood 3 kernels =="
mkdir -p /tmp/sgoiter_df
GOWORK=off go build -o /tmp/sgoiter_df/sgoiter ./sgoiter/cmd/sgoiter
for k in add mul_add xor_clear murmur3_lab fastlz_lab array_lab memmove_lab; do
  /tmp/sgoiter_df/sgoiter -in "sgoiter/testdata/c/${k}.c" -out "/tmp/sgoiter_df/${k}.go" -ir-out "/tmp/sgoiter_df/${k}.ir.json"
  dir="/tmp/sgoiter_df/build_$k"
  rm -rf "$dir" && mkdir -p "$dir"
  cp "/tmp/sgoiter_df/${k}.go" "$dir/"
  printf 'module t\n\ngo 1.26\n' >"$dir/go.mod"
  (cd "$dir" && GOWORK=off go build -o /dev/null .)
  echo "  OK $k"
done
python3 - <<'PY'
import json
m=json.load(open("/tmp/sgoiter_df/murmur3_lab.ir.json"))
names={f["name"] for f in m["funcs"]}
need={"rotl32","fmix32","murmur3_x86_32"}
assert need <= names, names
print("  harvest murmur3_lab:", sorted(names))
m=json.load(open("/tmp/sgoiter_df/fastlz_lab.ir.json"))
names={f["name"] for f in m["funcs"]}
need={"flz_hash","flz_readu32","flz_readu32_idx"}
assert need <= names, names
print("  harvest fastlz_lab:", sorted(names))
PY



echo "== reject bad_asm =="
if /tmp/sgoiter_df/sgoiter -in sgoiter/testdata/c/bad_asm.c -out /tmp/sgoiter_df/bad.go 2>/tmp/sgoiter_df/bad.err; then
  echo "FAIL: bad_asm should fail" >&2
  exit 1
fi
grep -q err_asm /tmp/sgoiter_df/bad.err

echo "== monocypher AEAD multi-bloc 1KB (si amalg présent) =="
AMALG="spec/c_sources/upstream/monocypher/4.0.2/monocypher_amalg.c"
if [[ -f "$AMALG" ]]; then
  GOWORK=off go test ./sgoiter/front/ -count=1 -timeout 180s -run TestMonoAEAD_MultiBlock_1KB
else
  echo "  skip (amalg absent)"
fi

echo "== dogfood sgoiter_out KATs =="
DOG_OUT="spec/dogfood/testdata/cycles/20260810k_monocypher/sgoiter_out"
if [[ -f "$DOG_OUT/go.mod" ]]; then
  (cd "$DOG_OUT" && GOWORK=off go test -count=1 .)
else
  echo "  skip (sgoiter_out absent)"
fi

echo "== monocypher_sgoiter parity (+ C si gcc) =="
SS_SGOI="../pkg/monocypher55"
if [[ ! -d "$SS_SGOI" ]]; then
  SS_SGOI="/devhoros/pkg/monocypher55"
fi
if [[ -d "$SS_SGOI" ]]; then
  AMALG="spec/c_sources/upstream/monocypher/4.0.2/monocypher_amalg.c"
  if [[ -f "$AMALG" ]]; then
    echo "  [ci_check] vérification de la régénération mécanique de monocypher..."
    EXCLUDES="Slide_init,Slide_step,Remove_l,Mod_l,Invsqrt,Lookup_add,Crypto_argon2,Crypto_elligator_key_pair,Crypto_chacha20_djb,Crypto_x25519_dirty_small,Poly_blocks,Ge_scalarmult_base,Crypto_eddsa_check_equation,Crypto_aead_write,Slide_ctx"
    /tmp/sgoiter_df/sgoiter -in "$AMALG" -out /tmp/sgoiter_df/monocypher_aead_fresh.go -exclude "$EXCLUDES"
    sed -i 's/^package .*/package monocypher55/' /tmp/sgoiter_df/monocypher_aead_fresh.go
    python3 sgoiter/scripts/postprocess_monocypher_go.py /tmp/sgoiter_df/monocypher_aead_fresh.go
    if ! diff -u "$SS_SGOI/monocypher_aead_sgoiter.go" /tmp/sgoiter_df/monocypher_aead_fresh.go >/tmp/sgoiter_df/mono_diff.patch; then
      echo "FAIL: monocypher_aead_sgoiter.go versionné ne correspond pas au Go émis frais par sgoiter (diff ci-dessous) :" >&2
      cat /tmp/sgoiter_df/mono_diff.patch >&2
      exit 1
    fi
    echo "  [ci_check] monocypher_aead_sgoiter.go conforme à l'émission sgoiter fraîche."
  fi
  (cd "$SS_SGOI" && GOWORK=off go test -count=1 -timeout 180s .)
else
  echo "  skip (monocypher55 absent)"
fi

echo "== libinjection KAT driver (gcc smoke, Gemini oracle tool) =="
LI_SRC="spec/c_sources/testdata/c_sources/libinjection"
LI_DRV="sgoiter/testdata/c_kat/libinjection_kat_driver.c"
if [[ -f "$LI_DRV" && -f "$LI_SRC/libinjection_sqli.c" ]] && command -v gcc >/dev/null; then
  gcc -O2 -I "$LI_SRC" -o /tmp/sgoiter_df/libinjection_kat_driver \
    "$LI_DRV" "$LI_SRC/libinjection_sqli.c"
  /tmp/sgoiter_df/libinjection_kat_driver all | grep -q '^NAME=sqli_or_1eq1'
  echo "  OK libinjection_kat_driver"
else
  echo "  skip (sources or gcc absent)"
fi

echo "== tribench: every kernel with a C oracle must be bit-exact (fail-closed) =="
GOWORK=off go build -o /tmp/sgoiter_df/sgoiter ./sgoiter/cmd/sgoiter
GOWORK=off go build -o /tmp/sgoiter_df/tribench ./sgoiter/tribench/cmd/tribench
TB_OUT="/tmp/sgoiter_df/tribench_ci"
rm -rf "$TB_OUT"
if ! /tmp/sgoiter_df/tribench -root "$MOD" -sgoiter /tmp/sgoiter_df/sgoiter -out "$TB_OUT" -skip-bench -skip-ccgo -v; then
  echo "FAIL: tribench non-match (see $TB_OUT/SUMMARY.md)" >&2
  exit 1
fi
grep -E '^- (summary|[0-9]+ kernel)' "$TB_OUT/SUMMARY.md" | sed 's/^/  /'

echo "sgoiter ci_check OK"
