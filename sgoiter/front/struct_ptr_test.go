package front_test

import (
	"strings"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/front"
)

func TestHarvestStructPointerFields(t *testing.T) {
	src := `
typedef struct {
	const uint8_t *pass;
	const uint8_t *salt;
	uint32_t pass_size;
	uint32_t salt_size;
} crypto_argon2_inputs;
void f(void) {}
`
	res, err := front.ParsePartial(src, "t")
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, st := range res.Module.Structs {
		if st.Name != "crypto_argon2_inputs" {
			continue
		}
		found = true
		names := map[string]int{}
		for _, f := range st.Fields {
			names[f.Name] = f.ArrayLen
		}
		if names["pass"] != -1 || names["salt"] != -1 {
			t.Fatalf("pointer fields ArrayLen want -1 got %+v", st.Fields)
		}
		if _, ok := names["pass_size"]; !ok {
			t.Fatal("missing pass_size")
		}
	}
	if !found {
		t.Fatal("struct not harvested")
	}
}

func TestParseArrIndexDotField(t *testing.T) {
	src := `
typedef struct { uint64_t a[4]; } blk;
void fill(blk *blocks, uint32_t i) {
	blocks[i].a[0] = 1;
}
`
	res, err := front.ParsePartial(src, "t")
	if err != nil {
		t.Fatal(err)
	}
	// may still skip on store path — at least no crash
	for _, s := range res.Skipped {
		if strings.Contains(s, "fill") {
			t.Log("fill skipped:", s)
		}
	}
}
