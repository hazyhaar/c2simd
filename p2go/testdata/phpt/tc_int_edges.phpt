--TEST--
Cas ciblé int_edges (EXPECT gravé depuis l'oracle php)
--FILE--
<?php
$max = 0x7FFFFFFFFFFFFFFF;
$min = ~$max;
echo $max, "\n";
echo $min, "\n";
echo $max >> 62, " ", $min >> 63, "\n";
echo ($max & $min) | 1, "\n";
echo -0x10 + 0x10, " ", 0xff ^ 0xFF;
--EXPECT--
9223372036854775807
-9223372036854775808
1 -1
1
0 0
