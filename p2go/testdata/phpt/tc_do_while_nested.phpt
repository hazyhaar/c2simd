--TEST--
Cas ciblé do_while_nested (EXPECT gravé depuis l'oracle php)
--FILE--
<?php
$total = 0;
$i = 0;
do {
    $j = 0;
    do {
        $total += $i * 10 + $j;
        $j++;
    } while ($j < 3);
    $i++;
} while ($i < 4);
echo $total, "\n";
$k = 10;
while ($k > 0) {
    $k -= 3;
}
echo $k;
--EXPECT--
192
-2
