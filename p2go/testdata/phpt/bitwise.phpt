--TEST--
Opérateurs bit-à-bit et littéraux hexadécimaux (F-p2go-bitwise-ops, F-p2go-hex-literals)
--FILE--
<?php
$x = 0xFF;
$y = 0x0F;
echo $x & $y, "\n";
echo $x | 0x100, "\n";
echo $x ^ $y, "\n";
echo ~$y & 0xFF, "\n";
echo 1 << 10, "\n";
echo 0x8000 >> 4, "\n";
echo -8 >> 1, "\n";
$crc = 0xEDB88320;
echo $crc & 0xFFFFFFFF, "\n";
echo (3 & 1) == 1 && (4 | 1) > 4 ? 1 : 0;
--EXPECT--
15
511
240
240
1024
2048
-4
3988292384
1
