--TEST--
v0.5 : break/continue de boucle — while, for, foreach, break de switch dans une boucle
--FILE--
<?php
$s = 0;
for ($i = 0; $i < 100; $i++) {
    if ($i % 2 == 0) {
        continue;
    }
    if ($i > 10) {
        break;
    }
    $s += $i;
}
echo $s, "\n";

$a = [5, 3, 99, 7, 99, 1];
$found = -1;
foreach ($a as $k => $v) {
    if ($v == 99) {
        $found = $k;
        break;
    }
}
echo $found, "\n";

$n = 0;
$out = "";
while (1 == 1) {
    $n++;
    switch ($n % 3) {
        case 0:
            $out .= "z";
            break;
        case 1:
            break;
        default:
            $out .= "x";
            break;
    }
    if ($n >= 7) {
        break;
    }
}
echo $out, " ", $n;
--EXPECT--
25
2
xzxz 7
