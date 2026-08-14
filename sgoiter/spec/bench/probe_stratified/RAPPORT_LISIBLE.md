# Rapport de performance — C · sgoiter · ccgo

**Date de mesure :** 2026-08-13T07:11:24Z (retranspile + rebench frais)

Médiane sur 7 mesures. Ratio 1,50× = une fois et demie plus long que la référence.

| Candidat | Rôle |
|----------|------|
| C (gcc -O2) | référence native |
| sgoiter | Go émis par le transpileur |
| ccgo | concurrent C→Go |

---

## FNV-1a 64 bits

### appel à vide

*coût d’appel à vide*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 0.88 ns | — |
| sgoiter | 2.05 ns | — |
| ccgo | 0.46 ns | — |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 2.34× | goulet (beaucoup plus lent que C) |
| ccgo / C | 0.52× | plus rapide que C |
| sgoiter / ccgo | 4.48× | sgoiter plus lent que ccgo |

### 16 octets

*petit message (démarrage)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 5.32 ns | 2,869 Mo/s |
| sgoiter | 7.05 ns | 2,165 Mo/s |
| ccgo | 8.46 ns | 1,805 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.33× | nettement plus lent que C |
| ccgo / C | 1.59× | goulet (beaucoup plus lent que C) |
| sgoiter / ccgo | 0.83× | sgoiter plus rapide que ccgo |

### 1 Kio

*chemin chaud (cache L1)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 730.0 ns | 1,338 Mo/s |
| sgoiter | 726.6 ns | 1,344 Mo/s |
| ccgo | 741.5 ns | 1,317 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.00× | équivalent à C |
| ccgo / C | 1.02× | équivalent à C |
| sgoiter / ccgo | 0.98× | équivalents |

### 4 Kio

*chemin chaud (cache L1)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 2.89 µs | 1,353 Mo/s |
| sgoiter | 3.02 µs | 1,294 Mo/s |
| ccgo | 2.96 µs | 1,318 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.05× | équivalent à C |
| ccgo / C | 1.03× | équivalent à C |
| sgoiter / ccgo | 1.02× | équivalents |

### 64 Kio

*données plus larges (cache L2)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 46.47 µs | 1,345 Mo/s |
| sgoiter | 46.87 µs | 1,334 Mo/s |
| ccgo | 47.20 µs | 1,324 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.01× | équivalent à C |
| ccgo / C | 1.02× | équivalent à C |
| sgoiter / ccgo | 0.99× | équivalents |

### 1 Mio

*gros volume (débit)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 743.90 µs | 1,344 Mo/s |
| sgoiter | 754.80 µs | 1,325 Mo/s |
| ccgo | 756.48 µs | 1,322 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.01× | équivalent à C |
| ccgo / C | 1.02× | équivalent à C |
| sgoiter / ccgo | 1.00× | équivalents |

**Bilan :** pire sgoiter/C = **1.33×** (« 16 octets ») — nettement plus lent que C.

---

## CRC-32 IEEE

### appel à vide

*coût d’appel à vide*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 0.91 ns | — |
| sgoiter | 0.28 ns | — |
| ccgo | 1.82 ns | — |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 0.31× | plus rapide que C |
| ccgo / C | 2.00× | goulet (beaucoup plus lent que C) |
| sgoiter / ccgo | 0.16× | sgoiter plus rapide que ccgo |

### 16 octets

*petit message (démarrage)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 91.3 ns | 167.2 Mo/s |
| sgoiter | 91.5 ns | 166.8 Mo/s |
| ccgo | 92.8 ns | 164.5 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.00× | équivalent à C |
| ccgo / C | 1.02× | équivalent à C |
| sgoiter / ccgo | 0.99× | équivalents |

### 1 Kio

*chemin chaud (cache L1)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 6.00 µs | 162.7 Mo/s |
| sgoiter | 6.16 µs | 158.6 Mo/s |
| ccgo | 6.02 µs | 162.2 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.03× | équivalent à C |
| ccgo / C | 1.00× | équivalent à C |
| sgoiter / ccgo | 1.02× | équivalents |

### 4 Kio

*chemin chaud (cache L1)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 24.12 µs | 161.9 Mo/s |
| sgoiter | 24.45 µs | 159.7 Mo/s |
| ccgo | 24.23 µs | 161.2 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.01× | équivalent à C |
| ccgo / C | 1.00× | équivalent à C |
| sgoiter / ccgo | 1.01× | équivalents |

### 64 Kio

*données plus larges (cache L2)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 385.50 µs | 162.1 Mo/s |
| sgoiter | 397.08 µs | 157.4 Mo/s |
| ccgo | 389.68 µs | 160.4 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.03× | équivalent à C |
| ccgo / C | 1.01× | équivalent à C |
| sgoiter / ccgo | 1.02× | équivalents |

### 1 Mio

*gros volume (débit)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 6.15 ms | 162.6 Mo/s |
| sgoiter | 6.29 ms | 159.1 Mo/s |
| ccgo | 6.17 ms | 162.0 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.02× | équivalent à C |
| ccgo / C | 1.00× | équivalent à C |
| sgoiter / ccgo | 1.02× | équivalents |

**Bilan :** pire sgoiter/C = **1.03×** (« 64 Kio ») — équivalent à C.

---

## XOR rapide

### 17 octets (queue)

*queue courte / non alignée*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 2.28 ns | 7,106 Mo/s |
| sgoiter | 3.42 ns | 4,740 Mo/s |
| ccgo | 5.24 ns | 3,094 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.50× | nettement plus lent que C |
| ccgo / C | 2.30× | goulet (beaucoup plus lent que C) |
| sgoiter / ccgo | 0.65× | sgoiter plus rapide que ccgo |

### 64 octets alignés

*chemin chaud (cache L1)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 5.02 ns | n/a (fixture) |
| sgoiter | 5.03 ns | n/a (fixture) |
| ccgo | 6.16 ns | 9,917 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.00× | équivalent à C |
| ccgo / C | 1.23× | un peu plus lent que C |
| sgoiter / ccgo | 0.82× | sgoiter plus rapide que ccgo |

### 1 Kio

*chemin chaud (cache L1)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 64.1 ns | n/a (fixture) |
| sgoiter | 32.7 ns | n/a (fixture) |
| ccgo | 77.4 ns | n/a (fixture) |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 0.51× | plus rapide que C |
| ccgo / C | 1.21× | un peu plus lent que C |
| sgoiter / ccgo | 0.42× | sgoiter plus rapide que ccgo |

### 64 Kio

*données plus larges (cache L2)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 2.51 µs | n/a (fixture) |
| sgoiter | 2.30 µs | n/a (fixture) |
| ccgo | 3.41 µs | n/a (fixture) |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 0.92× | équivalent à C |
| ccgo / C | 1.36× | nettement plus lent que C |
| sgoiter / ccgo | 0.68× | sgoiter plus rapide que ccgo |

### 1 Mio

*gros volume (débit)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 51.44 µs | n/a (fixture) |
| sgoiter | 43.95 µs | n/a (fixture) |
| ccgo | 63.65 µs | n/a (fixture) |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 0.85× | équivalent à C |
| ccgo / C | 1.24× | un peu plus lent que C |
| sgoiter / ccgo | 0.69× | sgoiter plus rapide que ccgo |

**Bilan :** pire sgoiter/C = **1.50×** (« 17 octets (queue) ») — nettement plus lent que C.

---

## SipHash-2-4

### appel à vide

*coût d’appel à vide*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 6.48 ns | — |
| sgoiter | 7.33 ns | — |
| ccgo | 7.55 ns | — |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.13× | un peu plus lent que C |
| ccgo / C | 1.16× | un peu plus lent que C |
| sgoiter / ccgo | 0.97× | équivalents |

### 16 octets

*petit message (démarrage)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 14.2 ns | 1,074 Mo/s |
| sgoiter | 12.1 ns | 1,265 Mo/s |
| ccgo | 12.2 ns | 1,252 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 0.85× | plus rapide que C |
| ccgo / C | 0.86× | équivalent à C |
| sgoiter / ccgo | 0.99× | équivalents |

### 1 Kio

*chemin chaud (cache L1)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 276.4 ns | 3,533 Mo/s |
| sgoiter | 299.3 ns | 3,262 Mo/s |
| ccgo | 317.3 ns | 3,078 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.08× | équivalent à C |
| ccgo / C | 1.15× | un peu plus lent que C |
| sgoiter / ccgo | 0.94× | sgoiter plus rapide que ccgo |

### 4 Kio

*chemin chaud (cache L1)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 1.07 µs | 3,649 Mo/s |
| sgoiter | 1.18 µs | 3,313 Mo/s |
| ccgo | 1.22 µs | 3,195 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.10× | un peu plus lent que C |
| ccgo / C | 1.14× | un peu plus lent que C |
| sgoiter / ccgo | 0.96× | équivalents |

### 64 Kio

*données plus larges (cache L2)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 16.44 µs | 3,802 Mo/s |
| sgoiter | 18.14 µs | 3,445 Mo/s |
| ccgo | 19.29 µs | 3,241 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.10× | un peu plus lent que C |
| ccgo / C | 1.17× | un peu plus lent que C |
| sgoiter / ccgo | 0.94× | sgoiter plus rapide que ccgo |

### 1 Mio

*gros volume (débit)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 268.96 µs | 3,718 Mo/s |
| sgoiter | 292.74 µs | 3,416 Mo/s |
| ccgo | 308.19 µs | 3,245 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.09× | équivalent à C |
| ccgo / C | 1.15× | un peu plus lent que C |
| sgoiter / ccgo | 0.95× | sgoiter plus rapide que ccgo |

**Bilan :** pire sgoiter/C = **1.10×** (« 64 Kio ») — un peu plus lent que C.

---

## MurmurHash3 32 bits

### appel à vide

*coût d’appel à vide*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 1.24 ns | — |
| sgoiter | 2.96 ns | — |
| ccgo | 2.48 ns | — |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 2.39× | goulet (beaucoup plus lent que C) |
| ccgo / C | 2.00× | goulet (beaucoup plus lent que C) |
| sgoiter / ccgo | 1.20× | sgoiter plus lent que ccgo |

### 16 octets

*petit message (démarrage)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 4.31 ns | 3,538 Mo/s |
| sgoiter | 4.66 ns | 3,277 Mo/s |
| ccgo | 5.61 ns | 2,721 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.08× | équivalent à C |
| ccgo / C | 1.30× | nettement plus lent que C |
| sgoiter / ccgo | 0.83× | sgoiter plus rapide que ccgo |

### 1 Kio

*chemin chaud (cache L1)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 199.3 ns | 4,900 Mo/s |
| sgoiter | 257.1 ns | 3,799 Mo/s |
| ccgo | 258.2 ns | 3,782 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.29× | nettement plus lent que C |
| ccgo / C | 1.30× | nettement plus lent que C |
| sgoiter / ccgo | 1.00× | équivalents |

### 4 Kio

*chemin chaud (cache L1)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 806.0 ns | 4,846 Mo/s |
| sgoiter | 1.03 µs | 3,799 Mo/s |
| ccgo | 1.00 µs | 3,894 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.28× | nettement plus lent que C |
| ccgo / C | 1.24× | un peu plus lent que C |
| sgoiter / ccgo | 1.02× | équivalents |

### 64 Kio

*données plus larges (cache L2)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 12.83 µs | 4,873 Mo/s |
| sgoiter | 16.41 µs | 3,809 Mo/s |
| ccgo | 16.60 µs | 3,766 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.28× | nettement plus lent que C |
| ccgo / C | 1.29× | nettement plus lent que C |
| sgoiter / ccgo | 0.99× | équivalents |

### 1 Mio

*gros volume (débit)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 209.60 µs | 4,771 Mo/s |
| sgoiter | 261.20 µs | 3,828 Mo/s |
| ccgo | 257.08 µs | 3,890 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.25× | un peu plus lent que C |
| ccgo / C | 1.23× | un peu plus lent que C |
| sgoiter / ccgo | 1.02× | équivalents |

**Bilan :** pire sgoiter/C = **1.29×** (« 1 Kio ») — nettement plus lent que C.

---

## BLAKE2b

### 1 bloc

*un bloc de calcul*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 98.6 ns | 1,238 Mo/s |
| sgoiter | 140.9 ns | 866.3 Mo/s |
| ccgo | 168.2 ns | 725.9 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.43× | nettement plus lent que C |
| ccgo / C | 1.70× | goulet (beaucoup plus lent que C) |
| sgoiter / ccgo | 0.84× | sgoiter plus rapide que ccgo |

### 1 000 blocs

*chemin chaud (cache L1)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 108.0 ns | 1,130 Mo/s |
| sgoiter | 148.3 ns | 823.4 Mo/s |
| ccgo | 176.1 ns | 693.1 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.37× | nettement plus lent que C |
| ccgo / C | 1.63× | goulet (beaucoup plus lent que C) |
| sgoiter / ccgo | 0.84× | sgoiter plus rapide que ccgo |

### 64 000 blocs

*gros volume (débit)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 161.5 ns | 755.7 Mo/s |
| sgoiter | 199.3 ns | 612.4 Mo/s |
| ccgo | 244.5 ns | 499.2 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.23× | un peu plus lent que C |
| ccgo / C | 1.51× | goulet (beaucoup plus lent que C) |
| sgoiter / ccgo | 0.82× | sgoiter plus rapide que ccgo |

**Bilan :** pire sgoiter/C = **1.43×** (« 1 bloc ») — nettement plus lent que C.

---

## ChaCha20 quarter-round

### 1 quarter-round

*un bloc de calcul*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 2.35 ns | 6,498 Mo/s |
| sgoiter | 2.40 ns | 6,364 Mo/s |
| ccgo | 5.39 ns | 2,831 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.02× | équivalent à C |
| ccgo / C | 2.30× | goulet (beaucoup plus lent que C) |
| sgoiter / ccgo | 0.44× | sgoiter plus rapide que ccgo |

### boucle ALU serrée

*chemin chaud (cache L1)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 2.70 ns | 5,659 Mo/s |
| sgoiter | 2.52 ns | 6,062 Mo/s |
| ccgo | 5.88 ns | 2,597 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 0.93× | équivalent à C |
| ccgo / C | 2.18× | goulet (beaucoup plus lent que C) |
| sgoiter / ccgo | 0.43× | sgoiter plus rapide que ccgo |

**Bilan :** pire sgoiter/C = **1.02×** (« 1 quarter-round ») — équivalent à C.

---

## MD5 réduit

### 1 bloc

*un bloc de calcul*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 8.37 ns | 7,295 Mo/s |
| sgoiter | 11.0 ns | 5,540 Mo/s |
| ccgo | 10.7 ns | 5,700 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.32× | nettement plus lent que C |
| ccgo / C | 1.28× | nettement plus lent que C |
| sgoiter / ccgo | 1.03× | équivalents |

### 1 000 blocs

*chemin chaud (cache L1)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 8.34 ns | 7,319 Mo/s |
| sgoiter | 11.2 ns | 5,443 Mo/s |
| ccgo | 11.8 ns | 5,159 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.34× | nettement plus lent que C |
| ccgo / C | 1.42× | nettement plus lent que C |
| sgoiter / ccgo | 0.95× | sgoiter plus rapide que ccgo |

### 64 000 blocs

*gros volume (débit)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 8.38 ns | 7,283 Mo/s |
| sgoiter | 11.3 ns | 5,386 Mo/s |
| ccgo | 11.9 ns | 5,124 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.35× | nettement plus lent que C |
| ccgo / C | 1.42× | nettement plus lent que C |
| sgoiter / ccgo | 0.95× | équivalents |

**Bilan :** pire sgoiter/C = **1.35×** (« 64 000 blocs ») — nettement plus lent que C.

---

## Poly1305

### 1 bloc poly

*un bloc de calcul*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 8.34 ns | 1,830 Mo/s |
| sgoiter | 9.07 ns | 1,682 Mo/s |
| ccgo | 14.5 ns | 1,053 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.09× | équivalent à C |
| ccgo / C | 1.74× | goulet (beaucoup plus lent que C) |
| sgoiter / ccgo | 0.63× | sgoiter plus rapide que ccgo |

### beaucoup de blocs poly

*chemin chaud (cache L1)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 10.1 ns | 1,518 Mo/s |
| sgoiter | 9.14 ns | 1,670 Mo/s |
| ccgo | 16.1 ns | 947.4 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 0.91× | équivalent à C |
| ccgo / C | 1.60× | goulet (beaucoup plus lent que C) |
| sgoiter / ccgo | 0.57× | sgoiter plus rapide que ccgo |

**Bilan :** pire sgoiter/C = **1.09×** (« 1 bloc poly ») — équivalent à C.

---

## Base64

### 17 octets (queue)

*queue courte / non alignée*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 7.75 ns | 2,092 Mo/s |
| sgoiter | 11.0 ns | 1,468 Mo/s |
| ccgo | 10.1 ns | 1,609 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.43× | nettement plus lent que C |
| ccgo / C | 1.30× | nettement plus lent que C |
| sgoiter / ccgo | 1.10× | sgoiter plus lent que ccgo |

### 64 octets alignés

*chemin chaud (cache L1)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 26.8 ns | 2,277 Mo/s |
| sgoiter | 31.1 ns | 1,963 Mo/s |
| ccgo | 34.9 ns | 1,751 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.16× | un peu plus lent que C |
| ccgo / C | 1.30× | nettement plus lent que C |
| sgoiter / ccgo | 0.89× | sgoiter plus rapide que ccgo |

### 1 Kio

*chemin chaud (cache L1)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 377.6 ns | 2,586 Mo/s |
| sgoiter | 456.7 ns | 2,138 Mo/s |
| ccgo | 487.3 ns | 2,004 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.21× | un peu plus lent que C |
| ccgo / C | 1.29× | nettement plus lent que C |
| sgoiter / ccgo | 0.94× | sgoiter plus rapide que ccgo |

### 64 Kio

*données plus larges (cache L2)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 24.34 µs | 2,568 Mo/s |
| sgoiter | 27.41 µs | 2,280 Mo/s |
| ccgo | 30.48 µs | 2,050 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.13× | un peu plus lent que C |
| ccgo / C | 1.25× | nettement plus lent que C |
| sgoiter / ccgo | 0.90× | sgoiter plus rapide que ccgo |

### 1 Mio

*gros volume (débit)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 436.37 µs | 2,292 Mo/s |
| sgoiter | 461.89 µs | 2,165 Mo/s |
| ccgo | 495.09 µs | 2,020 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.06× | équivalent à C |
| ccgo / C | 1.13× | un peu plus lent que C |
| sgoiter / ccgo | 0.93× | sgoiter plus rapide que ccgo |

**Bilan :** pire sgoiter/C = **1.43×** (« 17 octets (queue) ») — nettement plus lent que C.

---

## TweetNaCl vérification

### vérification égalité

*un bloc de calcul*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 2.51 ns | 6,087 Mo/s |
| sgoiter | 1.94 ns | 7,869 Mo/s |
| ccgo | 7.84 ns | 1,947 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 0.77× | plus rapide que C |
| ccgo / C | 3.13× | goulet (beaucoup plus lent que C) |
| sgoiter / ccgo | 0.25× | sgoiter plus rapide que ccgo |

### vérification inégalité

*un bloc de calcul*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 2.24 ns | 6,819 Mo/s |
| sgoiter | 2.20 ns | 6,937 Mo/s |
| ccgo | 7.96 ns | 1,917 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 0.98× | équivalent à C |
| ccgo / C | 3.56× | goulet (beaucoup plus lent que C) |
| sgoiter / ccgo | 0.28× | sgoiter plus rapide que ccgo |

**Bilan :** pire sgoiter/C = **0.98×** (« vérification inégalité ») — équivalent à C.

---

## strlen/strspn labo

### appel à vide

*coût d’appel à vide*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 1.14 ns | — |
| sgoiter | 0.28 ns | — |
| ccgo | 1.59 ns | — |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 0.24× | plus rapide que C |
| ccgo / C | 1.40× | nettement plus lent que C |
| sgoiter / ccgo | 0.17× | sgoiter plus rapide que ccgo |

### 16 octets

*petit message (démarrage)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 3.63 ns | 4,203 Mo/s |
| sgoiter | 0.92 ns | n/a (fixture) |
| ccgo | 3.08 ns | 4,950 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 0.25× | plus rapide que C |
| ccgo / C | 0.85× | plus rapide que C |
| sgoiter / ccgo | 0.30× | sgoiter plus rapide que ccgo |

### 1 Kio

*chemin chaud (cache L1)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 3.77 ns | n/a (fixture) |
| sgoiter | 1.00 ns | n/a (fixture) |
| ccgo | 5.11 ns | n/a (fixture) |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 0.27× | plus rapide que C |
| ccgo / C | 1.36× | nettement plus lent que C |
| sgoiter / ccgo | 0.20× | sgoiter plus rapide que ccgo |

### 4 Kio

*chemin chaud (cache L1)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 3.81 ns | n/a (fixture) |
| sgoiter | 0.91 ns | n/a (fixture) |
| ccgo | 5.43 ns | n/a (fixture) |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 0.24× | plus rapide que C |
| ccgo / C | 1.42× | nettement plus lent que C |
| sgoiter / ccgo | 0.17× | sgoiter plus rapide que ccgo |

### 64 Kio

*données plus larges (cache L2)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 2.98 ns | n/a (fixture) |
| sgoiter | 0.97 ns | n/a (fixture) |
| ccgo | 5.04 ns | n/a (fixture) |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 0.33× | plus rapide que C |
| ccgo / C | 1.69× | goulet (beaucoup plus lent que C) |
| sgoiter / ccgo | 0.19× | sgoiter plus rapide que ccgo |

### 1 Mio

*gros volume (débit)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 3.34 ns | n/a (fixture) |
| sgoiter | 1.46 ns | n/a (fixture) |
| ccgo | 5.83 ns | n/a (fixture) |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 0.44× | plus rapide que C |
| ccgo / C | 1.74× | goulet (beaucoup plus lent que C) |
| sgoiter / ccgo | 0.25× | sgoiter plus rapide que ccgo |

**Bilan :** pire sgoiter/C = **0.44×** (« 1 Mio ») — plus rapide que C.

---

## MD5 complet

### 1 bloc

*un bloc de calcul*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 62.8 ns | 971.7 Mo/s |
| sgoiter | 91.2 ns | 669.5 Mo/s |
| ccgo | 89.0 ns | 685.5 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.45× | nettement plus lent que C |
| ccgo / C | 1.42× | nettement plus lent que C |
| sgoiter / ccgo | 1.02× | équivalents |

### 1 000 blocs

*chemin chaud (cache L1)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 78.5 ns | 777.3 Mo/s |
| sgoiter | 112.0 ns | 545.0 Mo/s |
| ccgo | 112.2 ns | 544.2 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.43× | nettement plus lent que C |
| ccgo / C | 1.43× | nettement plus lent que C |
| sgoiter / ccgo | 1.00× | équivalents |

### 64 000 blocs

*gros volume (débit)*

| Candidat | Temps par opération | Débit |
|----------|--------------------:|------:|
| C (gcc -O2) | 78.4 ns | 778.7 Mo/s |
| sgoiter | 109.4 ns | 557.7 Mo/s |
| ccgo | 112.0 ns | 544.9 Mo/s |

| Comparaison | Ratio | Lecture |
|-------------|------:|---------|
| sgoiter / C | 1.40× | nettement plus lent que C |
| ccgo / C | 1.43× | nettement plus lent que C |
| sgoiter / ccgo | 0.98× | équivalents |

**Bilan :** pire sgoiter/C = **1.45×** (« 1 bloc ») — nettement plus lent que C.

---
## Vue d’ensemble

| Bibliothèque | Pire sgoiter/C | Étape | Lecture |
|--------------|---------------:|-------|---------|
| FNV-1a 64 bits | 1.33× | 16 octets | nettement plus lent que C |
| CRC-32 IEEE | 1.03× | 64 Kio | équivalent à C |
| XOR rapide | 1.50× | 17 octets (queue) | nettement plus lent que C |
| SipHash-2-4 | 1.10× | 64 Kio | un peu plus lent que C |
| MurmurHash3 32 bits | 1.29× | 1 Kio | nettement plus lent que C |
| BLAKE2b | 1.43× | 1 bloc | nettement plus lent que C |
| ChaCha20 quarter-round | 1.02× | 1 quarter-round | équivalent à C |
| MD5 réduit | 1.35× | 64 000 blocs | nettement plus lent que C |
| Poly1305 | 1.09× | 1 bloc poly | équivalent à C |
| Base64 | 1.43× | 17 octets (queue) | nettement plus lent que C |
| TweetNaCl vérification | 0.98× | vérification inégalité | équivalent à C |
| strlen/strspn labo | 0.44× | 1 Mio | plus rapide que C |
| MD5 complet | 1.45× | 1 bloc | nettement plus lent que C |
