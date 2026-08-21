--TEST--
Cas ciblé concat_compound (EXPECT gravé depuis l'oracle php)
--FILE--
<?php
$log = "";
for ($i = 1; $i <= 4; $i++) {
    $log .= $i;
    $log .= ",";
}
echo $log, "\n";
$sep = "";
$out = "";
$parts = [7, 8, 9];
foreach ($parts as $p) {
    $out = $out . $sep . $p;
    $sep = "-";
}
echo $out, "\n";
echo strlen($log . $out);
--EXPECT--
1,2,3,4,
7-8-9
13
