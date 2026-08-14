# Probebench LAYERS — flux + strates + goulets

Stamp: **2026-08-13T07:11:24Z**  
Workdir: `/tmp/sgoiter_probebench → / (/dev/nvme0n1p2, ext4, SSD/NVMe)`  
Backends: `c_gcc_O2` · `sgoiter` · `ccgo` (in-process).  
Ratios : sgo/C, ccgo/C, sgo/ccgo (>1 = numérateur plus lent).  

## Légende des phases

| Phase | Ce qu'elle isole |
|-------|------------------|
| overhead | Coût d'appel / len=0 — **bruit** si on parle kernel |
| setup / tiny | Petits messages : setup + prologue dominent |
| hot_l1 / block | Noyau chaud, working set ≤ L1 |
| hot_l2 | Traverse L1, encore CPU-bound souvent |
| bulk | Streaming 1 MiB : bande passante + boucle |
| tail | Longueurs non alignées / queue scalaire |

Doctrine : comparer **même strate**, ne pas cherry-picker le max ; optim = peephole fidèle au .c.

## Baseline B0 (pré vague A overrides)

Référence historique stamp `2026-08-11T22:31:38Z` / commit `49fc1fc` — ratios sgo/C rep. : Vn 2.83× · xor 2.41× · md5 2.12× · blake 1.71× · murmur 1.24× · siphash 1.07×.

Ce rapport (stamp courant) est le **B1+** post-overrides ; lire les ratios ci-dessous comme Δ implicite vs B0.

## Synthèse triangle C / sgoiter / ccgo — strate représentative

| lib | strate | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | verdict sgo vs C |
|-----|--------|------|--------|---------|-------|--------|----------|------------------|
| fnv1a_64 | l1_1k | 730.0 | 726.6 | 741.5 | **1.00×** | 1.02× | 0.98× | parité |
| crc32_ieee | l1_1k | 6000.9 | 6158.9 | 6019.6 | **1.03×** | 1.00× | 1.02× | parité |
| fast_xor | l1_1k | 64.1 | 32.7 | 77.4 | **0.51×** | 1.21× | 0.42× | sgo plus rapide |
| siphash24 | l1_1k | 276.4 | 299.3 | 317.3 | **1.08×** | 1.15× | 0.94× | parité |
| murmur3_x86_32 | l1_1k | 199.3 | 257.1 | 258.2 | **1.29×** | 1.30× | 1.00× | écart modéré |
| blake2b_compress | block_1k | 108.0 | 148.3 | 176.1 | **1.37×** | 1.63× | 0.84× | écart modéré |
| chacha20_qr | qr_1m | 2.7 | 2.5 | 5.9 | **0.93×** | 2.18× | 0.43× | sgo plus rapide |
| md5_transform | block_1k | 8.3 | 11.2 | 11.8 | **1.34×** | 1.42× | 0.95× | écart modéré |
| poly1305_block5 | poly_1m | 10.1 | 9.1 | 16.1 | **0.91×** | 1.60× | 0.57× | sgo plus rapide |
| base64_simd | l1_1k | 377.6 | 456.7 | 487.3 | **1.21×** | 1.29× | 0.94× | écart modéré |
| tweetnacl_dogfood | ver_eq | 2.5 | 1.9 | 7.8 | **0.77×** | 3.13× | 0.25× | sgo plus rapide |
| strlenspn_lab | l1_1k | 3.8 | 1.0 | 5.1 | **0.27×** | 1.36× | 0.20× | sgo plus rapide |
| md5_transform_full | block_1k | 78.5 | 112.0 | 112.2 | **1.43×** | 1.43× | 1.00× | écart modéré |

---

## FNV-1a 64-bit — `fnv1a_64`

**Contrat :** hash octet-par-octet h = (h^b)*prime ; prime fixe 1099511628211

### Flux de calcul

1. Entrée : buffer data[0..len)
2. Init h = offset_basis 14695981039346656037
3. Pour chaque octet : h ^= b ; h *= FNV_prime
4. Sortie : u64

**Emit sgoiter :** Override emit : unroll ×8 (même sémantique XOR*prime) + queue ; fStmts nil (pas de double corps).

**Lecture strates :** ov_empty = coût d'appel ; tiny = setup ; l1/l2/bulk = débit streaming octets.

### Couches (toutes strates)

| stratum | phase | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | lecture |
|---------|-------|------|--------|---------|-------|--------|----------|----------|
| ov_empty | overhead | 0.9 | 2.0 | 0.5 | 2.34× | 0.52× | 4.48× | GOULET, bruit appel |
| tiny_16 | setup | 5.3 | 7.0 | 8.5 | 1.33× | 1.59× | 0.83× | sgo +lent |
| l1_1k | hot_l1 | 730.0 | 726.6 | 741.5 | 1.00× | 1.02× | 0.98× | parité |
| l1_4k | hot_l1 | 2887.7 | 3017.7 | 2963.1 | 1.05× | 1.03× | 1.02× | parité |
| l2_64k | hot_l2 | 46470.4 | 46865.4 | 47204.3 | 1.01× | 1.02× | 0.99× | parité |
| bulk_1m | bulk | 743904.6 | 754803.7 | 756477.2 | 1.01× | 1.02× | 1.00× | parité |

**Goulet dominant (cette lib, hors overhead) :** `tiny_16` — ratio **1.33×** (sgo 7.0 vs C 5.3 ns/op), phase `setup`.

**Leviers fidèles (pas de changement d'algo) :**
- Unroll ×16 si gain mesuré
- BCE data[:len] si compteur int

---

## CRC-32 IEEE (bit-wise) — `crc32_ieee`

**Contrat :** poly 0xEDB88320, 8 tours de bit par octet — PAS de table, PAS de hardware CRC

### Flux de calcul

1. crc = 0xFFFFFFFF
2. Pour chaque octet : crc ^= b ; 8× { mask = -(crc&1) ; crc = (crc>>1) ^ (poly & mask) }
3. return ~crc

**Emit sgoiter :** Fidèle au .c bit-à-bit (boucle intérieure 8 bits).

**Lecture strates :** Débit plafonné ~130–140 MB/s des deux côtés = parité algo, pas défaut d'emit.

### Couches (toutes strates)

| stratum | phase | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | lecture |
|---------|-------|------|--------|---------|-------|--------|----------|----------|
| ov_empty | overhead | 0.9 | 0.3 | 1.8 | 0.31× | 2.00× | 0.16× | sgo gagne, bruit appel |
| tiny_16 | setup | 91.3 | 91.5 | 92.8 | 1.00× | 1.02× | 0.99× | parité |
| l1_1k | hot_l1 | 6000.9 | 6158.9 | 6019.6 | 1.03× | 1.00× | 1.02× | parité |
| l1_4k | hot_l1 | 24123.0 | 24454.0 | 24225.4 | 1.01× | 1.00× | 1.01× | parité |
| l2_64k | hot_l2 | 385499.0 | 397080.2 | 389680.2 | 1.03× | 1.01× | 1.02× | parité |
| bulk_1m | bulk | 6149115.5 | 6287015.6 | 6173029.1 | 1.02× | 1.00× | 1.02× | parité |

**Leviers fidèles (pas de changement d'algo) :**
- Aucun en transpile ; table slicing-by-8 = autre oracle C

---

## XOR bulk octets — `fast_xor`

**Contrat :** dst[i] = s1[i]^s2[i] pour i in 0..len

### Flux de calcul

1. Boucle principale par pas de 8 : load u64 s1, s2 → store u64 dst
2. Queue scalaire pour len%8

**Emit sgoiter :** LittleEndian.Uint64/PutUint64 déjà ; masque align &^7 retiré si safe.

**Lecture strates :** tail_17 = queue ; align_64/l1 = ALU+store ; l2/bulk = bande passante mémoire.

### Couches (toutes strates)

| stratum | phase | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | lecture |
|---------|-------|------|--------|---------|-------|--------|----------|----------|
| tail_17 | tail | 2.3 | 3.4 | 5.2 | 1.50× | 2.30× | 0.65× | sgo +lent |
| align_64 | hot_l1 | 5.0 | 5.0 | 6.2 | 1.00× | 1.23× | 0.82× | parité |
| l1_1k | hot_l1 | 64.1 | 32.7 | 77.4 | 0.51× | 1.21× | 0.42× | sgo gagne |
| l2_64k | hot_l2 | 2512.9 | 2302.4 | 3406.7 | 0.92× | 1.36× | 0.68× | sgo gagne |
| bulk_1m | bulk | 51440.7 | 43953.0 | 63646.4 | 0.85× | 1.24× | 0.69× | sgo gagne |

**Goulet dominant (cette lib, hors overhead) :** `tail_17` — ratio **1.50×** (sgo 3.4 vs C 2.3 ns/op), phase `tail`.

**Point fort :** strate `l1_1k` sgoiter **plus rapide** que C (ratio 0.51×).

**Leviers fidèles (pas de changement d'algo) :**
- Unroll ×4 mots (32 B/tour) fidèle
- Pas d'AVX hors source C

---

## SipHash-2-4 — `siphash24`

**Contrat :** PRFkeyed 64-bit, 2 compression rounds / 4 finalization

### Flux de calcul

1. Charge clé k0,k1 (16 B LE)
2. Init v0..v3 avec constantes ^ clés
3. Pour chaque bloc 8 B message : v3^=m ; 2×SipRound ; v0^=m
4. Padding dernier bloc + longueur ; 4×SipRound final ; return v0^v1^v2^v3

**Emit sgoiter :** Uint64 LE + rounds avec RotateLeft64 ; corps long (~170 lignes).

**Lecture strates :** ov/tiny = setup clé ; l1+ = boucle message dominante.

### Couches (toutes strates)

| stratum | phase | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | lecture |
|---------|-------|------|--------|---------|-------|--------|----------|----------|
| ov_empty | overhead | 6.5 | 7.3 | 7.5 | 1.13× | 1.16× | 0.97× | parité, bruit appel |
| tiny_16 | setup | 14.2 | 12.1 | 12.2 | 0.85× | 0.86× | 0.99× | sgo gagne |
| l1_1k | hot_l1 | 276.4 | 299.3 | 317.3 | 1.08× | 1.15× | 0.94× | parité |
| l1_4k | hot_l1 | 1070.4 | 1179.0 | 1222.5 | 1.10× | 1.14× | 0.96× | parité |
| l2_64k | hot_l2 | 16438.1 | 18143.7 | 19285.9 | 1.10× | 1.17× | 0.94× | parité |
| bulk_1m | bulk | 268963.6 | 292737.7 | 308191.2 | 1.09× | 1.15× | 0.95× | parité |

**Point fort :** strate `tiny_16` sgoiter **plus rapide** que C (ratio 0.85×).

**Leviers fidèles (pas de changement d'algo) :**
- Inline SipRound
- Éviter temp m si SSA le fait déjà

---

## MurmurHash3 x86_32 — `murmur3_x86_32`

**Contrat :** mix 32-bit blocs LE + fmix final

### Flux de calcul

1. nblocks = len/4 ; h1 = seed
2. Pour chaque bloc : k1=LE32 ; k1*=c1 ; rotl15 ; k1*=c2 ; h1^=k1 ; rotl13 ; h1=h1*5+const
3. Tail 0..3 octets ; h1 ^= len ; return fmix32(h1)

**Emit sgoiter :** Rotl32 = wrapper bits.RotateLeft32 ; boucle d'induction v18=-nblocks atypique.

**Lecture strates :** Dégradation progressive l1→bulk = coût mix/rot par bloc, pas mémoire seule.

### Couches (toutes strates)

| stratum | phase | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | lecture |
|---------|-------|------|--------|---------|-------|--------|----------|----------|
| ov_empty | overhead | 1.2 | 3.0 | 2.5 | 2.39× | 2.00× | 1.20× | GOULET, bruit appel |
| tiny_16 | setup | 4.3 | 4.7 | 5.6 | 1.08× | 1.30× | 0.83× | parité |
| l1_1k | hot_l1 | 199.3 | 257.1 | 258.2 | 1.29× | 1.30× | 1.00× | sgo +lent |
| l1_4k | hot_l1 | 806.0 | 1028.2 | 1003.2 | 1.28× | 1.24× | 1.02× | sgo +lent |
| l2_64k | hot_l2 | 12825.1 | 16409.9 | 16595.3 | 1.28× | 1.29× | 0.99× | sgo +lent |
| bulk_1m | bulk | 209602.4 | 261199.7 | 257075.9 | 1.25× | 1.23× | 1.02× | sgo +lent |

**Goulet dominant (cette lib, hors overhead) :** `l1_1k` — ratio **1.29×** (sgo 257.1 vs C 199.3 ns/op), phase `hot_l1`.

**Leviers fidèles (pas de changement d'algo) :**
- Inline Rotl32/Fmix32
- Normaliser la boucle for i:=0;i<nblocks

---

## BLAKE2b compress (1 bloc 128 B) — `blake2b_compress`

**Contrat :** 12 rondes G sur état v[16], message m[16], sigma[12][16]

### Flux de calcul

1. Charge m[0..15] depuis block (LE u64)
2. v[0..7]=h ; v[8..15]=IV ^ (t0,t1,f0,f1)
3. 12 rondes : 8×G avec m[sigma[r][i]]
4. h[i] ^= v[i] ^ v[i+8]

**Emit sgoiter :** sigma table runtime + indexation m[sigma[…]] ; RotateLeft64 présent.

**Lecture strates :** block_* = pure ALU (même payload) ; ratio = coût indirection sigma / forme Go.

### Couches (toutes strates)

| stratum | phase | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | lecture |
|---------|-------|------|--------|---------|-------|--------|----------|----------|
| block_1 | block | 98.6 | 140.9 | 168.2 | 1.43× | 1.70× | 0.84× | sgo +lent |
| block_1k | hot_l1 | 108.0 | 148.3 | 176.1 | 1.37× | 1.63× | 0.84× | sgo +lent |
| block_64k | bulk | 161.5 | 199.3 | 244.5 | 1.23× | 1.51× | 0.82× | sgo +lent |

**Goulet dominant (cette lib, hors overhead) :** `block_1` — ratio **1.43×** (sgo 140.9 vs C 98.6 ns/op), phase `block`.

**Leviers fidèles (pas de changement d'algo) :**
- Unroll 12 rondes + sigma littéraux (peephole fidèle majeur)

---

## ChaCha20 quarter-round (seule) — `chacha20_qr`

**Contrat :** Une QR sur 4 mots *a,*b,*c,*d — pas le cipher complet 20 rondes

### Flux de calcul

1. a+=b ; d^=a ; d<<<16
2. c+=d ; b^=c ; b<<<12
3. a+=b ; d^=a ; d<<<8
4. c+=d ; b^=c ; b<<<7

**Emit sgoiter :** Pointeurs *uint32 + bits.RotateLeft32 ; fixture = 1 QR.

**Lecture strates :** qr_1 / qr_1m = pure ALU ; comparer à C O2 sur le même symbole.

### Couches (toutes strates)

| stratum | phase | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | lecture |
|---------|-------|------|--------|---------|-------|--------|----------|----------|
| qr_1 | block | 2.3 | 2.4 | 5.4 | 1.02× | 2.30× | 0.44× | parité |
| qr_1m | hot_l1 | 2.7 | 2.5 | 5.9 | 0.93× | 2.18× | 0.43× | sgo gagne |

**Point fort :** strate `qr_1m` sgoiter **plus rapide** que C (ratio 0.93×).

**Leviers fidèles (pas de changement d'algo) :**
- Éliminer reloads via pointeurs si SSA le permet
- Ne pas inventer full ChaCha ici

---

## MD5 transform block — `md5_transform`

**Contrat :** 64 pas FF/GG/HH/II sur block 64 B → state[4]

### Flux de calcul

1. Charge X[16] LE depuis block
2. 64 pas : F/G/H/I + add const + rotl + add b
3. state[i] += a/b/c/d

**Emit sgoiter :** Expressions plates + RotateLeft32 + BCE block[15] ; graphe à vérifier (troncature fixture possible).

**Lecture strates :** block_* répète le même transform ; ratio stable = coût par transform.

### Couches (toutes strates)

| stratum | phase | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | lecture |
|---------|-------|------|--------|---------|-------|--------|----------|----------|
| block_1 | block | 8.4 | 11.0 | 10.7 | 1.32× | 1.28× | 1.03× | sgo +lent |
| block_1k | hot_l1 | 8.3 | 11.2 | 11.8 | 1.34× | 1.42× | 0.95× | sgo +lent |
| block_64k | bulk | 8.4 | 11.3 | 11.9 | 1.35× | 1.42× | 0.95× | sgo +lent |

**Goulet dominant (cette lib, hors overhead) :** `block_64k` — ratio **1.35×** (sgo 11.3 vs C 8.4 ns/op), phase `bulk`.

**Leviers fidèles (pas de changement d'algo) :**
- Vérifier 64 pas complets vs C
- Inline déjà fait

---

## Poly1305 block (5 limbs) — `poly1305_block5`

**Contrat :** h = (h + m) * r mod 2^130-5 sur limbs 26-bit

### Flux de calcul

1. Charge m en limbs
2. Multiplications croisées h_i * r_j → accumulateurs 64-bit
3. Propagation retenues >>26 + *5 sur wrap

**Emit sgoiter :** Scalaire 64-bit fidèle ; pas encore bits.Mul64/Add64 systématiques.

**Lecture strates :** poly_1 = 1 bloc ; poly_1m = throughput mul/add.

### Couches (toutes strates)

| stratum | phase | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | lecture |
|---------|-------|------|--------|---------|-------|--------|----------|----------|
| poly_1 | block | 8.3 | 9.1 | 14.5 | 1.09× | 1.74× | 0.63× | parité |
| poly_1m | hot_l1 | 10.1 | 9.1 | 16.1 | 0.91× | 1.60× | 0.57× | sgo gagne |

**Point fort :** strate `poly_1m` sgoiter **plus rapide** que C (ratio 0.91×).

**Leviers fidèles (pas de changement d'algo) :**
- Intrinsèques bits.Mul64/Add64 si équivalents bit-exact

---

## Base64 encode stream — `base64_simd`

**Contrat :** 3 octets → 4 chars table ; padding =

### Flux de calcul

1. Tant que ≥3 octets : pack u32 (b0<<16|b1<<8|b2) ; 4 index 6-bit → table
2. Reste 1–2 octets + padding

**Emit sgoiter :** Pack 32-bit déjà ; table string indexée.

**Lecture strates :** tail = padding path ; l1/bulk = boucle 3→4 dominante.

### Couches (toutes strates)

| stratum | phase | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | lecture |
|---------|-------|------|--------|---------|-------|--------|----------|----------|
| tail_17 | tail | 7.7 | 11.0 | 10.1 | 1.43× | 1.30× | 1.10× | sgo +lent |
| align_64 | hot_l1 | 26.8 | 31.1 | 34.9 | 1.16× | 1.30× | 0.89× | sgo +lent |
| l1_1k | hot_l1 | 377.6 | 456.7 | 487.3 | 1.21× | 1.29× | 0.94× | sgo +lent |
| l2_64k | hot_l2 | 24341.5 | 27408.1 | 30483.2 | 1.13× | 1.25× | 0.90× | parité |
| bulk_1m | bulk | 436366.3 | 461893.9 | 495091.9 | 1.06× | 1.13× | 0.93× | parité |

**Goulet dominant (cette lib, hors overhead) :** `tail_17` — ratio **1.43×** (sgo 11.0 vs C 7.7 ns/op), phase `tail`.

**Leviers fidèles (pas de changement d'algo) :**
- BCE dst/src
- Éviter bounds sur table si const

---

## TweetNaCl crypto_verify_16 → vn — `tweetnacl_dogfood`

**Contrat :** Comparaison temps-constant 16 B (surface) via vn(x,y,n)

### Flux de calcul

1. crypto_verify_16 appelle vn(x,y,16)
2. Si n%8==0 : OR des XOR de mots NativeEndian u64 (CT)
3. Sinon : boucle octet d |= x[i]^y[i]
4. return 0 si égal, -1 sinon (formule (1&((d-1)>>k))-1)

**Emit sgoiter :** Override Vn mots 64-bit + fallback octet + BCE [:n].

**Lecture strates :** ver_eq / ver_neq : même travail CT (pas d'early-exit data-dependent).

### Couches (toutes strates)

| stratum | phase | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | lecture |
|---------|-------|------|--------|---------|-------|--------|----------|----------|
| ver_eq | block | 2.5 | 1.9 | 7.8 | 0.77× | 3.13× | 0.25× | sgo gagne |
| ver_neq | block | 2.2 | 2.2 | 8.0 | 0.98× | 3.56× | 0.28× | parité |

**Point fort :** strate `ver_eq` sgoiter **plus rapide** que C (ratio 0.77×).

**Leviers fidèles (pas de changement d'algo) :**
- Asm check ver_neq
- Unroll fixe n=16 sans boucle

---

## strlenspn_lab — `strlenspn_lab`

### Couches (toutes strates)

| stratum | phase | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | lecture |
|---------|-------|------|--------|---------|-------|--------|----------|----------|
| ov_empty | overhead | 1.1 | 0.3 | 1.6 | 0.24× | 1.40× | 0.17× | sgo gagne, bruit appel |
| tiny_16 | setup | 3.6 | 0.9 | 3.1 | 0.25× | 0.85× | 0.30× | sgo gagne |
| l1_1k | hot_l1 | 3.8 | 1.0 | 5.1 | 0.27× | 1.36× | 0.20× | sgo gagne |
| l1_4k | hot_l1 | 3.8 | 0.9 | 5.4 | 0.24× | 1.42× | 0.17× | sgo gagne |
| l2_64k | hot_l2 | 3.0 | 1.0 | 5.0 | 0.33× | 1.69× | 0.19× | sgo gagne |
| bulk_1m | bulk | 3.3 | 1.5 | 5.8 | 0.44× | 1.74× | 0.25× | sgo gagne |

**Point fort :** strate `l1_4k` sgoiter **plus rapide** que C (ratio 0.24×).

---

## md5_transform_full — `md5_transform_full`

### Couches (toutes strates)

| stratum | phase | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | lecture |
|---------|-------|------|--------|---------|-------|--------|----------|----------|
| block_1 | block | 62.8 | 91.2 | 89.0 | 1.45× | 1.42× | 1.02× | sgo +lent |
| block_1k | hot_l1 | 78.5 | 112.0 | 112.2 | 1.43× | 1.43× | 1.00× | sgo +lent |
| block_64k | bulk | 78.4 | 109.4 | 112.0 | 1.40× | 1.43× | 0.98× | sgo +lent |

**Goulet dominant (cette lib, hors overhead) :** `block_1` — ratio **1.45×** (sgo 91.2 vs C 62.8 ns/op), phase `block`.

---

## Classement goulets globaux (ratio sgo/C, hors overhead)

| # | lib | stratum | phase | ratio | C ns | sgo ns | allocs |
|---|-----|---------|-------|-------|------|--------|--------|
| 1 | fast_xor | tail_17 | tail | **1.50×** | 2.3 | 3.4 | 0 |
| 2 | md5_transform_full | block_1 | block | **1.45×** | 62.8 | 91.2 | 0 |
| 3 | blake2b_compress | block_1 | block | **1.43×** | 98.6 | 140.9 | 0 |
| 4 | md5_transform_full | block_1k | hot_l1 | **1.43×** | 78.5 | 112.0 | 0 |
| 5 | base64_simd | tail_17 | tail | **1.43×** | 7.7 | 11.0 | 0 |
| 6 | md5_transform_full | block_64k | bulk | **1.40×** | 78.4 | 109.4 | 0 |
| 7 | blake2b_compress | block_1k | hot_l1 | **1.37×** | 108.0 | 148.3 | 0 |
| 8 | md5_transform | block_64k | bulk | **1.35×** | 8.4 | 11.3 | 0 |
| 9 | md5_transform | block_1k | hot_l1 | **1.34×** | 8.3 | 11.2 | 0 |
| 10 | fnv1a_64 | tiny_16 | setup | **1.33×** | 5.3 | 7.0 | 0 |
| 11 | md5_transform | block_1 | block | **1.32×** | 8.4 | 11.0 | 0 |
| 12 | murmur3_x86_32 | l1_1k | hot_l1 | **1.29×** | 199.3 | 257.1 | 0 |
| 13 | murmur3_x86_32 | l2_64k | hot_l2 | **1.28×** | 12825.1 | 16409.9 | 0 |
| 14 | murmur3_x86_32 | l1_4k | hot_l1 | **1.28×** | 806.0 | 1028.2 | 0 |
| 15 | murmur3_x86_32 | bulk_1m | bulk | **1.25×** | 209602.4 | 261199.7 | 0 |

### Priorité d'ingénierie (fidèle au graphe)

1. **blake2b_compress** — unroll + sigma littéral (indirection table).
2. **murmur3_x86_32** — inline Rotl/Fmix + boucle d'induction claire.
3. **tweetnacl Vn** — residual CT vs C (asm) ; n=16 unroll fixe.
4. **md5_transform** — confirmer complétude 64 pas puis forme rot.
5. **fnv1a_64** — déjà ×8 ; gains marginaux seulement.
6. **crc32_ieee** — **clos** (parité algo bit-wise) ; ne pas « optimiser » par table.
