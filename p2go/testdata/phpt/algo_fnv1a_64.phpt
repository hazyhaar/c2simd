--TEST--
Algorithme réel fnv1a_64 — EXPECT gravé depuis l'oracle CLI php 8.3.6 (TestAlgorithmsVsPhpOracle le re-vérifie en vif)
--FILE--
<?php
// FNV-1a 64-bit. PHP promeut les overflows arithmétiques en float : le
// multiply mod 2^64 est décomposé en colonnes 16-bit (produits ≤ 2^32,
// sommes < 2^62, jamais de float) ; la recomposition est purement bitwise.
// Go wrappe nativement — les deux chemins restent bit-exacts.
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
    // 0xcbf29ce484222325 dépasse PHP_INT_MAX en littéral : composé en bitwise.
    $h = (0xcbf29ce4 << 32) | 0x84222325;
    $prime = 0x100000001b3;
    $n = strlen($data);
    for ($i = 0; $i < $n; $i++) {
        $h = $h ^ ord(substr($data, $i, 1));
        $h = mul64($h, $prime);
    }
    return $h;
}

echo fnv1a_64(""), "\n";
echo fnv1a_64("a"), "\n";
echo fnv1a_64("foobar"), "\n";
echo fnv1a_64("p2go: transpilation thesaurus-first vers Go 1.27");
--EXPECT--
-3750763034362895579
-5808556873153909620
-8821353812377114648
5616168750439945314
