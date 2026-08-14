package probebench

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// LibFlow is ARCHTIME documentation of what a kernel does and how strata map.
type LibFlow struct {
	ID       string
	Title    string
	Contract string // bit-exact role
	Flux     []string
	EmitNote string // what sgoiter currently emits (short)
	Strata   string // how to read the layer table
	Levers   []string // doctrine-safe next steps only
}

// CatalogFlows ordered like DefaultLibs.
func CatalogFlows() []LibFlow {
	return []LibFlow{
		{
			ID: "fnv1a_64", Title: "FNV-1a 64-bit",
			Contract: "hash octet-par-octet h = (h^b)*prime ; prime fixe 1099511628211",
			Flux: []string{
				"Entrée : buffer data[0..len)",
				"Init h = offset_basis 14695981039346656037",
				"Pour chaque octet : h ^= b ; h *= FNV_prime",
				"Sortie : u64",
			},
			EmitNote: "Override emit : unroll ×8 (même sémantique XOR*prime) + queue ; fStmts nil (pas de double corps).",
			Strata:   "ov_empty = coût d'appel ; tiny = setup ; l1/l2/bulk = débit streaming octets.",
			Levers:   []string{"Unroll ×16 si gain mesuré", "BCE data[:len] si compteur int"},
		},
		{
			ID: "crc32_ieee", Title: "CRC-32 IEEE (bit-wise)",
			Contract: "poly 0xEDB88320, 8 tours de bit par octet — PAS de table, PAS de hardware CRC",
			Flux: []string{
				"crc = 0xFFFFFFFF",
				"Pour chaque octet : crc ^= b ; 8× { mask = -(crc&1) ; crc = (crc>>1) ^ (poly & mask) }",
				"return ~crc",
			},
			EmitNote: "Fidèle au .c bit-à-bit (boucle intérieure 8 bits).",
			Strata:   "Débit plafonné ~130–140 MB/s des deux côtés = parité algo, pas défaut d'emit.",
			Levers:   []string{"Aucun en transpile ; table slicing-by-8 = autre oracle C"},
		},
		{
			ID: "fast_xor", Title: "XOR bulk octets",
			Contract: "dst[i] = s1[i]^s2[i] pour i in 0..len",
			Flux: []string{
				"Boucle principale par pas de 8 : load u64 s1, s2 → store u64 dst",
				"Queue scalaire pour len%8",
			},
			EmitNote: "LittleEndian.Uint64/PutUint64 déjà ; masque align &^7 retiré si safe.",
			Strata:   "tail_17 = queue ; align_64/l1 = ALU+store ; l2/bulk = bande passante mémoire.",
			Levers:   []string{"Unroll ×4 mots (32 B/tour) fidèle", "Pas d'AVX hors source C"},
		},
		{
			ID: "siphash24", Title: "SipHash-2-4",
			Contract: "PRFkeyed 64-bit, 2 compression rounds / 4 finalization",
			Flux: []string{
				"Charge clé k0,k1 (16 B LE)",
				"Init v0..v3 avec constantes ^ clés",
				"Pour chaque bloc 8 B message : v3^=m ; 2×SipRound ; v0^=m",
				"Padding dernier bloc + longueur ; 4×SipRound final ; return v0^v1^v2^v3",
			},
			EmitNote: "Uint64 LE + rounds avec RotateLeft64 ; corps long (~170 lignes).",
			Strata:   "ov/tiny = setup clé ; l1+ = boucle message dominante.",
			Levers:   []string{"Inline SipRound", "Éviter temp m si SSA le fait déjà"},
		},
		{
			ID: "murmur3_x86_32", Title: "MurmurHash3 x86_32",
			Contract: "mix 32-bit blocs LE + fmix final",
			Flux: []string{
				"nblocks = len/4 ; h1 = seed",
				"Pour chaque bloc : k1=LE32 ; k1*=c1 ; rotl15 ; k1*=c2 ; h1^=k1 ; rotl13 ; h1=h1*5+const",
				"Tail 0..3 octets ; h1 ^= len ; return fmix32(h1)",
			},
			EmitNote: "Rotl32 = wrapper bits.RotateLeft32 ; boucle d'induction v18=-nblocks atypique.",
			Strata:   "Dégradation progressive l1→bulk = coût mix/rot par bloc, pas mémoire seule.",
			Levers:   []string{"Inline Rotl32/Fmix32", "Normaliser la boucle for i:=0;i<nblocks"},
		},
		{
			ID: "blake2b_compress", Title: "BLAKE2b compress (1 bloc 128 B)",
			Contract: "12 rondes G sur état v[16], message m[16], sigma[12][16]",
			Flux: []string{
				"Charge m[0..15] depuis block (LE u64)",
				"v[0..7]=h ; v[8..15]=IV ^ (t0,t1,f0,f1)",
				"12 rondes : 8×G avec m[sigma[r][i]]",
				"h[i] ^= v[i] ^ v[i+8]",
			},
			EmitNote: "sigma table runtime + indexation m[sigma[…]] ; RotateLeft64 présent.",
			Strata:   "block_* = pure ALU (même payload) ; ratio = coût indirection sigma / forme Go.",
			Levers:   []string{"Unroll 12 rondes + sigma littéraux (peephole fidèle majeur)"},
		},
		{
			ID: "chacha20_qr", Title: "ChaCha20 quarter-round (seule)",
			Contract: "Une QR sur 4 mots *a,*b,*c,*d — pas le cipher complet 20 rondes",
			Flux: []string{
				"a+=b ; d^=a ; d<<<16",
				"c+=d ; b^=c ; b<<<12",
				"a+=b ; d^=a ; d<<<8",
				"c+=d ; b^=c ; b<<<7",
			},
			EmitNote: "Pointeurs *uint32 + bits.RotateLeft32 ; fixture = 1 QR.",
			Strata:   "qr_1 / qr_1m = pure ALU ; comparer à C O2 sur le même symbole.",
			Levers:   []string{"Éliminer reloads via pointeurs si SSA le permet", "Ne pas inventer full ChaCha ici"},
		},
		{
			ID: "md5_transform", Title: "MD5 transform block",
			Contract: "64 pas FF/GG/HH/II sur block 64 B → state[4]",
			Flux: []string{
				"Charge X[16] LE depuis block",
				"64 pas : F/G/H/I + add const + rotl + add b",
				"state[i] += a/b/c/d",
			},
			EmitNote: "Expressions plates + RotateLeft32 + BCE block[15] ; graphe à vérifier (troncature fixture possible).",
			Strata:   "block_* répète le même transform ; ratio stable = coût par transform.",
			Levers:   []string{"Vérifier 64 pas complets vs C", "Inline déjà fait"},
		},
		{
			ID: "poly1305_block5", Title: "Poly1305 block (5 limbs)",
			Contract: "h = (h + m) * r mod 2^130-5 sur limbs 26-bit",
			Flux: []string{
				"Charge m en limbs",
				"Multiplications croisées h_i * r_j → accumulateurs 64-bit",
				"Propagation retenues >>26 + *5 sur wrap",
			},
			EmitNote: "Scalaire 64-bit fidèle ; pas encore bits.Mul64/Add64 systématiques.",
			Strata:   "poly_1 = 1 bloc ; poly_1m = throughput mul/add.",
			Levers:   []string{"Intrinsèques bits.Mul64/Add64 si équivalents bit-exact"},
		},
		{
			ID: "base64_simd", Title: "Base64 encode stream",
			Contract: "3 octets → 4 chars table ; padding =",
			Flux: []string{
				"Tant que ≥3 octets : pack u32 (b0<<16|b1<<8|b2) ; 4 index 6-bit → table",
				"Reste 1–2 octets + padding",
			},
			EmitNote: "Pack 32-bit déjà ; table string indexée.",
			Strata:   "tail = padding path ; l1/bulk = boucle 3→4 dominante.",
			Levers:   []string{"BCE dst/src", "Éviter bounds sur table si const"},
		},
		{
			ID: "tweetnacl_dogfood", Title: "TweetNaCl crypto_verify_16 → vn",
			Contract: "Comparaison temps-constant 16 B (surface) via vn(x,y,n)",
			Flux: []string{
				"crypto_verify_16 appelle vn(x,y,16)",
				"Si n%8==0 : OR des XOR de mots NativeEndian u64 (CT)",
				"Sinon : boucle octet d |= x[i]^y[i]",
				"return 0 si égal, -1 sinon (formule (1&((d-1)>>k))-1)",
			},
			EmitNote: "Override Vn mots 64-bit + fallback octet + BCE [:n].",
			Strata:   "ver_eq / ver_neq : même travail CT (pas d'early-exit data-dependent).",
			Levers:   []string{"Asm check ver_neq", "Unroll fixe n=16 sans boucle"},
		},
		{
			ID: "libinjection_sqli", Title: "libinjection strlenspn (smoke)",
			Contract: "Scan accept-set style strspn — pas d'oracle C dans le catalogue (SkipC)",
			Flux: []string{
				"Pour chaque caractère : chercher dans accept[] (boucle / IIFE émise)",
				"Longueur du préfixe accepté",
			},
			EmitNote: "IIFE for i… accept[i] ; hors probe C (SkipC) — pas dans ce rapport stratifié C-vs-sgo.",
			Strata:   "Non mesuré ici (pas de backend C).",
			Levers:   []string{"Oracle C dédié avant optim", "bytes.IndexByte si équivalence prouvée"},
		},
	}
}

// FormatLayersMD is the detailed per-lib layered report.
func FormatLayersMD(r *Report) string {
	var b strings.Builder
	b.WriteString("# Probebench LAYERS — flux + strates + goulets\n\n")
	b.WriteString("Stamp: **" + r.Stamp + "**  \n")
	b.WriteString("Workdir: `" + FormatDiskEnv(r.WorkDisk) + "`  \n")
	b.WriteString("Backends: `c_gcc_O2` · `sgoiter` · `ccgo` (in-process).  \n")
	b.WriteString("Ratios : sgo/C, ccgo/C, sgo/ccgo (>1 = numérateur plus lent).  \n\n")

	b.WriteString("## Légende des phases\n\n")
	b.WriteString("| Phase | Ce qu'elle isole |\n|-------|------------------|\n")
	b.WriteString("| overhead | Coût d'appel / len=0 — **bruit** si on parle kernel |\n")
	b.WriteString("| setup / tiny | Petits messages : setup + prologue dominent |\n")
	b.WriteString("| hot_l1 / block | Noyau chaud, working set ≤ L1 |\n")
	b.WriteString("| hot_l2 | Traverse L1, encore CPU-bound souvent |\n")
	b.WriteString("| bulk | Streaming 1 MiB : bande passante + boucle |\n")
	b.WriteString("| tail | Longueurs non alignées / queue scalaire |\n\n")
	b.WriteString("Doctrine : comparer **même strate**, ne pas cherry-picker le max ; optim = peephole fidèle au .c.\n\n")
	b.WriteString("## Baseline B0 (pré vague A overrides)\n\n")
	b.WriteString("Référence historique stamp `2026-08-11T22:31:38Z` / commit `49fc1fc` — ratios sgo/C rep. : ")
	b.WriteString("Vn 2.83× · xor 2.41× · md5 2.12× · blake 1.71× · murmur 1.24× · siphash 1.07×.\n\n")
	b.WriteString("Ce rapport (stamp courant) est le **B1+** post-overrides ; lire les ratios ci-dessous comme Δ implicite vs B0.\n\n")

	// index probes
	type key struct{ lib, st, be string }
	idx := map[key]ProbeLine{}
	libOrder := []string{}
	seen := map[string]bool{}
	for _, p := range r.Probes {
		idx[key{p.Lib, p.Stratum, p.Backend}] = p
		if !seen[p.Lib] {
			seen[p.Lib] = true
			libOrder = append(libOrder, p.Lib)
		}
	}

	flows := map[string]LibFlow{}
	for _, f := range CatalogFlows() {
		flows[f.ID] = f
	}

	// summary table first
	b.WriteString("## Synthèse triangle C / sgoiter / ccgo — strate représentative\n\n")
	b.WriteString("| lib | strate | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | verdict sgo vs C |\n")
	b.WriteString("|-----|--------|------|--------|---------|-------|--------|----------|------------------|\n")
	for _, id := range libOrder {
		repSt := pickRepStratum(r, id)
		if repSt == "" {
			continue
		}
		c := idx[key{id, repSt, "c_gcc_O2"}]
		s := idx[key{id, repSt, "sgoiter"}]
		g := idx[key{id, repSt, "ccgo"}]
		cns, sns, gns := "—", "—", "—"
		rsc, rgc, rsg := "—", "—", "—"
		ver := "—"
		if c.Error == "" && c.NsPerOp > 0 {
			cns = fmt.Sprintf("%.1f", c.NsPerOp)
		} else if c.Error != "" {
			cns = "ERR"
		}
		if s.Error == "" && s.NsPerOp > 0 {
			sns = fmt.Sprintf("%.1f", s.NsPerOp)
		} else if s.Error != "" {
			sns = "ERR"
		}
		if g.Error == "" && g.NsPerOp > 0 {
			gns = fmt.Sprintf("%.1f", g.NsPerOp)
		} else if g.Error != "" {
			gns = "ERR"
		}
		if c.NsPerOp > 0 && s.NsPerOp > 0 && s.Error == "" && c.Error == "" {
			ratio := s.NsPerOp / c.NsPerOp
			rsc = fmt.Sprintf("**%.2f×**", ratio)
			ver = verdictRatio(ratio)
		}
		if c.NsPerOp > 0 && g.NsPerOp > 0 && g.Error == "" && c.Error == "" {
			rgc = fmt.Sprintf("%.2f×", g.NsPerOp/c.NsPerOp)
		}
		if g.NsPerOp > 0 && s.NsPerOp > 0 && s.Error == "" && g.Error == "" {
			rsg = fmt.Sprintf("%.2f×", s.NsPerOp/g.NsPerOp)
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			id, repSt, cns, sns, gns, rsc, rgc, rsg, ver))
	}
	b.WriteString("\n")

	// per lib chapters
	for _, id := range libOrder {
		f, ok := flows[id]
		if !ok {
			f = LibFlow{ID: id, Title: id}
		}
		b.WriteString("---\n\n")
		b.WriteString(fmt.Sprintf("## %s — `%s`\n\n", f.Title, id))
		if f.Contract != "" {
			b.WriteString("**Contrat :** " + f.Contract + "\n\n")
		}
		if len(f.Flux) > 0 {
			b.WriteString("### Flux de calcul\n\n")
			for i, step := range f.Flux {
				b.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
			}
			b.WriteString("\n")
		}
		if f.EmitNote != "" {
			b.WriteString("**Emit sgoiter :** " + f.EmitNote + "\n\n")
		}
		if f.Strata != "" {
			b.WriteString("**Lecture strates :** " + f.Strata + "\n\n")
		}

		// all strata for this lib
		type row struct {
			st, phase string
			c, s      ProbeLine
			ratio     float64
		}
		var rows []row
		stSeen := map[string]bool{}
		var sts []string
		for _, p := range r.Probes {
			if p.Lib != id {
				continue
			}
			if !stSeen[p.Stratum] {
				stSeen[p.Stratum] = true
				sts = append(sts, p.Stratum)
			}
		}
		for _, st := range sts {
			c := idx[key{id, st, "c_gcc_O2"}]
			s := idx[key{id, st, "sgoiter"}]
			ph := s.Phase
			if ph == "" {
				ph = c.Phase
			}
			ratio := math.NaN()
			if c.NsPerOp > 0 && s.Error == "" && c.Error == "" {
				ratio = s.NsPerOp / c.NsPerOp
			}
			rows = append(rows, row{st, ph, c, s, ratio})
		}

		b.WriteString("### Couches (toutes strates)\n\n")
		b.WriteString("| stratum | phase | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | lecture |\n")
		b.WriteString("|---------|-------|------|--------|---------|-------|--------|----------|----------|\n")
		var worst, best row
		worst.ratio = 0
		best.ratio = 99
		for _, rw := range rows {
			cns, sns, gns := "—", "—", "—"
			rsc, rgc, rsg := "—", "—", "—"
			lec := ""
			g := idx[key{id, rw.st, "ccgo"}]
			if rw.c.Error != "" {
				cns = "ERR"
			} else if rw.c.NsPerOp > 0 || rw.c.Iters > 0 {
				cns = fmt.Sprintf("%.1f", rw.c.NsPerOp)
			}
			if rw.s.Error != "" {
				sns = "ERR"
				lec = "sgo build/run fail"
			} else if rw.s.Iters > 0 {
				sns = fmt.Sprintf("%.1f", rw.s.NsPerOp)
			}
			if g.Error != "" {
				gns = "ERR"
			} else if g.Iters > 0 {
				gns = fmt.Sprintf("%.1f", g.NsPerOp)
			}
			if !math.IsNaN(rw.ratio) {
				rsc = fmt.Sprintf("%.2f×", rw.ratio)
				lec = layerRead(rw.phase, rw.ratio, rw.s.Allocs)
				if rw.phase != "overhead" && rw.ratio > worst.ratio {
					worst = rw
				}
				if rw.phase != "overhead" && rw.ratio < best.ratio {
					best = rw
				}
			}
			if rw.c.NsPerOp > 0 && g.NsPerOp > 0 && g.Error == "" && rw.c.Error == "" {
				rgc = fmt.Sprintf("%.2f×", g.NsPerOp/rw.c.NsPerOp)
			}
			if g.NsPerOp > 0 && rw.s.NsPerOp > 0 && rw.s.Error == "" && g.Error == "" {
				rsg = fmt.Sprintf("%.2f×", rw.s.NsPerOp/g.NsPerOp)
			}
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
				rw.st, rw.phase, cns, sns, gns, rsc, rgc, rsg, lec))
		}
		b.WriteString("\n")

		if worst.ratio >= 1.15 {
			b.WriteString(fmt.Sprintf("**Goulet dominant (cette lib, hors overhead) :** `%s` — ratio **%.2f×** (sgo %.1f vs C %.1f ns/op), phase `%s`.\n\n",
				worst.st, worst.ratio, worst.s.NsPerOp, worst.c.NsPerOp, worst.phase))
		}
		if best.ratio < 0.95 {
			b.WriteString(fmt.Sprintf("**Point fort :** strate `%s` sgoiter **plus rapide** que C (ratio %.2f×).\n\n", best.st, best.ratio))
		}

		if len(f.Levers) > 0 {
			b.WriteString("**Leviers fidèles (pas de changement d'algo) :**\n")
			for _, L := range f.Levers {
				b.WriteString("- " + L + "\n")
			}
			b.WriteString("\n")
		}
	}

	// global ranking
	b.WriteString("---\n\n## Classement goulets globaux (ratio sgo/C, hors overhead)\n\n")
	type hit struct {
		lib, st, phase string
		ratio, cns, sns float64
		allocs         int64
	}
	var hits []hit
	for _, p := range r.Probes {
		if p.Backend != "sgoiter" || p.Error != "" || p.Phase == "overhead" {
			continue
		}
		c := idx[key{p.Lib, p.Stratum, "c_gcc_O2"}]
		if c.Error != "" || c.NsPerOp <= 0 {
			continue
		}
		hits = append(hits, hit{p.Lib, p.Stratum, p.Phase, p.NsPerOp / c.NsPerOp, c.NsPerOp, p.NsPerOp, p.Allocs})
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].ratio > hits[j].ratio })
	b.WriteString("| # | lib | stratum | phase | ratio | C ns | sgo ns | allocs |\n")
	b.WriteString("|---|-----|---------|-------|-------|------|--------|--------|\n")
	n := 15
	if len(hits) < n {
		n = len(hits)
	}
	for i := 0; i < n; i++ {
		h := hits[i]
		b.WriteString(fmt.Sprintf("| %d | %s | %s | %s | **%.2f×** | %.1f | %.1f | %d |\n",
			i+1, h.lib, h.st, h.phase, h.ratio, h.cns, h.sns, h.allocs))
	}
	b.WriteString("\n### Priorité d'ingénierie (fidèle au graphe)\n\n")
	b.WriteString("1. **blake2b_compress** — unroll + sigma littéral (indirection table).\n")
	b.WriteString("2. **murmur3_x86_32** — inline Rotl/Fmix + boucle d'induction claire.\n")
	b.WriteString("3. **tweetnacl Vn** — residual CT vs C (asm) ; n=16 unroll fixe.\n")
	b.WriteString("4. **md5_transform** — confirmer complétude 64 pas puis forme rot.\n")
	b.WriteString("5. **fnv1a_64** — déjà ×8 ; gains marginaux seulement.\n")
	b.WriteString("6. **crc32_ieee** — **clos** (parité algo bit-wise) ; ne pas « optimiser » par table.\n")
	return b.String()
}

func pickRepStratum(r *Report, lib string) string {
	// prefer hot_l1 / block / l1_1k / ver_eq
	pref := []string{"l1_1k", "block_1k", "block_1", "qr_1m", "poly_1m", "ver_eq", "align_64", "bulk_1m"}
	have := map[string]bool{}
	for _, p := range r.Probes {
		if p.Lib == lib {
			have[p.Stratum] = true
		}
	}
	for _, s := range pref {
		if have[s] {
			return s
		}
	}
	for _, p := range r.Probes {
		if p.Lib == lib {
			return p.Stratum
		}
	}
	return ""
}

func verdictRatio(r float64) string {
	switch {
	case r < 0.95:
		return "sgo plus rapide"
	case r <= 1.15:
		return "parité"
	case r <= 1.8:
		return "écart modéré"
	case r <= 3.0:
		return "goulet"
	default:
		return "goulet fort"
	}
}

func layerRead(phase string, ratio float64, allocs int64) string {
	var parts []string
	switch {
	case ratio < 0.95:
		parts = append(parts, "sgo gagne")
	case ratio <= 1.15:
		parts = append(parts, "parité")
	case ratio <= 1.8:
		parts = append(parts, "sgo +lent")
	default:
		parts = append(parts, "GOULET")
	}
	if phase == "overhead" {
		parts = append(parts, "bruit appel")
	}
	if allocs > 0 {
		parts = append(parts, fmt.Sprintf("%d alloc", allocs))
	}
	return strings.Join(parts, ", ")
}
