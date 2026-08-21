--TEST--
Cas ciblé logic_shortcircuit (EXPECT gravé depuis l'oracle php)
--FILE--
<?php
function t(string $tag): int {
    echo $tag;
    return 1;
}
function f(string $tag): int {
    echo $tag;
    return 0;
}
if (t("a") || t("b")) {
    echo "|or";
}
echo "\n";
if (f("c") && t("d")) {
    echo "non";
} else {
    echo "|and";
}
echo "\n";
if (f("e") || t("g")) {
    echo "|mix";
}
--EXPECT--
a|or
c|and
eg|mix
