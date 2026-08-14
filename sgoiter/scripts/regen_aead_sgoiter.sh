#!/usr/bin/env bash
# Regen monocypher AEAD → secretstream55 (wrapper around regen_monocypher_dogfood).
# Requires fix and_ones_u64: 0xffffffff is NOT u64 identity (2026-08-13).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
exec bash "$ROOT/regen_monocypher_dogfood.sh"
