# First-pass sgoiter — bibliothèques C additionnelles (2026-08-11)

Roster : **16** (12 tribench + 4 extra — [`../ccgo_pkg/ROSTER.md`](../ccgo_pkg/ROSTER.md)).  
Sous-projets **extra** (13–16) : `sgoiter-stb-image`, `sgoiter-cjson`, `sgoiter-yyjson`, `sgoiter-utf8proc`.

## Layout sources

| Canonique | Contenu |
|-----------|---------|
| `spec/c_sources/upstream/cjson/` | cJSON.c/h |
| `spec/c_sources/upstream/yyjson/` | yyjson.c/h |
| `spec/c_sources/upstream/utf8proc/` | utf8proc.c/h + utf8proc_data.c |
| `spec/c_sources/upstream/stb/` | stb_image.h + stb_image_impl.c |
| `spec/c_sources/testdata/c_sources/*_dogfood.c` | kernels subset émissibles |
| `c2simd/sources/` | dump brut — **ne plus enrichir** (voir README) |

## Front full upstream

| ID | exit | Diagnostic |
|----|------|------------|
| cJSON.c | 0 | emit ~30 L — structs + stubs (presque vide fonctionnel) |
| utf8proc.c | 0 | emit ~19 L — stubs (`...any`) ; data include non digéré |
| yyjson.c | 1 | **`err_asm`** inline assembly |
| stb_image_impl.c | 1 | **`err_empty`** après normalize (include STB non expansé) |

**Conclusion** : les 4 libs **complètes** ne sont **pas** encore candidats tribench/bit-exact.  
Le chantier correct est **dogfood par tranches** (comme monocypher), pas dump entier → bench.

## Dogfood kernels (emit OK)

| Kernel | Fichier | Fonctions |
|--------|---------|-----------|
| cjson_number_dogfood | testdata/… | `cjson_parse_u64_dogfood`, `cjson_is_null_lit_dogfood` |
| yyjson_digit_dogfood | testdata/… | `yyjson_count_digits_dogfood`, `yyjson_skip_ws_dogfood` |
| utf8_iterate_dogfood | testdata/… | `utf8_cl_dogfood`, `utf8_iterate2_dogfood` |
| stbi_crc_dogfood | testdata/… | `stbi_crc32_dogfood` |

Catalogue : `sgoiter/sgoiterbench.CatalogExtra`.  
Test : `go test ./sgoiter/ -run TestExtraLibsFrontPass`.

## Non fait (volontaire)

- Bench différentiel CGO/ccgo/sgoiter sur upstream full — **bloqué** tant que front ≠ subset viable
- Bit-exact tribench des 4 dogfood — **prochain palier** (harness + oracle C)
- Expansion include stb / strip asm yyjson — chantiers front séparés

## Prochaines itérations recommandées

1. Tribench kinds pour les 4 dogfood (oracle gcc -O2)
2. Front : `#include` local + skip/`__asm__` yyjson portable path
3. stb : préprocess amalgamation hors sgoiter puis ingest
4. Élargir dogfood utf8 3–4 bytes, cJSON string escape, etc.
