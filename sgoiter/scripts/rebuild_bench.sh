#!/usr/bin/env bash
# Rebuild sgoiter + probebench + tribench at repo c2simd root.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
export GOWORK=off
mkdir -p bin
go build -o bin/sgoiter ./sgoiter/cmd/sgoiter
go build -o bin/probebench ./sgoiter/cmd/probebench
go build -o bin/tribench ./sgoiter/tribench/cmd/tribench
echo "OK $ROOT/bin/{sgoiter,probebench,tribench}"
