# Note — banc stratifié C / ccgo / sgoiter

**Stamp probe :** `2026-08-12T20:23:46Z`  
**Artefacts :** `spec/bench/probe_stratified/` (`probe_report.json`, `PROBE.md`, `LAYERS.md`)  
**Bit-exact :** tribench **13/13** vs C (oracle `c_gcc_O2`)  
**Binaire :** `bin/sgoiter` HEAD post densify Gemini + base64/fnv (`e5c36f5`…`237f22d`)

---

## 1. Cadre

Mesure **in-process** par lib et par strate (tiny / tail / L1 / L2 / bulk / block), triangle :

| backend | rôle |
|---------|------|
| **c_gcc_O2** | oracle perf + bit-exact |
| **sgoiter** | Go émis kernel (`-mode kernel`) |
| **ccgo** | référence transpileur tiers |

Overhead `ov_empty` exclu des goulets « métier ». Workdir probes : `/tmp/sgoiter_probebench` (NVMe), pas `/data`.

Rejouer :

```bash
cd /devhoros/c2simd
GOWORK=off go build -o bin/sgoiter ./sgoiter/cmd/sgoiter
GOWORK=off go build -o bin/probebench ./sgoiter/cmd/probebench
./bin/probebench -root /devhoros/c2simd -sgoiter ./bin/sgoiter -ccgo "$(which ccgo)" \
  -work /tmp/sgoiter_probebench -out ./sgoiter/spec/bench/probe_stratified -skip-io -repeat 7
./bin/probebench -validate ./sgoiter/spec/bench/probe_stratified/probe_report.json -require-ccgo
```

---

## 2. Synthèse executive

| question | réponse (ce stamp) |
|----------|-------------------|
| sgoiter correct vs C ? | **oui** — 13/13 bit-exact |
| goulets sgo/C ≥ 1,5× ? | **1** : `fast_xor` / `tail_17` (**1,53×**) |
| sgoiter vs ccgo ? | sgo **devant ou égal** en médiane sur presque toutes les libs ; écart large sur nacl / strlenspn / chacha / poly / xor |
| où sgo **bat** C ? | xor bulk/L1 (**~0,66×**), strlenspn (**~0,25×**), poly_1 (**0,68×**), chacha 1m (**0,90×**) |

Le transpileur n’est plus en régime « partout 2× plus lent ». Le résidu est **local** (queues courtes, fixtures md5 réduites).

---

## 3. Tableau compact — pire strate sgo/C par lib

| lib | pire sgo/C | strate | médiane sgo/ccgo | commentaire |
|-----|------------|--------|------------------|-------------|
| fast_xor | **1,53×** | tail_17 | **0,63×** | bulk/L1 **fast** vs C ; queue 17 B seule ★ |
| md5_transform | 1,49× | block_64k | 0,94× | fixture 4×FF — pas le MD5 complet |
| base64_simd | 1,43× | tail_17 | 0,98× | string table + `dst[j:j+4]` ; bulk ~1,09× |
| md5_transform_full | 1,43× | block_64k | 0,99× | |
| murmur3_x86_32 | 1,40× | tiny_16 | 1,02× | |
| siphash24 | 1,37× | l1_4k | 0,98× | |
| blake2b_compress | 1,36× | block_1 | **0,84×** | ~143 L densifié ; sgo > ccgo |
| fnv1a_64 | 1,28× | tiny_16 | 0,99× | int index + subslice BCE ; bulk ~1,0× |
| crc32_ieee | 1,20× | tiny_16 | 0,99× | bit step 1 ligne ; bulk ~1,0× |
| chacha20_qr | 1,07× | qr_1 | **0,44×** | |
| poly1305_block5 | 0,99× | poly_1m | **0,52×** | poly_1 **0,68×** vs C |
| tweetnacl_dogfood | ~0,95× | ver_* | **0,27×** | |
| strlenspn_lab | **0,23–0,42×** | toutes | **0,19×** | dominant sgo |

Détail ligne à ligne : `probe_stratified/PROBE.md`.

---

## 4. Wins et dettes (lecture opposable)

### Ce qui tient

- **Absorptions IR** : fnv, murmur, blake sans override corps (overrides restants documentés `emit/OVERRIDES.md`).
- **Densify** : crc masque une ligne ; blake boucle lisible ; L32/R/Rotl32 → `bits.RotateLeft*`.
- **xor** : chemin SliceData 16 B + 8 B + switch queue — gagne le bulk, paye le tail.
- **Triangle** : sgoiter n’est plus le parent pauvre de ccgo sur le dogfood.

### Dettes ouvertes (ordre utile)

1. **xor tail_17** — seul ≥1,5× ; polish queue sans casser L1/bulk.
2. **base64 tail** — 1,43× ; LUT package testée puis abandonnée (régression L1).
3. **md5** reduced/full ~1,4× — en partie fixture / graphe FF.
4. **copy-prop live-range** safe en pipeline (unit-test only aujourd’hui).
5. **BCE subslice** généralisé hors fnv (Gemini rec. 3).

---

## 5. Dogfood code émis

Snapshots lisibles à l’œil :

| chemin | contenu |
|--------|---------|
| `spec/dogfood/.../dogfood_kernels_20260812_075100/` | premier lot post-drop overrides |
| `spec/dogfood/.../dogfood_kernels_20260812_e5c_rotl/` | post base64/fnv + Rotl module |

Re-émettre :

```bash
./bin/sgoiter -in spec/c_sources/testdata/c_sources/<k>.c -out /tmp/k.go -mode kernel
```

---

## 6. Critère « commit-ready » perf (rappel)

- `probebench -validate … -require-ccgo` vert  
- zéro strate métier avec **sgoiter error** / iters=0  
- goulets sgo/C ≥2× : **zéro** (atteint)  
- goulets ≥1,5× : **un** (xor tail) — acceptable comme dette nommée

---

*Note de session 2026-08-12 — joint au module sgoiter pour arbitrage et reprise.*
