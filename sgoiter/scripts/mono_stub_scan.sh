#!/usr/bin/env bash
set -euo pipefail
F="${1:-/devhoros/pkg/monocypher55/monocypher_aead_sgoiter.go}"
n=$(grep -c 'was not harvested' "$F" || true)
echo "stubs=$n file=$F"
if [[ "$n" -gt 0 ]]; then
  grep 'was not harvested' "$F" || true
  exit 1
fi
exit 0
