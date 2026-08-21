package front_test

import (
	"strings"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/emit"
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
	for _, s := range res.Skipped {
		if strings.Contains(s, "fill") {
			t.Fatalf("fill skipped: %s", s)
		}
	}
}

func TestStrcmpStringLiteralNotDoubled(t *testing.T) {
	src := `
int c2_validate_patch_mode(const char *mode_str) {
    if (mode_str == 0) return 1;
    if (strcmp(mode_str, "120000") == 0) return 0;
    if (strcmp(mode_str, "160000") == 0) return 0;
    return 1;
}
`
	m, err := front.Parse(src, "t")
	if err != nil {
		t.Fatal(err)
	}
	out, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "+ __str_") {
		t.Fatalf("string literal doubled:\n%s", out)
	}
	if !strings.Contains(out, "120000") {
		t.Fatalf("missing 120000:\n%s", out)
	}
}

func TestSwitchBreakNoFallthrough(t *testing.T) {
	src := `
int feed(int state, int b) {
    switch (state) {
    case 0:
        if (b == 27) return 1;
        break;
    case 1:
        return 2;
    }
    return 0;
}
`
	m, err := front.Parse(src, "t")
	if err != nil {
		t.Fatal(err)
	}
	out, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "fallthrough") {
		t.Fatalf("break case must not fallthrough:\n%s", out)
	}
}
