# Sources C téléchargées pour dogfood c2simd (2026-08-10)

**Hors périmètre :** aucun code d'exploit, shellcode, C2, brute-force, ou outillage red-team offensif.

| Fichier | Origine | Licence |
|---------|---------|---------|
| tweetnacl_dogfood.c + tweetnacl.h | https://tweetnacl.cr.yp.to/20140427/ ( + stub randombytes local) | public domain (tweetnacl) |
| (lab) crc32/chacha_qr/fnv/poly1305_block5 | écrits session | lab dogfood |

## Refus pentest/red-team

Le dogfood c2simd vise le **pipeline de transpile** (rotates, tls, endian, builds).  
Les sources offensives n'apportent pas de motifs AST utiles et sortent du cadre autorisé.

## 20260810f

| Fichier | Origine | Licence | Note |
|---------|---------|---------|------|
| libinjection/* | github.com/libinjection/libinjection v3.10.0 | BSD | détection SQLi **défensive** |
| adv_*.c | lab session | lab | motifs adversariaux transpile, **pas** d'exploit |

Refus explicite : dumps Metasploit, shellcode packs, C2, exploit-db comme corpus dogfood.

## 20260810g — extrême légitime

| Projet | URL tag | Licence typique | Usage dogfood |
|--------|---------|-----------------|---------------|
| cJSON | DaveGamble/cJSON v1.7.18 | MIT | JSON parse |
| lz4 | lz4/lz4 v1.9.4 | BSD-2 | compression |
| tiny-regex-c | kokke/tiny-regex-c | Unlicense | regex |
| mpc | orangeduck/mpc | MIT | parser combinators AST |

Cycles sous `spec/dogfood/cycles/20260810g/` (raw/opt générés, pas recopiés en testdata).
