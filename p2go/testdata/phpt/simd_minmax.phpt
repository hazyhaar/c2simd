--TEST--
Extremums vectorisables (F-p2go-simd-minmax) : max et min sur tableau signé
--FILE--
<?php
$a = [5, -3, 42, 7, -19, 42, 0, 13, 8, 21, -7, 99, 3];
$max = $a[0];
for ($i = 0; $i < count($a); $i++) {
    if ($a[$i] > $max) {
        $max = $a[$i];
    }
}
$min = $a[0];
for ($j = 0; $j < count($a); $j++) {
    if ($a[$j] < $min) {
        $min = $a[$j];
    }
}
echo $max, " ", $min;
--EXPECT--
99 -19
