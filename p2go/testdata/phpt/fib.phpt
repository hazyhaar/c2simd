--TEST--
Fibonacci récursif et itératif (subset v0.1 : fonctions pures, if, for)
--FILE--
<?php
function fib($n) {
    if ($n < 2) {
        return $n;
    }
    return fib($n - 1) + fib($n - 2);
}

function fib_iter($n) {
    $a = 0;
    $b = 1;
    for ($i = 0; $i < $n; $i++) {
        $t = $a + $b;
        $a = $b;
        $b = $t;
    }
    return $a;
}

for ($k = 0; $k <= 10; $k++) {
    if ($k > 0) {
        echo " ";
    }
    echo fib($k);
}
echo "\n";
echo fib_iter(30), "\n";
if (fib(20) == fib_iter(20)) {
    echo "parite ok";
}
--EXPECT--
0 1 1 2 3 5 8 13 21 34 55
832040
parite ok
