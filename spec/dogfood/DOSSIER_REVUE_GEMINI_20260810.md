# Dossier de revue globale — c2simd transpile / dogfood (2026-08-10)

**Destinataire :** Gemini (œil global)  
**Module :** `code.hazyhaar.fr/devhoros/c2simd`  
**VERSION produit pipeline :** `2.1.0` (post-audit : `__ccgo_up` landed + cycle h)  
**Poste mesure :** Intel Core i9-14900K, Linux amd64  
**Outils :** ccgo/v4.34.6, `c2simd-gen`, go1.26.5 / go1.27rc1 `GOEXPERIMENT=simd`  
**Politique :** CGO=0, pas de `.s`, pas de corpus exploit/red-team (lab adversarial + OSS permissif uniquement)

---

## 1. Objectif des sessions

1. Fermer la boucle **C → ccgo → c2simd-gen → go build → KAT → bench** (dogfood produit, pas seulement des règles inventées).
2. Thésauriser les leçons en `spec/findings/*.cue` (schéma `#Finding`).
3. Rendre `ArchtimeRulesTable` **honnête** (rewrite vs handwrite_pointer vs declared).
4. Interdiction opposable : règle générique « boucle → simd » (peer review Q2).
5. Domaines extrêmes légitimes : crypto/hash lab, adversarial patterns, scraping/parse/compress/AST.

**Hors scope dogfood :** SimSIMD port, horosvec fuse fp16 (autre chantier), exploits.

---

## 2. Architecture livrée (c2simd v2)

### 2.1 Pipeline

```
C source
  → ccgo (--package-name=… -o raw.go [-I …])
  → c2simd-gen -in raw.go -out opt.go   # internal/astmatch.TransformAST
  → go build / KAT / bench
  → metrics.json + REVIEW.md + finding?
```

### 2.2 Scripts

| Script | Rôle |
|--------|------|
| `scripts/dogfood_cycle.sh` | Cycle 1 kernel testdata + metrics + REVIEW.md |
| `scripts/transpile_one.sh` | gen seul |
| `scripts/regenerate_opt.sh` | raw/→opt/ commités (package `opt_*`) |

### 2.3 `ArchtimeRulesTable` (Kind)

| Symbol | Kind | Effet réel |
|--------|------|------------|
| `rotl32` | rewrite | appels → `bits.RotateLeft32` + DeadCode déf. |
| `L32` | rewrite | tweetnacl ; cast `u32(bits.RotateLeft32(uint32(x),…))` + DeadCode |
| `load32_le` / `store32_le` | rewrite | corps → unsafe `*uint32` |
| `crypto_wipe` | declared | pas encore de passe |
| `poly_blocks` / `chacha20_rounds` | handwrite_pointer | pointe vers `simd_*.go`, **pas** gen |

### 2.4 Passes structurelles (hors Symbol table)

- Motifs ROL/ROR constants (`|` ou `^`) → `bits.RotateLeft{32,64}`
- Élision `tls` T0 **unexported only** + **point fixe ≤8** (callers)
- `dropUnusedImports` (libc mort après T0)
- `x + uintptr(-N)` → `x - uintptr(N)`
- `iqlibc.__builtin_foo` → `libc.Xfoo`
- Fold partiel `libc.*FromInt32` littéraux non négatifs
- Rename `crypto_aead_lock` → exporté

### 2.5 Tests

- `internal/astmatch` : ~17 cas table-driven (rotl, L32, ROR, T0, ABI export, uintptr-neg, iqlibc, anti boucle→simd)
- `spec/c_sources` KAT raw≡opt (4 kernels commités)
- `kat/` AEAD produit

---

## 3. Chronologie des cycles dogfood

| Stamp | Contenu | Faits saillants |
|-------|---------|-----------------|
| **20260810** | md5, siphash, blake2b, xor, base64 | md5 +4 RotateLeft ; siphash +48 ; blake2b 0 (ROTR manquant) ; bug libc unused |
| **20260810b** | blake2b re-run | ROTR64 landed → **32×** RotateLeft64 |
| **20260810c** | chacha_qr, crc32, fnv, poly1305_block5 lab | chacha +4 ROL ; poly confirme hand-write |
| **20260810d** | re-run + T0 point fixe | chacha tls 1→**2** ; relecture hot path |
| **20260810e** | tweetnacl DL + murmur3 | L32 rule + cast u32=ulong ; murmur +3 |
| **20260810f** | adv_* lab + libinjection | AVOID punning/stack/var-ROL ; libinj 40k L |
| **20260810g** | cJSON, lz4, tiny-regex, mpc | fixette uintptr-neg + iqlibc ; extrême parse/compress |
| **20260810h** | re-gen g post-`__ccgo_up` | **0** `__ccgo_up` ; tiny/cjson/lz4/mpc build OK |

Rapports détaillés :  
`spec/dogfood/testdata/cycles/20260810{,d,e,f,g}/REPORT.md`

---

## 4. Catalogue des findings (`spec/findings/`)

Schéma : `schema.cue` — levier `c_source|ast_rule|handwrite` ; statut `proposed|landed|rejected|codified`.

| ID | Statut | Essence |
|----|--------|---------|
| F-20260810-q2-generic-simd | codified | Interdiction boucle→simd générique (peer review Q2) |
| F-20260810-rotl32-bits | landed | rotl32 → bits.RotateLeft32 |
| F-20260810-load32-le | landed | load/store32_le unsafe body |
| F-20260810-tls-t0 | landed | élision tls T0 (base) |
| F-20260810-tls-t0-exported-abi | landed | ne pas stripper exportés (ABI bench/KAT) |
| F-20260810-tls-t0-fixed-point | landed | point fixe callers (chacha double_round) |
| F-20260810-unused-libc-import | landed | dropUnusedImports |
| F-20260810-rotr64-gap | landed | ROR/XOR blake2b → RotateLeft(W-N) |
| F-20260810-tweetnacl-l32 | landed | L32 + cast uint32/u32 LP64 |
| F-20260810-uintptr-neg-iqlibc | landed | uintptr(-N) + iqlibc builtins |
| F-20260810-ccgo-pitfalls-research | codified | ABI Xmalloc, TLS/goroutine, vet#45, union#46, va_list#43 |
| F-20260810-ccgo-up-goulot | **landed** | `*(**T)(__ccgo_up(E))` → `(*T)(unsafe.Pointer(E))` ; 0 résiduel kernels h |
| F-20260810-uintptr-neg-second-pass | **landed** | 2e tour ADD après capture dans unsafe.Pointer |
| F-20260810-avoid-patterns-adversarial | codified | AVOID punning/align, var-ROL, stack 4k, data-heavy |

Validation : `cue vet ./spec/findings/`

---

## 5. Métriques consolidées (cycles)

### 5.1 Kernels « rotate-rich »

| Kernel | bits.Rotate gagnés | Notes |
|--------|-------------------:|-------|
| siphash24 | 48 | motif ROL constant |
| blake2b (post-ROTR) | 32 | ROTR64 via ^ |
| md5_transform | 4 | FF macro partial |
| chacha20_qr | 4 | QR only (double appelle) |
| murmur3_x86_32 | 3 | rotl local |
| tweetnacl L32/core | 4 | après règle L32 |

### 5.2 Kernels « ABI / volume »

| Kernel | L Go approx | __ccgo_up | tls raw→opt | build |
|--------|------------:|----------:|-------------|-------|
| tiny_regex | 944 | 75 | 18→8 | OK post-fix |
| cjson | 3440 | 266 | 112→81 | OK |
| lz4 | 3084 | 151 | 87→85 | OK post-fix |
| mpc | 6010 | 427 | 276→257 | OK |
| libinjection_sqli | 40444 | 305 | 60→41 | OK |
| tweetnacl | 2787 | (élevé) | 62→46 | OK post-L32 |

### 5.3 Bench packages commités (multi_bench, go1.27rc1+simd, ordres de grandeur)

| Bench | Observation |
|-------|-------------|
| SipHash raw→opt | souvent **+5–8 %** (bruit selon run) |
| MD5 | ~+4–5 % ou bruit |
| BLAKE2b | **~0 %** même après 32 RotateLeft (goulot tls/__ccgo_up) |
| FastXOR | ~0–9 % (mem-bound) |
| KAT raw≡opt | **PASS** stable |
| kat/ AEAD | **PASS** |

---

## 6. Checklist AVOID (opposable post-ccgo)

1. **Punning non aligné** `*(uint32_t*)p` — survit en `__ccgo_up` ; gen ne sécurise pas l’alignement.
2. **ROL distance variable** — pas de rewrite `bits.*`.
3. **Gros buffers stack C** (`ws[4096]`) → stack Go — AVOID services multi-goroutine.
4. **Data-heavy** (tables 10k+ lignes header) — bloat ccgo (issue #46) ; 40k L libinjection.
5. **Pointeurs Go natifs** vers API ccgo — interdit (README Memory ABI) ; Xmalloc/tls.Alloc only.
6. **va_list imbriqué** — bug ABI ccgo #43 ouvert.
7. **iqlibc / __builtin_*** sans rewrite — build cassé (fixé pour memmove).
8. **uintptr(-1)** arith pointeur — overflow Go (fixé → soustraction).
9. **typedef u32 = unsigned long** (tweetnacl LP64) — RotateLeft32 exige cast.
10. **Règle générique boucle→simd** — interdite (Q2).

---

## 7. Pièges ccgo recherchés (sources externes)

- pkg.go.dev `modernc.org/ccgo/v4` README Memory ABI  
- GitLab cznic/ccgo : #43 va_list, #45 vet unsafe, #46 union bloat, #11 volatile  
- `modernc.org/libc` : 1 TLS / goroutine, pas concurrent  

Finding : `F-20260810-ccgo-pitfalls-research`

---

## 8. Bugs produit découverts **uniquement** en dogfood (pas en unit test isolé)

| Bug | Symptôme | Fix |
|-----|----------|-----|
| libc import mort | build fail après T0 total | dropUnusedImports |
| T0 exportés | ABI multi_bench cassée | !IsExported |
| T0 single-pass | double_round garde tls | point fixe |
| ROTR64 manquant | blake2b 0 rotate | matchRotateBinary XOR/ROR |
| L32 + u32 ulong | 0 rotate / type error | règle L32 + cast |
| uintptr(-1) | constant overflows | ADD→SUB |
| iqlibc.memmove | undefined | → libc.Xmemmove |

---

## 9. Ce qui N’est PAS le gen (hand-write)

- `simd_chacha20.go`, `simd_poly1305_*.go`, `simd_fused.go` — produit AEAD oraclé KAT  
- Gains Go 1.27 archsimd lab (`c2simd-noncrypto-lab`) — hors pipeline AST  
- Peer review annonçait `rules_gen.go` / CUE thésaurus étendu — **non livré** ; livré = slice Go + findings CUE  

---

## 10. Corpus C disponible

### testdata (sources)

`spec/c_sources/testdata/c_sources/` — lab + DL attribués (`ATTRIBUTION_DOWNLOADED.md`)

- Historique : md5, siphash, blake2b, xor, base64  
- Lab : chacha_qr, crc32, fnv, poly1305_block5, murmur3, adv_*  
- DL : tweetnacl(+h), libinjection/*, (cycles g : cJSON/lz4/re/mpc en cycles seulement)

### cycles générés

`spec/dogfood/testdata/cycles/20260810*/**/{src.c,raw.go,opt.go,metrics.json,REVIEW.md?}`

### opt/ commités régénérés

`spec/c_sources/opt/{blake2b_compress,fast_xor,md5_transform,siphash24}/`

---

## 11. Questions ouvertes pour Gemini

1. **Priorité next rewrite** : `__ccgo_up` load/store scalaires → `*(*T)(unsafe.Pointer(p))` (F-ccgo-up-goulot) — ROI vs risque sémantique ?
2. **T1 tls** call-graph formel vs point fixe actuel — suffisant ?
3. **Variable-distance rotate** : faut-il une règle `bits.RotateLeft32(x, int(k&31))` avec preuve k ?
4. **Embarquer cJSON/lz4/mpc** en KAT CI — budget CI / flaky ?
5. **rules_gen.go** encore utile si findings CUE + table Go < 15 rewrite ?
6. **Article** « patterns extrêmes transpile » : angle parse/compress/AST vs crypto — lequel tranche mieux le récit ARCHTIME ?
7. Faut-il un **oracle différentiel C natif** (gcc -O0) systématique en plus de raw≡opt ?

---

## 12. Commandes de reprise rapide

```bash
cd /devhoros/c2simd
cue vet ./spec/findings/
GOWORK=off go test ./internal/astmatch/ ./spec/c_sources/ ./kat/ -count=1
DOGFOOD_STAMP=manual ./scripts/dogfood_cycle.sh md5_transform
./scripts/regenerate_opt.sh
# extrême déjà en cycles :
ls spec/dogfood/testdata/cycles/20260810g/
```

---

## 13. Index fichiers clés

| Chemin | Rôle |
|--------|------|
| `internal/astmatch/astmatch.go` | TransformAST |
| `internal/astmatch/rules.go` | ArchtimeRulesTable v2 |
| `internal/astmatch/astmatch_test.go` | gardes mécaniques |
| `cmd/c2simd-gen/main.go` | CLI |
| `spec/findings/` | thésaurus |
| `spec/dogfood/` | cycles + ce dossier |
| `spec/c2simd_transpiler_2026_peer_review.md` | doctrine Q1–Q5 |
| `spec/bench_matrix.cue` / `bench_history.json` | perf crypto produit |
| `VERSION` | 2.0.0 |

---

*Fin du dossier — prêt pour adjudication Gemini.*
