<?php
// Vert v0.2 → F-p2go-simd-sum-reduction : boucle de réduction abaissée en SumLoop.
$a = [1, 2, 3, 4, 5, 6, 7, 8, 9];
$s = 0;
for ($i = 0; $i < count($a); $i++) {
    $s += $a[$i];
}
echo $s;
