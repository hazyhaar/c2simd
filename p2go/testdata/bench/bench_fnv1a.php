<?php
// Charge scalaire pure : FNV-1a 64 (mul64 en colonnes 16-bit) sur 8 Kio, 20 passes.
function mul64($a, $b) {
    $a0 = $a & 0xFFFF;
    $a1 = ($a >> 16) & 0xFFFF;
    $a2 = ($a >> 32) & 0xFFFF;
    $a3 = ($a >> 48) & 0xFFFF;
    $b0 = $b & 0xFFFF;
    $b1 = ($b >> 16) & 0xFFFF;
    $b2 = ($b >> 32) & 0xFFFF;
    $b3 = ($b >> 48) & 0xFFFF;
    $c0 = $a0 * $b0;
    $c1 = (($c0 >> 16) & 0xFFFFFFFFFFFF) + $a0 * $b1 + $a1 * $b0;
    $c2 = (($c1 >> 16) & 0xFFFFFFFFFFFF) + $a0 * $b2 + $a1 * $b1 + $a2 * $b0;
    $c3 = (($c2 >> 16) & 0xFFFFFFFFFFFF) + $a0 * $b3 + $a1 * $b2 + $a2 * $b1 + $a3 * $b0;
    return (($c3 & 0xFFFF) << 48) | (($c2 & 0xFFFF) << 32) | (($c1 & 0xFFFF) << 16) | ($c0 & 0xFFFF);
}

function fnv1a_64(string $data) {
    $h = (0xcbf29ce4 << 32) | 0x84222325;
    $prime = 0x100000001b3;
    $n = strlen($data);
    for ($i = 0; $i < $n; $i++) {
        $h = $h ^ ord(substr($data, $i, 1));
        $h = mul64($h, $prime);
    }
    return $h;
}

$s = "fnv payload 0123456789 abcdefghijklmnopqrstuvwxyz";
for ($d = 0; $d < 8; $d++) {
    $s = $s . $s;
}
$acc = 0;
for ($r = 0; $r < 100; $r++) {
    $acc = $acc ^ fnv1a_64($s);
}
echo $acc, " ", strlen($s);
