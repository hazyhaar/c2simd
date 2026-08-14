#!/usr/bin/env bash
# Gate unique P0+ monocypher suite (TODO_MONOCYPHER_SUITE_10 point 3).
set -euo pipefail
export GOWORK=off
# Décret 2026-08-14 : toolchain Go 1.27 + GOEXPERIMENT=simd (voir c2simd/CLAUDE.md)
export GOTOOLCHAIN=go1.27rc3
export GOEXPERIMENT=simd
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

echo "== monocypher55 package tests =="
(cd ../pkg/monocypher55 && go test -count=1 -timeout 300s .)

echo "== secretstream55 make ci =="
(cd ../pkg/secretstream55 && make ci)

echo "== aead_sgoiter tag =="
(cd ../pkg/secretstream55 && go test -count=1 -tags aead_sgoiter ./...)

echo "== regen dogfood =="
./sgoiter/scripts/regen_monocypher_dogfood.sh

echo "== stub scan =="
./sgoiter/scripts/mono_stub_scan.sh

echo "== ci_check =="
./sgoiter/scripts/ci_check.sh

echo "gate_monocypher_suite OK"
