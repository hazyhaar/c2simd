<?php
// Charge simd_minmax : extremums sur 200k éléments, 100 passes.
$n = 200000;
$a = array_fill(0, $n, 0);
$seed = 4242;
for ($i = 0; $i < $n; $i++) {
    $seed = (1103515245 * $seed + 12345) % 2147483648;
    $a[$i] = ($seed % 2000000) - 1000000;
}
$acc = 0;
for ($r = 0; $r < 300; $r++) {
    $max = $a[0];
    for ($k = 0; $k < count($a); $k++) {
        if ($a[$k] > $max) {
            $max = $a[$k];
        }
    }
    $min = $a[0];
    for ($m = 0; $m < count($a); $m++) {
        if ($a[$m] < $min) {
            $min = $a[$m];
        }
    }
    $acc = ($acc + $max - $min) % 1000000007;
}
echo $acc;
