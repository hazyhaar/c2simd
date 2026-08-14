#!/usr/bin/env bash
# scan_ccgo_up_residuals.sh — C7 inventaire des formes __ccgo_up restantes
# Usage: ./scripts/scan_ccgo_up_residuals.sh [dir…]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
if [[ $# -eq 0 ]]; then
  set -- "$ROOT/spec/dogfood/testdata/cycles" "$ROOT/spec/c_sources/opt" "$ROOT/spec/c_sources/raw"
fi

python3 - "$ROOT" "$@" <<'PY'
import collections, pathlib, re, sys, json
root = pathlib.Path(sys.argv[1])
dirs = [pathlib.Path(p) for p in sys.argv[2:]]
forms = collections.Counter()
by_file = []
pat = re.compile(r'.{0,40}__ccgo_up\([^)]{0,120}\).{0,20}')
for d in dirs:
    if not d.exists():
        continue
    for p in d.rglob('*.go'):
        if 'build_raw' in p.parts or 'build_opt' in p.parts:
            continue
        try:
            t = p.read_text(errors='ignore')
        except Exception:
            continue
        n = t.count('__ccgo_up')
        if n == 0:
            continue
        # exclude func def line from residual "call" count
        call_n = len(re.findall(r'(?<!func )__ccgo_up\s*\(', t))
        shapes = collections.Counter()
        for m in pat.finditer(t):
            s = m.group(0)
            if 'func __ccgo_up' in s:
                shape = 'func_def'
            elif re.search(r'\*\*\(\*\*\[', s):
                shape = 'array_double_star'
            elif re.search(r'\*\*\(\*\*\w', s):
                shape = 'scalar_double_star'
            elif re.search(r'\(\*\*\[', s):
                shape = 'array_cast'
            else:
                shape = 'other'
            shapes[shape] += 1
            forms[shape] += 1
        by_file.append({
            "file": str(p.relative_to(root)),
            "total_token": n,
            "callish": call_n,
            "shapes": dict(shapes),
        })

by_file.sort(key=lambda x: -x["callish"])
report = {
    "forms_global": dict(forms),
    "files_with_ccgo_up": len(by_file),
    "top": by_file[:40],
}
out = root / "spec/dogfood/ccgo_up_residuals.json"
out.write_text(json.dumps(report, indent=2, ensure_ascii=False) + "\n")
print(json.dumps({"forms_global": report["forms_global"], "files": report["files_with_ccgo_up"], "report": str(out)}, indent=2))
PY
