# Harvest AGY dd7965 — post CP5 / post-compaction (build OK, KAT FAIL)

**Session :** `dd7965da-b873-4d48-b400-223e608ffc96`  
**Tranche :** après CHECKPOINT 5 (L1224) → fin ~23:48+ (paste usager « Conversation compacted »)  
**Paste usager :** analyse mono.go chacha / Gap / OpNot / ternary / zero assignment  

---

## Verdict progression

| Avant CP5 | Après CP5 (cette tranche) |
|-----------|---------------------------|
| `go build` AEAD **FAIL** (types) | **`go build` OK** (package monocypher_amalg ~9k+ L) |
| pas d’oracle runtime | **`TestMonoAEAD_KAT`** existe et **FAIL sémantique** |
| ci_check parfois cassé (isParam) | **ci_check OK** (répété 11×) |
| boucle types H/uint8 | **sortie du mur compile** → mur **correctness** |

→ **Progresse** (changement de phase), plus le rond CP1–CP3 sur `uint8→H`.  
Encore **pas clos** : KAT « Decrypted plaintext does not match original ».

---

## Oracle posé (enfin)

```go
// mono_test.go (tmp agent)
Crypto_aead_lock → unlock round-trip
// FAIL: Decrypted plaintext does not match original!
```

C’est le bon critère d’arrêt (remplace le martelage d’erreurs compile).

---

## Fixes atomiques observés (post-CP5, paste + transcript)

| Thème | Fichier | Statut |
|-------|---------|--------|
| ci_check garde vert | emit/front | landed-wip |
| isParam manquant (compile emit) | emit.go | fixed puis OK |
| ternary `a?b:c` / `__select` | front + emit | WIP |
| OpNot / `~` prefix | front | WIP (Gap) |
| NULL → 0 preprocess | front | présent |
| plain_text / zero conditionnel | front/emit chacha | **BUG RESTANT** |
| pointer / offSlot cipher_text | emit | **BUG RESTANT** (cursor) |
| wipe skip non-[]byte | emit | partiel |
| TestHarvestStructsMono | front_test | OK |
| Gap `(~x+1)&mask` | emit | **semble correct** (`^x` puis +1) |

---

## Bugs sémantiques encore visibles dans mono.go (sol)

### B1 — `plain_text` forcé à `zero`
```go
// Crypto_chacha20_djb
v7 = zero          // TOUJOURS
if v7 != nil {     // toujours vrai (zero est slice non-nil!)
  // XOR depuis zero, pas plain_text
  v46 = Load32_le(v7)
  ...
}
```
C attend : XOR avec `plain_text` si non-NULL ; `plain_text = zero` **seulement** dernier bloc incomplet si NULL (`monocypher_amalg.c` ~569).

**Double bug Go :**
1. assign initiale `v7 = zero` au lieu de `plain_text`  
2. `zero` global `make([]byte,128)` est **non-nil** → branche « plain présent » toujours prise avec contenu zéro → keystream XOR 0, pas le plain

Finding : `F-sgoiter-chacha-plain-zero`

### B2 — curseur `cipher_text` / `v50` reset dans la boucle mot
```go
Store32_le(cipher_text, v47)  // pas d’offset
v50 = uint64(0)
v50 = v50 + 4                 // reset chaque mot
```
→ écrasement du début du buffer ; pas d’avancée C `cipher_text += 4`.

Finding : `F-sgoiter-offslot-cursor-reset` (lié F-go-block-scope)

### B3 — KAT AEAD
lock/unlock round-trip échoue car chacha (et donc aead_write/read) faux.

---

## Gap (fausse piste si correct)

C : `return (~x + 1) & (pow_2 - 1);`  
Go émis : `v2:=^x; v4=v2+1; v7=v4&(pow_2-1)` — **équivalent**.  
Ne pas prioriser Gap pour le KAT FAIL.

---

## Artefacts

| Path | Note |
|------|------|
| `/tmp/tmp.eGCUxIiH9c/mono.go` | 170k, build OK |
| `…/mono_test.go` | TestMonoAEAD_KAT |
| amalg | `spec/c_sources/upstream/monocypher/4.0.2/monocypher_amalg.c` |
| WIP | emit.go +484, front.go +872 vs HEAD |

---

## Recommandation reprise (anti-rond)

1. **Ne plus** optimiser le message compile (déjà vert).  
2. Mini-oracle **chacha only** avant AEAD :  
   `crypto_chacha20_djb(ct, pt, …)` vs gcc / vecteur connu.  
3. Fix ordonnés :  
   - B1 plain/zero/nil (`[]byte(nil)` pour NULL, pas `zero` non-nil)  
   - B2 offSlot live hors reset for  
4. Garder KAT AEAD comme gate finale.  
5. ci_check à chaque patch (déjà fait).

---

## Findings CUE

| id | status |
|----|--------|
| F-sgoiter-aead-build-ok-kat-fail | codified (phase) |
| F-sgoiter-chacha-plain-zero | proposed |
| F-sgoiter-offslot-cursor-reset | proposed |
| F-sgoiter-dd7965-compaction-loop | MAJ : post-CP5 **sort** du rond types |
