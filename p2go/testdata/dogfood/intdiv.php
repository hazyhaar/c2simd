<?php
// Capture dogfood → F-p2go-intdiv-builtin (landed) : division entière PHP 7+.
echo intdiv(17, 5), "\n";
echo intdiv(-17, 5), "\n";
echo intdiv(100, intdiv(10, 3));
