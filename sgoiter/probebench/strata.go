// Package probebench — banc stratifié durable C / sgoiter / ccgo.
// Chaque strate isole une phase logique (overhead, L1, bulk, queue) pour
// guider la micro-optimisation sans confondre le coût process avec le kernel.
package probebench

import "code.hazyhaar.fr/devhoros/c2simd/sgoiter/tribench"

// Stratum is one named working set / phase for a lib kind.
type Stratum struct {
	ID       string // machine id
	Label    string // human
	Phase    string // overhead|setup|hot_l1|hot_l2|bulk|tail|block
	Bytes    int    // primary payload size (0 = empty / fixed block)
	Iters    int    // in-process iterations (scaled so ~50–200ms)
	Notes    string
}

// StrataFor returns durable strata for a tribench Kind.
func StrataFor(k tribench.Kind) []Stratum {
	switch k {
	case tribench.KindHash64, tribench.KindHash32, tribench.KindHash32Seed, tribench.KindSipHash, tribench.KindLibInj:
		return []Stratum{
			{ID: "ov_empty", Label: "overhead empty", Phase: "overhead", Bytes: 0, Iters: 2_000_000, Notes: "call + len only"},
			{ID: "tiny_16", Label: "tiny 16 B", Phase: "setup", Bytes: 16, Iters: 1_000_000, Notes: "setup dominates"},
			{ID: "l1_1k", Label: "L1 1 KiB", Phase: "hot_l1", Bytes: 1024, Iters: 200_000, Notes: "hot path L1"},
			{ID: "l1_4k", Label: "L1 4 KiB", Phase: "hot_l1", Bytes: 4096, Iters: 80_000, Notes: "L1 full-ish"},
			{ID: "l2_64k", Label: "L2 64 KiB", Phase: "hot_l2", Bytes: 64 * 1024, Iters: 8_000, Notes: "cross L1"},
			{ID: "bulk_1m", Label: "bulk 1 MiB", Phase: "bulk", Bytes: 1024 * 1024, Iters: 400, Notes: "bandwidth / streaming"},
		}
	case tribench.KindXor, tribench.KindBase64:
		return []Stratum{
			{ID: "tail_17", Label: "unaligned 17 B", Phase: "tail", Bytes: 17, Iters: 1_000_000, Notes: "scalar tail"},
			{ID: "align_64", Label: "aligned 64 B", Phase: "hot_l1", Bytes: 64, Iters: 500_000, Notes: "one cache line"},
			{ID: "l1_1k", Label: "L1 1 KiB", Phase: "hot_l1", Bytes: 1024, Iters: 100_000, Notes: "bulk short"},
			{ID: "l2_64k", Label: "L2 64 KiB", Phase: "hot_l2", Bytes: 64 * 1024, Iters: 4_000, Notes: "mem bound"},
			{ID: "bulk_1m", Label: "bulk 1 MiB", Phase: "bulk", Bytes: 1024 * 1024, Iters: 200, Notes: "throughput"},
		}
	case tribench.KindBlake2b, tribench.KindMD5:
		// one block compress; scale by block count via Iters
		return []Stratum{
			{ID: "block_1", Label: "1 block", Phase: "block", Bytes: 64, Iters: 500_000, Notes: "single compress"},
			{ID: "block_1k", Label: "1k blocks", Phase: "hot_l1", Bytes: 64, Iters: 50_000, Notes: "repeat compress"},
			{ID: "block_64k", Label: "64k blocks", Phase: "bulk", Bytes: 64, Iters: 2_000, Notes: "throughput blocks"},
		}
	case tribench.KindChaChaQR:
		return []Stratum{
			{ID: "qr_1", Label: "1 quarter-round", Phase: "block", Bytes: 0, Iters: 5_000_000, Notes: "ALU only"},
			{ID: "qr_1m", Label: "1e6 QR", Phase: "hot_l1", Bytes: 0, Iters: 1_000_000, Notes: "tight ALU"},
		}
	case tribench.KindPoly5:
		return []Stratum{
			{ID: "poly_1", Label: "1 block5", Phase: "block", Bytes: 16, Iters: 1_000_000, Notes: "mul limb"},
			{ID: "poly_1m", Label: "1e6 block5", Phase: "hot_l1", Bytes: 16, Iters: 200_000, Notes: "throughput"},
		}
	case tribench.KindTweetVer:
		return []Stratum{
			{ID: "ver_eq", Label: "verify eq", Phase: "block", Bytes: 16, Iters: 2_000_000, Notes: "const-time cmp"},
			{ID: "ver_neq", Label: "verify neq", Phase: "block", Bytes: 16, Iters: 2_000_000, Notes: "early diverge still full"},
		}
	default:
		return []Stratum{{ID: "default", Label: "default 1k", Phase: "hot_l1", Bytes: 1024, Iters: 100_000}}
	}
}
