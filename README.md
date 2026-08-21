# c2simd (`code.hazyhaar.fr/devhoros/c2simd`)

> **Préprocesseur de Transpilation C-to-Go & Restructuration SIMD (Go 1.27)**  
> *Zéro CGO • Matcher d'AST Déterministe • Moteur Fused SIMD256 • Fallback Scalaire Sûr `x/crypto`*

---

## 1. Description de l'Architecture V4

Le module `c2simd` met en œuvre un moteur cryptographique XChaCha20-Poly1305 et un pipeline d'optimisation d'AST C-vers-Go en Pure Go (`CGO_ENABLED=0`, zéro fichier `.s`).

### Composants Clés :
1. **Moteur Fused SIMD256 (`simd_fused.go`) :**  
   Fusion de boucle par tranches de 4 096 octets (cache CPU L1) combinant ChaCha20 SIMD256 4-blocs (256 octets/itération) et Poly1305 16-Way 3x44-bit. Atteint **1 228 Mo/s à 0 allocation (`0 B/op`)**.
2. **API Zéro-Allocation (`AEADLockDst` / `AEADUnlockDst`) :**  
   Permet le scellage et le déchiffrement authentifié avec réutilisation des tampons de destination fournis par l'appelant.
3. **Fallback Scalaire Sécurisé (`simd_fused_fallback.go`) :**  
   Sous `!goexperiment.simd`, le module bascule de manière transparente sur `golang.org/x/crypto/chacha20poly1305`, garantissant la sécurité cryptographique et le contrôle du tag MAC *fail-closed* (`subtle.ConstantTimeCompare`).
4. **Porte KAT Inconditionnelle (`kat/kat_test.go`) :**  
   Validation différentielle automatique contre 3 oracles (RFC 8439, `x/crypto`, Monocypher C transpilé) s'exécutant dans tous les modes de build.

---

## 2. Benchmark et Mesures au Sol (Intel Core i9-14900K)

| Composant | Baseline / Référence | Moteur `c2simd` Pure Go | Allocations Heap | KAT Status |
| :--- | :---: | :---: | :---: | :---: |
| **ChaCha20 Stream** | 1 016 Mo/s (`x/crypto` ASM Google) | **1 861 Mo/s** (+83,2 %) | **0 B/op** | PASS 100% |
| **Poly1305 MAC** | 4 863 Mo/s (`x/crypto` ASM Google) | **3 048 Mo/s** (62,7 % vs ASM) | **0 B/op** | PASS 100% |
| **AEAD Fused Lock** | 437 Mo/s (Monocypher C transpilé) | **1 228 Mo/s** (+180 %) | **0 B/op** | PASS 100% |
| **Fallback Scalaire** | N/A | **1 007 Mo/s** | 1 alloc/op | PASS 100% |

Bench transpile raw→opt (go1.27rc1+simd, post-`__ccgo_up`, 2026-08-10) : SipHash **+8,5 %**, BLAKE2b **+16,5 %**, MD5 **+13,5 %** ; FastXOR **−12 %** waiver mem-bound.

---

## 3. Ancrage HPM55

* **Projet ID :** `019fd633-6735-705c-8c57-e09608dca298`
* **Sous-projet sgoiter :** `019fec65-22c3-735a-aa9a-753b89ffa9a9` — transpileur itératif thésaurus-first (`sgoiter/`)
* **Module Go :** `code.hazyhaar.fr/devhoros/c2simd`
* **SDK :** `go1.27rc1` (`GOEXPERIMENT=simd`)
* **VERSION :** `2.1.0` (gen `__ccgo_up` + harnais dogfood/CI)

---

## 4. Pipeline transpile v2 (AST, hors produit AEAD)

Séparation stricte (peer review §1 + finding `F-20260810-q2-generic-simd`) :

| Couche | Rôle | Artefacts |
| :--- | :--- | :--- |
| **rules rewrite** | Passes AST appliquées | `rotl32`, `L32`, `load32_le`, `store32_le` (`Kind=rewrite`) |
| **passes structurelles** | Hors Symbol table | tls T0, rotate, fold libc.From*, `__ccgo_up`→`unsafe.Pointer`, `uintptr(-N)`, polish index |
| **handwrite_pointer** | Pointeurs, pas de gen | `poly_blocks` → `simd_poly1305_*.go`, `chacha20_rounds` → `simd_chacha20.go` |
| **findings** | Thésaurus append-only | `spec/findings/*.cue` (`cue vet ./spec/findings/`) |
| **gen CLI** | `ccgo` Go → opt Go | `bin/c2simd-gen -in … -out … [-stats]` |

```bash
./scripts/ci_check.sh
GOWORK=off go test ./internal/astmatch/ ./spec/c_sources/ ./kat/ -count=1
./scripts/dogfood_cycle.sh md5_transform          # DOGFOOD_CCGO_UP_MAX=0 par défaut
./scripts/bench_append.sh label_local             # go1.27rc1 + simd
./scripts/scan_ccgo_up_residuals.sh
```

Interdit en v2 : règle générique « boucle C → SIMD », `rules.db` runtime, fork ccgo, I/O au runtime AEAD.

---

## 5. Checklist AVOID (post-ccgo, opposable)

1. **Punning non aligné** `*(uint32_t*)p` — survit en accès `unsafe` ; le gen ne garantit pas l'alignement.
2. **ROL distance variable** — pas de rewrite `bits.*` sans preuve de borne.
3. **Gros buffers stack C** (`ws[4096]`) → stack Go — AVOID services multi-goroutine.
4. **Data-heavy** (tables 10k+ lignes header) — bloat ccgo (issue #46).
5. **Pointeurs Go natifs** vers API ccgo — interdit ; Xmalloc / tls.Alloc only.
6. **va_list imbriqué** — bug ABI ccgo #43 ouvert.
7. **`iqlibc.__builtin_*`** sans rewrite — build cassé (memmove landé ; autres au fil de l'eau).
8. **`uintptr(-N)`** arith pointeur — overflow Go (landé, y compris 2e tour post-`__ccgo_up`).
9. **typedef u32 = unsigned long** (tweetnacl LP64) — RotateLeft32 exige cast (règle L32).
10. **Règle générique boucle→simd** — interdite (Q2).

Détail CUE : `spec/findings/F-20260810-avoid-patterns-adversarial.cue`, `F-20260810-ccgo-pitfalls-research.cue`.

---

## 6. Contributeurs & Équipe de Développement

Le projet `c2simd` et l'ensemble de ses sous-modules (`sgoiter`, `c2painter`, `c2display`, `c2fynedriver`, `c2fyneterm`, `c2vte`) sont développés et maintenus par :

- **Hazyhaar** ([@hazyhaar](https://github.com/hazyhaar)) — Conception architecturale, doctrine d'ingénierie et gouvernance du projet.
- **Gemini** (Google DeepMind) — Audits contradictoires, recherche de code, preuves de parité et modélisation formelle.
- **Grok** (xAI) — Robustesse système, fuzzing intensif, résistance aux CVEs et analyse protocolaire.
- **Claude** (Anthropic) — Passes du transpileur `sgoiter`, pile souveraine Fyne55, protocoles X11/Wayland et schémas CUE ARCHTIME.



