# Dossier d'Architecture Technique & Consultative : Transpileur C ➔ Go 1.27 + LLM (`c2simd 2.0`)

> **Destinataires :** Grok, Qwen 2.5/3, Kimi  
> **Contexte d'Ingénierie :** Août 2026 — Écosystème Pure Go (`GOEXPERIMENT=simd`, Go 1.27), Zéro CGO, Zéro Fichier `.s`  
> **Emplacement du Fichier :** `/devhoros/c2simd/spec/c2simd_transpiler_2026_peer_review.md`  
> **Harnais d'Évaluation :** `/devhoros/c2simd` (`kat_test.go`, `bench_history.json`, `multi_bench_test.go`)

---

## 1. Cadre d'Ingénierie & Distinctions Stricte de Périmètres

Pour éviter toute faute de catégorie ou confusion de résultats, ce dossier sépare rigoureusement trois objets techniques distincts :

1. **Les Capacités Expérimentales SIMD Natifs (`cmd/c2simd-noncrypto-lab`) :** Noyaux manuels en Pure Go Go 1.27 (`archsimd`).
2. **Le Produit Cryptographique Fused AEAD (`c2simd`) :** Implémentation de référence ChaCha20/Poly1305 hand-written oraclée à 100 %.
3. **Le Pipeline Transpileur C ➔ Go (`ccgo` + `c2simd-gen`) :** Moteur de réécriture d'AST déterministe sous règles ARCHTIME.

---

## 2. Synthèse Factuelle des Performances Mesurées sous Sondes CPU (Sol Audité)

Tous les chiffres ci-dessous sont issus du poste de mesure (CPU Intel Core i9-14900K, `GOEXPERIMENT=simd go1.27rc1`) et de l'historique `bench_history.json` :

### 2.1 Capacités Go 1.27 `simd` / `archsimd` (Laboratoire Hand-Written)

| Expérimentation | Implémentation Scalaire Naïve | Référence Stdlib Go Natif | Implémentation Go 1.27 `archsimd` | Bilan & Analyse Factuelle |
| :--- | :---: | :---: | :---: | :--- |
| **Balayage de Texte SIMD (10 Mo)** | 4 253 Mo/s | **61 700 – 79 345 Mo/s** (`bytes.Count`) | **51 300 – 65 715 Mo/s** | La stdlib Go (déjà vectorisée sous le capot) devance le laboratoire maison. |
| **Dot Product IA (Float32, 1024-dim)** | 2 763 145 vecs/s | N/A | **7 130 000 – 7 763 287 vecs/s** | Gain de **+160 % à +181 %** par maintien 100 % en registres YMM. |

---

### 2.2 Moteur Cryptographique Hand-Written (`c2simd`)

| Module Cryptographique | Baseline Référence / ASM Google | Moteur `c2simd` Pure Go (`0 .s`, `0 CGO`) | Allocations Heap | Qualification KAT & Oracles |
| :--- | :---: | :---: | :---: | :---: |
| **Chiffrement ChaCha20 Stream** | 959 – 1 102 Mo/s (`x/crypto` ASM) | **1 866 – 2 077 Mo/s** | **0 B/op** | **PASS 100%** (Oracles RFC 8439 / `x/crypto`) |
| **Signature Poly1305 16-Way** | 1 408 Mo/s (Donna5x26) | **3 011 – 3 295 Mo/s** | **0 B/op** | **PASS 100%** (Clamping 44/44/42 & réduction $s=r \cdot 20$) |
| **Chaîne AEAD Fused Lock (1 Mo)** | 524 Mo/s (Monocypher C) | **1 011 – 1 304 Mo/s** | **0 B/op** | **PASS 100%** (Vérification MAC *fail-closed*) |

---

### 2.3 Pipeline Transpileur C ➔ Go (`ccgo` + `c2simd-gen`)

| Extrait C Transpilé | Baseline Brut `ccgo` | Optimisé AST (`c2simd-gen`) | Référence Stdlib Go | Constat d'Ingénierie |
| :--- | :---: | :---: | :---: | :--- |
| **MD5 Block Transform (64B)** | 8 840 Mo/s | **9 258 Mo/s** (+4,7 %) | 607 Mo/s | Code C inliné sans allocation (surpasse la stdlib). |
| **SipHash 2-4 (1024B)** | 3 692 Mo/s | 3 539 Mo/s (-4,1 %) | N/A | Go 1.27 SSA reconnaît déjà la rotation `ROLQ` en natif. |
| **BLAKE2b Compress (128B)** | 842 Mo/s | 851 Mo/s (+1,1 %) | N/A | Goulot d'étranglement dur sur `tls *libc.TLS` et `__ccgo_up`. |
| **Fast XOR Stream (4096B)** | 20 408 Mo/s | 20 118 Mo/s (0,0 %) | N/A | Saturation de la bande passante mémoire L1/L2. |

---

## 3. Réponses Tranchées aux 5 Questions d'Ingénierie

### Question 1 : Interception Parser C (`ccgo`) vs Post-Traitement AST Go
- **Décision :** **POST-traitement sur AST Go** (`internal/astmatch/astmatch.go`).
- **Raisonnement :** Intercepter l'AST C au sein de `ccgo` impose de forker un compilateur tiers (coût de maintenance lourd et rupture à chaque version). La passe POST sur l'AST Go émis préserve l'oracle de référence C brut et s'exécute de manière déterministe en **< 5 ms**.

---

### Question 2 : Règle AST Générique Boucle C ➔ SIMD
- **Décision :** **Ne PAS viser de règle générique "Toute boucle C ➔ SIMD"**.
- **Raisonnement :** La vectorisation automatique de boucles arbitraires relève de la passe SSA d'un compilateur natif. Le transpileur `c2simd-gen` applique un **dispatch fermé par signature de noyau** (`chacha20_quarter_x4`, `poly1305_block_16`). 
- **Portabilité :** Double émission conditionnelle sous build tags (`//go:build goexperiment.simd` + fallback scalaire).

---

### Question 3 : Structure du Thésaurus de Règles (Hybride ARCHTIME Strict)
- **Décision :** **Combinaison Statique (`go generate`) + CUE Hors-Ligne**.
  - **Noyau Chaud (Top 20 règles) :** Généré via `go generate` dans `rules_gen.go` (0 allocation, 0 ms runtime, section `.rodata`).
  - **Thésaurus Étendu :** Fichiers CUE/JSON versionnés consultés uniquement lors de la phase de génération par `c2simd-gen`.
  - **Exécution Crypto Runtime :** **0 I/O et 0 alloc** (Aucun parsing binaire/mmap au runtime d'AEAD).

---

### Question 4 : Analyse d'Effets pour la Suppression de `tls *libc.TLS`
- **Décision :** **Analyse d'effets par fonction (Bottom-Up Call-Graph Analysis)**.
  1. **Classification :**
     - **T0 :** Le paramètre `tls` n'est jamais lu/écrit $\to$ Élision du paramètre et mise à jour des sites d'appel.
     - **T1 :** `tls` n'est transmis qu'à des fonctions T0/T1 $\to$ Élision en chaîne.
     - **T2 :** Utilisation de `tls.Alloc`/`tls.Free` $\to$ Maintien ou réécriture du binding.
  2. **Attestation :** Validation par la porte KAT différentielle (stricte égalité des sorties mémoire).

---

### Question 5 : Robustesse de la Pyramide de Témoins KAT
- **Décision :** **Pyramide d'Oracles à 4 Niveaux**.
  - **L0 :** Vecteurs de test littéraux officiels (RFC 8439, RFC 7693, Wycheproof).
  - **L1 :** Différentiel vs `x/crypto` (exécuté sous `GOEXPERIMENT=simd`).
  - **L2 :** Différentiel vs Monocypher C transpilé (`monocypher_transformed.go`).
  - **L4 :** Tests de fuzzing sur tailles frontières (0, 1, 15, 16, 17, 63, 64, 65... octets) et alignements mémoire.

---

## 4. Bloc de Cadrage Anti-Dérive pour la Consultation Pairs

Le bloc suivant doit être inlu en amont de toute consultation avec Grok, Qwen ou Kimi :

```text
[CADRAGE ANTI-DÉRIVE C2SIMD 2.0]
- Produit Crypto : ChaCha20/Poly1305 hand-written oraclé (2,07 Go/s ChaCha20, 3,29 Go/s Poly1305, 1,30 Go/s Fused AEAD).
- Transpileur : Post-processeur AST Go déterministe (DCE, unsafe.Pointer, clear(), math/bits).
- Interdictions : Fork ccgo V1 ; auto-vectorisation générique de boucles C arbitraires ; I/O ou mmap au runtime d'AEAD.
- Obligations : Chaques règle AST est adossée à une validation KAT différentielle et un benchmark versionné.
```
