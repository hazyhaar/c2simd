#!/usr/bin/env bash
# dogfood_cycle.sh — cycle produit c2simd v2 :
#   C source → ccgo (raw) → c2simd-gen (opt) → métriques + (opt) go build
# Usage:
#   ./scripts/dogfood_cycle.sh              # tous les .c de testdata
#   ./scripts/dogfood_cycle.sh md5_transform # un seul kernel (basename sans .c)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CSRC_DIR="$ROOT/spec/c_sources/testdata/c_sources"
STAMP="${DOGFOOD_STAMP:-$(date -u +%Y%m%d)}"
OUT_ROOT="$ROOT/spec/dogfood/testdata/cycles/$STAMP"
GEN="$ROOT/bin/c2simd-gen"
CCGO="${CCGO:-$(command -v ccgo)}"

mkdir -p "$ROOT/bin" "$OUT_ROOT"
if [[ ! -x "$GEN" ]]; then
  (cd "$ROOT" && GOWORK=off go build -o bin/c2simd-gen ./cmd/c2simd-gen)
fi
if [[ -z "$CCGO" || ! -x "$CCGO" ]]; then
  echo "ccgo introuvable (PATH ou CCGO=)" >&2
  exit 1
fi

count() { # count regex in file (grep -c exits 1 when 0 matches)
  local f="$1" pat="$2" n
  n=$(grep -cE "$pat" "$f" 2>/dev/null || true)
  # collapse accidental multi-line
  n=$(echo "$n" | awk 'NR==1{print; exit}')
  echo "${n:-0}"
}

run_one() {
  local base="$1"
  local cfile="$CSRC_DIR/${base}.c"
  if [[ ! -f "$cfile" ]]; then
    echo "SKIP missing $cfile" >&2
    return 1
  fi
  local dir="$OUT_ROOT/$base"
  mkdir -p "$dir"
  cp "$cfile" "$dir/src.c"

  local pkg="df_${base}"
  local raw="$dir/raw.go"
  local opt="$dir/opt.go"
  local metrics="$dir/metrics.json"

  echo "=== dogfood $base ==="
  # ccgo v4 : --package-name= --o=
  (cd "$ROOT" && "$CCGO" --package-name="$pkg" -o "$raw" "$dir/src.c")
  # capi satellite éventuel à côté de -o : le déplacer dans dir
  local capi_side
  capi_side="$(dirname "$raw")/capi_linux_amd64.go"
  # parfois écrit cwd
  if [[ -f "$ROOT/capi_linux_amd64.go" ]]; then
    mv "$ROOT/capi_linux_amd64.go" "$dir/capi_linux_amd64.go"
  elif [[ -f "$capi_side" ]]; then
    : # already in place if -o dir
  fi
  # si ccgo a écrit capi next to raw with different path
  find "$(dirname "$raw")" -maxdepth 1 -name 'capi_*.go' -exec true \; 2>/dev/null || true

  "$GEN" -in "$raw" -out "$opt"

  local raw_rotl opt_rotl raw_bits opt_bits raw_tls_param opt_tls_param raw_lines opt_lines
  local raw_ccgo opt_ccgo
  raw_rotl=$(count "$raw" '\brotl32\s*\(|\brotl64\s*\(')
  opt_rotl=$(count "$opt" '\brotl32\s*\(|\brotl64\s*\(')
  raw_bits=$(count "$raw" 'bits\.RotateLeft')
  opt_bits=$(count "$opt" 'bits\.RotateLeft')
  # approx : defs still taking tls as first param
  raw_tls_param=$(count "$raw" 'func [A-Za-z0-9_]+\(tls \*libc\.TLS')
  opt_tls_param=$(count "$opt" 'func [A-Za-z0-9_]+\(tls \*libc\.TLS')
  raw_ccgo=$(count "$raw" '__ccgo_up')
  opt_ccgo=$(count "$opt" '__ccgo_up')
  raw_lines=$(wc -l <"$raw")
  opt_lines=$(wc -l <"$opt")

  # build both as packages under module (copy into buildable path)
  local build_raw="$dir/build_raw"
  local build_opt="$dir/build_opt"
  rm -rf "$build_raw" "$build_opt"
  mkdir -p "$build_raw" "$build_opt"
  cp "$raw" "$build_raw/"
  # include capi if present for build
  if compgen -G "$dir/capi_*.go" > /dev/null; then
    cp "$dir"/capi_*.go "$build_raw/" 2>/dev/null || true
    cp "$dir"/capi_*.go "$build_opt/" 2>/dev/null || true
  fi
  cp "$opt" "$build_opt/"
  # rewrite package path isolation: keep package name from file
  local build_ok_raw=0 build_ok_opt=0
  if (cd "$ROOT" && GOWORK=off go build -o /dev/null "./spec/dogfood/testdata/cycles/$STAMP/$base/build_raw/" 2>"$dir/build_raw.err"); then
    build_ok_raw=1
    rm -f "$dir/build_raw.err"
  fi
  if (cd "$ROOT" && GOWORK=off go build -o /dev/null "./spec/dogfood/testdata/cycles/$STAMP/$base/build_opt/" 2>"$dir/build_opt.err"); then
    build_ok_opt=1
    rm -f "$dir/build_opt.err"
  fi

  cat >"$metrics" <<EOF
{
  "kernel": "$base",
  "stamp": "$STAMP",
  "src_c": "spec/c_sources/testdata/c_sources/${base}.c",
  "ccgo": "$CCGO",
  "raw_lines": $raw_lines,
  "opt_lines": $opt_lines,
  "raw_rotl_calls": $raw_rotl,
  "opt_rotl_calls": $opt_rotl,
  "raw_bits_rotate": $raw_bits,
  "opt_bits_rotate": $opt_bits,
  "raw_tls_first_param_funcs": $raw_tls_param,
  "opt_tls_first_param_funcs": $opt_tls_param,
  "tls_params_elided": $((raw_tls_param - opt_tls_param)),
  "bits_rotate_gained": $((opt_bits - raw_bits)),
  "raw_ccgo_up": $raw_ccgo,
  "opt_ccgo_up": $opt_ccgo,
  "ccgo_up_removed": $((raw_ccgo - opt_ccgo)),
  "build_raw_ok": $build_ok_raw,
  "build_opt_ok": $build_ok_opt
}
EOF
  echo "metrics → $metrics"
  cat "$metrics"
  if [[ "$build_ok_raw" -ne 1 || "$build_ok_opt" -ne 1 ]]; then
    echo "WARN build fail raw=$build_ok_raw opt=$build_ok_opt (see *.err)" >&2
  fi
  # D1 : seuil résiduel __ccgo_up (défaut 0 pour kernels lab simples ; gros parseurs : DOGFOOD_CCGO_UP_MAX)
  local max_ccgo="${DOGFOOD_CCGO_UP_MAX:-0}"
  if [[ "$opt_ccgo" -gt "$max_ccgo" ]]; then
    echo "FAIL $base: opt_ccgo_up=$opt_ccgo > max=$max_ccgo (export DOGFOOD_CCGO_UP_MAX pour waiver)" >&2
    return 2
  fi

  # --- REVIEW auto (agent doit relire opt.go hot path à chaque passe) ---
  local review="$dir/REVIEW.md"
  {
    echo "# REVIEW dogfood — $base ($STAMP)"
    echo
    echo "## Métriques"
    echo '```json'
    cat "$metrics"
    echo '```'
    echo
    echo "## Diff structurel raw → opt (grep signatures / rotate / tls / unsafe)"
    echo '```'
    echo "--- raw funcs ---"
    grep -nE '^func ' "$raw" | head -40 || true
    echo "--- opt funcs ---"
    grep -nE '^func ' "$opt" | head -40 || true
    echo "--- opt RotateLeft ---"
    grep -nE 'bits\.RotateLeft' "$opt" | head -20 || echo "(none)"
    echo "--- opt remaining rotl/>> << patterns (sample) ---"
    grep -nE 'rotl32|>>[^=].*<<|<<[^=].*>>' "$opt" | head -15 || echo "(none)"
    echo '```'
    echo
    echo "## Hot path opt (première fonction non-__ccgo, extrait 80 lignes)"
    echo '```go'
    # extract first real func body-ish
    awk '
      /^func __ccgo/ {skip=1}
      /^func / && !/^func __ccgo/ {if(!found){found=1; skip=0}}
      found && !skip {print; n++; if(n>=80) exit}
    ' "$opt"
    echo '```'
    echo
    echo "## Checklist relecture agent (obligatoire)"
    echo "- [ ] Sémantique rotate : ROL/ROR → RotateLeft correct (W-N)"
    echo "- [ ] ABI tls : exportés gardent tls ; unexported T0 strip OK"
    echo "- [ ] Pas de Go pointer passé en uintptr (callers futurs)"
    echo "- [ ] Imports morts absents"
    echo "- [ ] Motifs ratés documentés (finding proposed si récurrent)"
    echo "- [ ] __ccgo_up résiduel ≤ seuil (opt_ccgo_up=$opt_ccgo max=${DOGFOOD_CCGO_UP_MAX:-0})"
    echo "- [ ] build_ok raw+opt"
    echo
    echo "## Verdict agent"
    echo "_À remplir après Read de opt.go / raw.go._"
  } >"$review"
  echo "review → $review"
}

FILTER="${1:-}"
if [[ -n "$FILTER" ]]; then
  run_one "$FILTER"
else
  shopt -s nullglob
  for c in "$CSRC_DIR"/*.c; do
    run_one "$(basename "$c" .c)"
  done
fi

# index
{
  echo "{"
  echo "  \"stamp\": \"$STAMP\","
  echo "  \"cycles\": ["
  first=1
  for m in "$OUT_ROOT"/*/metrics.json; do
    [[ -f "$m" ]] || continue
    if [[ $first -eq 0 ]]; then echo ","; fi
    first=0
    sed 's/^/    /' "$m" | tr -d '\n'
    echo
  done
  echo "  ]"
  echo "}"
} >"$OUT_ROOT/index.json"

echo "INDEX $OUT_ROOT/index.json"
