# Probebench LAYERS — flux + strates + goulets

Stamp: **2026-08-12T07:17:47Z**  
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
| fnv1a_64 | l1_1k | 978.2 | 1017.4 | 974.7 | **1.04×** | 1.00× | 1.04× | parité |
| crc32_ieee | l1_1k | 6847.0 | 7235.5 | 7163.7 | **1.06×** | 1.05× | 1.01× | parité |
| fast_xor | l1_1k | 64.9 | 33.5 | 79.5 | **0.52×** | 1.22× | 0.42× | sgo plus rapide |
| siphash24 | l1_1k | 385.6 | 322.6 | 358.3 | **0.84×** | 0.93× | 0.90× | sgo plus rapide |
| murmur3_x86_32 | l1_1k | 238.2 | 314.2 | 347.7 | **1.32×** | 1.46× | 0.90× | écart modéré |
| blake2b_compress | block_1k | 164.5 | 202.4 | 230.5 | **1.23×** | 1.40× | 0.88× | écart modéré |
| chacha20_qr | qr_1m | 2.9 | 2.9 | 6.8 | **1.02×** | 2.39× | 0.43× | parité |
| md5_transform | block_1k | 9.4 | 12.8 | 12.1 | **1.36×** | 1.28× | 1.06× | écart modéré |
| poly1305_block5 | poly_1m | 10.7 | 10.6 | 19.5 | **0.99×** | 1.82× | 0.54× | parité |
| base64_simd | l1_1k | 568.5 | 765.7 | 650.6 | **1.35×** | 1.14× | 1.18× | écart modéré |
| tweetnacl_dogfood | ver_eq | 2.8 | 2.3 | 11.5 | **0.82×** | 4.06× | 0.20× | sgo plus rapide |
| strlenspn_lab | l1_1k | 4.3 | 1.0 | 5.3 | **0.24×** | 1.21× | 0.20× | sgo plus rapide |
| md5_transform_full | block_1k | 80.4 | 117.3 | 115.1 | **1.46×** | 1.43× | 1.02× | écart modéré |

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
| ov_empty | overhead | 0.8 | 2.1 | 0.5 | 2.56× | 0.62× | 4.12× | GOULET, bruit appel |
| tiny_16 | setup | 8.2 | 7.5 | 13.0 | 0.91× | 1.59× | 0.58× | sgo gagne |
| l1_1k | hot_l1 | 978.2 | 1017.4 | 974.7 | 1.04× | 1.00× | 1.04× | parité |
| l1_4k | hot_l1 | 3461.8 | 3961.1 | 3493.5 | 1.14× | 1.01× | 1.13× | parité |
| l2_64k | hot_l2 | 63526.0 | 73458.6 | 53075.2 | 1.16× | 0.84× | 1.38× | sgo +lent |
| bulk_1m | bulk | 976468.8 | 923262.5 | 1192620.7 | 0.95× | 1.22× | 0.77× | sgo gagne |

**Goulet dominant (cette lib, hors overhead) :** `l2_64k` — ratio **1.16×** (sgo 73458.6 vs C 63526.0 ns/op), phase `hot_l2`.

**Point fort :** strate `tiny_16` sgoiter **plus rapide** que C (ratio 0.91×).

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
| ov_empty | overhead | 0.9 | 2.1 | 2.0 | 2.34× | 2.24× | 1.05× | GOULET, bruit appel |
| tiny_16 | setup | 98.6 | 105.1 | 115.2 | 1.07× | 1.17× | 0.91× | parité |
| l1_1k | hot_l1 | 6847.0 | 7235.5 | 7163.7 | 1.06× | 1.05× | 1.01× | parité |
| l1_4k | hot_l1 | 28575.6 | 29406.7 | 28455.1 | 1.03× | 1.00× | 1.03× | parité |
| l2_64k | hot_l2 | 455073.7 | 450831.3 | 462523.6 | 0.99× | 1.02× | 0.97× | parité |
| bulk_1m | bulk | 7342837.1 | 7490604.1 | 7461067.7 | 1.02× | 1.02× | 1.00× | parité |

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
| tail_17 | tail | 2.6 | 3.2 | 4.5 | 1.23× | 1.73× | 0.71× | sgo +lent |
| align_64 | hot_l1 | 5.1 | 5.2 | 6.9 | 1.02× | 1.34× | 0.77× | parité |
| l1_1k | hot_l1 | 64.9 | 33.5 | 79.5 | 0.52× | 1.22× | 0.42× | sgo gagne |
| l2_64k | hot_l2 | 4329.0 | 2947.3 | 3581.5 | 0.68× | 0.83× | 0.82× | sgo gagne |
| bulk_1m | bulk | 66468.1 | 50168.7 | 68481.5 | 0.75× | 1.03× | 0.73× | sgo gagne |

**Goulet dominant (cette lib, hors overhead) :** `tail_17` — ratio **1.23×** (sgo 3.2 vs C 2.6 ns/op), phase `tail`.

**Point fort :** strate `l1_1k` sgoiter **plus rapide** que C (ratio 0.52×).

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
| ov_empty | overhead | 9.9 | 8.0 | 8.9 | 0.80× | 0.90× | 0.89× | sgo gagne, bruit appel |
| tiny_16 | setup | 13.8 | 15.5 | 15.9 | 1.12× | 1.15× | 0.98× | parité |
| l1_1k | hot_l1 | 385.6 | 322.6 | 358.3 | 0.84× | 0.93× | 0.90× | sgo gagne |
| l1_4k | hot_l1 | 1264.9 | 1371.0 | 1531.1 | 1.08× | 1.21× | 0.90× | parité |
| l2_64k | hot_l2 | 21581.2 | 21985.4 | 24946.3 | 1.02× | 1.16× | 0.88× | parité |
| bulk_1m | bulk | 340186.6 | 369216.8 | 421483.2 | 1.09× | 1.24× | 0.88× | parité |

**Point fort :** strate `l1_1k` sgoiter **plus rapide** que C (ratio 0.84×).

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
| ov_empty | overhead | 2.3 | 3.1 | 2.4 | 1.34× | 1.02× | 1.31× | sgo +lent, bruit appel |
| tiny_16 | setup | 6.6 | 7.2 | 7.8 | 1.09× | 1.18× | 0.92× | parité |
| l1_1k | hot_l1 | 238.2 | 314.2 | 347.7 | 1.32× | 1.46× | 0.90× | sgo +lent |
| l1_4k | hot_l1 | 1131.9 | 1256.4 | 1170.9 | 1.11× | 1.03× | 1.07× | parité |
| l2_64k | hot_l2 | 14803.9 | 18118.4 | 18554.6 | 1.22× | 1.25× | 0.98× | sgo +lent |
| bulk_1m | bulk | 231772.2 | 350272.8 | 312903.7 | 1.51× | 1.35× | 1.12× | sgo +lent |

**Goulet dominant (cette lib, hors overhead) :** `bulk_1m` — ratio **1.51×** (sgo 350272.8 vs C 231772.2 ns/op), phase `bulk`.

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
| block_1 | block | 117.6 | 145.5 | 219.7 | 1.24× | 1.87× | 0.66× | sgo +lent |
| block_1k | hot_l1 | 164.5 | 202.4 | 230.5 | 1.23× | 1.40× | 0.88× | sgo +lent |
| block_64k | bulk | 166.5 | 204.5 | 274.2 | 1.23× | 1.65× | 0.75× | sgo +lent |

**Goulet dominant (cette lib, hors overhead) :** `block_1` — ratio **1.24×** (sgo 145.5 vs C 117.6 ns/op), phase `block`.

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
| qr_1 | block | 2.8 | 3.0 | 7.8 | 1.07× | 2.76× | 0.39× | parité |
| qr_1m | hot_l1 | 2.9 | 2.9 | 6.8 | 1.02× | 2.39× | 0.43× | parité |

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
| block_1 | block | 9.2 | 11.3 | 12.2 | 1.23× | 1.32× | 0.93× | sgo +lent |
| block_1k | hot_l1 | 9.4 | 12.8 | 12.1 | 1.36× | 1.28× | 1.06× | sgo +lent |
| block_64k | bulk | 8.6 | 12.8 | 12.5 | 1.49× | 1.46× | 1.02× | sgo +lent |

**Goulet dominant (cette lib, hors overhead) :** `block_64k` — ratio **1.49×** (sgo 12.8 vs C 8.6 ns/op), phase `bulk`.

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
| poly_1 | block | 10.2 | 8.2 | 15.5 | 0.81× | 1.52× | 0.53× | sgo gagne |
| poly_1m | hot_l1 | 10.7 | 10.6 | 19.5 | 0.99× | 1.82× | 0.54× | parité |

**Point fort :** strate `poly_1` sgoiter **plus rapide** que C (ratio 0.81×).

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
| tail_17 | tail | 10.4 | 18.8 | 13.5 | 1.81× | 1.30× | 1.39× | GOULET |
| align_64 | hot_l1 | 37.9 | 49.9 | 41.4 | 1.32× | 1.09× | 1.20× | sgo +lent |
| l1_1k | hot_l1 | 568.5 | 765.7 | 650.6 | 1.35× | 1.14× | 1.18× | sgo +lent |
| l2_64k | hot_l2 | 32779.8 | 38495.8 | 34136.3 | 1.17× | 1.04× | 1.13× | sgo +lent |
| bulk_1m | bulk | 488406.3 | 655788.0 | 531330.7 | 1.34× | 1.09× | 1.23× | sgo +lent |

**Goulet dominant (cette lib, hors overhead) :** `tail_17` — ratio **1.81×** (sgo 18.8 vs C 10.4 ns/op), phase `tail`.

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
| ver_eq | block | 2.8 | 2.3 | 11.5 | 0.82× | 4.06× | 0.20× | sgo gagne |
| ver_neq | block | 2.6 | 2.3 | 10.8 | 0.89× | 4.14× | 0.22× | sgo gagne |

**Point fort :** strate `ver_eq` sgoiter **plus rapide** que C (ratio 0.82×).

**Leviers fidèles (pas de changement d'algo) :**
- Asm check ver_neq
- Unroll fixe n=16 sans boucle

---

## strlenspn_lab — `strlenspn_lab`

### Couches (toutes strates)

| stratum | phase | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | lecture |
|---------|-------|------|--------|---------|-------|--------|----------|----------|
| ov_empty | overhead | 1.2 | 0.3 | 1.7 | 0.25× | 1.49× | 0.17× | sgo gagne, bruit appel |
| tiny_16 | setup | 4.2 | 0.9 | 5.6 | 0.22× | 1.34× | 0.17× | sgo gagne |
| l1_1k | hot_l1 | 4.3 | 1.0 | 5.3 | 0.24× | 1.21× | 0.20× | sgo gagne |
| l1_4k | hot_l1 | 4.0 | 0.9 | 5.3 | 0.24× | 1.33× | 0.18× | sgo gagne |
| l2_64k | hot_l2 | 4.0 | 1.0 | 5.2 | 0.25× | 1.30× | 0.19× | sgo gagne |
| bulk_1m | bulk | 3.5 | 1.5 | 6.2 | 0.43× | 1.79× | 0.24× | sgo gagne |

**Point fort :** strate `tiny_16` sgoiter **plus rapide** que C (ratio 0.22×).

---

## md5_transform_full — `md5_transform_full`

### Couches (toutes strates)

| stratum | phase | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | lecture |
|---------|-------|------|--------|---------|-------|--------|----------|----------|
| block_1 | block | 68.9 | 113.8 | 115.2 | 1.65× | 1.67× | 0.99× | sgo +lent |
| block_1k | hot_l1 | 80.4 | 117.3 | 115.1 | 1.46× | 1.43× | 1.02× | sgo +lent |
| block_64k | bulk | 80.2 | 111.1 | 131.3 | 1.39× | 1.64× | 0.85× | sgo +lent |

**Goulet dominant (cette lib, hors overhead) :** `block_1` — ratio **1.65×** (sgo 113.8 vs C 68.9 ns/op), phase `block`.

---

## Classement goulets globaux (ratio sgo/C, hors overhead)

| # | lib | stratum | phase | ratio | C ns | sgo ns | allocs |
|---|-----|---------|-------|-------|------|--------|--------|
| 1 | base64_simd | tail_17 | tail | **1.81×** | 10.4 | 18.8 | 0 |
| 2 | md5_transform_full | block_1 | block | **1.65×** | 68.9 | 113.8 | 0 |
| 3 | murmur3_x86_32 | bulk_1m | bulk | **1.51×** | 231772.2 | 350272.8 | 0 |
| 4 | md5_transform | block_64k | bulk | **1.49×** | 8.6 | 12.8 | 0 |
| 5 | md5_transform_full | block_1k | hot_l1 | **1.46×** | 80.4 | 117.3 | 0 |
| 6 | md5_transform_full | block_64k | bulk | **1.39×** | 80.2 | 111.1 | 0 |
| 7 | md5_transform | block_1k | hot_l1 | **1.36×** | 9.4 | 12.8 | 0 |
| 8 | base64_simd | l1_1k | hot_l1 | **1.35×** | 568.5 | 765.7 | 0 |
| 9 | base64_simd | bulk_1m | bulk | **1.34×** | 488406.3 | 655788.0 | 0 |
| 10 | murmur3_x86_32 | l1_1k | hot_l1 | **1.32×** | 238.2 | 314.2 | 0 |
| 11 | base64_simd | align_64 | hot_l1 | **1.32×** | 37.9 | 49.9 | 0 |
| 12 | blake2b_compress | block_1 | block | **1.24×** | 117.6 | 145.5 | 0 |
| 13 | blake2b_compress | block_1k | hot_l1 | **1.23×** | 164.5 | 202.4 | 0 |
| 14 | blake2b_compress | block_64k | bulk | **1.23×** | 166.5 | 204.5 | 0 |
| 15 | fast_xor | tail_17 | tail | **1.23×** | 2.6 | 3.2 | 0 |

### Priorité d'ingénierie (fidèle au graphe)

1. **blake2b_compress** — unroll + sigma littéral (indirection table).
2. **murmur3_x86_32** — inline Rotl/Fmix + boucle d'induction claire.
3. **tweetnacl Vn** — residual CT vs C (asm) ; n=16 unroll fixe.
4. **md5_transform** — confirmer complétude 64 pas puis forme rot.
5. **fnv1a_64** — déjà ×8 ; gains marginaux seulement.
6. **crc32_ieee** — **clos** (parité algo bit-wise) ; ne pas « optimiser » par table.
