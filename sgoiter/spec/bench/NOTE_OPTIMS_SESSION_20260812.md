# Note — optimisations session 2026-08-12 (sgoiter)

**Fenêtre :** session `ses_013ba8bb6ffeOWFomoaQUz0GL4`  
**Commits :** `e28dd1e` … `423a712` (chaîne sgoiter du jour)  
**Oracle perf :** probebench triangle C gcc-O2 / sgoiter / ccgo  
**Oracle correctitude :** tribench bit-exact **13/13** vs C

Mesures finales (stamp probe stratifié `2026-08-12T20:23:46Z`) sauf mention contraire. Les ratios sont **sgo/C** (plus bas = mieux). Les gains « avant → après » reprennent les probes de la session quand disponibles ; sinon état final seulement.

---

## 1. Carte des optimisations

| # | Optimisation | Lieu | Mécanisme | Gain / effet mesuré |
|---|--------------|------|-----------|---------------------|
| 1 | **xor SliceData + mots 16 B / 8 B** | `emit/overrides.go` | Pointeurs `unsafe.SliceData`, boucles `i+16` / `i+8`, plus de reslices par mot | **l1_1k ~0,67×**, bulk **~0,66×** vs C (sgo gagne) ; médiane sgo/ccgo **0,63×** |
| 2 | **xor queue switch fallthrough 0–7** | idem | Remplace boucle octet pour le reste après 8 B | tail_17 ramené vers **~1,5×** (final **1,53×** — seul goulet ≥1,5) ; sans ça tail était ~2–4× selon itérations |
| 3 | **N6 `unroll_const_trip_load`** | `rules/unroll_const_trip.go` | Unroll boucles `for i<N` corps load simple N≤16 | Support folds table / const-trip ; base pour absorb murmur/fnv |
| 4 | **Drop override murmur** | IR + `maybeRewriteMurmurLoop` | Boucle d’induction négative → `j` forward + LE loads | Absorb IR ; l1 final **~1,36×** ; tiny **1,40×** ; bit-exact |
| 5 | **Drop override fnv** | IR + `maybePostUnrollHashes` | Corps IR + post-pass ×8 | Puis #6 |
| 6 | **fnv int index + subslice BCE** | `emit/post_unroll.go` | `n:=int(len_)`, `i int`, `b:=data[i:i+8]` | tiny **1,28×**, l1 **1,11×**, bulk **~1,02×** ; plus de `int(v4)` par octet |
| 7 | **Drop override blake** | IR G-expand + folds rotate | Corps IR (plus d’override 800 L) | Puis #8–9 |
| 8 | **Rotate load-alias / xor-pair** | `emit/emit.go` | `sameRotSrc` via Mov, xor pair, load(base,idx) | ROTR C → `bits.RotateLeft64` (blake, chacha) |
| 9 | **Densify blake (boucle, pas unroll 12)** | single-use + folds | Unroll général **désactivé** (cassait CRC bit-loop) | **~5675 L → ~143 L** ; block_1 **1,36×**, block_1k **1,34×** ; sgo/ccgo médiane **0,84×** |
| 10 | **CRC masque 1 ligne** | `foldNegatedMask` + single-use | `crc=(crc>>1)^(POLY&-(crc&1))` | **~96 L → 23 L** ; bulk/l1 **~1,0–1,04×** ; tiny **1,20×** |
| 11 | **L32 / R → RotateLeft\*** | `emit/gemini_folds.go` + module emit | Helpers tweetnacl inlinés | nacl sans defs L32/R ; ver_* **~0,95×** vs C, sgo/ccgo **~0,27×** |
| 12 | **Rotl32 module-level drop** | `archInlineRotlWrappers` sur module | Wrapper murmur mort supprimé | murmur **58 L** (plus de `func Rotl32`) |
| 13 | **base64 string table + `dst[j:j+4]`** | `emit/overrides.go` | Plus de `[64]byte` stack par appel ; BCE sous-tranche sortie | tail **1,83× → ~1,38–1,43×** ; bulk **~1,09×** ; l1 **~1,36×**. *LUT package testée puis abandonnée (régression l1 ~1,8×).* |
| 14 | **stripDeadAssigns multi-assign** | `emit/emit.go` | LHS d’autres `vN=` n’est plus un « use » | Débloque DCE après unroll / folds (blake, etc.) |
| 15 | **Plafond foldSingleUse** | `emit/emit.go` | budget ≤400 | Évite hang O(n²) sur corps monstrueux |
| 16 | **Unroll général body désactivé** | `rules/unroll_const_trip.go` | Garde anti bit-serial CRC `b<8` | Correctitude CRC restaurée ; remapBodyFresh conservé pour réactivation future |

---

## 2. Détail par famille

### 2.1 `fast_xor` — plus gros win bulk

| strate | sgo/C final | lecture |
|--------|-------------|---------|
| tail_17 | **1,53×** ★ | seul goulet ≥1,5 ; coût de la queue générale vs C |
| align_64 | 0,99× | |
| l1_1k | **0,67×** | sgo plus rapide que C |
| l2_64k | **0,68×** | |
| bulk_1m | **0,66×** | |
| sgo/ccgo médiane | **0,63×** | |

**Levier :** mots 16/8 + `SliceData` (évite headers de slice à chaque pas).  
**Dette :** spécialiser encore tail_17 (n petit / rem=1) sans régresser L1.

### 2.2 Hashes (fnv, murmur, siphash, crc)

| lib | avant session (ordre de grandeur) | après |
|-----|-----------------------------------|--------|
| fnv | override ×8 ; puis IR | int+BCE ; bulk **~1,0×**, l1 **1,11×** |
| murmur | override long | IR forward ; l1 **1,36×**, tiny **1,40×** |
| crc32 | multi-temps `int(-(crc&1))` | **1 ligne** bit step ; bulk **1,02×** |
| siphash | override dense | l1 **1,00×**, tiny **1,05×**, l1_4k **1,37×** |

### 2.3 Crypto blocs (blake, chacha, poly, nacl)

| lib | effet code | sgo/C | sgo/ccgo |
|-----|------------|-------|----------|
| blake | 5,6 kL unrolled → **143 L** boucle densifiée | 1,24–1,36× | **0,84×** |
| chacha | 4× `RotateLeft32` | 0,90–1,07× | **0,44×** |
| poly | inchangé majeur | 0,68–0,99× | **0,52×** |
| nacl Vn / ver | L32/R inlinés | ~0,95× | **0,27×** |

**Leçon :** unroll agressif sans remap/DCE = illisible + folds lents ; densify **dans la boucle** + peephole rotate > dump 12 rounds.

### 2.4 `base64_simd`

| itération | tail_17 | l1_1k | bulk |
|-----------|---------|-------|------|
| table stack `[64]byte` | ~1,8× | ~1,4× | ~1,3× |
| string + `j:j+4` (**retenu**) | **~1,38–1,43×** | ~1,36× | **~1,09×** |
| LUT package + `outN` BCE | ~1,8× | **~1,8×** régression | ~1,14× |

**Retenu :** string const + sous-tranche sortie.  
**Rejeté :** LUT package (pire sur L1 dans ce harness).

### 2.5 Divers

| lib | sgo/C | note |
|-----|-------|------|
| strlenspn | **0,23–0,42×** | prédicat inline ; sgo domine C et ccgo |
| md5 reduced | 1,34–1,49× | fixture 4×FF |
| md5 full | 1,36–1,43× | |

---

## 3. Correctitude (non négociable)

Chaque optimisation ci-dessus a été gardée **seulement** si :

- `tribench` bit-exact vs C (13/13 en fin de session),
- `go test ./sgoiter/...` vert,
- pas de régression type « CRC xor-only » (unroll bit-loop) ou mono undefined (copy-prop trop large).

Échecs évités / corrigés en session :

- unroll `for b<8` CRC → corps vidé → **unroll général off** ;
- `foldLiveRangeCopies` en pipeline → mono/CRC cassés → **unit-test only** ;
- `foldManualRotHelpers` matchait `func L32(` → ordre **def puis calls**.

---

## 4. Bilan quantitatif (fin de session)

| métrique | valeur |
|----------|--------|
| Goulets sgo/C ≥ **2,0×** | **0** |
| Goulets sgo/C ≥ **1,5×** | **1** (`fast_xor` tail_17) |
| Libs où sgo médiane **&lt; ccgo** | quasi toutes (sauf murmur ~1,02) |
| Bit-exact | **13/13** |
| Dogfood émis | `dogfood_kernels_20260812_e5c_rotl/` |
| Note triangle | `NOTE_STRATIFIED_20260812.md` |

---

## 5. Prochaines optimisations (si reprise)

1. **xor tail_17** — unique ★ ; viser ≤1,25× sans toucher bulk 0,66×.  
2. **base64 tail** — 1,43× ; autre angle que LUT package.  
3. **md5** ~1,4× — graphe / fixture.  
4. **live-range copy** safe (1-use + pas de for-header).  
5. **itérateur int** généralisé hors fnv (Gemini rec. 4).

---

## 6. Fichiers touchés (principaux)

```
sgoiter/emit/overrides.go      xor, base64, …
sgoiter/emit/emit.go           rotate alias, stripDead, budget folds, module Rotl
sgoiter/emit/post_unroll.go    fnv int+BCE
sgoiter/emit/gemini_folds.go   L32/R, live-range (tests)
sgoiter/rules/unroll_const_trip.go  N6 + garde unroll général
sgoiter/spec/bench/probe_stratified/
sgoiter/spec/bench/NOTE_STRATIFIED_20260812.md
```

---

*Note d’ingénierie session 2026-08-12 — gains opposables au sol (probe + tribench), pas aux intentions.*
