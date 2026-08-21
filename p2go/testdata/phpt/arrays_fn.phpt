--TEST--
Tableaux en signature (F-p2go-array-signatures) : param array lecture seule, retour array copié
--FILE--
<?php
function firstlast(array $a) {
    return $a[0] + $a[count($a) - 1];
}

function make(int $n): array {
    $out = [0, 0, 0];
    for ($i = 0; $i < 3; $i++) {
        $out[$i] = $n + $i;
    }
    return $out;
}

$a = [10, 20, 30];
echo firstlast($a), "\n";

$b = make(5);
echo $b[0], $b[1], $b[2], "\n";

$c = make(9);
$c[0] = 77;
echo $c[0] + $c[2], "\n";

echo firstlast(make(100));
--EXPECT--
40
567
88
202
