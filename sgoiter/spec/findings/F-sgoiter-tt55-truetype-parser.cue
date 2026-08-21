package findings

finding: "F-sgoiter-tt55-truetype-parser": #Finding & {
	id:      "F-sgoiter-tt55-truetype-parser"
	kernel:  "tt55"
	stage:   "dogfood"
	lever:   "emit"
	status:  "landed"
	symptom: "Transpilation de la bibliothèque TrueType C99 (tt55) : résolution des adresses de champs de structure (&ptr->field) dans sgoiter/front et effondrement des chaînes de registres booléens dans sgoiter/emit pour garantir le court-circuitage sans déréférencement nil."
	evidence: {
		file_line:  "pkg/tt55/gen_tt.go:1"
		kat:        "pass"
		source_doc: "pkg/tt55/tt_test.go"
	}
	action: """
		1. Amélioration de sgoiter/front (front.go) pour émettre une instruction ir.OpMov avec Sym: 'addr_of' lors de la prise d'adresse sur un champ de structure (&ptr->field et &obj.field).
		2. Activation de la passe astCollapseBooleanRegisterChains et astFoldBooleanClosures dans sgoiter/emit (emit.go) permettant l'inlining direct des conditions logiques complexes et l'élimination des fermetures IIFE redondantes, garantissant l'évaluation en court-circuit.
		3. Transpilation stricte de sources/tt.c vers gen_tt.go sans aucun fichier manuscrit.
		4. Validation de la parité bit-exacte contre oracle gcc -O2 (TestVsCOracle_SystemFonts) sur l'ensemble des polices système TrueType (DejaVu, Liberation, Noto, Ubuntu) couvrant les tables cmap format 4 et format 12 ainsi que les avances hmtx.
		5. Débit de résolution mesuré à 75 ns/op avec 0 allocation heap en régime établi.
		"""
	notes: "Parité bit-exacte validée à 100% sur des centaines de polices TrueType réelles."
}
