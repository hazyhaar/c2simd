--TEST--
Cas ciblé minmax_chain (EXPECT gravé depuis l'oracle php)
--FILE--
<?php
$vals = [14, 3, 27, 9];
$lo = $vals[0];
$hi = $vals[0];
for ($i = 0; $i < count($vals); $i++) {
    if ($vals[$i] < $lo) {
        $lo = $vals[$i];
    }
}
for ($j = 0; $j < count($vals); $j++) {
    if ($vals[$j] > $hi) {
        $hi = $vals[$j];
    }
}
echo min(max($lo, 0), 100), " ", max($hi, abs(-30)), "\n";
echo pow(min(2, 3), max(2, 3));
--EXPECT--
3 30
8
