<?php
// Capture dogfood → F-p2go-do-while (landed) : construct courant des ports C.
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
echo $len;
