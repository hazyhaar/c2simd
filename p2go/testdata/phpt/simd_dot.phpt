--TEST--
Produit scalaire vectorisable (F-p2go-simd-dot) : deux tableaux, vecteur + queue
--FILE--
<?php
$a = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13];
$b = [2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26];
$s = 0;
for ($i = 0; $i < count($a); $i++) {
    $s += $a[$i] * $b[$i];
}
echo $s, "\n";
$t = 1000;
for ($j = 0; $j < count($a); $j++) {
    $t += $a[$j] * $a[$j];
}
echo $t;
--EXPECT--
1638
1819
