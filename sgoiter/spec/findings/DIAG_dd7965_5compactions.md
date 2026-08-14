# Diagnostic dd7965 — 5 compactions : progrès ou rond ?

**Session :** `dd7965da-b873-4d48-b400-223e608ffc96` (sgoiter monocypher AEAD)  
**Checkpoints disque :** CP0 … **CP5** (6 compactages indexés 0–5)  
**Transcript :** ~1254 events (fin ~23:43)

---

## Verdict : **tourne en rond avec dérive lente** (pas bloqué figé, pas breakthrough)

| Signal | Observation |
|--------|-------------|
| Objectif CP1→CP5 | **Identique** : liste req figée se terminant par monocypher REPORT + « reprends » |
| Fichiers | Toujours `emit.go` + `front.go` (martelage) |
| Build AEAD | **Jamais vert** en fin de chaque tranche CP |
| Classes d’erreur | **Changent** (progrès local) puis **nouvelles** apparaissent |
| Compaction | Chaque CP **re-résume** le même brief → perte de nuance, re-attaque du même mur |

---

## Évolution des erreurs par compaction

| Segment | CA | Pattern dominant | Lecture |
|---------|---:|------------------|---------|
| CP0→1 | 7 | `byte` → `uint32` (Load LE) | démarrage hoist |
| CP1→2 | 29 | `uint8(v)` → `ctx.H[]` uint32 ; `ctx.H` as scalar | structs/field |
| CP2→3 | 31 | **encore** `uint8→uint32` H ; + undefined v ; uint32→byte | **boucle** même classe H |
| CP3→4 | 35 | int/uint32 ; undefined Crypto_aead_* ; binop slice | élargit surface, casse autre |
| CP4→5 | 21 | `Crypto_*_ctx` value vs `*ptr` ; `**ctx` | nouveau mur pointeurs |
| CP5→fin | 2 | (peu d’échantillons) | essoufflement / re-compact |

**Signature du rond :** CP1–CP2 et CP2–CP3 partagent le top error `cannot use uint8(vN) as uint32` sur stores H — **même bug non clos** après des dizaines de patches.

**Signature du progrès (faible) :**
- Load byte/LE largement poussé (binary path)
- TestHarvestStructsMono apparu
- Classe d’erreur migre (H scalar → uint8 cast → *ctx) = on avance dans le graphe de bugs, **sans sortir**

---

## Pourquoi 5 compactions aggravent

1. **Brief CP figé** : les 10 dernières user requests recopient biscuit/ccgo/dogfood j/ctx/REPORT/reprends — le monocypher emit n’a pas de **critère d’arrêt** ni de mini-oracle dans le checkpoint.  
2. **Pas de gate** `go build AEAD` dans un test CI → l’agent optimise le dernier message d’erreur, pas un package vert.  
3. **Patches atomiques** (isRead cassé, hoist on/off) sans golden IR → régressions.  
4. **Contexte tronqué** : findings/FIXLOG hors session non rechargés → re-découverte.

---

## Comparer ab517 (horosvec) vs dd7965 (sgoiter)

| | ab517 | dd7965 |
|---|--------|--------|
| Compactions | CP0–1 seulement | **CP0–5** |
| Fin | plan 4 pistes **validé**, stop propre 22:30 | encore FAIL emit 23:43 |
| Progrès | **oui** (doctrine + priorisation) | **local only**, objectif build AEAD non atteint |
| Rond ? | non (après correction usager) | **oui** sur stores/types ctx |

---

## Recommandation

**dd7965 :** arrêter le martelage ; figer un **oracle** unique :
1. mini C `poly_store_h.c` / IR golden  
2. ou `go build` package AEAD filtré en CI  
3. un finding = un test qui rouge/vert  

Sans ça, CP6–CP10 = même film.

**ab517 22:31–22:42 :** rien à attendre du transcript ; enchaîner **sonde prefetch batch** hors git si reprise humaine.
