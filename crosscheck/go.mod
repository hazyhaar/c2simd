module code.hazyhaar.fr/devhoros/c2simd/crosscheck

go 1.27rc1

require (
	code.hazyhaar.fr/devhoros/c2simd v0.0.0
	code.hazyhaar.fr/devhoros/pkg/monocypher55 v0.0.0
	golang.org/x/crypto v0.54.0
)

require golang.org/x/sys v0.47.0 // indirect

replace code.hazyhaar.fr/devhoros/c2simd => ../

replace code.hazyhaar.fr/devhoros/pkg/monocypher55 => ../../pkg/monocypher55
