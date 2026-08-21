<?php
// CRC32 IEEE 802.3 (réflexe, polynôme 0xEDB88320), sans table.
// Vecteur canonique : crc32("123456789") = 0xCBF43926 = 3421780262.
function crc32_ieee(string $data) {
    $crc = 0xFFFFFFFF;
    $n = strlen($data);
    for ($i = 0; $i < $n; $i++) {
        $crc = $crc ^ ord(substr($data, $i, 1));
        for ($j = 0; $j < 8; $j++) {
            if (($crc & 1) == 1) {
                $crc = ($crc >> 1) ^ 0xEDB88320;
            } else {
                $crc = $crc >> 1;
            }
        }
    }
    return $crc ^ 0xFFFFFFFF;
}

echo crc32_ieee("123456789"), "\n";
echo crc32_ieee(""), "\n";
echo crc32_ieee("p2go transpile PHP vers Go 1.27 avec archsimd"), "\n";
echo crc32_ieee("The quick brown fox jumps over the lazy dog");
