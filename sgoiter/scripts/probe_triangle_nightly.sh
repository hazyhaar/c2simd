#!/usr/bin/env bash
# Nightly: full C/sgoiter/ccgo triangle + validate both backends.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
export GOWORK=off
./sgoiter/scripts/rebuild_bench.sh
go test ./sgoiter/... -count=1
./bin/tribench -root "$ROOT" -sgoiter ./bin/sgoiter -skip-ccgo -skip-bench
OUT="$ROOT/sgoiter/spec/bench/probe_microopt"
CCGO="$(command -v ccgo || true)"
if [[ -z "$CCGO" ]]; then
  echo "ccgo not in PATH" >&2
  exit 1
fi
./bin/probebench -root "$ROOT" -sgoiter ./bin/sgoiter -ccgo "$CCGO" \
  -work /tmp/sgoiter_probebench -out "$OUT" -skip-io
./bin/probebench -validate "$OUT/probe_report.json" -require-ccgo
echo "probe_triangle_nightly OK → $OUT"
