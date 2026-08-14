package tribench

import (
	"fmt"
	"strings"
)

// GenHarnessSgo produces package main calling sgoiter-exported symbols (same package merge).
// kernel.go is package "kernel"; harness is package main importing it — simpler: single package main.
// We rewrite kernel package to "main" and append harness.
func GenHarnessSgo(lib Lib, kernelGo string) string {
	// force package main
	kg := rewritePackage(kernelGo, "main")
	var b strings.Builder
	b.WriteString(kg)
	if !strings.Contains(kg, "encoding/binary") && lib.Kind == KindXor {
		// already handled by rewrite if import present
	}
	b.WriteString("\n// ---- tribench harness ----\n")
	b.WriteString(`import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)
`)
	// dedupe imports is hard if kernel already imported — use separate file instead
	_ = b
	return genSgoSeparateMain(lib)
}

// GenHarnessSgoMain is a standalone main.go importing package kernel.
func GenHarnessSgoMain(lib Lib) string {
	return genSgoSeparateMain(lib)
}

func genSgoSeparateMain(lib Lib) string {
	var b strings.Builder
	b.WriteString("package main\n\nimport (\n")
	if lib.Kind == KindXor {
		b.WriteString("\t\"crypto/sha256\"\n\t\"encoding/hex\"\n")
	}
	b.WriteString("\t\"fmt\"\n\n\t\"trib/kernel\"\n)\n\n")
	if lib.Kind == KindXor {
		b.WriteString("func hx(b []byte) string { return hex.EncodeToString(b) }\n")
	}
	b.WriteString(`func u64(v uint64) string { return fmt.Sprintf("%016x", v) }
func u32(v uint32) string { return fmt.Sprintf("%08x", v) }

func main() {
`)
	fixs := FixturesFor(lib.Kind)
	usesData := lib.Kind != KindChaChaQR && lib.Kind != KindPoly5
	for i, f := range fixs {
		fmt.Fprintf(&b, "\t// %s\n", f.Name)
		if usesData {
			fmt.Fprintf(&b, "\td%d := %s\n", i, goByteLit(f.Data))
		}
		if len(f.Data2) > 0 {
			fmt.Fprintf(&b, "\te%d := %s\n", i, goByteLit(f.Data2))
		}
	}
	for i, f := range fixs {
		switch lib.Kind {
		case KindHash64:
			fmt.Fprintf(&b, "\tfmt.Printf(\"%s %%s\\n\", u64(kernel.%s(d%d, uint64(len(d%d)))))\n", f.Name, lib.SgoFunc, i, i)
		case KindHash32:
			fmt.Fprintf(&b, "\tfmt.Printf(\"%s %%s\\n\", u32(kernel.%s(d%d, uint64(len(d%d)))))\n", f.Name, lib.SgoFunc, i, i)
		case KindHash32Seed:
			fmt.Fprintf(&b, "\tfmt.Printf(\"%s %%s\\n\", u32(kernel.%s(d%d, uint64(len(d%d)), uint32(%d))))\n", f.Name, lib.SgoFunc, i, i, f.Seed)
		case KindSipHash:
			fmt.Fprintf(&b, "\tfmt.Printf(\"%s %%s\\n\", u64(kernel.%s(d%d, uint64(len(d%d)), e%d)))\n", f.Name, lib.SgoFunc, i, i, i)
		case KindXor:
			fmt.Fprintf(&b, `	{ dst := make([]byte, len(d%d)); kernel.%s(dst, d%d, e%d, uint64(len(d%d)));
		sum := sha256.Sum256(dst); fmt.Printf("%s %%s\n", hx(sum[:])) }
`, i, lib.SgoFunc, i, i, i, f.Name)
		case KindBlake2b:
			fmt.Fprintf(&b, `	{ h := make([]uint64, 8); block := make([]byte, 128); copy(block, d%d);
		kernel.%s(h, block, 0, 0, 0xffffffffffffffff, 0xffffffffffffffff);
		var s string; for _, v := range h { s += u64(v) }; fmt.Printf("%s %%s\n", s) }
`, i, lib.SgoFunc, f.Name)
		case KindChaChaQR:
			fmt.Fprintf(&b, `	{ a, b, c, d := uint32(0x11111111), uint32(0x22222222), uint32(0x33333333), uint32(0x44444444);
		kernel.%s(&a, &b, &c, &d); fmt.Printf("%s %%s%%s%%s%%s\n", u32(a), u32(b), u32(c), u32(d)) }
`, lib.SgoFunc, f.Name)
		case KindMD5:
			fmt.Fprintf(&b, `	{ st := []uint32{0x67452301, 0xefcdab89, 0x98badcfe, 0x10325476}; block := make([]byte, 64); copy(block, d%d);
		kernel.%s(st, block); var s string; for _, v := range st { s += u32(v) }; fmt.Printf("%s %%s\n", s) }
`, i, lib.SgoFunc, f.Name)
		case KindPoly5:
			fmt.Fprintf(&b, `	{ h := make([]uint32, 5); r := []uint32{1,2,3,4,5}; m := []uint32{9,8,7,6};
		kernel.%s(h, r, m); var s string; for _, v := range h { s += u32(v) }; fmt.Printf("%s %%s\n", s) }
`, lib.SgoFunc, f.Name)
		case KindBase64:
			fmt.Fprintf(&b, `	{ dst := make([]byte, (len(d%d)+2)/3*4+8); n := kernel.%s(d%d, uint64(len(d%d)), dst);
		fmt.Printf("%s %%s\n", string(dst[:n])) }
`, i, lib.SgoFunc, i, i, f.Name)
		case KindTweetVer:
			fmt.Fprintf(&b, "\tfmt.Printf(\"%s %%d\\n\", kernel.%s(d%d, e%d))\n", f.Name, lib.SgoFunc, i, i)
		case KindLibInj:
			// 2-arg form (strlenspn_lab); accept set is internal to the kernel.
			fmt.Fprintf(&b, "\tfmt.Printf(\"%s %%s\\n\", u64(kernel.%s(d%d, uint64(len(d%d)))))\n", f.Name, lib.SgoFunc, i, i)
		}
	}
	b.WriteString("}\n")
	return b.String()
}

func goByteLit(data []byte) string {
	if len(data) == 0 {
		return "[]byte{}"
	}
	var b strings.Builder
	b.WriteString("[]byte{")
	for i, v := range data {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%d", v)
	}
	b.WriteByte('}')
	return b.String()
}

func rewritePackage(src, pkg string) string {
	lines := strings.Split(src, "\n")
	for i, ln := range lines {
		if strings.HasPrefix(strings.TrimSpace(ln), "package ") {
			lines[i] = "package " + pkg
			break
		}
	}
	return strings.Join(lines, "\n")
}
