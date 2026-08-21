# c2simd — règles du pôle transpilation (c2simd, sgoiter, pkg émis)

## Doctrine toolchain — décret 2026-08-14, opposable à toute session et tout subagent

LE BUT de l'exercice c2simd/sgoiter est la transpilation C → Go-SIMD via le package
`simd/archsimd` de **Go 1.27** (`GOEXPERIMENT=simd`). Le scalaire émis est une étape,
jamais la cible finale.

1. **Toolchain obligatoire : go1.27.0** (stable août 2026), avec
   `GOEXPERIMENT=simd` actif. `GOTOOLCHAIN=go1.27.0`. Le workspace `go.work` épingle 1.27.0.
2. **Workspace dédié : `/devhoros/c2simd/go.work`** — épingle la toolchain et couvre
   c2simd + `/devhoros/pkg/monocypher55` + `/devhoros/pkg/secretstream55`. Tout travail
   sous c2simd le prend automatiquement ; depuis les pkg, exporter
   `GOWORK=/devhoros/c2simd/go.work`.
3. **Interdit de builder/benchmarker la strate SIMD sous go1.26** : le codegen 1.26.5
   régresse ×4–5 sous le scalaire (sonde mesurée 154 MB/s vs 1 172 MB/s en 1.27rc3,
   i9-14900K AVX2). Tout chiffre de perf SIMD produit sous 1.26 est invalide.
4. **Parité bit-exact obligatoire** pour chaque kernel SIMD contre l'oracle C
   (drivers gcc des `*_c_test.go`) ET contre la variante scalaire émise.
5. **Fallback scalaire systématique** : tout kernel `archsimd` vit derrière le build tag
   `goexperiment.simd` avec garde runtime (`archsimd.X86.AVX2()` — noter que
   `RotateAllLeft` exige AVX-512, absent du poste : rotations émulées 2 shifts + or).
6. **arm64** : couvert par Go 1.27 (Neon 128-bit) pour les types 128-bit seulement ;
   les types 256/512-bit restent amd64.

## Renvois

- Autorité CUE (ARCHTIME, findings, cuegen) : `sgoiter/AGENTS.md`.
- Spec transpileur : `sgoiter/SPEC.md` (marche 6 = strate SIMD).
- Protocole Dogfooding : `sgoiter/spec/PROTOCOLE_DOGFOODING.md` (cycle canonique en 6 étapes & CUE).
- Gates : `sgoiter/scripts/gate_monocypher_suite.sh` (suite), `sgoiter/scripts/ci_check.sh`
  (G1–G3) — tous deux épinglent GOTOOLCHAIN=go1.27rc3.
- Cœur SIMD de référence : `simd_fused.go`, `simd_chacha20.go`, garde fail-loud
  `require_simd_test.go` (échec explicite si build sans `goexperiment.simd`).
