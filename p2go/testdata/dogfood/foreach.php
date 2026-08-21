<?php
// Vert vague E → F-p2go-foreach : foreach désucré en for indexé.
$a = [4, 8, 15, 16, 23, 42];
$s = 0;
foreach ($a as $v) {
    $s += $v;
}
foreach ($a as $i => $v) {
    $s += $i;
}
echo $s;
