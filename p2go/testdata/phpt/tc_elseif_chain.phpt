--TEST--
Cas ciblé elseif_chain (EXPECT gravé depuis l'oracle php)
--FILE--
<?php
function grade($n): string {
    if ($n >= 90) {
        return "A";
    } elseif ($n >= 80) {
        return "B";
    } elseif ($n >= 70) {
        return "C";
    } elseif ($n >= 60) {
        return "D";
    } else {
        return "F";
    }
}
echo grade(95), grade(85), grade(75), grade(65), grade(20);
--EXPECT--
ABCDF
