package emit

import "testing"

// T15 — a guard sitting directly under `for {` is the loop's condition.
func TestHoistLoopGuards(t *testing.T) {
	cases := []struct{ in, want string }{
		{
			"\tfor {\n\t\tif !(v3 < len_) { break }\n\t\tv15 := s[int(v3)]\n\t}",
			"\tfor v3 < len_ {\n\t\tv15 := s[int(v3)]\n\t}",
		},
		{
			// two guards back to back merge into one condition
			"\tfor {\n\t\tif !(a < b) { break }\n\t\tif !(c != 0) { break }\n\t\tx = 1\n\t}",
			"\tfor a < b && c != 0 {\n\t\tx = 1\n\t}",
		},
		{
			// a definition comes first: the guard may read it, so it stays put
			"\tfor {\n\t\tv9 := v4 + 2\n\t\tif !(v9 < len_) { break }\n\t}",
			"\tfor {\n\t\tv9 := v4 + 2\n\t\tif !(v9 < len_) { break }\n\t}",
		},
		{
			// a guard further down the body is not the loop condition
			"\tfor {\n\t\tx = 1\n\t\tif !(y) { break }\n\t}",
			"\tfor {\n\t\tx = 1\n\t\tif !(y) { break }\n\t}",
		},
	}
	for _, c := range cases {
		if got := hoistLoopGuards(c.in); got != c.want {
			t.Errorf("in  =%q\ngot =%q\nwant=%q", c.in, got, c.want)
		}
	}
}
