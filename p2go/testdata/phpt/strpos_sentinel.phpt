--TEST--
strpos absent → SENTINELLE -1 (F-p2go-strpos-sentinel) — ÉCART ASSUMÉ vs PHP (false), EXPECT épinglé main, PAS généré par l'oracle
--FILE--
<?php
$h = "le corbeau et le renard";
echo strpos($h, "loup"), "\n";
if (strpos($h, "loup") < 0) {
    echo "absent", "\n";
}
if (strpos($h, "loup") == -1) {
    echo "sentinelle -1";
}
--EXPECT--
-1
absent
sentinelle -1
