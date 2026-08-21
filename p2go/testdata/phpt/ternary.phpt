--TEST--
Ternaire en position statement (F-p2go-ternary-expr) : affectation, ?:, return, echo
--FILE--
<?php
function classe($x) {
    return $x > 5 ? 100 : 200;
}

$x = 7;
$y = $x > 5 ? 1 : 0;
echo $y, "\n";

$z = 0;
$w = $z ?: 42;
echo $w, "\n";

$v = 9;
$k = $v ?: 42;
echo $k, "\n";

echo classe(10), "\n";
echo classe(2), "\n";

echo $x == 7 ? "sept" : "autre";
--EXPECT--
1
42
9
100
200
sept
