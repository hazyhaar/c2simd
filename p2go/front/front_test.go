package front

import "testing"

func mustCode(t *testing.T, src, wantCode string) {
	t.Helper()
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("refus %s attendu, parse accepté", wantCode)
	}
	fe, ok := err.(*Err)
	if !ok {
		t.Fatalf("erreur non typée : %v", err)
	}
	if fe.Code != wantCode {
		t.Fatalf("code %s attendu, obtenu %s (%v)", wantCode, fe.Code, err)
	}
}

func TestFailLoud(t *testing.T) {
	cases := map[string]string{
		"<?php $x = $$y;":              "err_varvar",
		"<?php global $x;":             "err_global",
		"<?php include \"a.php\";":     "err_include",
		"<?php $o = new Foo();":        "err_oop",
		"<?php $o->m();":               "err_oop",
		"<?php $a = array(1, 2);":      "err_array",
		"<?php foreach ($a as &$v) {}": "err_parse",
		"<?php $x = 1; switch ($x) { case 1: $y = 2; case 2: break; }": "err_parse",
		"<?php while ($x ? 1 : 0) {}":                                  "err_parse",
		"<?php $x = 1.5;":                                              "err_float",
		"<?php echo \"a ${x}\";":                                       "err_interp",
		"<?php echo \"a {$x}\";":                                       "err_interp",
		"<?php":                                                        "err_empty",
		"<?php $x = ;":                                                 "err_parse",
	}
	for src, code := range cases {
		mustCode(t, src, code)
	}
}

func TestParseOK(t *testing.T) {
	src := `<?php
// commentaire
function add($a, $b) { return $a + $b; }
$x = add(1, 2) * 3;
echo $x, "\n";
for ($i = 0; $i < 3; ++$i) { $x += $i; }
while ($x > 0) { $x--; }
if (!($x == 0)) { echo 0; } else { echo 1; }
`
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse : %v", err)
	}
	if len(prog.Funcs) != 1 || len(prog.Main) != 5 {
		t.Fatalf("structure inattendue : %d funcs, %d stmts", len(prog.Funcs), len(prog.Main))
	}
}
