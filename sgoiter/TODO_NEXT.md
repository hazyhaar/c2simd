# TODO_NEXT — reste qualité (housekeep 2026-08-13)

Gate permanent : `go test ./sgoiter/... -count=1` et, pour le banc,
**tout noyau disposant d'un oracle C doit être bit-exact**.

Mesure opposable (2026-08-13T06:16Z) :

```
./bin/tribench -root /devhoros/c2simd -sgoiter ./bin/sgoiter -skip-ccgo -skip-bench
# 13/13 compared bit-exact vs C ; 1 kernel sans oracle C : libinjection_sqli
```

Doctrine : patch emit → relu yeux du noyau touché → tribench → cycle si palier.
Métrique `int(` : **toujours** `grep -o 'int(' | wc -l` (jamais `int(v` seul).

---

## Soldé (ne pas rouvrir sans régression mesurée)

| # | Sujet | Preuve au sol |
|---|---|---|
| Q3 | copies d'identité siphash | 0 id sur corpus |
| Q4 | fixtures base64 pad0–3 | `TestBase64FixturesMod3` |
| Q6 | pack RFC monocypher | `TestMonocypherRFCPack` + `TestMonocypherCOracleOnPack` 6/6 |
| Q10 | conversions d'index | blake **17** `int(` (&lt;20) ; `TestBlakeIndexConversionsUnderTarget` |
| S2 | barrières `__asm__` vides | strip + yyjson full émet |
| S3 | `#include` local | `TestIncludeLocal` |
| T1 | identity cast | landed |
| T2 | shift lit bare | landed |
| T5 | postinc → store;inc | landed |
| T6 | rodata array / const | landed (+ garde `globalsPassedToCalls`) |
| T11 | harvest roots opt-in | `-roots` |
| T13 | Q10 index int | landed (`emit/indexonly.go`) |
| T16 | narrow shift+mask index | landed |
| Masque CRC 1 ligne | `foldNegatedMask` | crc 21 L |
| RotL module | `archInlineRotlWrappers` | blake 32× `RotateLeft64` |
| Bit-exact cœur | tribench | **13/13** (2026-08-13) |
| Overrides fnv/murmur/blake | droppés | session 2026-08-12 |

Snapshots dogfood : `spec/dogfood/.../dogfood_kernels_20260812_e5c_rotl/`.  
Notes session : `spec/bench/NOTE_OPTIMS_SESSION_20260812.md`, `NOTE_STRATIFIED_20260812.md`.

---

## Mesures emit fraîches (2026-08-13, `./bin/sgoiter -in …`)

| noyau | lines | `int(` | RotateLeft* |
|---|---:|---:|---:|
| fnv1a_64 | 25 | 1 | 0 |
| crc32_ieee | 21 | 1 | 0 |
| fast_xor | 22 | 1 | 0 |
| base64_simd | 39 | 1 | 0 |
| chacha20_qr | 29 | 0 | 4 |
| md5_transform | 42 | 0 | 4 |
| siphash24 | 67 | 2 | 32 |
| murmur3_x86_32 | 56 | 2 | 3 |
| poly1305_block5 | 53 | 4 | 0 |
| blake2b_compress | 141 | **17** | 32 |
| tweetnacl_dogfood | 115† | 5 | 4 |

† tribench peut compter plus de lignes selon roots/mode ; bit-exact OK.

---

## Ouvert — forme (P2/P3)

| Id | Sujet | P | Notes |
|----|-------|---|-------|
| T15 | loop-cond-combine | P2 | **partiel** — gardes avec deps corps non remontables (cjson) |
| T3–T4 | CRC parens, `&^7` hoist | P2 | harvest yeux |
| T7–T8 | stack arr forme, rot inline résidu | P2 | |
| T9–T10 | poly×5 forme ; BE load | P2/P3 | |
| T12 | chacha ABI no | P3 | |
| T14 | typedef dogfood | — | |
| P3 | parens `&&` associatifs | P3 | lisibilité seule |

Perf (débit vs C) → **`TODO_NIGHT.md`**.  
Libs extra (cjson/yyjson/utf8/stb) → **`TODO_EXTRA_LIBS.md`**.  
Capacités banc → **`TODO_TRIBENCH_FINDINGS.md`** (stable).

---

## Ordre recommandé (reprise)

```
1. TODO_NIGHT — perf mesurable (base64 tail, xor tail, md5, live-range)
2. TODO_EXTRA — B*.2 dogfood élargi (pas full upstream)
3. Queue forme T3–T10 / parens P3 si session « yeux »
```

---

## Gates

```bash
cd /devhoros/c2simd && export GOWORK=off
go test ./sgoiter/... -count=1
go build -o bin/sgoiter ./sgoiter/cmd/sgoiter
go build -o bin/tribench ./sgoiter/tribench/cmd/tribench
./bin/tribench -root /devhoros/c2simd -sgoiter ./bin/sgoiter -skip-ccgo -skip-bench
# exit 0 ; « 13/13 compared » ; libinjection annoncé no-oracle
./sgoiter/scripts/ci_check.sh
```
