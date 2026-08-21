--TEST--
Builtins tableaux (F-p2go-stdlib-arrays) : push/pop, reverse, slice, fill, in_array
--FILE--
<?php
$a = [1, 2, 3];
array_push($a, 4);
array_push($a, 5);
echo count($a), "\n";
$last = array_pop($a);
echo $last, " ", count($a), "\n";
$r = array_reverse($a);
echo $r[0], $r[1], $r[2], $r[3], "\n";
$s = array_slice($a, 1, 2);
echo $s[0], $s[1], "\n";
$t = array_slice($a, -2, 2);
echo $t[0], $t[1], "\n";
$f = array_fill(0, 4, 7);
echo $f[0] + $f[3], "\n";
if (in_array(3, $a)) {
    echo "present", "\n";
}
if (!in_array(99, $a)) {
    echo "absent";
}
--EXPECT--
5
5 4
4321
23
34
14
present
absent
