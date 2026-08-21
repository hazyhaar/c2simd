--TEST--
Cas ciblé array_edge_slices (EXPECT gravé depuis l'oracle php)
--FILE--
<?php
$a = [1, 2, 3, 4, 5, 6];
$s1 = array_slice($a, -3, 2);
echo $s1[0], $s1[1], "\n";
$s2 = array_slice($a, 2, -1);
echo count($s2), " ", $s2[0], "\n";
$r = array_reverse(array_slice($a, 0, 3));
echo $r[0], $r[1], $r[2], "\n";
$f = array_fill(0, 3, -1);
$t = 0;
foreach ($f as $v) {
    $t += $v;
}
echo $t;
--EXPECT--
45
3 3
321
-3
