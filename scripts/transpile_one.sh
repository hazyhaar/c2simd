#!/usr/bin/env bash
# transpile_one.sh — boucle bornée c2simd v2 : Go ccgo-like → c2simd-gen → go test slice
# Usage: ./scripts/transpile_one.sh <in.go> <out.go> [package_test_path]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
IN="${1:?usage: transpile_one.sh <in.go> <out.go> [test_pkg]}"
OUT="${2:?}"
TEST_PKG="${3:-}"

GEN="$ROOT/bin/c2simd-gen"
if [[ ! -x "$GEN" ]]; then
  mkdir -p "$ROOT/bin"
  (cd "$ROOT" && GOWORK=off go build -o bin/c2simd-gen ./cmd/c2simd-gen)
fi

"$GEN" -in "$IN" -out "$OUT"
echo "OK gen: $IN -> $OUT"

if [[ -n "$TEST_PKG" ]]; then
  (cd "$ROOT" && GOWORK=off go test "$TEST_PKG" -count=1)
fi
