--TEST--
Cas ciblé foreach_nested (EXPECT gravé depuis l'oracle php)
--FILE--
<?php
$rows = [1, 2, 3];
$cols = [10, 20];
$acc = 0;
foreach ($rows as $r) {
    foreach ($cols as $ci => $c) {
        $acc += $r * $c + $ci;
    }
}
echo $acc;
--EXPECT--
183
