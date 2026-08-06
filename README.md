# c2simd (`code.hazyhaar.fr/devhoros/c2simd`)

> **Préprocesseur de Transpilation C-to-Go & Restructuration SIMD (Go 1.27)**  
> *Zéro CGO • Matcher d'AST Déterministe • Moteur Fused SIMD256 • Fallback Scalaire Sûr `x/crypto`*

---

## 1. Description de l'Architecture V4

Le module `c2simd` met en œuvre un moteur cryptographique XChaCha20-Poly1305 et un pipeline d'optimisation d'AST C-vers-Go en Pure Go (`CGO_ENABLED=0`, zéro fichier `.s`).

### Composants Clés :
1. **Moteur Fused SIMD256 (`simd_fused.go`) :**  
   Fusion de boucle par tranches de 4 096 octets (cache CPU L1) combinant ChaCha20 SIMD256 4-blocs (256 octets/itération) et Poly1305 16-Way 3x44-bit. Atteint **1 228 Mo/s à 0 allocation (`0 B/op`)**.
2. **API Zéro-Allocation (`AEADLockDst` / `AEADUnlockDst`) :**  
   Permet le scellage et le déchiffrement authentifié avec réutilisation des tampons de destination fournis par l'appelant.
3. **Fallback Scalaire Sécurisé (`simd_fused_fallback.go`) :**  
   Sous `!goexperiment.simd`, le module bascule de manière transparente sur `golang.org/x/crypto/chacha20poly1305`, garantissant la sécurité cryptographique et le contrôle du tag MAC *fail-closed* (`subtle.ConstantTimeCompare`).
4. **Porte KAT Inconditionnelle (`kat/kat_test.go`) :**  
   Validation différentielle automatique contre 3 oracles (RFC 8439, `x/crypto`, Monocypher C transpilé) s'exécutant dans tous les modes de build.

---

## 2. Benchmark et Mesures au Sol (Intel Core i9-14900K)

| Composant | Baseline / Référence | Moteur `c2simd` Pure Go | Allocations Heap | KAT Status |
| :--- | :---: | :---: | :---: | :---: |
| **ChaCha20 Stream** | 1 016 Mo/s (`x/crypto` ASM Google) | **1 861 Mo/s** (+83,2 %) | **0 B/op** | PASS 100% |
| **Poly1305 MAC** | 4 863 Mo/s (`x/crypto` ASM Google) | **3 048 Mo/s** (62,7 % vs ASM) | **0 B/op** | PASS 100% |
| **AEAD Fused Lock** | 437 Mo/s (Monocypher C transpilé) | **1 228 Mo/s** (+180 %) | **0 B/op** | PASS 100% |
| **Fallback Scalaire** | N/A | **1 007 Mo/s** | 1 alloc/op | PASS 100% |

---

## 3. Ancrage HPM55

* **Projet ID :** `019fd633-6735-705c-8c57-e09608dca298`
* **Module Go :** `code.hazyhaar.fr/devhoros/c2simd`
* **SDK :** `go1.27rc1` (`GOEXPERIMENT=simd`)
