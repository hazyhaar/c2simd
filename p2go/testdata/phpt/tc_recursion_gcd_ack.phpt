--TEST--
Cas ciblé recursion_gcd_ack (EXPECT gravé depuis l'oracle php)
--FILE--
<?php
function gcd($a, $b) {
    if ($b == 0) {
        return $a;
    }
    return gcd($b, $a % $b);
}
function ack($m, $n) {
    if ($m == 0) {
        return $n + 1;
    }
    if ($n == 0) {
        return ack($m - 1, 1);
    }
    return ack($m - 1, ack($m, $n - 1));
}
echo gcd(252, 105), " ", gcd(17, 5), "\n";
echo ack(2, 3), " ", ack(3, 3);
--EXPECT--
21 1
9 61
