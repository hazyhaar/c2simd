--TEST--
Cas ciblé switch_string_dispatch (EXPECT gravé depuis l'oracle php)
--FILE--
<?php
function famille(string $op) {
    switch ($op) {
        case "add":
        case "sub":
            return 1;
        case "mul":
        case "div":
        case "mod":
            return 2;
        case "concat":
            return 3;
        default:
            return 0;
    }
}
echo famille("add"), famille("sub"), famille("mul"), famille("mod"), famille("concat"), famille("xyz");
--EXPECT--
112230
