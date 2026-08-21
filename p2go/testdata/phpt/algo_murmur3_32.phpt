--TEST--
Algorithme réel murmur3_32 — EXPECT gravé depuis l'oracle CLI php 8.3.6 (TestAlgorithmsVsPhpOracle le re-vérifie en vif)
--FILE--
<?php
// MurmurHash3 x86 32-bit, seed paramétrable. Multiplications 32-bit
// décomposées en demi-mots 16-bit (pas de promotion float PHP) ; tout état
// est masqué & 0xFFFFFFFF.
function mul32($a, $b) {
    $a0 = $a & 0xFFFF;
    $a1 = ($a >> 16) & 0xFFFF;
    $b0 = $b & 0xFFFF;
    $b1 = ($b >> 16) & 0xFFFF;
    return ($a0 * $b0 + ((($a0 * $b1 + $a1 * $b0) & 0xFFFF) << 16)) & 0xFFFFFFFF;
}

function rotl32($x, $r) {
    return (($x << $r) | (($x & 0xFFFFFFFF) >> (32 - $r))) & 0xFFFFFFFF;
}

function murmur3_32(string $key, $seed) {
    $c1 = 0xcc9e2d51;
    $c2 = 0x1b873593;
    $h = $seed & 0xFFFFFFFF;
    $n = strlen($key);
    $blocks = intdiv($n, 4);

    for ($i = 0; $i < $blocks; $i++) {
        $o = $i * 4;
        $k = ord(substr($key, $o, 1))
            | (ord(substr($key, $o + 1, 1)) << 8)
            | (ord(substr($key, $o + 2, 1)) << 16)
            | (ord(substr($key, $o + 3, 1)) << 24);
        $k = mul32($k, $c1);
        $k = rotl32($k, 15);
        $k = mul32($k, $c2);
        $h = $h ^ $k;
        $h = rotl32($h, 13);
        $h = (mul32($h, 5) + 0xe6546b64) & 0xFFFFFFFF;
    }

    $tail = $n - $blocks * 4;
    $k = 0;
    if ($tail >= 3) {
        $k = $k ^ (ord(substr($key, $blocks * 4 + 2, 1)) << 16);
    }
    if ($tail >= 2) {
        $k = $k ^ (ord(substr($key, $blocks * 4 + 1, 1)) << 8);
    }
    if ($tail >= 1) {
        $k = $k ^ ord(substr($key, $blocks * 4, 1));
        $k = mul32($k, $c1);
        $k = rotl32($k, 15);
        $k = mul32($k, $c2);
        $h = $h ^ $k;
    }

    $h = $h ^ $n;
    $h = $h ^ (($h & 0xFFFFFFFF) >> 16);
    $h = mul32($h, 0x85ebca6b);
    $h = $h ^ (($h & 0xFFFFFFFF) >> 13);
    $h = mul32($h, 0xc2b2ae35);
    $h = $h ^ (($h & 0xFFFFFFFF) >> 16);
    return $h & 0xFFFFFFFF;
}

echo murmur3_32("", 0), "\n";
echo murmur3_32("hello", 0), "\n";
echo murmur3_32("hello, world", 0), "\n";
echo murmur3_32("The quick brown fox jumps over the lazy dog", 0x9747b28c);
--EXPECT--
0
613153351
345750399
799549133
