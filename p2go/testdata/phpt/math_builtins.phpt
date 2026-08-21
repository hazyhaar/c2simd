--TEST--
Builtins mathématiques int (F-p2go-stdlib-math) : abs, min, max, pow, floor/ceil/round identité
--FILE--
<?php
echo abs(-42), " ", abs(7), "\n";
echo min(3, 9), " ", max(3, 9), "\n";
echo pow(2, 10), " ", pow(3, 0), " ", pow(-2, 3), "\n";
echo floor(7), " ", ceil(7), " ", round(7), "\n";
echo intdiv(pow(2, 20), 1024), "\n";
echo min(abs(-5), max(1, 2));
--EXPECT--
42 7
3 9
1024 1 -8
7 7 7
1024
2
