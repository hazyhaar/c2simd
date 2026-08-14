package emit

import "testing"

// T5 — `dst[j++] = v` expands into a snapshot, an increment and a store. When
// the snapshot has a single reader, the three lines say what two say.
func TestFoldPostIncStore(t *testing.T) {
	in := "\t\tv31 := v5\n\t\tv5 = v5 + 1\n\t\tdst[int(v31)] = b64_table[int(v12 & 63)]\n"
	want := "\t\tdst[int(v5)] = b64_table[int(v12 & 63)]\n\t\tv5++\n"
	if got := foldPostIncStore(in); got != want {
		t.Errorf("got =%q\nwant=%q", got, want)
	}
}

func TestFoldPostIncStoreKeepsExtraReaders(t *testing.T) {
	// the snapshot is read again after the store: the increment cannot move
	in := "\t\tv31 := v5\n\t\tv5 = v5 + 1\n\t\tdst[int(v31)] = 1\n\t\tn = v31\n"
	if got := foldPostIncStore(in); got != in {
		t.Errorf("folded a snapshot with a second reader:\n%q", got)
	}
}

func TestFoldPostIncStoreKeepsCounterInValue(t *testing.T) {
	// the stored value reads the counter: moving the increment first changes it
	in := "\t\tv31 := v5\n\t\tv5 = v5 + 1\n\t\tdst[int(v31)] = uint8(v5)\n"
	if got := foldPostIncStore(in); got != in {
		t.Errorf("folded a store whose value reads the counter:\n%q", got)
	}
}

// T16 — an index does not need a wider type than the value it comes from.
func TestNarrowIndexShiftMask(t *testing.T) {
	types := map[string]string{"v12": "uint32", "v9": "uint64"}
	cases := []struct{ in, want string }{
		{"x = t[int(uint64(v12) >> 18 & uint64(63))]", "x = t[int((v12 >> 18) & 63)]"},
		{"x = t[int(uint64(v12) >> 6 & uint64(63))]", "x = t[int((v12 >> 6) & 63)]"},
		// already the value's own type: nothing was widened
		{"x = t[int(uint64(v9) >> 3 & uint64(7))]", "x = t[int(uint64(v9) >> 3 & uint64(7))]"},
		// unknown declaration: no evidence of the value's width
		{"x = t[int(uint64(zz) >> 3 & uint64(7))]", "x = t[int(uint64(zz) >> 3 & uint64(7))]"},
		// a shift that reaches past the value's own width needs the wide type
		{"x = t[int(uint64(v12) >> 40 & uint64(63))]", "x = t[int(uint64(v12) >> 40 & uint64(63))]"},
	}
	for _, c := range cases {
		if got := narrowIndexShiftMask(c.in, types); got != c.want {
			t.Errorf("in  =%q\ngot =%q\nwant=%q", c.in, got, c.want)
		}
	}
}
