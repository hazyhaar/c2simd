// Package p2go — transpileur PHP→Go 1.27 itératif (modèle sgoiter) : pipeline
// déterministe 5 passes front → types → ir → rules → emit. Voir SPEC.md.
package p2go

import (
	"code.hazyhaar.fr/devhoros/c2simd/p2go/emit"
	"code.hazyhaar.fr/devhoros/c2simd/p2go/front"
	"code.hazyhaar.fr/devhoros/c2simd/p2go/ir"
	"code.hazyhaar.fr/devhoros/c2simd/p2go/rules"
	"code.hazyhaar.fr/devhoros/c2simd/p2go/types"
)

// TranspileFiles transforme une source PHP (subset v0.1/v0.2 partiel) en
// programme Go complet : map nom de fichier → contenu ("main.go" toujours ;
// helpers duals scalaire/SIMD ajoutés si une réduction vectorisable est
// reconnue). Toute sortie du subset retourne une *front.Err (code err_*).
func TranspileFiles(src, srcName string) (map[string]string, error) {
	prog, err := front.Parse(src)
	if err != nil {
		return nil, err
	}
	info, err := types.Check(prog)
	if err != nil {
		return nil, err
	}
	lowered := rules.Apply(ir.Lower(prog, info))
	return emit.EmitFiles(lowered, srcName), nil
}

// Transpile conserve l'API mono-fichier : refuse (err_simd_multifile) un
// programme dont l'emit produit des helpers à build tags, insérables
// uniquement en fichiers séparés.
func Transpile(src, srcName string) (string, error) {
	files, err := TranspileFiles(src, srcName)
	if err != nil {
		return "", err
	}
	if len(files) > 1 {
		return "", &front.Err{Code: "err_simd_multifile", Line: 1,
			Msg: "programme vectorisé multi-fichiers : utiliser TranspileFiles (CLI : -outdir)"}
	}
	return files["main.go"], nil
}
