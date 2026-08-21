--TEST--
Tableaux v0.2 + réduction vectorisable (F-p2go-simd-sum-reduction)
--FILE--
<?php
$a = [5, 1, 9, 2, 8, 3, 7, 4, 6, 10, 11, 13, 12];
$a[3] = 20;
$s = 0;
for ($i = 0; $i < count($a); $i++) {
    $s += $a[$i];
}
echo $s, "\n";
echo count($a), "\n";
echo $a[0] + $a[12], "\n";

$b = [100, 200, 300];
$t = 1000;
for ($j = 0; $j < count($b); $j++) {
    $t += $b[$j];
}
echo $t;
--EXPECT--
109
13
17
1600
