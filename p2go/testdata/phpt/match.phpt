--TEST--
match PHP 8 (F-p2go-match) désucré en switch : arms multiples, default, en argument
--FILE--
<?php
$code = 2;
$msg = match ($code) {
    1 => "un",
    2, 3 => "deux-ou-trois",
    default => "autre",
};
echo $msg, "\n";

function double($n) {
    return $n * 2;
}
echo double(match ($code) { 2 => 21, default => 0 }), "\n";

echo match ("b") {
    "a" => 1,
    "b" => 2,
    default => 0,
};
--EXPECT--
deux-ou-trois
42
2
