<?php
// Quicksort itératif (partition de Lomuto, pile explicite array_push/array_pop)
// sur tableau int64 local — les tableaux mutables vivent au top-level.
$a = [93, -7, 42, 0, 13, 42, -100, 8, 77, 3, 21, -19, 55, 1, 99, -3, 64, 30, 12, 5];

$stack = [];
array_push($stack, 0);
array_push($stack, count($a) - 1);

while (count($stack) > 0) {
    $hi = array_pop($stack);
    $lo = array_pop($stack);
    if ($lo < $hi) {
        $p = $a[$hi];
        $i = $lo - 1;
        for ($j = $lo; $j < $hi; $j++) {
            if ($a[$j] <= $p) {
                $i++;
                $t = $a[$i];
                $a[$i] = $a[$j];
                $a[$j] = $t;
            }
        }
        $i++;
        $t = $a[$i];
        $a[$i] = $a[$hi];
        $a[$hi] = $t;
        array_push($stack, $lo);
        array_push($stack, $i - 1);
        array_push($stack, $i + 1);
        array_push($stack, $hi);
    }
}

foreach ($a as $k => $v) {
    if ($k > 0) {
        echo " ";
    }
    echo $v;
}
echo "\n";
echo count($a);
