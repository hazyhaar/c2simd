--TEST--
foreach désucré en for indexé (F-p2go-foreach) : valeur seule, clé => valeur, corps sans accolades
--FILE--
<?php
$a = [4, 8, 15];
$s = 0;
foreach ($a as $v) {
    $s += $v;
}
echo $s, "\n";

foreach ($a as $i => $v) {
    echo $i, ":", $v, "\n";
}

$total = 0;
foreach ($a as $v)
    $total += $v * 2;
echo $total, "\n";

function somme(array $t) {
    $acc = 0;
    foreach ($t as $x) {
        $acc += $x;
    }
    return $acc;
}
echo somme([1, 2, 3, 4]);
--EXPECT--
27
0:4
1:8
2:15
54
10
