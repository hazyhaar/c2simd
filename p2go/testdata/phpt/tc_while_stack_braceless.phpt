--TEST--
Cas ciblé while_stack_braceless (EXPECT gravé depuis l'oracle php)
--FILE--
<?php
$s = 0;
$i = 0;
while ($i < 5)
    $i += 1;
echo $i, "\n";
for ($j = 0; $j < 3; $j++)
    if ($j % 2 == 0)
        $s += 100;
echo $s;
--EXPECT--
5
200
