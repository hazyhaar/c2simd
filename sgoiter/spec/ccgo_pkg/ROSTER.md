# Roster HPM55 `ccgo-pkg` — 16 sous-projets sgoiter

> **Parent :** `ccgo-pkg`  
> **Règle de comptage :** **16 = 12 banc cœur (tribench) + 4 extra parsers/media**.  
> **Machine :** `roster.json` · **code :** `sgoiterbench.CatalogCCGOPkg`  
> **Date de formalisation :** 2026-08-11  
> **Source du « 16 » :** claim harvest AGY + alignement corpus `/devhoros/c2simd` (pas une liste HPM55 live — `hpm55.db` vide).

Doctrine : chaque sous-projet = **une lib / surface C** avec dogfood, oracle (si applicable), et statut front.  
sgoiter ≠ ccgo : fail-loud hors subset ; pas de promesse parity full upstream avant dogfood bit-exact.

---

## Vue d’ensemble

| # | HPM55 id | Famille | Surface C | Oracle tribench | Statut |
|---|----------|---------|-----------|:---------------:|--------|
| 01 | `sgoiter-fnv1a` | hash | `fnv1a_64.c` | oui | **banc vert** |
| 02 | `sgoiter-crc32` | hash | `crc32_ieee.c` | oui | **banc vert** |
| 03 | `sgoiter-fast-xor` | mem | `fast_xor.c` | oui | **banc vert** |
| 04 | `sgoiter-siphash` | hash | `siphash24.c` | oui | **banc vert** |
| 05 | `sgoiter-murmur3` | hash | `murmur3_x86_32.c` | oui | **banc vert** |
| 06 | `sgoiter-blake2b` | crypto | `blake2b_compress.c` | oui | **banc vert** |
| 07 | `sgoiter-chacha20` | crypto | `chacha20_qr.c` | oui | **banc vert** |
| 08 | `sgoiter-md5` | crypto | `md5_transform.c` | oui | **banc vert** (noyau tronqué volontaire) |
| 09 | `sgoiter-poly1305` | crypto | `poly1305_block5.c` | oui | **banc vert** |
| 10 | `sgoiter-base64` | codec | `base64_simd.c` | oui | **banc vert** |
| 11 | `sgoiter-tweetnacl` | crypto | `tweetnacl_dogfood.c` | oui (verify_16) | **banc vert** ; helpers Ld32/Pack **dette P0** |
| 12 | `sgoiter-libinjection` | sqli | cycle `libinjection_sqli` | **no_oracle** | emit OK ; multi-header |
| 13 | `sgoiter-cjson` | parse | upstream + `cjson_number_dogfood` | non | dogfood OK ; full stub |
| 14 | `sgoiter-yyjson` | parse | upstream + `yyjson_digit_dogfood` | non | dogfood OK ; full asm/stub |
| 15 | `sgoiter-utf8proc` | unicode | upstream + `utf8_iterate_dogfood` | non | dogfood OK ; full stub |
| 16 | `sgoiter-stb-image` | image | upstream + `stbi_crc_dogfood` | non | dogfood OK ; full fail include |

**Comptage :** 12 cœur + 4 extra = **16**. Rien d’autre n’entre dans le quota sans retirer une entrée.

---

## Détail — cœur 01–12 (tribench)

Chemins C relatifs à `spec/c_sources/testdata/c_sources/` sauf libinjection (cycle dogfood).

| HPM55 | Tribench ID | CFunc / SgoFunc | Notes / dettes |
|-------|-------------|-----------------|----------------|
| `sgoiter-fnv1a` | `fnv1a_64` | `fnv1a_64` | stable |
| `sgoiter-crc32` | `crc32_ieee` | `crc32_ieee` | mask lisible = P1 forme |
| `sgoiter-fast-xor` | `fast_xor` | `fast_xor_bytes` | |
| `sgoiter-siphash` | `siphash24` | `siphash24` | |
| `sgoiter-murmur3` | `murmur3_x86_32` | `murmur3_x86_32` | ROT variable = P1 |
| `sgoiter-blake2b` | `blake2b_compress` | `blake2b_compress_block` | |
| `sgoiter-chacha20` | `chacha20_qr` | `chacha20_quarter_round` | |
| `sgoiter-md5` | `md5_transform` | `md5_transform_block` | élargir `.c` si RFC complète |
| `sgoiter-poly1305` | `poly1305_block5` | `poly1305_block5` | |
| `sgoiter-base64` | `base64_simd` | `base64_encode_stream` | padding `'='` = P1 |
| `sgoiter-tweetnacl` | `tweetnacl_dogfood` | `crypto_verify_16` | **P0** Ld32 typage ; Pack25519 stub |
| `sgoiter-libinjection` | `libinjection_sqli` | `strlenspn` (cycle) | `SkipC` ; headers stub |

Catalogue exécutable : `sgoiter/tribench/catalog.go`.

---

## Détail — extra 13–16

| HPM55 | Upstream | Dogfood | Full front | Doc |
|-------|----------|---------|------------|-----|
| `sgoiter-cjson` | DaveGamble/cJSON | `cjson_number_dogfood` | stub ~30 L | `upstream/cjson/PROVENANCE.md` |
| `sgoiter-yyjson` | ibireme/yyjson | `yyjson_digit_dogfood` | barrier/stub | `upstream/yyjson/PROVENANCE.md` |
| `sgoiter-utf8proc` | JuliaStrings/utf8proc | `utf8_iterate_dogfood` | stub ; data | `upstream/utf8proc/PROVENANCE.md` |
| `sgoiter-stb-image` | nothings/stb | `stbi_crc_dogfood` | err_include | `upstream/stb/PROVENANCE.md` |

Chantier produit : `sgoiter/TODO_EXTRA_LIBS.md` · first-pass : `sgoiter/spec/extra_libs/FIRST_PASS_20260811.md`.

---

## Hors roster (captés, **non** dans les 16)

Ne pas les compter dans le « 16 » sans arbitrage HPM55 explicite (swap ou extension du parent).

| Id proposé | Corpus | Motif hors quota |
|------------|--------|------------------|
| `sgoiter-monocypher` | `upstream/monocypher/4.0.2/` | full crypto stack ; dogfood/RFC à part ; emit large |
| `sgoiter-simsimd` | `upstream/SimSIMD/` | SIMD/dispatch multi-ISA ; hors corridor sgoiter |
| `sgoiter-ann-lab` | `ann_lab/*.c` | float/FHT ; front fail |
| *(adv_*)* | `adv_*.c` | tests adversariaux producteur, pas libs produit |

PROVENANCE hors roster : `upstream/monocypher/PROVENANCE.md`, `upstream/SimSIMD/PROVENANCE.md`, `ann_lab/PROVENANCE.md`.

---

## Manquants formalisés (avant ce fichier)

| Élément | Avant | Après |
|---------|-------|-------|
| Noms HPM55 des 12 cœur | absents (seuls 4 extra) | table 01–12 ci-dessus + `roster.json` |
| Lien tribench ID ↔ HPM55 | implicite | 1:1 dans roster |
| Hors roster nommé | monocypher/SimSIMD orphelins | section dédiée + PROVENANCE |
| Claim « 16 » | non inventorié (`hpm55.db` vide) | règle 12+4 opposable |

---

## Gates par sous-projet

| Famille | Gate minimum |
|---------|----------------|
| 01–11 | `tribench` bit-exact vs gcc -O2 sur surface cataloguée |
| 12 | emit + compile Go ; pas d’oracle C obligatoire |
| 13–16 dogfood | `TestExtraLibsFrontPass` + (cible) oracle C dédié |
| 13–16 full | front rapport honnête (stub/fail) — **pas** parity promise |
| Cœur global | `go test ./sgoiter/...` + `tribench -skip-ccgo` **11/11 + 1 no_oracle** |

---

## Enregistrement HPM55 (ops)

Quand `hpm55.db` est de nouveau utilisable :

```
parent  : ccgo-pkg
enfants : les 16 id de la table (slug = colonne HPM55)
hors    : monocypher, simsimd, ann-lab (parent ccgo-pkg OU projet frère — arbitrage humain)
```

Reminder par enfant : lien `ROSTER.md` + cycle dogfood + dettes P0/P1 si crypto.

---

## Interdits

- Ajouter un 17ᵉ sans retirer ou sans bump explicite du quota parent.
- Confondre **pôle modernc/ccgo upstream** et ce roster **sgoiter**.
- Promettre full cJSON/yyjson/stb/utf8proc avant dogfood bit-exact + front A0–A5 (`TODO_EXTRA_LIBS.md`).
```
