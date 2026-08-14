package findings

F_sgoiter_cpp_ifdef_amalg: #Finding & {
	id:     "F-sgoiter-cpp-ifdef-amalg"
	kernel: "monocypher.h+c amalg"
	stage:  "front"
	symptom: "stripIfDefs garde #ifdef __cplusplus → extern C { } casse le brace balance; update introuvable."
	evidence: {
		file_line: "amalg monocypher_amalg.c; normalize voit prototypes only si guards C++ gardés"
		kat:       "n/a"
	}
	lever:  "front"
	action: "Amalgame: strip #ifdef __cplusplus / MONOCYPHER_CPP_NAMESPACE avant sgoiter; ou stripIfDefs skip ces macros."
	status: "codified"
	notes:  "Fichier: spec/c_sources/upstream/monocypher/4.0.2/monocypher_amalg.c (guards retirés)."
}
