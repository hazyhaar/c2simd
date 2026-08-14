#!/usr/bin/env bash
# regenerate_opt.sh — raw/ → opt/ via c2simd-gen v2, package renommé opt_*
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GEN="$ROOT/bin/c2simd-gen"
RAW="$ROOT/spec/c_sources/raw"
OPT="$ROOT/spec/c_sources/opt"

mkdir -p "$ROOT/bin"
if [[ ! -x "$GEN" ]]; then
  (cd "$ROOT" && GOWORK=off go build -o bin/c2simd-gen ./cmd/c2simd-gen)
fi

kernels=(blake2b_compress fast_xor md5_transform siphash24)
for k in "${kernels[@]}"; do
  in="$RAW/$k/${k}.go"
  out="$OPT/$k/${k}.go"
  tmp="$(mktemp)"
  echo "regen $k"
  "$GEN" -in "$in" -out "$tmp"
  # package raw_FOO → opt_FOO
  sed -E 's/^package raw_/package opt_/' "$tmp" >"$out"
  rm -f "$tmp"
  # gofmt
  (cd "$ROOT" && GOWORK=off gofmt -w "$out")
  echo "  → $out ($(wc -l <"$out") lines)"
done
echo "OK regenerate_opt"
