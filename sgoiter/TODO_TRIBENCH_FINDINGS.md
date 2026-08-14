# Banc tribench — ce qu'il attrape, ce qu'il n'attrape pas

Mis à jour le 2026-08-11 (durcissement banc) ; score sol **2026-08-13 : 13/13
compared** bit-exact vs C (`libinjection_sqli` no-oracle à part). Aucun finding
n'est « gratuit » : soit le banc peut l'opposer, soit il ne le peut pas par
construction, et cela se dit.

## Ce que le banc oppose

| Classe de défaut | Attrapé ? | Par quoi |
|---|---|---|
| Sémantique fausse (index, snapshot, réévaluation) | **oui** | comparaison bit-exact des sorties contre gcc -O2 |
| Code Go qui ne compile pas | **oui** | build du noyau ; c'est ce qui a arrêté deux régressions le 11/08 (`tweetnacl` sur un faux double-cast, `murmur3` sur une négation non signée) |
| Forme du code (rotation pliée, hoist, parenthèses) | **non** | sorties identiques, le banc ne bouge pas — relecture et compteurs de forme |
| Non-déterminisme de l'émission | **non** | deux sorties différentes peuvent être toutes deux bit-exactes ; couvert désormais par `TestEmitIsDeterministic` |

## Défauts du banc lui-même, corrigés le 11/08

| # | Défaut | Correctif |
|---|---|---|
| B1 | Un noyau sans oracle C était comparé **à sa propre sortie** et compté comme réussi (`sg.MatchOracle = true` posé en dur). Le score affiché « 12/12 » recouvrait onze comparaisons réelles. | `run.go` : un noyau sans référence externe est marqué `no_oracle`, n'entre ni au numérateur ni au dénominateur, et le résumé l'annonce à part. Verrouillé par `TestNoOracleKernelIsNotCountedAsMatch`. |
| B2 | La colonne « ns/op » chronométrait `fork`+`exec` du binaire entier, d'où ~1 ms pour les douze noyaux quelle que soit leur charge. | Colonne retirée du résumé. Une mesure de débit n'est reportée que si le harnais la produit lui-même. |
| B3 | Le rapport `sgoiterbench` imprimait « 100 % », un ratio de vitesse « 1.05x - 1.25x » en littéral, « Idiomatic Go (0 unsafe) » et « <1ms (native) » pour des valeurs jamais mesurées. | Rapport réécrit : chaque colonne vient d'une mesure enregistrée, l'absence de mesure s'écrit « not measured ». Verrouillé par `TestSummaryReportsOnlyMeasuredValues`. |
| B4 | `RunBenchmarkMeasurement` divisait le temps du processus par un nombre d'itérations **supposé**, sans vérifier que le binaire les avait faites. | La fonction lit la ligne `BENCH:` que le harnais imprime ; un binaire muet renvoie une erreur, plus un chiffre reconstruit. Le nombre d'itérations est défini une seule fois et partagé entre le C généré et le Go. |

## Défaut du thésaurus, corrigé le 11/08

Deux entrées (`fold_rot_bits`, `bce_slice_guard`) figuraient au catalogue des
règles sans rien transformer, et satisfaisaient le gate qui se contentait de
**compter** les entrées. Elles sont remplacées par trois règles réelles
(`const_fold_sub`, `sub_zero`, `and_or_self`) et le gate exige désormais qu'une
règle transforme son module témoin (`TestEveryRewriteRuleTransformsItsWitness`,
vérifié comme mordant par injection d'une règle vide).

Inventaire de ce que les règles touchent réellement sur les dix noyaux du corpus
(`TestRuleInventoryOnKernels`) :

| règle | noyaux touchés |
|---|---|
| `const_fold_sub` | 4 / 10 |
| `add_zero` | 3 / 10 |
| `const_fold_mul` | 2 / 10 |
| `const_fold_mul0` | 2 / 10 |
| `strength_mul1` | 2 / 10 |
| `const_fold_add`, `xor_self`, `sub_zero`, `and_or_self` | aucun — témoin unitaire seulement |

## Doctrine

- **Bit-exact** = filet sémantique. Le banc doit l'attraper, et il l'attrape.
- **Forme du code** = relecture et compteurs. Le banc ne remplace pas les yeux.
- **Un noyau sans oracle ne se compte jamais comme une réussite.** Un chiffre de
  parité doit toujours porter son dénominateur réel.
- **Un compteur de forme se vérifie avant d'être cité** : `grep 'int(v'` a
  longtemps servi de mesure Q10 alors qu'une parenthèse intercalée le fait mentir
  (voir l'avertissement en tête de `TODO_NEXT.md`).
