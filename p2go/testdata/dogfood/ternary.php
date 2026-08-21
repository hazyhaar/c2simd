<?php
// Vert vague B → F-p2go-ternary-expr (landed) : ternaire en position statement.
$x = 7;
$y = $x > 5 ? 1 : 0;
$w = $y ?: 42;
echo $y + $w;
