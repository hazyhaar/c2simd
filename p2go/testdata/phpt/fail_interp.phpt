--TEST--
Refus fail-loud : interpolation ${…} et {$…} hors subset ($ident seul supporté)
--FILE--
<?php
$x = 5;
echo "valeur ${x}";
--EXPECT_ERR--
err_interp
