--TEST--
Refus fail-loud : eval hors subset
--FILE--
<?php
$x = 1;
eval("echo 1;");
--EXPECT_ERR--
err_eval
