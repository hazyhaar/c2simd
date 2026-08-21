--TEST--
do…while désucré (F-p2go-do-while) : corps exécuté au moins une fois
--FILE--
<?php
$n = 27;
$len = 0;
do {
    if ($n % 2 == 0) {
        $n = intdiv($n, 2);
    } else {
        $n = 3 * $n + 1;
    }
    $len++;
} while ($n != 1);
echo $len, "\n";

$k = 100;
do {
    echo "une fois";
    $k++;
} while ($k < 100);
--EXPECT--
111
une fois
