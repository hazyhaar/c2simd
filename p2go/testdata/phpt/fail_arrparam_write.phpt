--TEST--
Refus fail-loud : écriture dans un paramètre tableau (sémantique de copie PHP non imitée)
--FILE--
<?php
function mutate(array $a) {
    $a[0] = 1;
    return $a[0];
}
$b = [5];
echo mutate($b);
--EXPECT_ERR--
err_parse
