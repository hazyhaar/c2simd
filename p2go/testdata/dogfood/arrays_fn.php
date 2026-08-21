<?php
// Vert vague D → F-p2go-array-signatures : array en param (lecture seule) et retour (copie).
function somme_bornes(array $a) {
    return $a[0] + $a[count($a) - 1];
}
$t = [3, 5, 8];
echo somme_bornes($t);
