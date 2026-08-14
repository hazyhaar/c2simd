#!/usr/bin/env bash
# bench_append.sh — bench raw/opt (goexperiment.simd) → append bench_history.jsonl
# Usage: ./scripts/bench_append.sh [label]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
LABEL="${1:-$(date -u +%Y%m%dT%H%M%SZ)}"
GO="${GO_BIN:-$(command -v go1.27rc1 || command -v go)}"
OUT="${BENCH_HISTORY:-$ROOT/bench_history.jsonl}"
TMP=$(mktemp)

echo "bench with $GO label=$LABEL"
GOEXPERIMENT=simd GOWORK=off "$GO" test ./spec/c_sources/ \
  -bench='Benchmark(SipHash24|BLAKE2b|MD5_Raw|MD5_AST|FastXOR)_' \
  -benchtime=300ms -count=1 2>"$TMP.err" | tee "$TMP"

# parse ns/op lines into one JSON object
python3 - "$LABEL" "$OUT" "$TMP" <<'PY'
import json, re, sys, datetime
label, out_path, bench_path = sys.argv[1], sys.argv[2], sys.argv[3]
text = open(bench_path).read()
rows = {}
for m in re.finditer(r'^(Benchmark\S+)\s+\d+\s+([\d.]+)\s+ns/op(?:\s+([\d.]+)\s+MB/s)?', text, re.M):
    rows[m.group(1)] = {"ns_op": float(m.group(2)), "mb_s": float(m.group(3)) if m.group(3) else None}
entry = {
    "ts": datetime.datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ"),
    "label": label,
    "benches": rows,
}
with open(out_path, "a") as f:
    f.write(json.dumps(entry, ensure_ascii=False) + "\n")
print("appended", out_path, "keys", len(rows))
if not rows:
    sys.exit(2)
PY
rm -f "$TMP" "$TMP.err"
