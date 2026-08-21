package emit

import (
	"strings"
	"testing"
)

func TestAstEliminateDeadCodeAfterReturn(t *testing.T) {
	in := `
	if x > 0 {
		return 42
		x = x + 1
		x++
	}
	return 0
`
	got := astEliminateDeadCodeAfterReturn(in)
	if strings.Contains(got, "x = x + 1") || strings.Contains(got, "x++") {
		t.Fatalf("Dead code not eliminated after return:\n%s", got)
	}
	if !strings.Contains(got, "return 42") || !strings.Contains(got, "return 0") {
		t.Fatalf("Return statements missing:\n%s", got)
	}
}

func TestAstEliminateDeadCodeAfterLoopReturn(t *testing.T) {
	in := `
	for i := 0; i < n; i++ {
		if i == 5 {
			return i
			i++
		}
	}
	return -1
`
	got := astEliminateDeadCodeAfterReturn(in)
	if strings.Contains(got, "return i\n\t\t\ti++") {
		t.Fatalf("Dead loop increment not eliminated after return:\n%s", got)
	}
}
