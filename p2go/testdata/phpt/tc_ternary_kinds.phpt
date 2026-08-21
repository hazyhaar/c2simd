--TEST--
Cas ciblé ternary_kinds (EXPECT gravé depuis l'oracle php)
--FILE--
<?php
$n = 3;
$mot = $n > 2 ? "grand" : "petit";
echo $mot, "\n";
$v = ($n ?: 7) + (0 ?: 5) + ($n > 0 ? $n * 10 : -1);
echo $v, "\n";
echo ($n == 3 ? "trois" : "autre") . "-" . ($n % 2 == 1 ? "impair" : "pair");
--EXPECT--
grand
38
trois-impair
