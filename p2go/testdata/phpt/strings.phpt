--TEST--
Strings première classe (F-p2go-string-concat, F-p2go-string-interp) : concat, .=, interpolation, strlen, égalité
--FILE--
<?php
$prenom = "ada";
$nom = "lovelace";
$plein = $prenom . " " . $nom;
echo $plein, "\n";

$s = "n=";
$n = 42;
$s .= $n;
$s = $s . "!";
echo $s, "\n";

echo "somme: " . (1 + 2) . " fois\n";

$msg = "compte $n unités, $prenom";
echo $msg, "\n";

echo strlen($plein), "\n";

if ($plein == "ada lovelace") {
    echo "egal", "\n";
}
if ($prenom != $nom) {
    echo "differents";
}
--EXPECT--
ada lovelace
n=42!
somme: 3 fois
compte 42 unités, ada
12
egal
differents
