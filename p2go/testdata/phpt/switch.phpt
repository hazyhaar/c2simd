--TEST--
switch strict (F-p2go-switch) : break obligatoire, cases empilés, default, sujet string
--FILE--
<?php
function nom($n) {
    switch ($n) {
        case 1:
            return 100;
        case 2:
        case 3:
            return 230;
        default:
            return -1;
    }
}

echo nom(1), " ", nom(2), " ", nom(3), " ", nom(9), "\n";

$fruit = "pomme";
switch ($fruit) {
    case "poire":
        echo "non";
        break;
    case "pomme":
        echo "oui";
        break;
    default:
        echo "inconnu";
        break;
}
echo "\n";

$c = 0;
switch (2 + 1) {
    case 3:
        $c = 42;
        break;
}
echo $c;
--EXPECT--
100 230 230 -1
oui
42
