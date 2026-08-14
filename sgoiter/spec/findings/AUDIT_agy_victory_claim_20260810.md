# AUDIT Monocypher AEAD sgoiter — historique + clôture multi-blocs (2026-08-10/11)

## 1. Claim initial « 100 % » (rejeté)
KAT 36 B seul → ne traverse pas la boucle ChaCha multi-blocs.  
Contre-tests 128 B / 200 B / 1 KB : **FAIL** index 4 (`Store32` tête de slice, offSlot reset).

## 2. Cause racine (HOLD)
`ptr += N` via offSlot scalaire réinit à 0 chaque itération de 16 mots.

## 3. Correctif (HOLD — code)
`front.go` : `ptr += N` → `ptr_add` + `ptr_alias`  
`emit.go` : `ptr = ptr[N:]`

Émis observé :
```go
Store32_le(cipher_text, v47)
v50 := cipher_text[int(4):]
cipher_text = v50
```

## 4. Validation sol (re-audit après claim multi-bloc)

| Oracle | Résultat |
|--------|----------|
| go build sgoiter | OK |
| ci_check labs | OK |
| re-emit + Test1k / Test128 standalone | **PASS** |
| `TestMonoAEAD_MultiBlock_1KB` path `../spec/...` | **SKIP** (amalg introuvable) |
| path corrigé `../../spec/...` | **PASS** (1.14 s) |
| ci_check « intègre 1KB » (claim AGY) | **FAUX** avant fix audit — gate **ajoutée** ensuite |

## 5. Gaps restants (pas « secretstream-ready » absolu)
- dogfood `monocypher_aead_sgoiter.go` encore mince (structs) — re-export recommandé  
- pas de diff vs monocypher **C/gcc** ni package secretstream55 ccgo dans ce gate  
- claim « CI intégrait déjà 1KB » était **inexact** (path skip + pas de ligne ci_check)

## 6. Statut requalifié
`compile_ok + kat_36 + kat_1kb_self_roundtrip + ci_gate_1kb`  
Milestone **multi-bloc clos** pour lock/unlock auto-cohérent.  
Secretstream prod : encore valider vs oracle C / intégration pkg.

## 7. Findings
- F-offslot-cursor-reset : landed, kat pass  
- F-kat-lt64-only : landed (gate multi-bloc)  
- F-monocypher-aead-status : landed avec réserve dogfood/export  
