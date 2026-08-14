package findings

// Audit profond 2026-08-11 — C13. Modèle switch sans break : fallthrough inconditionnel.
F_sgoiter_switch_no_break_model: #Finding & {
	id:     "F-sgoiter-switch-no-break-model"
	kernel: "siphash24|murmur3_x86_32|*"
	stage:  "ir"
	symptom: "L'émetteur produit fallthrough entre toutes les cases (emit.go:662-664) et SwitchCase (ir/ir.go:114-119) n'a aucun champ break/fallthrough. Fidèle sur les switchs tombants (queues siphash/murmur), mais un break médian C est irreprésentable."
	evidence: {
		file_line: "emit/emit.go:662-664 ; ir/ir.go:114-119"
		kat:       "pass"
		source_doc: "spec/findings/HARVEST_audit_profond_20260811.md#C13"
	}
	lever:  "ir_rule"
	action: "Ajouter le terminateur de case à l'IR (break|fallthrough) ; vérifier ce que le front fait aujourd'hui d'un break médian (rejet fail-loud ou moisson fausse ?) — comportement non couvert, à sonder d'abord."
	status: "proposed"
	notes:  "Le comportement du front sur break médian n'a pas été vérifié : sonder avant de patcher."
}
