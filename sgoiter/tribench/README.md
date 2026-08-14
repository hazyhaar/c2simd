# tribench — banc C × ccgo × sgoiter

Compare **bit-exact** (stdout fixtures) et **observabilité** pour les 12 kernels dogfood.

## Backends

| id | description |
|----|-------------|
| `c_gcc_O0` | harness C + kernel, `gcc -O0` |
| `c_gcc_O2` | idem `gcc -O2` (**oracle** par défaut) |
| `sgoiter` | `sgoiter -in kernel.c` + harness Go idiomatique |
| `ccgo` | `ccgo kernel.c` + harness `modernc.org/libc` TLS |

## Fixtures

Vecteurs déterministes par `Kind` (`empty`, `hello`, `16b`, `17b`, `64b`, `1k`, `64k`, …).  
Protocole stdout : `name digest` une ligne par fixture. Concordance = SHA256(stdout) identique à l'oracle.

## Observabilité (par backend)

- `compile_ms`, `binary_bytes`, `run_ms`
- `stdout_sha256`, `lines` (map fixture→digest)
- `match_oracle`
- `code_lines` (Go généré)
- `bench_ns_per_op` (boucle process, coarse)
- hook `pprof_cpu` path

## Usage

```bash
cd /devhoros/c2simd
go build -o bin/sgoiter ./sgoiter/cmd/sgoiter
go run ./sgoiter/tribench/cmd/tribench -root /devhoros/c2simd -v

# sous-ensemble
go run ./sgoiter/tribench/cmd/tribench -only fnv1a_64,fast_xor,crc32_ieee

# sans ccgo
go run ./sgoiter/tribench/cmd/tribench -skip-ccgo
```

Rapport : `$out/report.json` + `SUMMARY.md` + un dossier par lib.

## Doctrine

Oracle = **gcc -O2** (comportement C). sgoiter et ccgo sont jugés contre cet oracle, pas l'un contre l'autre.
