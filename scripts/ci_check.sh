#!/usr/bin/env bash
# ci_check.sh — gate locale c2simd (sans horosvec).
# Exit 0 = vert. Usage: ./scripts/ci_check.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "== cue vet findings =="
cue vet ./spec/findings/

echo "== go test astmatch + c_sources KAT + kat =="
GOWORK=off go test ./internal/astmatch/ ./spec/c_sources/ ./kat/ -count=1

echo "== build c2simd-gen =="
GOWORK=off go build -o bin/c2simd-gen ./cmd/c2simd-gen

echo "== opt/ : zéro __ccgo_up résiduel =="
fail=0
for f in spec/c_sources/opt/*/*.go; do
  n=$(grep -c '__ccgo_up' "$f" 2>/dev/null || true)
  n=${n:-0}
  if [[ "$n" -ne 0 ]]; then
    echo "FAIL $f still has $n __ccgo_up" >&2
    fail=1
  fi
done
if [[ "$fail" -ne 0 ]]; then
  exit 1
fi

echo "ci_check OK"
