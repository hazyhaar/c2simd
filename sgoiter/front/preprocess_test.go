package front

import (
	"regexp"
	"testing"
)

func TestFoldDefinesHash(t *testing.T) {
	src := `
#define HASH_LOG 13
#define HASH_SIZE (1 << HASH_LOG)
#define HASH_MASK (HASH_SIZE - 1)
#define FASTLZ_LIKELY(c) (__builtin_expect(!!(c), 1))
int dummy(int c) { return FASTLZ_LIKELY(c); }
uint32_t flz_hash(uint32_t v) {
  uint32_t h = (v * 2654435769LL) >> (32 - HASH_LOG);
  return h & HASH_MASK;
}
`
	out := foldDefines(stripComments(src))
	for _, id := range []string{"HASH_LOG", "HASH_MASK", "HASH_SIZE", "FASTLZ_LIKELY", "__builtin_expect"} {
		if word(out, id) {
			t.Fatalf("macro %s still present:\n%s", id, out)
		}
	}
}

func TestFoldDefinesParseFlzHash(t *testing.T) {
	src := `
#define HASH_LOG 13
#define HASH_SIZE (1 << HASH_LOG)
#define HASH_MASK (HASH_SIZE - 1)
static uint16_t flz_hash(uint32_t v) {
  uint32_t h = (v * 2654435769LL) >> (32 - HASH_LOG);
  return h & HASH_MASK;
}
`
	m, err := Parse(src, "flz")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Funcs) != 1 || m.Funcs[0].Name != "flz_hash" {
		t.Fatalf("%+v", m.Funcs)
	}
}

func word(s, id string) bool {
	return regexp.MustCompile(`\b` + regexp.QuoteMeta(id) + `\b`).MatchString(s)
}
