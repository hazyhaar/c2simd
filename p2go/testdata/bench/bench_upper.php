<?php
// Charge simd_ascii_case : strtoupper/strtolower sur 64 Kio, 200 passes.
$s = "The quick brown Fox jumps over the lazy Dog 0123456789 !";
for ($d = 0; $d < 10; $d++) {
    $s = $s . $s;
}
$acc = 0;
for ($r = 0; $r < 2000; $r++) {
    $u = strtoupper($s);
    $l = strtolower($u);
    $acc = ($acc + ord(substr($u, $r, 1)) + ord(substr($l, $r + 1, 1))) % 1000003;
}
echo $acc, " ", strlen($s);
