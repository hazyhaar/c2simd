<?php
// Charge scalaire pure : CRC32 IEEE sans table sur 8 Kio, 20 passes.
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

$s = "p2go benchmark payload 0123456789 abcdefghijklmnopqrstuvwxyz !";
for ($d = 0; $d < 7; $d++) {
    $s = $s . $s;
}
$acc = 0;
for ($r = 0; $r < 100; $r++) {
    $acc = ($acc + crc32_ieee($s)) % 1000000007;
}
echo $acc, " ", strlen($s);
