# Couverture findings sgoiter ↔ bancs (2026-08-10)

## 1. Banc AGY trouvé

| Chemin | Rôle |
|--------|------|
| [`/devhoros/pkg/ccgobench/ccgobench.go`](file:///devhoros/pkg/ccgobench/ccgobench.go) | API 5 piliers |
| [`…/ccgobench_test.go`](file:///devhoros/pkg/ccgobench/ccgobench_test.go) | tests **dummy** (XOR, Point, no-alloc) |
| [`…/run_banc_essai.sh`](file:///devhoros/pkg/ccgobench/run_banc_essai.sh) | script gcc+go+ccgo |

**Créé session dd7965 ~21:58** (début, pôle **ccgo** — pas sgoiter monocypher).

### API réelle

| Pilier | Fonction | Oracle |
|--------|----------|--------|
| 1 Différentiel | `RunDifferentialTest(cSrc, goSrc, args)` | gcc −O2 vs `go build` CGO=0 ; SHA256 stdout + exit |
| 2 KAT | `RunKATValidation(t, fn, vectors)` | `bytes.Equal` sur `func([]byte)[]byte` |
| 3 Déterminisme | `CheckDeterminism(cSrc, archivedGo)` | **`ccgo -o`** puis diff octet |
| 4 ABI struct | `ValidateStructLayout(cDef, goDef)` | size/offset/name fournis par l’appelant |
| 5 Fuite | `RunMemoryLeakCheck(n, fn)` | HeapAlloc après GC, seuil 1 MiB |

### Ce que le banc AGY **n’est pas**

- Pas branché sur `sgoiter` (déterminisme appelle **ccgo**).
- Pas de fixture monocypher / AEAD / poly / chacha.
- Tests unitaires = jouets (ne valident aucun finding emit).
- `run_banc_essai.sh` avale les échecs KAT (`|| true`) et ment « OK ».
- Pas de gate `go build` sur package library sans `main`.
- Pas d’IR golden, pas de harvest assert, pas de fail-closed `err_*`.

**Verdict banc AGY :** beau **squelette méthodo** pour qualification **ccgo** ; **zéro opposition mécanique** des findings debug monocypher/sgoiter tant qu’on ne le câble pas (ou qu’on n’ajoute pas un `sgoiterbench`).

---

## 2. Bancs sgoiter / c2simd déjà en dépôt (vrais opposants)

| Banc | Chemin | Ce qu’il oppose |
|------|--------|-----------------|
| **ci_check** | `sgoiter/scripts/ci_check.sh` | cue vet findings ; no ccgo import ; `go test ./sgoiter/...` ; dogfood build 7 labs C ; harvest names murmur/fastlz ; reject `err_asm` |
| **Murmur KAT** | `sgoiter/murmur_kat_test.go` | transpile + emit + vecteurs vs oracle gcc (pilier 1+2 sgoiter) |
| **front tests** | `front/*_test.go` | parse, preprocess, fastlz, v05, **TestHarvestStructsMono** |
| **emit/ir/rules tests** | `emit|ir|rules/*_test.go` | unités IR/rules/emit |
| **Dogfood 12** | `spec/dogfood/.../20260810j_sgoiter12_fresh/` | harvest+build 11/12 kernels |
| **Dogfood monocypher** | `…/20260810k_monocypher/` | amalg + REPORT (build AEAD **manuel**, pas gate CI) |
| **c2simd/kat** | `c2simd/kat/` | KAT kernels opt/ccgo maison (hors sgoiter emit path direct) |

---

## 3. Matrice finding → opposition

Légende couverture :
- **O** = opposé par un test/CI qui **échouerait** si régression
- **P** = partiel (proxy voisin, pas le bug exact)
- **A** = opposable par API ccgobench **si câblé** (fixture manquante)
- **∅** = aucun banc ne l’attrape aujourd’hui

### 3.1 Findings monocypher / emit debug (session + FIXLOG)

| Finding | status | ci_check / unit | murmur KAT | dogfood j/k | ccgobench | Note |
|---------|--------|-----------------|------------|-------------|-----------|------|
| F-type-before-assign | landed | P (build labs) | P | P | ∅ | Pas de test dédié ordre regType |
| F-call-rettype-callee | landed | P | P Load32 path murmur? | P | A KAT | Murmur peut casser si retType dom |
| F-cmp-prefix-variants | landed | ∅ | ∅ | P mono | ∅ | Besoin C avec `_cmp_` |
| F-field-regname-alias | landed | P TestHarvest | ∅ | P mono build | A layout | mono build FAIL autre cause |
| F-arg-field-slice | landed | ∅ | ∅ | P | A | Fixture `Load*_buf(ctx.R)` |
| F-param-ptrmeta-init | landed | P HarvestStructsMono | ∅ | P | A layout | |
| F-issimplebase | landed | ∅ | ∅ | P | ∅ | Mini C `(ctx->h)[i]` |
| F-struct-equal-fold | landed | **O** TestHarvestStructsMono | ∅ | P | A ValidateStruct | |
| F-opload-scalar-not-ptr | landed | P | P | P | ∅ | |
| F-hoist-regtype | proposed | ∅ mono | P | **∅ FAIL** | A go build | **Besoin gate build AEAD** |
| F-opstore-elem-default-u8 | proposed | ∅ | ∅ | **∅ FAIL** | A | poly store H[i] |
| F-ptrmeta-alias-lost | proposed | ∅ | ∅ | **∅ FAIL** | ∅ | front alias pe |
| F-field-array-slice-arg | proposed | ∅ | ∅ | P/∅ | A | chevauche arg-field-slice |
| F-pad-index-add | proposed | ∅ | ∅ | **∅ FAIL** | A KAT poly | |
| F-writeassign-non-lvalue | proposed | ∅ | ∅ | P | ∅ | `ctx.R[:] :=` |
| F-stub-cmp-redecl | proposed | ∅ | ∅ | P full amalg | ∅ | filter AEAD |
| F-hoist-unused | proposed | ∅ | ∅ | P | ∅ | unused var |
| F-isread-broken | proposed | **O?** go test compile | — | — | ∅ | si code isRead shippé |
| F-go-block-scope | landed | ∅ | ∅ | ∅ sémantique | **A diff** | curseur djb — KAT only |
| F-aead-stub-blacklist | landed | ∅ | ∅ | P harvest 0 stub | ∅ | assert IR no stub aead |
| F-cpp-ifdef-amalg | codified | ∅ | ∅ | P amalg exists | ∅ | |
| F-monocypher-aead-status | codified | ∅ | ∅ | REPORT only | A×5 | **meta** |
| F-agy-session-dd7965 | codified | ∅ | ∅ | ∅ | ∅ | doc |
| F-struct-arrow-monocypher | landed | P | ∅ | P harvest | ∅ | |
| F-findop-arrow-gt | landed | P | ∅ | P | ∅ | |
| F-field-plusplus | landed | P | ∅ | P | ∅ | |
| F-ptr-add-callarg | landed | P | ∅ | P | ∅ | |
| F-call-arg-cast | landed | P | P | P | ∅ | |
| F-call-balanced-paren | landed | P | P load64 | P | ∅ | |
| F-addr-struct-call | landed | P | ∅ | P | ∅ | |
| F-global-shadow-param | landed | P | ∅ | P | ∅ | |
| F-global-inject-unused | landed | P | ∅ | P | ∅ | |
| F-unsigned-strip | landed | P | P | P | ∅ | |

### 3.2 Findings dogfood / doctrine (hors monocypher debug)

| Finding | Opposition réelle |
|---------|-------------------|
| F-v02…F-v06 | **O** ci_check labs + cycle 20260810j index + murmur KAT (v04) |
| F-dogfood-xor-self | **O** xor_clear dogfood ci + rules test |
| F-blake2b-2d-index | **O** harvest_fail visible index.json j (pas de test unitaire dédié) |
| F-q2-no-generic-simd | doctrine — pas de banc (volontaire) |

---

## 4. Synthèse couverture

| Panier | N findings | Vraiment opposés (O) | Partiel (P) | ccgobench seul suffirait si câblé (A) | Angle mort (∅) |
|--------|----------:|---------------------:|------------:|--------------------------------------:|---------------:|
| Monocypher/emit debug ~33 | ~33 | ~2–4 | majorité labs | ~12 si fixtures | **résidus AEAD build/KAT** |
| Dogfood v0x + xor | 6 | oui | — | non prioritaire | — |
| Meta/session/doctrine | 3 | non | — | non | doc only |

**ccgobench aujourd’hui :** oppose **0** finding sgoiter monocypher (tests dummy verts sans lien).  
**sgoiter ci_check + murmur + HarvestStructsMono :** opposent le **corridor lab** et une partie des landed front ; **n’opposent pas** le package AEAD monocypher (build encore hors CI).

---

## 5. Gaps prioritaires pour rendre les findings opposables

Ordre de câblage (réutiliser l’esprit ccgobench, cible **sgoiter**) :

1. **`sgoiter` gate monocypher_aead** (nouveau dans ci ou script dogfood k)  
   - filter IR AEAD → emit → `go build` package  
   - oppose : hoist-regtype, opstore-u8, pad-index, stub-redecl, hoist-unused, writeassign-non-lvalue  

2. **KAT minimal** (pilier 2, style `RunKATValidation` ou murmur pattern)  
   - `Load64_le`, `crypto_chacha20_h`, plus tard `crypto_aead_lock`  
   - oppose : go-block-scope curseur, sémantique rounds, call-rettype  

3. **Differential gcc** (pilier 1 ccgobench `RunDifferentialTest`)  
   - harness C qui print hex MAC/cipher vs Go  
   - **ne pas** appeler ccgo dans CheckDeterminism pour sgoiter → `sgoiter -in -out`  

4. **Struct layout poly/aead ctx** (pilier 4)  
   - offsetof C vs reflect Go sur `Crypto_poly1305_ctx` / `Crypto_aead_ctx`  
   - oppose : struct-equal-fold, field harvest, padding  

5. **IR asserts** (spécifique sgoiter, hors ccgobench)  
   - 0 stub `crypto_aead_*`  
   - OpField Elem uint32 sur `h`  
   - pas de OpAdd(scalar, slice) sur pad  

6. **Réparer run_banc_essai.sh** si réutilisé : retirer `|| true` sur KAT ; brancher sgoiter.

---

## 6. Réponse courte

| Question | Réponse |
|----------|---------|
| Banc AGY trouvé ? | **Oui** : `/devhoros/pkg/ccgobench/` (+ script) |
| Beau ? | API 5 piliers claire ; exécution encore **démo** |
| Findings déjà opposés par ce banc ? | **Non** (0 fixture sgoiter/monocypher) |
| Findings opposés ailleurs ? | **Corridor lab** oui (ci_check, murmur KAT, HarvestStructsMono, dogfood j) ; **AEAD monocypher debug** non |
| Action | Câbler gate build AEAD + KAT + (option) adapter ccgobench en `sgoiter` determinism |

---

## 7. Inventaire fichiers banc

```
/devhoros/pkg/ccgobench/ccgobench.go       # framework
/devhoros/pkg/ccgobench/ccgobench_test.go  # dummy PASS
/devhoros/pkg/ccgobench/run_banc_essai.sh  # wrapper ccgo-centric
/devhoros/c2simd/sgoiter/scripts/ci_check.sh
/devhoros/c2simd/sgoiter/murmur_kat_test.go
/devhoros/c2simd/sgoiter/front/front_test.go  # TestHarvestStructsMono
```
