# Roster des Projets de Transpilation C — 16 modules sgoiter

> **Règle de classification :** **16 = 12 banc cœur (tribench) + 4 extra parsers/media**.  
> **Spécification machine :** `roster.json` · **Catalogue Go :** `sgoiterbench.CatalogCCGOPkg`  
> **Date :** 2026-08-11  

Doctrine : chaque sous-projet correspond à **une bibliothèque ou surface C** testée via banc dogfood, oracle formel et validation du frontend de transpilation.

---

## Vue d’ensemble

| # | Identifiant | Famille | Surface C | Oracle tribench | Statut |
|---|-------------|---------|-----------|:---------------:|--------|
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
| 11 | `sgoiter-tweetnacl` | crypto | `tweetnacl_dogfood.c` | oui (verify_16) | **banc vert** ; helpers Ld32/Pack |
| 12 | `sgoiter-libinjection` | sqli | cycle `libinjection_sqli` | **no_oracle** | emit OK ; multi-header |
| 13 | `sgoiter-cjson` | parse | upstream + `cjson_number_dogfood` | non | dogfood OK ; full stub |
| 14 | `sgoiter-yyjson` | parse | upstream + `yyjson_digit_dogfood` | non | dogfood OK ; full asm/stub |
| 15 | `sgoiter-utf8proc` | unicode | upstream + `utf8_iterate_dogfood` | non | dogfood OK ; full stub |
| 16 | `sgoiter-stb-image` | image | upstream + `stbi_crc_dogfood` | non | dogfood OK ; full fail include |

---

## Détail — Cœur 01–12 (tribench)

Chemins C relatifs à `spec/c_sources/testdata/c_sources/` (sauf libinjection en cycle dogfood).

| Identifiant | Tribench ID | CFunc / SgoFunc | Notes |
|-------------|-------------|-----------------|-------|
| `sgoiter-fnv1a` | `fnv1a_64` | `fnv1a_64` | stable |
| `sgoiter-crc32` | `crc32_ieee` | `crc32_ieee` | masque lisible |
| `sgoiter-fast-xor` | `fast_xor` | `fast_xor_bytes` | |
| `sgoiter-siphash` | `siphash24` | `siphash24` | |
| `sgoiter-murmur3` | `murmur3_x86_32` | `murmur3_x86_32` | |
| `sgoiter-blake2b` | `blake2b_compress` | `blake2b_compress_block` | |
| `sgoiter-chacha20` | `chacha20_qr` | `chacha20_quarter_round` | |
| `sgoiter-md5` | `md5_transform` | `md5_transform_block` | |
| `sgoiter-poly1305` | `poly1305_block5` | `poly1305_block5` | |
| `sgoiter-base64` | `base64_simd` | `base64_encode_stream` | |
| `sgoiter-tweetnacl` | `tweetnacl_dogfood` | `crypto_verify_16` | |
| `sgoiter-libinjection` | `libinjection_sqli` | `strlenspn` (cycle) | `SkipC` ; headers stub |

Catalogue exécutable : `sgoiter/tribench/catalog.go`.

---

## Détail — Extra 13–16

| Identifiant | Upstream | Dogfood | Frontend | Documentation |
|-------------|----------|---------|----------|---------------|
| `sgoiter-cjson` | DaveGamble/cJSON | `cjson_number_dogfood` | stub ~30 L | `upstream/cjson/PROVENANCE.md` |
| `sgoiter-yyjson` | ibireme/yyjson | `yyjson_digit_dogfood` | barrier/stub | `upstream/yyjson/PROVENANCE.md` |
| `sgoiter-utf8proc` | JuliaStrings/utf8proc | `utf8_iterate_dogfood` | stub ; data | `upstream/utf8proc/PROVENANCE.md` |
| `sgoiter-stb-image` | nothings/stb | `stbi_crc_dogfood` | err_include | `upstream/stb/PROVENANCE.md` |

---

## Hors roster

| Identifiant | Corpus | Motif |
|-------------|--------|-------|
| `sgoiter-monocypher` | `upstream/monocypher/4.0.2/` | Stack cryptographique complète ; bancs RFC dédiés |
| `sgoiter-simsimd` | `upstream/SimSIMD/` | Dispatch multi-architecture matériel |
| `sgoiter-ann-lab` | `ann_lab/*.c` | Algorithmes vectoriels en virgule flottante |

---

## Gates de validation

| Famille | Exigence de validation |
|---------|------------------------|
| 01–11 | `tribench` bit-exact vs gcc -O2 sur la surface cataloguée |
| 12 | Émission + compilation Go ; pas d’oracle C obligatoire |
| 13–16 dogfood | `TestExtraLibsFrontPass` + oracle C dédié |
| 13–16 full | Rapport frontend honnête (stub/fail) |
| Cœur global | `go test ./sgoiter/...` + `tribench -skip-ccgo` (11/11 bit-exact + 1 sans oracle) |
