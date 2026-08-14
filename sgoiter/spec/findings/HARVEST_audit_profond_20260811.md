# HARVEST — audit profond du code transpilé (12 noyaux + monocypher) — 2026-08-11

> Périmètre : les douze noyaux du banc tribench (cycle frais `tribench_20260811_203146`,
> 11/11 compared bit-exact, `libinjection_sqli` sans oracle) et le module monocypher
> régénéré ce jour (`20260810k_monocypher/sgoiter_out/monocypher_aead_sgoiter.go`,
> 1 781 lignes, KAT AEAD vert, 8 stubs annoncés).
> Corpus audité : `spec/dogfood/testdata/cycles/20260811_audit_fable/<lib>/{src.c,out.go}`
> (copies du cycle tribench du jour) + l'émis monocypher versionné.
> Méthode : treize lectures profondes indépendantes (subagents Fable, un par lib,
> brief commun), puis adjudication au sol par la session — chaque P1 ci-dessous a été
> rejoué sur le texte émis et, où nécessaire, sur la source C. Une seule affirmation
> reste non tranchée et est marquée PLAUSIBLE.

Rapports bruts par lib : scratchpad de session `audit_sgoiter/rapport_*.md`
(hors dépôt ; la substance arbitrée vit ici et dans les CUE).

## Verdict par noyau

| lib | P1 sémantiques | dominante |
|---|---|---|
| fnv1a_64 | 0 | étalon de forme ; dernier `int(` déposable (§C16) |
| crc32_ieee | 1 | compteur rétréci uint32 → non-terminaison (§C11) |
| fast_xor | 0 | garde de boucle non remontée (§C15) ; aliasing fidèle |
| chacha20_qr | 2 | fonction silencieusement non moissonnée (§C12) ; caches de pointeurs à travers écritures (§C5) |
| md5_transform | 1 (fixture) | la source C n'est pas un MD5 complet (§C17) |
| murmur3_x86_32 | 1 | `(int)(len/4)` non tronqué (§C10) |
| base64_simd | 0 | sur-élargissement u64 de l'assemblage (§C15) |
| siphash24 | 0 | verbosité = macro SIPROUND inline 8× ; modèle switch sans break (§C13) |
| poly1305_block5 | 1 | `5*r[k]` sans wrap u32 (§C10) |
| blake2b_compress | 0 | immédiats sur-typés (§C15) ; LE figé documenté (§C18) |
| tweetnacl_dogfood | 2 | `b = n` fusionné avec paramètre réaffecté (§C6) ; `i64`→uint64 (§C7) |
| libinjection_sqli | 3 | tautologie par pliage de comparaison de pointeurs (§C8) ; memset/memcpy en éléments (§C9) ; sans oracle |
| monocypher | 6 | §C1–§C4 + §C5-adjacent + §C7 — tous hors couverture KAT |

Constat transverse dominant : **le banc bit-exact ne couvre que ce qu'il exécute**.
Les quatorze classes P1 confirmées vivent soit hors des vecteurs (grandes longueurs,
limbes hors contrat, aliasing), soit dans des zones que l'oracle n'exerce pas
(fe/eddsa monocypher, fonctions au-dessus de stubs, noyau sans oracle C).

---

## P1 — classes sémantiques confirmées

### C1 — wipe émis avant l'expression de retour qui lit le buffer — emit — CONFIRMÉ

```go
// monocypher_aead_sgoiter.go:1381-1383
Fe_tobytes(v1, f)
for _i := range v1 { v1[_i] = 0 }
return int(v1[0] & uint8(1))
```

Le C (`monocypher_amalg.c:1639-1659`) lit le temporaire PUIS le wipe. L'émis wipe
d'abord : `Fe_isodd` retourne toujours 0, `Fe_isequal` (l.1393-1395) compare deux
buffers remis à zéro et retourne toujours 1. CUE : `F-sgoiter-wipe-before-return`.

### C2 — `uint32(x << 32)` : le mot haut vaut toujours 0 — front — CONFIRMÉ

```go
// monocypher_aead_sgoiter.go:416
uint64(ctr + uint32(uint64(Load32_le(nonce)) << 32))
```

Trois sites : nonce IETF (l.416), reconstruction du compteur djb (l.409), carry de
`Multiply` tronqué (l.1435 : `uint64(p[...] + uint32(a*b))`). Le shift 64 bits du C
est rejoué en 32 bits après troncature. CUE : `F-sgoiter-shift32-trunc`.

### C3 — regroupement de précédence `&` / `>>` — emit — CONFIRMÉ

C (`monocypher_amalg.c:1441,1451`) : `load24 & (0xffffff >> nb_mask)`.
Émis (l.972) : `Load24_le(s[29:]) & uint32(0xffffff) >> uint8(nb_mask) << 2` — en Go,
`&`, `>>`, `<<` partagent le niveau multiplicatif, associativité gauche :
`((load24 & 0xffffff) >> nb_mask) << 2`. Tout `fe_frombytes` (nb_mask=1) décode faux.
CUE : `F-sgoiter-precedence-mask-regroup`.

### C4 — avancement de pointeur perdu (`*message++`) — front — CONFIRMÉ

```go
// monocypher_aead_sgoiter.go:517-521
for v15 < v14 {
    ctx.C[int(ctx.C_idx)] = uint8(uint32(message[0]))  // message jamais avancé
```

La boucle d'alignement de `Crypto_poly1305_update` relit `message[0]` à chaque tour
(C : `*message++`, `monocypher_amalg.c:699-701`). Tout usage incrémental non aligné
produit un MAC faux ; le chemin AEAD masque le cas (padding zéro).
CUE : `F-sgoiter-ptr-advance-lost`.

### C5 — caches de valeurs pointées réutilisés à travers des écritures — emit/ir — CONFIRMÉ

`chacha20_qr/out.go:7-22` : `v4 := *b` puis réutilisation après `*a = …`/`*d = …` là
où le C relit. Sans `restrict` côté C, l'appel aliasé (`qr(&x[0], &x[0], …)`)
diverge. Le banc ne le voit pas (le double-round appelle quatre adresses
distinctes). CUE : `F-sgoiter-alias-cache-across-store`.

### C6 — variable de sauvegarde fusionnée avec un paramètre réaffecté — emit — CONFIRMÉ

`tweetnacl_dogfood/out.go:220-222` : le `b = n` de `crypto_hash` a disparu, le
padding SHA-512 encode `n << 3` avec `n` déjà réaffecté (taille de bloc constante)
au lieu de la longueur réelle. L'IR est correct ; `canPureAlias`
(`emit/emit.go:3365-3368`) ne vérifie les réécritures de la source d'un alias que si
c'est un vreg — un nom de paramètre échappe au contrôle. Classe : tout `T b = param;`
suivi d'une réaffectation du paramètre. CUE : `F-sgoiter-param-snapshot-fused`.

### C7 — `i64` mappé non signé — front — CONFIRMÉ

`front/front.go:2819` range `i64`/`int64_t` sous `TypUint64`. Effets : signatures
`Neq25519`/`Par25519` en `[]uint64` (tweetnacl out.go:120,130) ; carries fe
monocypher émis en `uint64` avec shifts logiques (`Fe_frombytes_mask`,
l.952-962, l.975) là où le C fait des shifts arithmétiques i64 — faux sur tout limbe
négatif (routinier après `fe_sub`). CUE : `F-sgoiter-i64-mapped-unsigned`.

### C8 — comparaison de pointeurs pliée en constante + offset perdu — front — CONFIRMÉ

```go
// libinjection_sqli/out.go:53
if 1 != 0 && (... uint32(cur[0]) == uint32(cur[0]) ...)
```

`(cur+1) < end` plié en `1`, `*(cur+1)` devenu `cur[0]` : `Is_double_delim_escaped`
retourne toujours 1. Seul noyau SANS oracle — le défaut est exactement dans l'angle
mort. CUE : `F-sgoiter-ptr-cmp-folded`.

### C9 — `sizeof` en octets appliqué comme nombre d'éléments, borne silencieuse — emit — CONFIRMÉ

`libinjection_sqli/out.go:34-40` : `memset`/`memcpy` de `stoken_t` (64 octets) émis
en boucles de 64 **éléments** sur `[]int`, avec garde `i < len(p)` qui rend toute
copie partielle silencieuse — contraire au fail-loud. CUE : `F-sgoiter-memsize-as-elems`.

### C10 — arithmétique 32 bits C élargie en `int` Go — front/emit — CONFIRMÉ

Deux visages : `5 * r[k]` de poly1305 émis `5 * int(v25)` (out.go:15) sans le wrap
unsigned 32 bits du C (divergence dès `r[k] ≥ 858993460` — hors contrat limbes
26 bits, mais la classe transpileur est réelle) ; `(int)(len / 4)` de murmur3 émis
`int(len_ / 4)` (out.go:27) sans troncature 32 bits (divergence dès 8 Gio).
CUE : `F-sgoiter-narrow-arith-widened`.

### C11 — compteur `size_t` rétréci en uint32 — emit — CONFIRMÉ

`crc32_ieee/out.go:5,10` : `var v4 uint32` face à `len_ uint64`, garde
`for uint64(v4) < len_` — non-terminaison pour `len_ ≥ 2^32` (v4 wrappe).
`front.go:2819` mappe pourtant `size_t`→uint64 ; la perte est côté hoist/declType
(`emit/emit.go:493-496, 918-920`, point exact non mesuré).
CUE : `F-sgoiter-counter-narrowed`.

### C12 — fonction non moissonnée sans trace dans l'émis — front — CONFIRMÉ

`chacha20_qr` : `chacha20_double_round` (src.c:14-23) absente de l'émis, sans stub
ni commentaire (`grep -n 'double_round' out.go` vide). La moisson range les rejets
dans `Skipped` (`front/front.go:114-136`) sans les matérialiser — violation du
principe fail-loud qui gouverne les stubs. CUE : `F-sgoiter-harvest-silent-skip`.

---

## P2 — classes confirmées

### C13 — modèle switch sans break : `fallthrough` inconditionnel — ir/emit

`emit.go:662-664` émet `fallthrough` entre toutes les cases ; `SwitchCase`
(`ir/ir.go:114-119`) n'a aucun champ break. Fidèle sur siphash24/murmur3 (switchs
tombants), mais un `break` médian C est irreprésentable — le comportement du front
sur ces formes n'est pas couvert. CUE : `F-sgoiter-switch-no-break-model`.

### C14 — `strcmp` émis en comparaison de tranches entières — emit

`libinjection_sqli/out.go:31` : `Streq` compare les tranches complètes là où
`strcmp` s'arrête au NUL ; les appelants amont (empreintes en tampon fixe) sont la
classe divergente. Incohérence interne : `strchr` gère le NUL, `strcmp` non
(`emit.go:3905-3907`). CUE : `F-sgoiter-strcmp-slice-vs-nul`.

### C15 — résidus T1/T16 sur expressions, résultats d'appel et immédiats — emit

T1 landed ne couvre que les identifiants nus (`selfCastPat`, `emit.go:2139` ;
`binLitCastPat`, `emit.go:2162` omet `%`/`/` et exige un identifiant à gauche).
Sites confirmés : `uint32(bits.RotateLeft32(...))` et `uint32(Fmix32(v8))`
(murmur3 out.go:10,59), cast au return composite (siphash out.go:170), 9
`return uint64(...)` (tweetnacl), immédiats IV `uint64(0x…)` (blake2b, `formatImm`
emit.go:3402-3409), sur-élargissement u64 de l'assemblage base64 (out.go:15,27,29),
`uint64(7)` sur `&^` (fast_xor out.go:12), casts littéraux poly1305.
Même famille : garde de boucle non remontée quand la définition intercalée est une
arithmétique pure dupliquable (fast_xor out.go:9-11, `hoistLoopGuards`
emit.go:2185-2212). CUE : `F-sgoiter-identity-cast-expr`.

### C16 — angle neuf Q10 : déposer le wrapper `int(` au site d'index sans retyper — emit

La spec Go accepte tout type entier comme index : `data[v4]` avec `v4 uint64` est
légal. Le dépôt du wrapper au site d'index ne requiert pas le retypage de registre
que Q10 documente (et que `indexonly` refuse dès qu'un registre est comparé à un
paramètre). Nuance sémantique confirmée sur fnv1a (out.go:10) : sur plateforme
32 bits, `int(v4)` tronque silencieusement là où `data[v4]` paniquerait — le
wrapper actuel est pire que son absence. 21 sites base64, 40 blake2b.
Couplage : les passes postinc (emit.go:2246) et T16 (emit.go:2282) matchent
textuellement `int(`. CUE : `F-sgoiter-index-cast-droppable`.

---

## Fixtures et documentation

### C17 — la fixture md5_transform n'est pas un MD5 — c_source

`md5_transform/src.c:29-32` : quatre étapes FF seulement, macros G/H/I définies
jamais invoquées, pas de table K. L'émis est fidèle à cette troncature ; l'oracle
bit-exact valide un fragment. CUE : `F-sgoiter-md5-fixture-truncated`.

### C18 — casts UB d'endianness des fixtures figés LE par l'émis — doctrine

`((uint64_t*)block)[i]` (blake2b src.c:39, md5 src.c:27) est UB strict-aliasing et
dépendant de l'hôte ; l'émis fige `binary.LittleEndian` (emit.go:1349-1361) —
correct pour ces algorithmes, divergent du C sur hôte big-endian. À documenter
comme décision, pas à corriger. CUE : `F-sgoiter-endian-fixed-le`.

---

## Régression mesurée au gate — émetteur du jour vs snapshot 20260810k

La régénération monocypher de ce jour (`regen_monocypher_dogfood.sh`, émetteur
rebuildé) produit 53 982 octets là où le snapshot versionné (commit `119d8c3`) en
portait 170 429, et la copie produit synchronisée fait échouer
`TestParityVsCCGO_Sizes/pt65_ad_empty` (MAC sgoiter ≠ ccgo) dans
`/devhoros/pkg/secretstream55/internal/monocypher_sgoiter/` — le KAT AEAD du script,
lui, passe. L'émetteur a donc régressé sur ce chemin entre le 10/08 22:27 et les
paliers du 11/08 (déterminisme, pliage rotations, nettoyage parenthèses, règles
do-while) ; le suspect n'est pas mesuré en face. L'émis audité ici est celui du
jour, copié en évidence stable :
`spec/dogfood/testdata/cycles/20260811_audit_fable/monocypher/out.go`.
Restauration des fichiers versionnés vs conservation de l'état frais : arbitrage
usager en attente.

## PLAUSIBLE — non tranché

- `Slide_step` (monocypher l.1528-1545) : retour du digit signé i8 zéro-étendu
  (251 au lieu de −5) selon le rapport ; le site de retour n'a pas été rejoué au
  sol. À trancher avant toute gravure.

## Déjà bon (calibrage, ne pas retravailler)

- Rotations pliées partout (`bits.RotateLeft32/64`) : blake2b 32/32, siphash 40,
  chacha20 4/4, murmur3, monocypher chemin chaud.
- Stubs bruyants conformes : les 8 stubs monocypher paniquent en nommant le
  symbole (l.1742-1779) ; stubs tweetnacl idem.
- Aliasing fidèle là où le C le tolère : fast_xor (tout recouvrement), tweetnacl.
- `-(v2 & 1)` (crc32) : wrap unsigned Go = sémantique C exacte.
- Étalons de forme : fnv1a_64, crc32_ieee (hors C11), chacha20_qr (hors C5/C12).
- La note TODO_NEXT « murmur garde `-int(v7)` » est périmée : l'émis du jour porte
  `v18 = -v7`, `v7` déjà `int` (out.go:31).

## Couverture d'oracle — trous mesurés

- KAT AEAD monocypher : n'exerce ni chacha20_ietf, ni verify32/64, ni crypto_wipe,
  ni la zone fe/eddsa (l.874-1631), ni la continuation poly1305 non alignée.
  Les six P1 monocypher vivent tous dans ces trous.
- libinjection_sqli : aucun oracle. Harnais C minimal faisable via `-Dstatic=` sur
  src.c + vecteurs NUL embarqués / délimiteurs doublés — détecterait C8 et C14.
- tweetnacl : `Crypto_hash` et `Neq25519`/`Par25519` au-dessus de stubs, jamais
  exécutées — C6 et C7 y sont invisibles par construction.
- Fixtures : md5 tronquée (C17) ; les vecteurs du banc ne dépassent jamais 2^32
  octets ni des limbes hors contrat (C10, C11 invisibles).

## Preuves rejouables (extraits)

```bash
# C1 — wipe avant return
grep -n 'range v1 { v1\[_i\] = 0 }' spec/dogfood/testdata/cycles/20260810k_monocypher/sgoiter_out/monocypher_aead_sgoiter.go
# C3 — précédence masque
grep -n '0xffffff) >> uint8(nb_mask)' spec/dogfood/testdata/cycles/20260810k_monocypher/sgoiter_out/monocypher_aead_sgoiter.go
# C8 — tautologie
grep -n 'cur\[0\]) == uint32(cur\[0\])' spec/dogfood/testdata/cycles/20260811_audit_fable/libinjection_sqli/out.go
# C11 — compteur rétréci
grep -n 'var v4 uint32' spec/dogfood/testdata/cycles/20260811_audit_fable/crc32_ieee/out.go
# C12 — moisson silencieuse
grep -c 'double_round' spec/dogfood/testdata/cycles/20260811_audit_fable/chacha20_qr/out.go   # 0
```
