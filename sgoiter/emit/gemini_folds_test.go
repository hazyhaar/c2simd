package emit

import (
	"strings"
	"testing"
)

func TestFoldLiveRangeCopies(t *testing.T) {
	in := "" +
		"\tv65 = v8[12]\n" +
		"\tv67 = v8[0]\n" +
		"\tv72 = v65\n" +
		"\tv74 = v67\n" +
		"\tv75 = v72 ^ v74\n" +
		"\tv80 = bits.RotateLeft64(v75, 32)\n"
	got := foldLiveRangeCopies(in)
	if strings.Contains(got, "v72 =") || strings.Contains(got, "v74 =") {
		t.Fatalf("copies not eliminated:\n%s", got)
	}
	if !strings.Contains(got, "v75 = v65 ^ v67") {
		t.Fatalf("expected inlined xor, got:\n%s", got)
	}
}

func TestFoldCrcBitSteps(t *testing.T) {
	in := "" +
		"\tv17 = v2 & 1\n" +
		"\tv18 = int(v17)\n" +
		"\tv20 = -v18\n" +
		"\tv21 = uint32(v20)\n" +
		"\tv15 = v21\n" +
		"\tv23 = v2 >> 1\n" +
		"\tv25 = 0xedb88320 & v15\n" +
		"\tv26 = v23 ^ v25\n" +
		"\tv2 = v26\n"
	got := foldCrcBitSteps(in)
	if strings.Count(got, "\n") > 2 {
		t.Fatalf("expected single-line fold, got:\n%s", got)
	}
	if !strings.Contains(got, "0xedb88320") || !strings.Contains(got, "v2 =") {
		t.Fatalf("bad fold:\n%s", got)
	}
}

func TestFoldManualRotHelpers(t *testing.T) {
	in := "" +
		"func L32(x uint64, c int) uint64 {\n" +
		"\treturn uint64((x << uint8(c)) | (x >> uint8(32 - c)))\n" +
		"}\n" +
		"func R(x uint64, c int) uint64 {\n" +
		"\treturn uint64(bits.RotateLeft64(x, 64-c))\n" +
		"}\n" +
		"func F(x uint64) uint64 { return R(x, 1) ^ L32(x, 7) }\n"
	got := foldManualRotHelpers(in)
	if strings.Contains(got, "func L32") || strings.Contains(got, "func R(") {
		t.Fatalf("helpers remain:\n%s", got)
	}
	if !strings.Contains(got, "RotateLeft64") {
		t.Fatalf("missing RotateLeft64:\n%s", got)
	}
}
