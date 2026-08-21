--TEST--
Fonctions scalaires (F-p2go-scalar-signatures) : params/retours string par hints, mixte int
--FILE--
<?php
function greet(string $qui): string {
    return "salut " . $qui;
}

function double($n) {
    return $n * 2;
}

function repeter(string $s, $n): string {
    $out = "";
    for ($i = 0; $i < $n; $i++) {
        $out .= $s;
    }
    return $out;
}

echo greet("monde"), "\n";
echo double(21), "\n";
echo repeter("ab", 3), "\n";
echo strlen(greet("x"));
--EXPECT--
salut monde
42
ababab
7
