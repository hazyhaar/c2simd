--TEST--
v0.5 : arrays_v05 (EXPECT gravé depuis l'oracle php)
--FILE--
<?php
$a = [1, 2, 3];
$b = $a;
$b[0] = 99;
echo $a[0], " ", $b[0], "\n";

function fabrique(int $n): array {
    $out = [0, 0];
    $out[0] = $n;
    $out[1] = $n * 2;
    return $out;
}
$t = 0;
foreach (fabrique(7) as $v) {
    $t += $v;
}
echo $t, "\n";
foreach ([10, 20, 30] as $i => $v) {
    echo $i, ":", $v, " ";
}
echo "\n";
foreach (array_reverse($a) as $v) {
    echo $v;
}
--EXPECT--
1 99
21
0:10 1:20 2:30 
321
