# Dogfood kernels — lecture à l’œil post-62ab7c5

stamp emit: 2026-08-12T07:51:00Z  
tribench: 13/13 bit-exact (`tribench_20260812_095109`)  
binaire: `bin/sgoiter` rebuild depuis HEAD `62ab7c5`

## Corpus (21 fichiers)

| kernel | lignes | verdict lecture |
|--------|-------:|-----------------|
| fnv1a_64 | 27 | **propre** — ×8 IR + queue ; primes hex |
| fast_xor | 49 | **propre** — SliceData 16B+8B+switch fallthrough |
| murmur3 | 61 | **lisible** — j forward, LE load ; wrapper `Rotl32` encore présent |
| chacha20_qr | 30 | **excellent** — 4× RotateLeft32 |
| blake2b | 5675 | **correct mais monstrueux** — 12 rounds unrolled, RotateLeft64 OK, reloads v8[i], `int(sigma)` |
| base64 | 36 | **propre** — table + tail `=` ; goulet perf tail seul ★ |
| strlenspn | 20 | **propre** — prédicat h/e/l |
| md5 reduced | 43 | override compact 4×FF |
| md5_full | 291 | long structuré |
| siphash | 68 | override dense, lisible |
| crc32 | 96 | bit-serial unrolled ×8/byte — verbeux mais OK |
| poly1305 | 54 | OK |
| tweetnacl | 153 | stub Pack25519 |

## Findings œil

1. **blake 5.6k** — bit-exact ; dogfood humain seulement par échantillon ; densifier (sigma littéral, moins de copies vN).
2. **murmur Rotl32** — `func Rotl32` + calls encore là (archInline n’a pas tout mangé sur ce chemin).
3. **crc32** — masque `-(crc&1)` encore en multi-temps ; perf ~1.0× donc non prioritaire.
4. **base64** — seule ★ perf (tail_17 ~1.8×) ; code OK.
5. **fnv / xor / chacha** — qualité cible pour prochains absorbs.

## Stub emit

```
1 symbol stubbed: Pack25519 (tweetnacl)
```
