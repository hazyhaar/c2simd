# TODO — 4 libs HPM55 extra (cjson / yyjson / utf8proc / stb)

> Parent HPM55 : `ccgo-pkg`  
> Roster 16 : `spec/ccgo_pkg/ROSTER.md` · `roster.json` · `sgoiterbench.CatalogCCGOPkg`  
> Sous-projets extra : `sgoiter-cjson`, `sgoiter-yyjson`, `sgoiter-utf8proc`, `sgoiter-stb-image`  
> First-pass : `sgoiter/spec/extra_libs/FIRST_PASS_20260811.md`  
> Catalogue : `sgoiterbench.CatalogExtra`  
> Gate cœur : **ne pas casser** `tribench -skip-ccgo` (**13/13** au 2026-08-13)

Doctrine : **dogfood par tranches** → front manquant → oracle → élargir.  
Pas de bench CGO full tant que front full = stub / harvest mince.

---

## État (housekeep 2026-08-13)

| Lib | Full upstream | Dogfood mince | Mesure emit full |
|-----|:-------------:|:-------------:|------------------|
| cJSON | stub mince | `cjson_number_dogfood` **OK+KAT** | ~**30** L (harvest silencieux / structs) |
| yyjson | emit partiel + stubs | `yyjson_digit_dogfood` **OK+KAT** | ~**153** L ; stubs `Fread_s, Free, U128_*` — **plus `err_asm`** (barriers strippées) |
| utf8proc | stub | `utf8_iterate_dogfood` **OK+KAT** | ~**12** L + stub Free |
| stb_image | **err_empty** | `stbi_crc_dogfood` **OK+KAT** | amalgamation / include data |

Tests verts : `TestExtraLibsFrontPass`, `TestExtraLibsDogfoodKATs`, `TestIncludeLocal`.

---

## Voie A — Front producteur

### A0 — Préprocesseur

| # | Item | État |
|---|------|------|
| A0.1 | `#include "file.h"` local | **partiel** — `TestIncludeLocal` vert ; stb full encore `err_empty` |
| A0.2 | `#define` / `#ifdef` utiles | ouvert |
| A0.3 | amalgamation stb optionnelle | ouvert |
| A0.4 | test include chain 2 fichiers | **soldé** (`TestIncludeLocal`) |

### A1 — Asm & chemins portables

| # | Item | État |
|---|------|------|
| A1.1 | `__asm__` barrier vide no-op | **soldé** — yyjson full émet |
| A1.2 | asm non vide → `err_asm` | **soldé** (fail-loud) |
| A1.3 | doc SPEC | à vérifier / sync si besoin |
| A1.4 | test barrier + add | couvert via extra / front |

### A2 — Harvest honnête — **ouvert**

| # | Item |
|---|------|
| A2.1 | compteur `skipped[]` par raison |
| A2.2 | `-strict-harvest` si ratio skip &gt; seuil |
| A2.3 | log 1 ligne / func skip |

### A3 — Contrôle de flux parsers — **ouvert**

goto simple · switch dense · break/continue gaps

### A4 — Mémoire & API C — **ouvert**

malloc/free · callbacks bornés · FILE\* / STBI_NO_STDIO · float subset

### A5 — Structs & pointeurs — **ouvert**

listes cJSON · buffers · tables const larges (utf8proc_data)

---

## Voie B — Dogfood élargi

Chaque item : `.c` testdata + emit OK + **oracle** + 13/13 cœur intact.

### Présents (B*.1) — **faits**

`cjson_number_dogfood` · `yyjson_digit_dogfood` · `utf8_iterate_dogfood` · `stbi_crc_dogfood`

### Manquants (fichiers absents sous testdata)

| # | Kernel | Contenu |
|---|--------|---------|
| B1.2 | `cjson_string_dogfood` | escape `\"` `\\` `\n` |
| B1.3 | `cjson_skip_ws` | espaces JSON |
| B2.2 | `yyjson_lit_dogfood` | true/false/null |
| B2.3 | `yyjson_str_simple` | string simple |
| B3.2–3 | utf8 3–4 bytes + encode | |
| B4.2–3 | `stbi_paeth` · PNG 1×1 | |

B*.4 full amputé : après A0/A1/A2.

---

## Voie C — Banc & HPM55 (après B bit-exact)

C1 kinds tribench extra · C2 wall/ns · C3 ccgo optionnel · C4 reminder HPM55  
**C5 interdit** : promettre parity full avant A0–A5.

---

## Ordre reprise

```
1. B1.2 B1.3 B2.2 B2.3 — dogfood + KAT (quick)
2. A2 harvest honnête (cJSON 30 L ne doit plus mentir)
3. A0 stb amalgamation / include data
4. A3–A5 parsers seulement si besoin d'un full amputé
5. C1 tribench extra
```

---

## Gates

```bash
cd /devhoros/c2simd && export GOWORK=off
go test ./sgoiter/... -count=1
./bin/tribench -root /devhoros/c2simd -sgoiter ./bin/sgoiter -skip-ccgo -skip-bench   # 13/13
go test ./sgoiter/ -run 'TestExtraLibs' -count=1
```

---

## Hors scope

- Fork ccgo pour ces libs  
- SIMD/asm réel yyjson/stb  
- Runtime secretstream (projet HPM55 `secretstream_go`, hors ce TODO)  
- Cloner repos sans tranche dogfood  

## Liens

| Ressource | Chemin |
|-----------|--------|
| Sources full (local) | `spec/c_sources/upstream/{cjson,yyjson,utf8proc,stb}/` |
| Dogfood | `spec/c_sources/testdata/c_sources/*_dogfood.c` |
| Qualité emit cœur | `TODO_NEXT.md` |
| Perf | `TODO_NIGHT.md` |
