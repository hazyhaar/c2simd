--TEST--
Builtin intdiv (F-p2go-intdiv-builtin) : troncature vers zéro, imbrication
--FILE--
<?php
echo intdiv(17, 5), "\n";
echo intdiv(-17, 5), "\n";
echo intdiv(100, intdiv(10, 3)), "\n";
$a = 7;
$b = intdiv($a * 10, 4);
echo $b;
--EXPECT--
3
-3
33
17
