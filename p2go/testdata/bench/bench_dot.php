<?php
// Charge simd_dot : produit scalaire 100k éléments, 100 passes.
$n = 100000;
$a = array_fill(0, $n, 0);
$b = array_fill(0, $n, 0);
$seed = 777;
for ($i = 0; $i < $n; $i++) {
    $seed = (1103515245 * $seed + 12345) % 2147483648;
    $a[$i] = $seed % 512;
    $seed = (1103515245 * $seed + 12345) % 2147483648;
    $b[$i] = $seed % 512;
}
$total = 0;
for ($r = 0; $r < 500; $r++) {
    $s = 0;
    for ($k = 0; $k < count($a); $k++) {
        $s += $a[$k] * $b[$k];
    }
    $total = ($total + $s) % 1000000007;
}
echo $total;
