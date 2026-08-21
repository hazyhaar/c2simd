# NOTE_BENCH_V04 — banc de mesure p2go (campagne v0.3 → v0.5, 2026-08-20)

## Dispositif

- Machine : Intel Core i9-14900K (AVX2, pas d'AVX-512), Linux.
- Voies mesurées par `cmd/p2go-bench` (meilleur de 3 runs, temps wall du process entier) :
  1. **php** — CLI PHP 8.3.6 (NTS) interprétant la charge telle quelle ;
  2. **go scalaire** — la charge transpilée par p2go, compilée go1.27rc3 **sans**
     `GOEXPERIMENT=simd` (les build tags sélectionnent les helpers scalaires) ;
  3. **go simd** — le même code transpilé, compilé go1.27rc3 **avec**
     `GOEXPERIMENT=simd` (helpers `simd/archsimd`, garde runtime AVX2).
- Même toolchain sur les deux voies Go : la colonne `simd/scalaire` isole
  l'effet des règles vectorielles, pas un effet de compilateur.
- **Gate de parité intégré au banc** : les trois stdout sont comparés octet à
  octet AVANT toute mesure ; toute divergence invalide la ligne (exit 1).
- Charges : `testdata/bench/*.php` — tableaux 100k-200k éléments synthétisés
  par LCG (dans le domaine int64 signé PHP, doctrine F-p2go-int-overflow-emulation),
  centaines de passes pour que le calcul domine le démarrage de process.

## Mesures (2026-08-20, meilleur de 3 runs)

| charge | php (ms) | go scalaire (ms) | go simd (ms) | scalaire/php | simd/scalaire |
|---|---|---|---|---|---|
| bench_sum.php (Σ 200k × 500) | 661.0 | 21.9 | 9.5 | ×30.2 | **×2.32** |
| bench_dot.php (a·b 100k × 500) | 506.6 | 15.7 | 11.3 | ×32.3 | **×1.39** |
| bench_minmax.php (min+max 200k × 300) | 736.3 | 46.8 | 37.9 | ×15.7 | **×1.23** |
| bench_upper.php (2 casses 58 Kio × 2000) | 15.3 | 171.0 | 51.7 | ×0.1 | **×3.31** |
| bench_crc32.php (8 Kio × 100, scalaire pur) | 94.1 | 21.0 | 20.8 | ×4.5 | ×1.01 |
| bench_fnv1a.php (12,8 Kio × 100, scalaire pur) | 119.6 | 7.0 | 6.6 | ×17.1 | ×1.07 |

Débits de la voie casse ASCII (≈233 Mo traités) : php ≈ 15,2 Go/s ;
go scalaire ≈ 1,4 Go/s ; go simd ≈ 4,5 Go/s.

## Lecture

1. **Transpilation seule** : ×4,5 à ×32 vs PHP interprété sur les charges de
   calcul — le gain vient de l'aplatissement archtime (slots typés int64, zéro
   dispatch dynamique). La borne basse (crc32, ×4,5) correspond au code où
   l'interpréteur PHP est le moins pénalisé (boucle d'octets serrée).
2. **Règles SIMD** : sum ×2,32, dot ×1,39, minmax ×1,23, casse ASCII ×3,31 —
   mesurées à toolchain identique. dot paie l'émulation du multiply 64-bit
   (3 VPMULUDQ + shifts, VPMULLQ étant AVX-512) ; minmax paie le blend émulé.
3. **Contre-mesure honnête (bench_upper)** : le `strtoupper` NATIF de PHP est
   du C vectorisé au memcpy-speed (≈15 Go/s) et bat les deux voies Go — le
   PHP interprété ne perd que quand il interprète. La règle simd_ascii_case
   ramène l'écart de ×11 à ×3,4 ; combler le reste (déplacement du []byte↔string,
   écriture in-place) est une marche v0.5 (F-p2go-upper-native-php-gap).
4. **Témoins scalaires** (crc32, fnv1a : aucune règle SIMD applicable) :
   simd/scalaire ≈ ×1,0 — le banc ne s'auto-mesure pas, les gains affichés
   viennent bien des règles.

## Addendum v0.5 (2026-08-20)

Chantier 7 (gap strtoupper) : déroulage 2×32 octets/itération appliqué aux
helpers de casse et re-mesuré sur bench_upper — go simd 51,7 → **48,9 ms**
(×3,44 vs scalaire), soit ≈5 % de gain. Verdict : le goulot dominant est la
paire d'allocations `[]byte(s)` / `string(b)` par appel, pas la boucle
vectorielle ; la résorption (conversion in-place sous analyse de vie) est la
marche v0.6. L'hypothèse « le déroulage suffit » a été testée et infirmée au
banc plutôt qu'affirmée.

## Reproduction

```bash
cd /devhoros/c2simd/p2go
go run ./cmd/p2go-bench -bench testdata/bench -runs 3
```
