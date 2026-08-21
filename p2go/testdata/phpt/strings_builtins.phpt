--TEST--
Builtins strings (F-p2go-stdlib-strings) : casse ASCII, substr PHP, str_replace, trim, ord/chr
--FILE--
<?php
$s = "Hello, World";
echo strtoupper($s), "\n";
echo strtolower($s), "\n";
echo substr($s, 7, 5), "\n";
echo substr($s, -5, 5), "\n";
echo substr($s, 3, -4), "\n";
echo str_replace("World", "PHP", $s), "\n";
echo trim("  du texte\t\n"), "\n";
echo ord("A"), " ", chr(66), chr(67), "\n";
echo ord(substr($s, 1, 1)), "\n";
echo strlen(trim(" a "));
--EXPECT--
HELLO, WORLD
hello, world
World
World
lo, W
Hello, PHP
du texte
65 BC
101
1
