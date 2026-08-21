--TEST--
Cas ciblé string_escapes (EXPECT gravé depuis l'oracle php)
--FILE--
<?php
echo "tab\there\n";
echo "dollar \$var et backslash \\ fin\n";
echo 'simple $pas_interpole \n litteral', "\n";
echo 'quote \' ok', "\n";
$a = "x";
echo strlen("a\tb\nc"), "\n";
echo "fin";
--EXPECT--
tab	here
dollar $var et backslash \ fin
simple $pas_interpole \n litteral
quote ' ok
5
fin
