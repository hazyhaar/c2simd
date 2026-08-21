package findings

F_sgoiter_dead_after_return: #Finding & {
	id:      "F-sgoiter-dead-after-return"
	kernel:  "élimination du code mort après return"
	stage:   "emit"
	symptom: "Dans tt55/gen_tt.go, les incréments de boucle ForPost (v26 = v5 + 1; v5 = v26) étaient émis après un return inconditionnel dans le corps de boucle."
	evidence: {
		file_line: "sgoiter/emit/emit.go:stmtsEndReturn, sgoiter/emit/ast_more_passes.go:astEliminateDeadCodeAfterReturn"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Conditionnement de l'émission de ForPost à !stmtsEndReturn(s.ForBody), arrêt de emitStmts dès un return terminal de bloc, et passe AST astEliminateDeadCodeAfterReturn tronquant les nœuds après un ReturnStmt."
	status: "landed"
}
