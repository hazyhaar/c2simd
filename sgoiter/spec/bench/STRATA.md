# Probebench stratifié — doctrine disques & goulets

## Piège machine (mesuré 2026-08-11)

| Montage | Device | Nature | Usage autorisé pour microbench |
|---------|--------|--------|--------------------------------|
| `/` | nvme0n1p2 | NVMe ext4 | OK (via `/tmp`) |
| `/devhoros` | nvme1n1p1 | NVMe ext4 **distinct** | OK code ; workdir CPU préférable `/tmp` |
| `/data` | sda | gros volume bulk | **INTERDIT** workdir probe CPU |

Comparer un binaire construit/écrit sur `/data` à un autre sur NVMe fausse CPU et RSS.

## Workdir

- Défaut : `/tmp/sgoiter_probebench` (même disque que `/`, NVMe).
- Jamais `/data/...` pour artefacts de compile/run.
- Le rapport commence toujours par la **carte disques** + probe I/O 64 MiB par montage.

## Strates logiques (par kind)

### Hash / sip / libinj
| id | phase | rôle |
|----|-------|------|
| ov_empty | overhead | coût d’appel |
| tiny_16 | setup | petit message |
| l1_1k / l1_4k | hot_l1 | chemin chaud L1 |
| l2_64k | hot_l2 | sortie L1 |
| bulk_1m | bulk | débit |

### xor / base64
tail_17 → align_64 → l1_1k → l2_64k → bulk_1m

### blake / md5
block_1 → block_1k → block_64k (répétition compress)

### chacha QR / poly / tweet verify
strates ALU / block serrées

## Métriques

| sonde | C | sgoiter |
|-------|---|---------|
| temps hot loop | `clock_gettime` MONOTONIC | `time.Now` in-process |
| débit | MB/s payload×iters | idem |
| RAM | `ru_maxrss`, minflt/majflt | `MemStats` allocs + heap_inuse |
| I/O machine | write+read 64 MiB sync par mount | séparé des probes kernel |

## Lancement

```bash
cd /devhoros/c2simd && export GOWORK=off
go build -o bin/sgoiter ./sgoiter/cmd/sgoiter
go build -o bin/probebench ./sgoiter/cmd/probebench
./bin/probebench -root /devhoros/c2simd -sgoiter ./bin/sgoiter \
  -out sgoiter/spec/bench/probe_latest
# option: -only fnv1a_64,crc32_ieee
```

## Micro-opt

1. Lire goulets `cpu` ratio ≥ 3.5× sur `hot_l1`/`block`.
2. Lire goulets `alloc` sur bulk.
3. Vérifier que work_disk n’est pas rotational.
4. Patch emit/front → re-probe **même** `-work` et mêmes strates.
