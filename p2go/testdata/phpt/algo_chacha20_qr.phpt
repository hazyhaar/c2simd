--TEST--
Algorithme réel chacha20_qr — EXPECT gravé depuis l'oracle CLI php 8.3.6 (TestAlgorithmsVsPhpOracle le re-vérifie en vif)
--FILE--
<?php
// Quarter-round ChaCha20 (RFC 7539 §2.1) sur mots 32-bit : additions mod 2^32
// masquées (sommes < 2^33, jamais de float PHP), rotations composées.
// Vecteur §2.1.1 : QR(0x11111111, 0x01020304, 0x9b8d6f43, 0x01234567)
// → (0xea2a92f4, 0xcb1cf8ce, 0x4581472e, 0x5881c4bb).
function rotl32($x, $r) {
    return (($x << $r) | (($x & 0xFFFFFFFF) >> (32 - $r))) & 0xFFFFFFFF;
}

// Le quarter-round opère sur l'état 16 mots, indices en place.
function qr(array $st, int $ia, int $ib, int $ic, int $id): array {
    $a = $st[$ia];
    $b = $st[$ib];
    $c = $st[$ic];
    $d = $st[$id];

    $a = ($a + $b) & 0xFFFFFFFF;
    $d = rotl32($d ^ $a, 16);
    $c = ($c + $d) & 0xFFFFFFFF;
    $b = rotl32($b ^ $c, 12);
    $a = ($a + $b) & 0xFFFFFFFF;
    $d = rotl32($d ^ $a, 8);
    $c = ($c + $d) & 0xFFFFFFFF;
    $b = rotl32($b ^ $c, 7);

    $out = array_slice($st, 0, count($st));
    $out[$ia] = $a;
    $out[$ib] = $b;
    $out[$ic] = $c;
    $out[$id] = $d;
    return $out;
}

$st = [0x11111111, 0x01020304, 0x9b8d6f43, 0x01234567,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0];
$st = qr($st, 0, 1, 2, 3);
echo $st[0], " ", $st[1], " ", $st[2], " ", $st[3], "\n";

// Vecteur RFC §2.2.1 : QR sur l'état complet, indices 2,7,8,13.
$st2 = [0x879531e0, 0xc5ecf37d, 0x516461b1, 0xc9a62f8a,
    0x44c20ef3, 0x3390af7f, 0xd9fc690b, 0x2a5f714c,
    0x53372767, 0xb00a5631, 0x974c541a, 0x359e9963,
    0x5c971061, 0x3d631689, 0x2098d9d6, 0x91dbd320];
$st2 = qr($st2, 2, 7, 8, 13);
foreach ($st2 as $k => $w) {
    if ($k > 0) {
        echo " ";
    }
    echo $w;
}
--EXPECT--
3928658676 3407673550 1166100270 1484899515
2274701792 3320640381 3182986972 3383111562 1153568499 865120127 3657197835 3484200914 3832277632 2953467441 2538361882 899586403 1553404001 3435166841 546888150 2447102752
