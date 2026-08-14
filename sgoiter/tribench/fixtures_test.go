package tribench_test

import (
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/tribench"
)

func TestBase64FixturesMod3(t *testing.T) {
	fixs := tribench.FixturesFor(tribench.KindBase64)
	modMap := map[int]bool{}
	lenMap := map[int]bool{}

	for _, f := range fixs {
		l := len(f.Data)
		modMap[l%3] = true
		lenMap[l] = true
	}

	for _, m := range []int{0, 1, 2} {
		if !modMap[m] {
			t.Errorf("missing base64 fixture for len %% 3 == %d", m)
		}
	}

	for _, l := range []int{0, 1, 2, 3} {
		if !lenMap[l] {
			t.Errorf("missing base64 fixture for len == %d", l)
		}
	}
}
