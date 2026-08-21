<?php
// Base64 (RFC 4648) : encodage et décodage via l'alphabet en string,
// composition d'octets par ord/substr/chr — zéro builtin base64.
function b64_val(string $c) {
    $o = ord($c);
    if ($o >= 65 && $o <= 90) {
        return $o - 65;
    }
    if ($o >= 97 && $o <= 122) {
        return $o - 97 + 26;
    }
    if ($o >= 48 && $o <= 57) {
        return $o - 48 + 52;
    }
    if ($o == 43) {
        return 62;
    }
    if ($o == 47) {
        return 63;
    }
    return -1;
}

function b64_encode(string $data): string {
    $alpha = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    $out = "";
    $n = strlen($data);
    $i = 0;
    while ($i + 3 <= $n) {
        $w = (ord(substr($data, $i, 1)) << 16)
            | (ord(substr($data, $i + 1, 1)) << 8)
            | ord(substr($data, $i + 2, 1));
        $out = $out . substr($alpha, ($w >> 18) & 63, 1)
            . substr($alpha, ($w >> 12) & 63, 1)
            . substr($alpha, ($w >> 6) & 63, 1)
            . substr($alpha, $w & 63, 1);
        $i = $i + 3;
    }
    $rest = $n - $i;
    if ($rest == 1) {
        $w = ord(substr($data, $i, 1)) << 16;
        $out = $out . substr($alpha, ($w >> 18) & 63, 1)
            . substr($alpha, ($w >> 12) & 63, 1) . "==";
    }
    if ($rest == 2) {
        $w = (ord(substr($data, $i, 1)) << 16) | (ord(substr($data, $i + 1, 1)) << 8);
        $out = $out . substr($alpha, ($w >> 18) & 63, 1)
            . substr($alpha, ($w >> 12) & 63, 1)
            . substr($alpha, ($w >> 6) & 63, 1) . "=";
    }
    return $out;
}

function b64_decode(string $s): string {
    $out = "";
    $n = strlen($s);
    $i = 0;
    while ($i + 4 <= $n) {
        $v0 = b64_val(substr($s, $i, 1));
        $v1 = b64_val(substr($s, $i + 1, 1));
        $v2 = b64_val(substr($s, $i + 2, 1));
        $v3 = b64_val(substr($s, $i + 3, 1));
        $w = ($v0 << 18) | ($v1 << 12);
        $out = $out . chr(($w >> 16) & 255);
        if ($v2 >= 0) {
            $w = $w | ($v2 << 6);
            $out = $out . chr(($w >> 8) & 255);
        }
        if ($v3 >= 0) {
            $w = $w | $v3;
            $out = $out . chr($w & 255);
        }
        $i = $i + 4;
    }
    return $out;
}

echo b64_encode("f"), " ", b64_encode("fo"), " ", b64_encode("foo"), "\n";
echo b64_encode("light work."), "\n";
echo b64_decode("bGlnaHQgd29yay4="), "\n";
$msg = "p2go: PHP -> Go 1.27, archsimd inside!";
$enc = b64_encode($msg);
echo $enc, "\n";
echo b64_decode($enc), "\n";
if (b64_decode($enc) == $msg) {
    echo "roundtrip ok";
}
