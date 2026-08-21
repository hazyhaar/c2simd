# Harvest dogfood yeux — post P0/P1 (2026-08-11)

> Relecture des 11 noyaux bit-exact (+ tweetnacl helpers) **après** `e5bab32`.  
> But : améliorer le **transpilé** et **thésauriser** dans sgoiter (findings CUE + TODO_NEXT).  
> Banc inchangé attendu : 11/11 ; toute passe = bit-exact + yeux.

Émis de référence : `/tmp/eye_{fnv,crc,xor,mur,blake,chacha,b64,sip,poly,md5,tn}.go`.

---

## Déjà bon (ne pas retravailler)

| Motif | Noyaux | Note |
|-------|--------|------|
| ROT const → `bits.RotateLeft*` | blake, chacha, sip, md5 | acquis |
| ROT variable | murmur Rotl32, tn R | P1 landed |
| CRC mask `-(v2&1)` | crc32 | P1 landed (TODO_NEXT encore « bloqué » → corriger) |
| base64 `'='` | base64 | P1 |
| Ld32 accumulateur large | tweetnacl | P0 |
| Pack25519 panic + stderr | tweetnacl | P0 |
| binary.LittleEndian loads | murmur, blake, sip, md5, xor | idiomatique |
| fallthrough switch tail | murmur, sip | fidèle C |

---

## Améliorations classées (levier → action)

### T1 — Casts d’identité / litteraux sur-typés — **emit P1**

```go
// fnv — v2 déjà uint64
return uint64(v2)
v4 = v4 + uint64(1)          // + 1 suffit si v4 uint64
v2 = uint64(0xcbf29…)        // 0xcbf29… suffit

// murmur Fmix
return uint32(h)             // h déjà uint32

// crc
return uint32(^v2)
for v5 < uint32(8)           // < 8
```

**Levier :** emit — si `regType[v]==T` et cast `T(v)` → `v` ; littéral entier nu quand le type du slot est déjà connu.  
**Finding :** `F-sgoiter-identity-cast`  
**Risque :** bas (forme pure si types IR stables).  
**Preuve :** `grep -E 'return uint(32|64)\(v' /tmp/eye_*.go` → 0.

### T2 — Quantité de shift littérale sans `uint8(...)` — **emit P2**

```go
v2 >> uint8(1)    →  v2 >> 1
h >> uint8(16)    →  h >> 16
```

Go accepte untyped int.  
**Finding :** `F-sgoiter-shift-lit-bare`  
**Risque :** bas.

### T3 — Parenthèses CRC (lisibilité / precedence) — **emit P2**

```go
// aujourd’hui (banc vert — même asso que gcc sur ce .c)
(v2 >> 1) ^ poly & mask
// cible lisible, sémantique CRC classique explicite
(v2 >> 1) ^ (poly & mask)
```

Si le banc reste vert avec parens forcées sur `^ (…&…)`, figer le motif dans `foldNegatedMask` / writeBinop.  
**Finding :** `F-sgoiter-crc-paren-mask`  
**Ne pas** changer l’ordre des ops sans re-oracle.

### T4 — `&^ 7` redondant (fast_xor) — **emit/ir P2**

```go
// v4 est déjà multiple de 8 dans la boucle (pas de +1 ; +8)
dst[int(v4 &^ uint64(7)):]
```

CSE / fold : si induction `v = 0; v += 8` et align known, drop `&^ (W-1)`.  
**Finding :** `F-sgoiter-align-mask-redundant`  
**Risque :** moyen (preuve d’alignement requise).

### T5 — Snapshots post-inc base64 — **emit P2**

```go
v31 := v5; v5 = v5 + 1; dst[int(v31)] = …
// cible
dst[int(v5)] = …; v5++
```

Aujourd’hui forcé car Go n’a pas de `dst[v5++]`. Emit peut ordonner **use then inc** quand le seul use du snapshot est un index store et qu’aucune autre lecture de l’ancienne valeur n’existe entre-temps.  
**Finding :** `F-sgoiter-postinc-store`  
**Gain :** base64 ~12 lignes / bruit j++.  
**Risque :** moyen (alias v5).

### T6 — Tables const → array non exporté — **emit P1**

```go
var blake2b_sigma = []byte{…}     // mutable + heap slice header
// cible
var blake2b_sigma = […]byte{…}    // ou [N]byte ; non exporté déjà si minuscule
var b64_table = […]byte{…}
```

tweetnacl `K`, `iv`, `L`, `minusp` idem.  
**Finding :** `F-sgoiter-rodata-array`  
**Risque :** bas si indices restent `int(…)`.  
**Pub :** réduit la surface « package-level mutable table ».

### T7 — Stack `[N]T` sans `[:]` systématique — **emit P2**

```go
var _arr_v7 [16]uint64
v7 := _arr_v7[:]
// si tous les uses sont v7[i] / pas de passage slice API
var v7 [16]uint64
v7[i] = …
```

blake, md5, Par25519 buffer.  
**Finding :** `F-sgoiter-stack-array-direct`  
**Risque :** moyen si call attend `[]T`.

### T8 — Inline `Rotl32(x, C)` const — **emit/ir P2**

```go
Rotl32(v23, uint8(15))  →  bits.RotateLeft32(v23, 15)
```

Helper peut rester pour signature C ; sites const inlinés.  
**Finding :** `F-sgoiter-inline-rot-helper`

### T9 — `Dl64` → `binary.BigEndian.Uint64` — **emit P3 / builtin**

Boucle 8 bytes → un load BE si pattern reconnu (comme LE déjà).  
**Finding :** `F-sgoiter-be-load64`

### T10 — poly1305 `5 * int(r[i])` — **front/emit P2**

```go
v26 := 5 * int(v25)   // v25 uint32
// cible
uint64(v25) * 5
```

Évite `int` intermédiaire (overflow signé théorique).  
**Finding :** `F-sgoiter-mul5-widen`  
**Risque :** sémantique — valider vs C oracle poly.

### T11 — Harvest surplus tweetnacl — **front P2**

Émis : Randombytes, L32, Sigma*, Maj, Ch, Pack panic, tables SHA — **hors** surface `crypto_verify_16`.  
Options :
1. `-harvest-roots=Crypto_verify_16,Vn,…` (closure d’appels)
2. stub panic OK mais **ne pas** émettre tables mortes non référencées

**Finding :** `F-sgoiter-harvest-root-closure`  
**Doctrine :** oracle étroit ≠ module entier ; réduit bruit pub + faux findings audit.

### T12 — chacha `*uint32` ×4 — **ABI P3 (doc only)**

Idiomatique Go serait `func QR(s *[16]uint32, a,b,c,d int)` ou valeurs.  
Changer = rupture fidélité pointeur C. **Thésauriser comme non-but** tant que dogfood = C ABI.  
**Finding :** `F-sgoiter-chacha-ptr-abi` status `rejected` ou `codified` doctrine.

### T13 — Q10 index `int` — **déjà TODO** (P1 fond)

blake 40× `int(` ; base64 21 ; …  
Pas de `_i := int(_i)`.  
Inchangé.

### T14 — L32 dogfood `u32=unsigned long` — **c_source P3**

Pas un bug emit ; typedef dogfood. Corriger le `.c` si on veut L32 32-bit.

---

## Complément Gemini (confirmé sol, même session)

| Id | Motif | Sol | Finding |
|----|-------|-----|---------|
| **T15** | cjson : 3× `if !(…){break}` en tête de `for{}` | `/tmp/eye_cjson2.go` | `F-sgoiter-loop-cond-combine` |
| **T16** | base64 : `int(uint64(v12)>>uint8(18)&uint64(63))` | eye_b64 | `F-sgoiter-narrow-shift-mask` |
| **T6bis** | `const b64_table = "…"` (alt. à `[…]byte`) | Gemini | rodata-array notes |

Gemini sur **e5bab32** (Ld32, Pack panic, Rotl32, CRC mask, gates 11/11) : **aligné** avec vérif locale — rien à contester.

---

## Ordre d’attaque recommandé (producteur)

```
P1  T1 identity cast + bare lit     (rapide, multi-noyaux, tests emit)
P1  T6 rodata array / const string  (pub + forme ; b64 = const string)
P2  T15 loop-cond-combine           (cjson + parsers)
P2  T16 narrow-shift-mask           (base64 ; lie T1/T2)
P2  T2 shift lit bare
P2  T3 CRC parens (si oracle OK)
P2  T5 postinc store (base64)
P2  T8 inline rot helper
P2  T11 harvest roots               (surtout tweetnacl)
P1  T13 Q10                         (fond, TODO_NEXT)
P2  T4 align mask xor
P2  T7 stack array direct
P2  T10 poly mul5
P3  T9 BE load
—   T12 chacha ABI                  (ne pas faire)
—   T14 typedef dogfood             (source, hors emit)
```

CRC mask « bloqué » dans TODO_NEXT : **obsolète** post-e5bab32 — marquer soldé.

---

## Comment thésauriser (pipeline sgoiter)

| Artefact | Rôle |
|----------|------|
| `spec/findings/F-sgoiter-*.cue` | 1 finding = 1 levier opposable (`schema.cue`) |
| Ce HARVEST | preuve yeux + extraits avant |
| `TODO_NEXT.md` | file d’exécution + preuves de done chiffrées |
| Tests `emit/*_test.go` | motif textuel → string out (rot_var, negmask, charlit) |
| `TestEmitIsDeterministic` / tribench | garde-fous |
| Cycle dogfood optionnel | snapshot `out.go` si palier forme majeur |

**Règle :** finding `proposed` → patch → `landed` + commit ref + kat pass.  
Pas de rule IR no-op. Pas de promesse perf sans benchstat.

---

## Extraits prioritaires (copie thésaurus)

### fnv — T1

```go
// bruit
v2 = uint64(0xcbf29ce484222325)
v4 = v4 + uint64(1)
return uint64(v2)
```

### base64 — T5 + T6

```go
var b64_table = []byte("…")   // T6
v31 := v5; v5 = v5 + 1; dst[int(v31)] = b64_table[int(…)]  // T5
```

### blake — T6 + T7 + T13

```go
var blake2b_sigma = []byte{…}           // T6
var _arr_v7 [16]uint64; v7 := _arr_v7[:] // T7
v7[int(v9)] = …                         // T13
```

### tweetnacl — T11

```go
// surface oracle : Vn / Crypto_verify_16
// bruit : K, iv, Sigma*, Pack25519 panic, Randombytes
```
