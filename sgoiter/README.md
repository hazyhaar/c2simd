# sgoiter

Transpileur **C→Go itératif** (thésaurus-first), domicile sous le module `c2simd`.

**Cible (décret 2026-08-14) : transpilation C → Go-SIMD sous Go 1.27 + `GOEXPERIMENT=simd`**
(`simd/archsimd`). Toolchain épinglée `go1.27rc3` — workspace dédié `/devhoros/c2simd/go.work`,
règles complètes dans `/devhoros/c2simd/CLAUDE.md`, roadmap SIMD = marche 6 de `SPEC.md`.

## HPM55

| | |
|--|--|
| **Projet** | `019fec65-22c3-735a-aa9a-753b89ffa9a9` |
| **Parent** | c2simd `019fd633-6735-705c-8c57-e09608dca298` |
| **Slug** | `sgoiter` |

## Pipeline

```text
C subset v0  →  front  →  IR  →  rules  →  emit go127  →  .go
                 ↓ fail-loud hors liste blanche
```

```bash
cd /devhoros/c2simd
./sgoiter/scripts/ci_check.sh
GOWORK=off go build -o bin/sgoiter ./sgoiter/cmd/sgoiter
./bin/sgoiter -in sgoiter/testdata/c/add.c -out /tmp/add.go -ir-out /tmp/add.ir.json
```

## Banc & statut (sol 2026-08-13)

- Tribench : **13/13** bit-exact vs C (`libinjection_sqli` sans oracle)
- Triangle stratifié : [`spec/bench/NOTE_STRATIFIED_20260812.md`](spec/bench/NOTE_STRATIFIED_20260812.md)
- Optimisations + gains : [`spec/bench/NOTE_OPTIMS_SESSION_20260812.md`](spec/bench/NOTE_OPTIMS_SESSION_20260812.md)
- Probe : `spec/bench/probe_stratified/`

## Documentation & Pilotage

| Fichier | Rôle |
|---------|------|
| [`TODO_NEXT.md`](TODO_NEXT.md) | Synthèse qualité et règles de forme émission (P2/P3) |
| [`TODO_EXTRA_LIBS.md`](TODO_EXTRA_LIBS.md) | Extension vers librairies C tierces (cJSON, yyjson, utf8, stb) |
| [`TODO_TRIBENCH_FINDINGS.md`](TODO_TRIBENCH_FINDINGS.md) | Constats factuels et oracles opposables du banc Tribench |

## Layout

```text
sgoiter/
  SPEC.md  README.md  TODO_*.md
  cmd/sgoiter/  front/  ir/  rules/  emit/
  spec/bench/  spec/findings/
  tribench/  sgoiterbench/  probebench/
  scripts/ci_check.sh
```

## Goals HPM55

| # | goal_id | Statut |
|---|---------|--------|
| G1 SPEC + findings | `019fec69-e7cd-…` | rendu |
| G2 IR + emit | `019fec69-e7d4-…` | rendu |
| G3 règles + dogfood | `019fec69-e7d9-…` | rendu |
| G4 front C | `019fec69-e7de-…` | rendu |

Projet HPM55 **Fini** (G1–G4) ; suite = qualité/perf/extra sans nouveaux goals tant que non forgés.

## Non-objectifs

Fork ccgo · C ISO · SIMD générique · merge LLM sans oracle
