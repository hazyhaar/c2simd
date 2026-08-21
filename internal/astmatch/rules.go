package astmatch

// RuleKind classe une entrée de la table ARCHTIME.
//
//	rewrite            — appliquée par TransformAST (DeadCode et/ou réécriture de corps/appels)
//	handwrite_pointer  — pointe vers une implémentation manuelle (simd_*.go) ; PAS un rewrite AST
//	declared           — documentée, pas encore de passe AST (ne pas prétendre le contraire)
type RuleKind string

const (
	KindRewrite          RuleKind = "rewrite"
	KindHandwritePointer RuleKind = "handwrite_pointer"
	KindDeclared         RuleKind = "declared"
)

// RuleDef définit le contrat statique d'une règle de transformation d'AST
// ou d'un pointeur vers une optimisation hors-gen.
type RuleDef struct {
	Symbol      string   // Symbole C/Go d'origine (ex: "rotl32")
	Replacement string   // Cible documentaire (bits.RotateLeft32, fichier hand-write, …)
	DeadCode    bool     // Purge la définition du symbole si true (rewrite only)
	Category    string   // Catégorie ARCHTIME
	Kind        RuleKind // rewrite | handwrite_pointer | declared
	Finding     string   // id finding spec/findings (optionnel)
}

// ArchtimeRulesTable — faits précalculés ARCHTIME (AGENTS.md §1).
//
// HONNÊTETÉ v2 (2026-08-10) :
//   - Seules les entrées Kind=rewrite sont consommées par TransformAST
//     (DeadCode + passes codées sur les Symbol concernés).
//   - handwrite_pointer : gains ChaCha/Poly dans simd_*.go, hors gen.
//   - declared : intention notée, pas de passe encore.
//   - Interdiction : RuleDef générique « boucle → simd »
//     (F-20260810-q2-generic-simd, peer review Q2).
//
// Passes structurelles hors table (hard-coded dans TransformAST) :
//
//	élision tls T0/T1, motif (x<<N)|(x>>(W-N)) → RotateLeft, fold libc.From*,
//	* (**T)(__ccgo_up(E)) → (*T)(unsafe.Pointer(E)).
var ArchtimeRulesTable = []RuleDef{
	{
		Symbol:      "rotl32",
		Replacement: "bits.RotateLeft32",
		DeadCode:    true,
		Category:    "bit_manipulation",
		Kind:        KindRewrite,
		Finding:     "F-20260810-rotl32-bits",
	},
	{
		// tweetnacl : static u32 L32(u32 x,int c) — appels L32(tls,x,c) → RotateLeft32
		Symbol:      "L32",
		Replacement: "bits.RotateLeft32",
		DeadCode:    true,
		Category:    "bit_manipulation",
		Kind:        KindRewrite,
		Finding:     "F-20260810-tweetnacl-l32",
	},
	{
		Symbol:      "load32_le",
		Replacement: "unsafe.Pointer(*uint32) // body rewrite (not encoding/binary call)",
		DeadCode:    false,
		Category:    "endianness",
		Kind:        KindRewrite,
		Finding:     "F-20260810-load32-le",
	},
	{
		Symbol:      "store32_le",
		Replacement: "unsafe.Pointer(*uint32) // body rewrite",
		DeadCode:    false,
		Category:    "endianness",
		Kind:        KindRewrite,
		Finding:     "F-20260810-load32-le",
	},
	{
		Symbol:      "crypto_wipe",
		Replacement: "runtime_memclr",
		DeadCode:    false,
		Category:    "security",
		Kind:        KindDeclared,
	},
	{
		Symbol:      "poly_blocks",
		Replacement: "simd_poly1305_*.go (hand-write dual-chain / ymm)",
		DeadCode:    false,
		Category:    "handwrite_pointer",
		Kind:        KindHandwritePointer,
	},
	{
		Symbol:      "chacha20_rounds",
		Replacement: "simd_chacha20.go (hand-write simd256 unrolled)",
		DeadCode:    false,
		Category:    "handwrite_pointer",
		Kind:        KindHandwritePointer,
	},
}

// AppliedRules retourne les règles réellement consommées par TransformAST.
func AppliedRules() []RuleDef {
	out := make([]RuleDef, 0, len(ArchtimeRulesTable))
	for _, r := range ArchtimeRulesTable {
		if r.Kind == KindRewrite {
			out = append(out, r)
		}
	}
	return out
}

// HandwritePointers retourne les pointeurs vers implémentations manuelles.
func HandwritePointers() []RuleDef {
	out := make([]RuleDef, 0, len(ArchtimeRulesTable))
	for _, r := range ArchtimeRulesTable {
		if r.Kind == KindHandwritePointer {
			out = append(out, r)
		}
	}
	return out
}
