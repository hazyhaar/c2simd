# sgoiter

Transpileur **C→Go itératif** (thésaurus-first), sous-module de `c2simd`.

**Cible : transpilation C → Go-SIMD sous Go 1.27 + `GOEXPERIMENT=simd`** (`simd/archsimd`). Roadmap SIMD = marche 6 de `SPEC.md`.

## Pipeline

```text
C subset v0  →  front  →  IR  →  rules  →  emit go127  →  .go
                 ↓ fail-loud hors liste blanche
```

```bash
./sgoiter/scripts/ci_check.sh
GOWORK=off go build -o bin/sgoiter ./sgoiter/cmd/sgoiter
./bin/sgoiter -in sgoiter/testdata/c/add.c -out /tmp/add.go -ir-out /tmp/add.ir.json
```

## Banc & Statut

- Tribench : **13/13** bit-exact vs C (`libinjection_sqli` sans oracle)
- Triangle stratifié : [`spec/bench/NOTE_STRATIFIED_20260812.md`](spec/bench/NOTE_STRATIFIED_20260812.md)
- Optimisations + gains : [`spec/bench/NOTE_OPTIMS_SESSION_20260812.md`](spec/bench/NOTE_OPTIMS_SESSION_20260812.md)
- Probe : `spec/bench/probe_stratified/`

## Layout

```text
sgoiter/
  SPEC.md  README.md
  cmd/sgoiter/  front/  ir/  rules/  emit/
  spec/bench/  spec/findings/
  tribench/  sgoiterbench/  probebench/
  scripts/ci_check.sh
```

## Non-objectifs

Fork ccgo · C ISO complet · SIMD générique sans oracle · fusion sans validation formelle
