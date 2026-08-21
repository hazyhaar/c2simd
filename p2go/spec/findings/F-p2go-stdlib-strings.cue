package findings

F_p2go_stdlib_strings: #Finding & {
	id:      "F-p2go-stdlib-strings"
	kernel:  "substr, str_replace, trim, strtoupper, strtolower, ord, chr"
	stage:   "emit"
	symptom: "Aucune primitive de chaîne — l'accès aux octets (ord/substr) est le prérequis des hachages et de base64."
	evidence: {
		file_line: "emit/builtins.go helperSrc (p2goSubstr, p2goToUpper/Lower, p2goOrd)"
		fixture:   "testdata/phpt/strings_builtins.phpt"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Helpers à sémantique PHP EXACTE, jamais l'équivalent Go approché : substr avec offsets négatifs et clamp PHP 8 ; trim sur la charlist PHP \" \\t\\n\\r\\0\\x0B\" (PAS TrimSpace unicode) ; strtoupper/strtolower ASCII octet à octet (PAS strings.ToUpper unicode) ; chr = octet mod 256 ; str_replace → ReplaceAll avec permutation d'arguments (search,replace,subject → subject,search,replace)."
	status: "landed"
	notes:  "strpos différé : son retour false PHP n'a pas d'équivalent int honnête (v0.5, avec un kind bool|int ou une variante à sentinelle documentée)."
}
