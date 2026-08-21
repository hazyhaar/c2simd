--TEST--
Refus fail-loud : array(…) hors subset (la voie normée est le littéral [ … ])
--FILE--
<?php
$a = array(1, 2, 3);
echo $a[0];
--EXPECT_ERR--
err_array
