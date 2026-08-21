--TEST--
Ternaire en sous-expression (F-p2go-ternary-subexpr) : arithmétique, argument d'appel, imbrication parenthésée
--FILE--
<?php
function double($n) {
    return $n * 2;
}

$x = 7;
$y = 10 + ($x > 5 ? 1 : 2) * 100;
echo $y, "\n";

echo double($x > 5 ? 3 : 4), "\n";

$z = ($x > 100 ? 1 : ($x > 5 ? 2 : 3)) + ($x ?: 9);
echo $z, "\n";

$a = [$x > 5 ? 11 : 22, 33];
echo $a[0] + $a[1], "\n";

echo 1 + (0 ?: 5);
--EXPECT--
110
6
9
44
6
