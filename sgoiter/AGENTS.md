# sgoiter — autorité CUE (ARCHTIME)

Toolchain Go 1.27 / SIMD : `../CLAUDE.md`. Cycle d’ingénierie : `spec/PROTOCOLE_DOGFOODING.md`.

## 1. Qui décide

La vérité du transpileur n’est pas dans un `switch` Go improvisé. Elle est **déclarée en CUE**, unifiée (`cue vet`), puis **aplatie avant compilation**. Le runtime Go n’indexe que des tables.

```
spec/*.cue  +  spec/findings/F-*.cue
        │  cue vet · cue export
        ▼
sgoiter/cmd/cuegen  →  tables Go rodata
        ▼
front / rules / emit  —  indexation, pas de découverte
```

CUE n’est pas Turing-complet ici : pas de boucle, pas d’effet de bord. Deux déclarations du même fait **s’unifient** ou le `cue vet` échoue. L’ordre des fichiers n’importe pas.

## 2. Magasins (faits, pas de la doc)

| Fichier | Rôle |
| :--- | :--- |
| `spec/types.cue` | Matrice C → Go (`#CType` : taille, signe, scalaire, `go_type`). |
| `spec/rules.cue` | Réécritures AST déclarées (`#RewriteRule`). Interdiction d’une règle magique « boucle → SIMD ». |
| `spec/fsm_vt.cue` | Automate VT : transitions fermées (`#State` × plage d’octets → `#Action`). |
| `spec/findings/schema.cue` | Contrat `#Finding`. |
| `spec/findings/F-sgoiter-*.cue` | Thésaurus : chaque motif, CVE ou passe landée. |
| `cmd/cuegen` | `cue export --out json ./spec` → `rodata` Go. Ne pas éditer l’émis. |

`types.cue` / `rules.cue` / `fsm_vt.cue` sont le **modèle** posé au commit `cd5235a6` (Vague 1 VTE). Ils restent minces ; les **enrichir** quand un fait devient stable, ne pas les inventer dans `emit.go`. La masse opposable aujourd’hui, ce sont les findings.

## 3. Finding — clôture obligatoire (étape 5)

Tout cas limite résolu, toute règle d’emit/front, tout CVE harnais : une fiche `F-sgoiter-<slug>.cue` conforme à `#Finding`.

- `id` : `^F-sgoiter-[a-z0-9-]+$`
- `lever` : `c_source` \| `ir_rule` \| `emit` \| `front` \| `handwrite`
- `status` : `proposed` \| `landed` \| `rejected` \| `codified`
- `evidence.kat` : `pass` \| `fail` \| `n/a` — ancré à un test ou un commit, pas à de la prose

Sans fiche + `cue vet` vert, le dogfood **n’est pas clos**. `handwrite` dans `lever` documente un noyau SIMD manuscrit (`simd_*.go`) : ce n’est pas un rewrite AST ; ne pas prétendre le contraire.

## 4. Commandes opposables

```bash
cd /devhoros/c2simd/sgoiter/spec && cue vet .
cd /devhoros/c2simd/sgoiter/spec && cue vet ./findings
# émission tables (quand le générateur est le chemin pris) :
cd /devhoros/c2simd/sgoiter && go run ./cmd/cuegen
```

Gates transpile : `GOEXPERIMENT=simd` + `GOTOOLCHAIN` selon `../CLAUDE.md`. Oracle : `Test*VsCOracle` contre `gcc -O2`. Jamais de chiffre SIMD sous go1.26.

## 5. Interdits

1. Modifier à la main un fichier émis par `sgoiter -in … -out …` ou par `cuegen`. Corriger le C, le CUE, le front, les rules ou l’emit.
2. Landir une passe compilateur sans finding CUE.
3. Inventer un type C→Go ou une transition VT dans le Go alors qu’elle peut vivre en `types.cue` / `fsm_vt.cue`.
4. Qualifier de transpilé un `.go` manuscrit.
