--TEST--
v0.5 : strpos (cas trouvés) et ordre de strings (EXPECT gravé depuis l'oracle php)
--FILE--
<?php
$h = "le corbeau et le renard";
echo strpos($h, "corbeau"), " ", strpos($h, "le"), " ", strpos($h, "renard"), "\n";
if (strpos($h, "renard") >= 0) {
    echo "trouve", "\n";
}
echo "abc" < "abd" ? 1 : 0, " ", "b" > "a" ? 1 : 0, " ", "Z" < "a" ? 1 : 0, "\n";
$a = "pomme";
$b = "poire";
if ($a >= $b) {
    echo "pomme apres poire";
}
--EXPECT--
3 0 17
trouve
1 1 1
pomme apres poire
