--TEST--
Casse ASCII vectorisée (F-p2go-simd-ascii-case) : string > 32 octets, multioctets UTF-8 intacts
--FILE--
<?php
$s = "Portez ce vieux whisky au juge blond qui fume: 0123456789 & ÉÀ";
echo strtoupper($s), "\n";
echo strtolower($s);
--EXPECT--
PORTEZ CE VIEUX WHISKY AU JUGE BLOND QUI FUME: 0123456789 & ÉÀ
portez ce vieux whisky au juge blond qui fume: 0123456789 & ÉÀ
