# Micro-opt colocalisé — fast_xor / Vn / md5

Workdir: `/tmp/sgoiter_probebench` (NVMe0, pas `/data`).

## Changements emit
1. `dropRedundantAlignMask` — `dst[int(v &^ 7):]` → `dst[int(v):]` (boucle +8)
2. `bceIndexLoops` — Vn: `x[:n], y[:n]` + compteur int (BCE)
3. stack array `[N]T` pur **annulé** (casse appels mono `[]uint32`)

## Résultats probe (post-opt, in-process)

| lib | strate | C ns | sgo ns | ratio |
|-----|--------|-----:|-------:|------:|
| fast_xor | l1_1k | 72 | 110 | **1.5×** (avant ~3.8× process-bench) |
| fast_xor | l2_64k | 2259 | 8715 | 3.9× (LE vs native) |
| tweetnacl Vn | ver_eq | 2.1 | 8.3 | 3.9× |
| md5 | block_1k | 8.4 | 18.8 | 2.2× |

## Suite
- xor L2: unsafe word load ou NativeEndian si doctrine l’autorise
- Vn: encore du bruit BCE/const-time
- md5: noyau C tronqué (4 rounds) — gain limité
