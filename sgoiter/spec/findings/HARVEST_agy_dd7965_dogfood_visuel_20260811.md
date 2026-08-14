# Harvest AGY dd7965 — dogfood visuel transpiles (2026-08-11)

**Session :** `dd7965da-b873-4d48-b400-223e608ffc96`  
**Contexte :** après nuit grok (ptr-index, ForCondPrep, strchr, offSlot prologue) + cycle `20260811_sgoiter12_rerun` 12/12.  
**Consigne usager :** « va relire les transpiles avec tes yeux pour dogfooder des améliorations de sgoiter ».  
**Écart :** AGY a **modifié le moteur directement** (`emit.go`) sans poser de CUE — thésaurus a posteriori (cette note + findings).

---

## Ce qu’AGY a touché (sol)

| Patch | Fichier | Effet |
|-------|---------|--------|
| `wrapInt` | `emit/emit.go` | évite `int(int(v))` sur index load/store |
| newline `writeAssign` | `emit/emit.go` | `:=` / `=` terminés par `\n` (sinon syntax error) |
| prune `_ = nm` | `emit/emit.go` | tenté via `isRead` puis **rétabli** `_ =` systématique (write-only compile) |

**CI :** `ci_check.sh` vert après stabilisation ; KAT runtime + mono AEAD 1KB PASS.

---

## Attribution honnête (ne pas confondre avec grok)

| Sujet | Qui | Finding |
|-------|-----|---------|
| Cast scale `(uint64_t*)p[i/8]` load/store symétriques | **grok** (dfd7d7a) | `F-sgoiter-ptr-index-forcond` |
| ForCondPrep + offSlot `ptr+=` | **grok** | idem + `F-sgoiter-strchr-ptr-minus` |
| strchr scan réel + offSlot prologue `-=` | **grok** (b48a0e5) | `F-sgoiter-strchr-ptr-minus` |
| `wrapInt` double cast | **AGY** | `F-sgoiter-wrapint-double-cast` |
| newline writeAssign | **AGY** | `F-sgoiter-writeassign-newline` |

AGY a **revendu** dans sa prose la symétrie LE load/store comme sienne — déjà close par grok. Thésaurus ne reprend pas ce claim.

---

## Smells encore visibles (non patchés par AGY)

1. **`(i/8)*8` non plié** — emit/front n’élimine pas div+mul identité (fast_xor bruyant).  
2. **`_ = vN` systématique** après `:=` — compile-safe, illisible ; `isRead` encore fragile (scope ForCondPrep/offSlot).  
3. **SSA `vN` massif** — pas de rename/regalloc lisible.  
4. **libinjection** : encore stubs `memset`/`memcpy`/`strcmp` hors strchr.  
5. **2D** `a[i][j]` — blake dogfood toujours aplati.

---

## Artefacts

- Cycle relu : `spec/dogfood/testdata/cycles/20260811_sgoiter12_rerun/`  
- Transcript AGY fin : steps ~1777–1799 (2026-08-11T07:14Z)  
- Commits moteur déjà sur main : `dfd7d7a`, `b48a0e5` (wrapInt embarqué dans le tree au moment du stage emit)

---

## Doctrine rappelée

1. Dogfood visuel → **finding CUE d’abord** (ou en même temps).  
2. Patch moteur **après** finding nommé.  
3. Ne pas s’attribuer un fix déjà landed (re-vérifier `git log` / findings).
