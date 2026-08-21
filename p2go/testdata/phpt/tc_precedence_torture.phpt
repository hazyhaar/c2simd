--TEST--
Cas ciblé precedence_torture (EXPECT gravé depuis l'oracle php)
--FILE--
<?php
echo 2 + 3 * 4, " ", (2 + 3) * 4, "\n";
echo 1 << 2 + 1, " ", (1 << 2) + 1, "\n";
echo 7 & 3 | 4 ^ 1, "\n";
echo "r=" . 2 + 3 * 4 . "x", "\n";
echo 10 - 4 - 3, " ", 100 / 5 / 2, " ", 17 % 7 % 4, "\n";
--EXPECT--
14 20
8 5
7
r=14x
3 10 3
