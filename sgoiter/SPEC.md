# sgoiter — SPEC v0

Version : **0.9.0** (overrides perf + IR folds + probe triangle C/sgo/ccgo)  
Module : `code.hazyhaar.fr/devhoros/c2simd/sgoiter`  
HPM55 : `019fec65-22c3-735a-aa9a-753b89ffa9a9`  

**Nuit / CI :** `make -C sgoiter smoke|nightly|probe-b3` · checklist `TODO_NIGHT.md` · emit `-mode safe|kernel`

**Doctrine toolchain (décret 2026-08-14) :** la cible de tout l'exercice de transpilation est
**Go 1.27 + `GOEXPERIMENT=simd`** (`simd/archsimd`). Toolchain épinglée `go1.27rc3` jusqu'à la
sortie stable (workspace dédié `/devhoros/c2simd/go.work`, exports dans les scripts de gate).
Oracle fondateur : la même sonde ChaCha20 `archsimd` mesure 154 MB/s sous go1.26.5 et
1 172 MB/s sous go1.27rc3 (i9-14900K, AVX2) — la strate SIMD n'est viable qu'en 1.27.
Le code émis scalaire doit builder et passer ses KAT sous cette toolchain, expériment actif.

## 1. Subset C v0.2 (liste blanche + harvest partiel)

**Harvest partiel :** un `.c` réel peut livrer un **sous-ensemble** de fonctions (les autres sont `Skipped`). Zéro fonction → `err_empty`.

| Construit | Notes |
|-----------|--------|
| Fonctions | `static? T name(params) { body }` |
| Types | `int`, `int8_t`…, `uint8_t`/`u8`, `uint32_t`/`u32`, `uint64_t`/`u64`/`size_t`, `void` |
| Qualifiers strip | `const`, `static`, `unsigned` (normalise) |
| Preprocess | lignes `#…` **ignorées** (plus `err_preprocess`) |
| Params / locals | scalaires ; decl `T x;` / `T x = e;` |
| Expr | littéraux, idents, `+ - * & \| ^ << >>`, casts `(T)e`, appels `f(a,b)` |
| Stmts | `return`, `=`, composés `+= -= *= ^= &= \|= <<= >>=` |
| Commentaires | `//` et `/* */` |

## 2. Refus fail-loud (hors-subset)

| Motif | Code |
|-------|------|
| `asm` / `__asm__` | `err_asm` |
| `va_list` / `, ...` | `err_varargs` |
| `goto` | `err_goto` |
| pointeurs / `[]` / `->` / `struct` | `err_memory` |
| `for`/`while`/`switch` (v0.2) | `err_parse` (contrôle de flux) |
| `float` / `double` | `err_float` |
| rien d’harvestable | `err_empty` |
| parse local | `err_parse` |

## 1bis. v0.3 ajouté

| Construit | Notes |
|-----------|--------|
| Pointeurs scalaires | `T *p`, `const` strip ; `[]T` à l’emit |
| Index | `p[i]` → Load ; scale 1/4/8 ; baseOff pour `(T*)(p+off)` |
| Load u32/u64 | `encoding/binary.LittleEndian` |
| `for (init;cond;post)` | Stmt SKFor |
| `switch` + cases + fallthrough | Stmt SKSwitch |
| `/` `%` `ULL`/`LL` | arith + parseIntLit |
| unary `-` | en `int` (bornes de boucle C) |

## 1ter. v0.4 ajouté

| Item | Détail |
|------|--------|
| `#define` object-like | eval entier multi-passe (`HASH_MASK` etc.) |
| `#define F(c)` simple | LIKELY/UNLIKELY → `(c)` |
| `void*` | `[]byte` |
| switch fallthrough | `fallthrough` Go |
| load byte | widen `uint32` (promotions C) |
| KAT Murmur | vs gcc -O0 (`murmur_kat_test.go`) |

## 1quater. v0.6 ajouté

| Item | Détail |
|------|--------|
| `++i` / `i++` prefix+suffix | stmt + clause for |
| `#define FOR(i,n)…` multi-args | placeholders `@ARGn@` (pas `$` regexp) |
| `for` corps sans `{}` | stmt unique jusqu'à `;` |
| `if` / `else` / `!cond` | Stmt SKIf |
| `h[i] &=` compound indexé | load/binop/store |
| FillStubs `...any` | build partiel valué/void |
| strip `#if/#else/#endif` | branche else préférée si présente |
| keywords package | `if` → `if_` |
| globals string / j++ / char / offSlot / LE | dogfood 11/12 |

## 1quinquies. v0.7 monocypher struct

| Item | Finding |
|------|---------|
| typedef struct + `ctx->f` / `(ctx->f)[i]` | F-sgoiter-struct-arrow-monocypher |
| findOp ne matche pas `>` dans `->` ni `<` dans `<<` | F-sgoiter-findop-arrow-gt |
| unsigned/volatile normalize | F-sgoiter-unsigned-strip |
| amalg sans `#ifdef __cplusplus` | F-sgoiter-cpp-ifdef-amalg |
| blacklist stubs `crypto_aead_*` | F-sgoiter-aead-stub-blacklist |

## 1sexies. v0.8 monocypher AEAD (landed)

| Item | Détail |
|------|--------|
| `ptr += N` sur `[]byte` | emit `ptr = ptr[N:]` (Sym `ptr_add` + `ptr_alias`) — multi-bloc ChaCha |
| binop avant unary | `~x + 1` / `gap()` |
| isParam | garde noms paramètres contre alias |
| crypto_wipe | `for _i := range buf { buf[_i] = 0 }` |
| hoist function-scope | vars multi-blocs Go |
| Harvest AEAD amalg 4.0.2 | lock/unlock/chacha/poly **0 stub** aead |
| KAT | self 36 B+1 KB ; **bit-exact ccgo** ; **vs gcc -O0** monocypher.c |
| Dual package | `pkg/secretstream55/internal/monocypher_sgoiter` (oracle) |
| CI | `ci_check` MultiBlock 1 KB + dogfood `sgoiter_out` + parity package |
| Régén dogfood | `sgoiter/scripts/regen_monocypher_dogfood.sh` |

Todos : `TODO_NEXT.md` (forme) · `TODO_NIGHT.md` (perf) · `TODO_EXTRA_LIBS.md` · secretstream V2 → `pkg/secretstream55/TODO_V2_SGOITER_LIBSODIUM.md` · **Monocypher complet sgoiter** → `TODO_MONOCYPHER_FULL.md`.

## 2bis. Marches

1. ~~Tables 2D (blake2b)~~  
2. ~~Tribench bit-exact~~ — **13/13** vs gcc -O2 (2026-08-13) ; libinjection no-oracle  
3. Secretstream V2 — **libsodium-compatible** + backend AEAD sgoiter (c2simd = perf V1 dual) ; plan 2j dans `TODO_V2_SGOITER_LIBSODIUM.md`  
4. ~~Qualité emit I1–I3 + Q10 + densify 2026-08-12~~ — ROT plié, drop overrides, CRC 1 ligne  
5. Reste : perf (`TODO_NIGHT`) · forme P2/P3 (`TODO_NEXT`) · dogfood extra (`TODO_EXTRA_LIBS`)  
6. Strate SIMD (LE BUT) — kernels chauds (ChaCha20, Poly1305, AEAD) portés sur `simd/archsimd`
   (Go 1.27, `GOEXPERIMENT=simd`) :
   - Validés en hands dans `pkg/monocypher55/` (`chacha20_simd_amd64.go`, `hand_poly1305_simd.go`, `hand_aead_fused.go`, avec garde `archsimd.X86.AVX2()` + fallback scalaire) :
     - **Patron 1 :** Projections vectorielles directes sans `StoreArray` sur pile (`F-sgoiter-simd-direct-projections`, débit ChaCha 2,30 Go/s).
     - **Patron 2 :** Boucle fusionnée 1-pass micro-entrelacée par blocs de 256 octets (`F-sgoiter-aead-fused-streaming-256b`, débit AEAD 1,87 Go/s).
     - **Patron 3 :** Réduction d'arité Poly1305 en limbs 64-bit saturés 6 mults/bloc (`F-sgoiter-poly1305-radix64-reduction`, débit Poly 3,93 Go/s).
   - Parité bit-exact vs oracle C obligatoire pour chaque variante ; règles d'émissions ciblées dans `sgoiter/emit` en perspective.

## 2ter. Banc tribench

- Backends : `c_gcc_O0`, `c_gcc_O2` (oracle), `sgoiter`, `ccgo`
- Gate CI : `ci_check.sh` lance tribench fail-closed (score **13/13** compared)
- Métriques sgoiter : `code_lines`, `identity_assigns`, `var_count`, `int_cast_index`, `rot_left_calls`
- Cycles preuve gitignorés : `spec/dogfood/testdata/cycles/tribench_*/`




## 3. Non-objectifs

- Fork `modernc.org/ccgo`
- C ISO / libc complète
- Auto-vectorisation générique de toute boucle C (la voie SIMD est un OBJECTIF — marche 6 —
  mais par kernels ciblés et règles de motif, jamais par une passe générique aveugle)
- Merge LLM sans golden + test

## 4. IR

Nœuds minimaux : `Module`, `Func`, `Block`, `Instr`  
Opcodes : `Nop`, `Const`, `Mov`, `Add`, `Sub`, `Mul`, `And`, `Or`, `Xor`, `Shl`, `Shr`, `Return`, `Phi` (réservé), `Call` (réservé).

Invariant : **même IR sérialisé JSON ⇒ même emit go127** (test golden round-trip).

## 5. Émission profils

| Profil | Rôle |
|--------|------|
| `go127` | Go idiomatique subset (package + funcs) |
| `compat` | réservé (oracle / dual-emit) |

## 6. Thésaurus

- Findings : `spec/findings/*.cue` (`cue vet`)
- Règles : package `rules` — RuleDef + golden in→want IR
- Boucle : finding → règle + test → land | reject

## 7. Oracle non-fork

Aucun `import` Go de `modernc.org/ccgo` sous `sgoiter/`.
