# c2myers

Diff de lignes (Myers O(ND)) et application de patch unifié **en mémoire**. Hors pile terminale hazhar TUI.

`DiffLines` est un manuscrit Go. `C2_myers_ses` (sgoiter) calcule la distance d’édition sur `[]int`, pas le backtrack. `ParsePatchSet` s’appuie sur `C2_validate_patch_mode` (rejet 120000/160000).

Ce n’est pas `git apply` : pas de binaire, pas de `\ No newline`, pas d’en-tête `diff --git` à l’émission. Round-trip `apply(diff(a,b), a) == b` sur texte avec ou sans newline finale.

Distance d’édition > 50 000 : `DiffLines` rend `nil` (fail-loud).
