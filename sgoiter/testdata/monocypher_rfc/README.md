# Pack de vecteurs Monocypher / RFC 8439

Vecteurs de test formels issus de la RFC 8439 (ChaCha20-Poly1305 AEAD, IETF) et
des suites de tests de Monocypher (`poly1305`, `blake2b`, `chacha20`).

## Fichiers

- `vectors.jsonl` — une ligne = un objet JSON autonome :
  ```json
  {"id": "rfc8439_aead_1", "alg": "aead_ietf", "key_hex": "…", "nonce_hex": "…", "ad_hex": "…", "pt_hex": "…", "ct_hex": "…", "mac_hex": "…"}
  ```
  Dix vecteurs : `poly1305` ×3, `blake2b` ×3, `aead_ietf` ×2, `chacha20` ×2.

## Ce que chaque test prouve, et ce qu'il ne prouve pas

| Test | Fichier | Oracle | Portée |
|---|---|---|---|
| `TestMonocypherRFCPack` | `/devhoros/c2simd/sgoiter/monocypher_rfc_test.go` | `golang.org/x/crypto` | Les dix vecteurs. Établit qu'ils sont conformes aux RFC. |
| `TestMonocypherCOracleOnPack` | `/devhoros/c2simd/sgoiter/monocypher_c_oracle_test.go` | Monocypher 4.0.2 compilé en C | Six vecteurs (`poly1305`, `blake2b`). Établit que la bibliothèque C s'accorde avec le pack. |

Aucun de ces deux tests n'exerce le code sgoiter : ils qualifient le pack lui-même,
qui sert d'oracle à d'autres travaux. Le passage du transpileur sur les noyaux
monocypher est mesuré ailleurs (`monocypher_dogfood_test.go`, cycles de dogfood).

Quatre vecteurs restent hors de portée de l'oracle C : `crypto_aead_lock` de
Monocypher 4.0.2 attend un nonce XChaCha20 de 24 octets alors que la RFC 8439
en spécifie 12, et la version 4.0.2 n'expose pas de point d'entrée ChaCha20 seul.
Ces quatre-là ne sont donc validés que contre l'implémentation Go de référence.

Le test C se met en `skip` — jamais en échec silencieux — quand les sources
upstream sont absentes (elles sont ignorées par git) ou qu'aucun compilateur C
n'est disponible.

## Provenance

1. **RFC 8439 §2.8.2** — vecteurs officiels ChaCha20-Poly1305 AEAD.
2. **Monocypher upstream** — vecteurs Poly1305 MAC, BLAKE2b (avec et sans clé),
   ChaCha20. Sources attendues sous
   `/devhoros/c2simd/spec/c_sources/upstream/monocypher/4.0.2/`.
