--TEST--
v0.5 : composés bitwise &= |= ^= <<= >>=, opérateur **, min/max variadiques
--FILE--
<?php
$x = 0xFF;
$x &= 0x3C;
echo $x, "\n";
$x |= 0x81;
echo $x, "\n";
$x ^= 0xFF;
echo $x, "\n";
$x <<= 4;
echo $x, "\n";
$x >>= 2;
echo $x, "\n";
echo 2 ** 10, " ", 3 ** 0, " ", 2 ** 3 ** 2, "\n";
echo -2 ** 2, " ", (-2) ** 2, "\n";
echo min(9, 4, 7, 2, 8), " ", max(9, 4, 7, 2, 8), "\n";
echo min(1, 2), " ", max(min(5, 3), 2, 1);
--EXPECT--
60
189
66
1056
264
1024 1 512
-4 4
2 9
1 3
