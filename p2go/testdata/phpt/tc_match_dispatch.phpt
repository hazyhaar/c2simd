--TEST--
Cas ciblé match_dispatch (EXPECT gravé depuis l'oracle php)
--FILE--
<?php
function nom_code($c): string {
    return match ($c) {
        200, 201, 204 => "ok",
        301, 302 => "redirect",
        404 => "notfound",
        default => "autre",
    };
}
$codes = [200, 302, 404, 500, 204];
foreach ($codes as $k => $c) {
    if ($k > 0) {
        echo " ";
    }
    echo nom_code($c);
}
--EXPECT--
ok redirect notfound autre ok
