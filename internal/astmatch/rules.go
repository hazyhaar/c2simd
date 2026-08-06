package astmatch

// RuleDef définit le contrat statique d'une règle de transformation d'AST
type RuleDef struct {
	Symbol      string // Symbole C d'origine (ex: "rotl32", "poly_blocks")
	Replacement string // Symbol Go cible ou nom de passe d'inlining
	DeadCode    bool   // Vrai si le symbole C d'origine doit être entièrement purgé du binaire
	Category    string // Catégorie d'optimisation ARCHTIME
}

// ArchtimeRulesTable est la table de faits précalculés statique (ARCHTIME Doctrine - AGENTS.md §1)
// Elle thésaurise les règles de réécriture et d'optimisation identifiées lors des audits et benchmarks :
// 1. Substitution des rotations rotl32 -> math/bits.RotateLeft32 (Inlining matériel ROL).
// 2. Élimination des wrappers libc et suppression du code mort.
// 3. Transformation des boucles d'accumulation séquentielle Poly1305 en Dual-Chain parallèle (ADCX/ADOX ILP).
// 4. Insertion d'ancres d'Élimination des Vérifications de Bornes (BCE).
var ArchtimeRulesTable = []RuleDef{
	{
		Symbol:      "rotl32",
		Replacement: "bits.RotateLeft32",
		DeadCode:    true,
		Category:    "bit_manipulation",
	},
	{
		Symbol:      "load32_le",
		Replacement: "binary.LittleEndian.Uint32",
		DeadCode:    false,
		Category:    "endianness",
	},
	{
		Symbol:      "store32_le",
		Replacement: "binary.LittleEndian.PutUint32",
		DeadCode:    false,
		Category:    "endianness",
	},
	{
		Symbol:      "crypto_wipe",
		Replacement: "runtime_memclr",
		DeadCode:    false,
		Category:    "security",
	},
	{
		Symbol:      "poly_blocks",
		Replacement: "poly1305_dual_chain_interleaved",
		DeadCode:    false,
		Category:    "superscalar_ilp",
	},
	{
		Symbol:      "chacha20_rounds",
		Replacement: "simd256_unrolled_4block",
		DeadCode:    false,
		Category:    "simd_unroll",
	},
}
