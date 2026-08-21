# p2go — SPEC v0.5

Version : **0.5.0** (marche v0.5 2026-08-20 : composés bitwise, **, break/continue,
strpos à sentinelle, copie de tableau, foreach sur expression, min/max variadiques,
ordre de strings, tuning casse mesuré)
Packages : `code.hazyhaar.fr/devhoros/c2simd/p2go/{front,types,ir,rules,emit,phpt}`
HPM55 : `01a01bd6-3fb0-7598-9ee1-6ac5a747acc4` (parent c2simd)
Jalon 1 : `01a01bd6-c991-75eb-a25b-a7dcf6a6360b`

## 0. Doctrine

p2go transpile un **sous-ensemble strict** de PHP vers du Go 1.27 idiomatique, selon le
modèle sgoiter (thésaurus-first, fail-loud, pipeline déterministe 5 passes) :

```
front/  parsing lexical + AST, whitelist fail-loud (codes err_*)
types/  inférence de types stricts + slots indexés fixes par fonction
ir/     abaissement en IR structurée désucrée (compound assign, for → forme canonique)
rules/  passes de réécriture pattern-matching sur l'IR (const-fold v0.1)
emit/   génération Go typé ; cible finale simd/archsimd (Go 1.27, GOEXPERIMENT=simd)
```

Principe archtime : tout le dynamisme PHP (table de symboles, types latents) est aplati
avant l'emit — le programme Go généré n'a **aucune** `map[string]any`, aucun `any`,
aucune résolution runtime. Les variables PHP deviennent des variables Go `int64` locales,
adressées par slot fixe déterminé à la passe `types`.

## 1. Subset PHP v0.1 (liste blanche)

| Construit | Notes |
|-----------|-------|
| Ouverture | `<?php` obligatoire en tête ; `?>` final optionnel |
| Types | `int` uniquement (mappé `int64`) ; bool implicite dans les conditions |
| Littéraux | entiers décimaux ; strings `"…"`/`'…'` première classe (escapes `\n \t \\ \" \$`) |
| Strings (v0.3) | slot `string` ; concat `.` / `.=` (précédence PHP 8, sous `+ -`), opérande int converti explicitement ; interpolation `$ident` désucrée au lexer (`${…}`/`{$…}` → `err_interp`) ; `==`/`!=` de strings ; `strlen` ; truthiness en condition REFUSÉE (`''` et `'0'` falsy PHP, piège non imité) |
| Ternaire (v0.3) | `c ? a : b` et `c ?: b` en positions STATEMENT (affectation, return, echo mono-argument), désucré en if/else paresseux ; refusé en clauses de for et sous-expression |
| foreach (v0.3) | `foreach ($a as [$i =>] $v)`, désucré en for indexé (compteur gensym) ; by-ref `&$v` refusé |
| Signatures (v0.3) | hints `int`/`string`/`array` en param, `: type` en retour (contrat explicite, int par défaut) ; param `array` en LECTURE SEULE, retour `array` copié (`ArrCopy`) — la divergence copie PHP / partage Go est gardée, pas imitée |
| Contrôle (v0.4) | `switch` strict (break/return obligatoire, cases vides empilés, fallthrough implicite refusé) ; `match` PHP 8 (default obligatoire) désucré en switch ; ternaire/match en TOUTE sous-expression par hoisting de temporaires (interdit en condition de boucle et valeur de case) |
| Bitwise (v0.4) | `& \| ^ ~ << >>` aux précédences PHP 8 ; littéraux `0x…` (réinterprétation int64) ; emit parenthésé selon la table de précédence GO |
| Stdlib (v0.4) | math : `abs min max pow intdiv floor ceil round` ; strings : `strlen substr str_replace trim strtoupper strtolower ord chr` (sémantique PHP exacte, ASCII octet à octet) ; tableaux : `count array_push array_pop array_reverse array_slice array_fill in_array` (mutations hors expression, copies au retour). Refus gardés : `sqrt`, `array_map`, `strpos` (false PHP sans équivalent int honnête) |
| SIMD (v0.4) | 4 règles : `simd_sum`, `simd_dot` (mul 64-bit émulé VPMULUDQ), `simd_minmax` (Greater+IfElse), `simd_ascii_case` (VPCMPGTB signé) — helpers duals scalaire/archsimd, parité bit-exacte testée sous go1.27rc3+GOEXPERIMENT=simd |
| Oracle & bench (v0.4) | corpus `testdata/algorithms/` 6/6 bit-exact vs CLI php 8.3 (TestAlgorithmsVsPhpOracle) ; banc `cmd/p2go-bench` tri-voies avec gate de parité, chiffres dans `spec/bench/NOTE_BENCH_V04.md` |
| v0.5 | composés `&= \|= ^= <<= >>=` ; `**` (droite-assoc, plus fort que l'unaire, plié en pow, exposant négatif = panic) ; `break`/`continue` de boucle (niveaux `break n` refusés ; garde do…while par jumpEscapes) ; `strpos` à SENTINELLE -1 (écart assumé vs false PHP, jamais `=== false`) ; copie de tableau `$b = $a` (ArrCopy, valeur PHP) ; `foreach (expr as …)` (temporaire p2go_faN via Block) ; min/max ≥ 2 args ; `< <= > >=` sur string×string (ordre octet à octet) ; casse SIMD déroulée 2×32 (48,9 ms, goulot = allocations, v0.6) |
| Variables | `$x` locales ; slot fixe par fonction ; pas de portée globale |
| Expr | `+ - * / % `, comparaisons `< <= > >= == !=`, logique `&& \|\| !`, unaire `-`, parenthèses, appels `f(a,b)` |
| Stmts | `$v = e;`, composés `+= -= *= /= %=`, `echo e1, e2;`, `if/elseif/else`, `while`, `do…while`, `for(init;cond;post)`, `return e;`, `return;` |
| Fonctions | `function f($a, $b) { … }` pures, params int, retour int ou void |
| Builtins | `intdiv($a,$b)` (plié en `/`, troncature vers zéro identique) ; `count($a)` |
| Tableaux (v0.2 partiel) | littéral `[e1,…]` en RHS d'un `=` simple, `$a[$i]` lecture/écriture, `count($a)` ; `[]int64` local, ni param ni retour ni copie (v0.3) |
| Incréments | `$i++`, `$i--`, `++$i`, `--$i` en statement et en clause for |
| Commentaires | `//`, `#`, `/* */` |

Division `/` et modulo `%` : sémantique entière Go (troncature vers zéro), documentée
comme écart assumé vs PHP (qui promeut `/` en float) — le subset v0.1 refuse toute
expression dont la division n'est pas entière exacte au sens Go, sans vérification
runtime. Écart tracé pour v0.2 (`intdiv` explicite).

## 2. Refus fail-loud (hors-subset)

| Motif | Code |
|-------|------|
| `eval` | `err_eval` |
| `$$var` (variable variable) | `err_varvar` |
| `global`, `static` | `err_global` |
| `include`/`require`(`_once`) | `err_include` |
| `class`/`new`/`->`/`::` | `err_oop` |
| `array(`, `list(` | `err_array` (littéral `[…]` seul) |
| `foreach` by-ref, sur expression | `err_parse` (v0.3 : `$var` tableau seulement, valeur) |
| sortie mono-fichier d'un programme vectorisé | `err_simd_multifile` (API Transpile ; TranspileFiles/-outdir requis) |
| float (littéral à point) | `err_float` |
| interpolation `${…}` / `{$…}` | `err_interp` (`$ident` seul supporté) |
| string en condition / opérande logique | `err_parse` (truthiness `'0'` falsy non imitée) |
| aucun statement harvestable | `err_empty` |
| parse local | `err_parse` |

Chaque refus porte le code, la ligne et le lexème fautif. Zéro fallback silencieux.

## 3. Harnais .phpt

Format supporté : sections `--TEST--`, `--FILE--`, `--EXPECT--` (comparaison exacte de
stdout). Exécuteur : `phpt/phpt.go` — transpile la section FILE, matérialise un
`main.go` + `go.mod` dans un répertoire temporaire, `go run`, compare stdout octet à
octet avec EXPECT. Fixtures : `testdata/phpt/*.phpt`. KAT : `phpt_kat_test.go`.

Les fixtures de refus portent `--EXPECT_ERR--` (code `err_*` attendu) au lieu
d'`--EXPECT--` : le harnais vérifie que le front rejette avec exactement ce code.

Oracle PHP réel : `TestAlgorithmsVsPhpOracle` (algo_oracle_test.go) exécute chaque
source de `testdata/algorithms/` par le CLI php ET par le Go transpilé, stdout
comparés octet à octet. Les EXPECT des fixtures `algo_*`/`tc_*` sont GÉNÉRÉS
depuis la sortie de l'oracle, jamais écrits de tête (F-p2go-php-oracle-harness).

## 4. Findings

Échecs et écarts documentés en CUE sous `spec/findings/F-p2go-*.cue`, schéma repris de
sgoiter (`spec/findings/schema.cue`, préfixe `F-p2go-`).

## 4bis. Dogfooding (Jalon 3) et strate SIMD (Jalon 4)

- **Dogfood** : `cmd/p2go-dogfood` balaye `testdata/dogfood/*.php` — verdict par
  source (`ok` / `refused` code err_* / `build_fail`), rapport JSON ; un Go émis
  qui ne compile pas = échec dur. Cycle fondateur : do…while et intdiv capturés
  refusés puis résolus ; ternaire et concat, d'abord proposed, sont landed
  depuis les vagues A-B (corpus 8/8 ok).
- **SIMD** : la règle `F-p2go-simd-sum-reduction` (rules/simd_sum.go) reconnaît
  la boucle de réduction canonique `for ($i=0; $i<count($a); $i++) { $s += $a[$i]; }`
  (garde : le compteur n'est lu nulle part ailleurs) et l'abaisse en nœud IR
  `SumLoop`, émis `s += p2goSumI64(a)` avec helpers duals : scalaire
  (`!goexperiment.simd`) et `simd/archsimd` (`goexperiment.simd`, garde runtime
  `archsimd.X86.AVX2()`, corps Int64x4 ×4, queue scalaire). KAT de parité :
  `simd_parity_test.go` exécute la même fixture sous go1.27rc3 + GOEXPERIMENT=simd
  et exige le stdout identique (doctrine pôle, `/devhoros/c2simd/CLAUDE.md` règles 3-5).

## 5. Marches

1. ~~v0.1 : subset scalaire, Fibonacci, refus fail-loud~~ (Jalons 1-2 rendus 2026-08-19).
2. ~~v0.2 partiel : do…while, intdiv/count, tableaux indexés, réduction SIMD~~ (Jalons 3-4).
3. ~~v0.3 : strings première classe + interpolation, ternaire statement, hints de
   signature (string/array en params et retours), foreach~~ (mission nocturne 2026-08-19,
   corpus dogfood 8/8 ok).
4. ~~v0.4 : switch/match, ternaire généralisé, bitwise+hexa, stdlib typée, 4 règles
   SIMD à parité prouvée, 6 algorithmes réels bit-exacts vs oracle php, banc tri-voies
   chiffré~~ (campagne 2026-08-20 : 45 fixtures, 27 findings landed, NOTE_BENCH_V04).
5. ~~v0.5 : composés bitwise, break/continue, strpos (sentinelle -1 tranchée),
   copie tableau var-à-var, foreach sur expression, min/max variadiques, ordre de
   strings, **, tuning casse mesuré~~ (marche close 2026-08-20 : 50 fixtures,
   31 findings landed ; hypothèse « déroulage suffit » infirmée au banc).
6. v0.6 : casse ASCII in-place (résorber []byte↔string, goulot mesuré 48,9 ms),
   `0b…` et séparateurs `_`, `array_push` variadique, kind bool|int si un corpus
   exige l'idiome `=== false`, `usort`/callables si jamais requis (à instruire).
