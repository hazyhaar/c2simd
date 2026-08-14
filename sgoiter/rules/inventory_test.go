package rules_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/front"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/rules"
)

// Counts what each rule rewrites across the real kernels. Before the walk covered
// do/while and if bodies, every macro-expanded block was invisible to the rules;
// this test is the ledger of what they now touch, and it fails if a rule goes
// silent on the whole corpus.
func TestRuleInventoryOnKernels(t *testing.T) {
	srcRoot := filepath.Join("..", "..", "spec", "c_sources", "testdata", "c_sources")
	kernels := []string{
		"fnv1a_64.c", "crc32_ieee.c", "fast_xor.c", "siphash24.c",
		"murmur3_x86_32.c", "blake2b_compress.c", "chacha20_qr.c",
		"md5_transform.c", "poly1305_block5.c", "base64_simd.c",
	}
	hits := map[string]int{}
	for _, k := range kernels {
		p, err := filepath.Abs(filepath.Join(srcRoot, k))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(p); err != nil {
			t.Skipf("kernel corpus missing: %v", err)
		}
		m, err := front.ParseFile(p)
		if err != nil {
			t.Fatalf("parse %s: %v", k, err)
		}
		for _, d := range rules.Table {
			if d.Kind != rules.KindRewrite || d.Apply == nil {
				continue
			}
			// each rule against the untouched module, so counts are independent
			out, err := d.Apply(m)
			if err != nil {
				t.Fatalf("%s on %s: %v", d.ID, k, err)
			}
			eq, err := ir.EqualJSON(m, out)
			if err != nil {
				t.Fatal(err)
			}
			if !eq {
				hits[d.ID]++
			}
		}
	}
	ids := make([]string, 0, len(hits))
	for id := range hits {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		t.Logf("%-16s fires on %d/%d kernels", id, hits[id], len(kernels))
	}
	for _, d := range rules.Table {
		if d.Kind != rules.KindRewrite || d.Apply == nil {
			continue
		}
		if hits[d.ID] == 0 {
			t.Logf("%-16s fires on no kernel of the corpus (unit witness only)", d.ID)
		}
	}
	if len(hits) == 0 {
		t.Fatal("no rule fires on any kernel: the walk is blind again")
	}
}
