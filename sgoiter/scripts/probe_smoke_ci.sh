#!/usr/bin/env bash
# CI smoke: rebuild, unit tests, tribench 11/11, short probe + validate.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
export GOWORK=off
export PATH="/home/cl-ment/sdk/go1.27rc3/bin:/home/cl-ment/go/bin:$PATH"
./sgoiter/scripts/rebuild_bench.sh
go test ./sgoiter/... -count=1
./bin/tribench -root "$ROOT" -sgoiter ./bin/sgoiter -skip-ccgo -skip-bench
OUT="${TMPDIR:-/tmp}/sgoiter_probe_ci_$$"
mkdir -p "$OUT"
./bin/probebench -root "$ROOT" -sgoiter ./bin/sgoiter \
  -work /tmp/sgoiter_probebench -out "$OUT" \
  -only fnv1a_64,blake2b_compress,fast_xor -skip-io -skip-ccgo
./bin/probebench -validate "$OUT/probe_report.json"
echo "probe_smoke_ci OK"
