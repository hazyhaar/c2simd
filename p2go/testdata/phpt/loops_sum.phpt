--TEST--
Boucles while/for, composés, elseif, logique, division entière
--FILE--
<?php
function collatz_len($n) {
    $len = 0;
    while ($n != 1) {
        if ($n % 2 == 0) {
            $n /= 2;
        } else {
            $n = 3 * $n + 1;
        }
        $len++;
    }
    return $len;
}

$sum = 0;
for ($i = 1; $i <= 100; $i++) {
    $sum += $i;
}
echo $sum, "\n";

echo collatz_len(27), "\n";

$x = 7;
if ($x > 10) {
    echo "grand";
} elseif ($x > 5 && $x < 10) {
    echo "moyen";
} else {
    echo "petit";
}
echo "\n";

$j = 10;
$acc = 0;
while ($j > 0 || !$acc) {
    $acc += $j;
    --$j;
    if ($j == 0) {
        $acc = $acc * 2 - 55;
    }
}
echo $acc;
--EXPECT--
5050
111
moyen
55
