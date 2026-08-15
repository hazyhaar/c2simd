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
	if lib.Kind == KindXor || lib.Kind == KindStbiPng {
		b.WriteString("\t\"crypto/sha256\"\n\t\"encoding/hex\"\n")
	}
	b.WriteString("\t\"fmt\"\n\n\t\"trib/kernel\"\n)\n\n")
	if lib.Kind == KindXor || lib.Kind == KindStbiPng {
		b.WriteString("func hx(b []byte) string { return hex.EncodeToString(b) }\n")
	}
	b.WriteString(`func u64(v uint64) string { return fmt.Sprintf("%016x", v) }
func u32(v uint32) string { return fmt.Sprintf("%08x", v) }

func main() {
`)
	fixs := FixturesFor(lib.Kind)
	usesData := lib.Kind != KindChaChaQR && lib.Kind != KindPoly5 && lib.Kind != KindPolyDonna32 && lib.Kind != KindCurveDonna64 && lib.Kind != KindYyjsonInt && lib.Kind != KindTweetHsalsa
	for i, f := range fixs {
		fmt.Fprintf(&b, "\t// %s\n", f.Name)
		if usesData {
			fmt.Fprintf(&b, "\td%d := %s\n\t_ = d%d\n", i, goByteLit(f.Data), i)
		}
		if len(f.Data2) > 0 {
			fmt.Fprintf(&b, "\te%d := %s\n\t_ = e%d\n", i, goByteLit(f.Data2), i)
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
			if len(f.Data) == 0 || len(f.Data2) == 0 {
				fmt.Fprintf(&b, "\tfmt.Printf(\"%s e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\\n\")\n", f.Name)
			} else {
				fmt.Fprintf(&b, `	{ dst := make([]byte, len(d%d)); kernel.%s(dst, d%d, e%d, uint64(len(d%d)));
		sum := sha256.Sum256(dst); fmt.Printf("%s %%s\n", hx(sum[:])) }
`, i, lib.SgoFunc, i, i, i, f.Name)
			}
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
		case KindPolyDonna32:
			fmt.Fprintf(&b, `	{ h := make([]uint32, 5); r := []uint32{0x3ffffff,0x3fffffe,0x3fffffd,0x3fffffc,0x3fffffb};
		block := make([]byte, 16); for j := range block { block[j] = 0x5a };
		kernel.%s(h, r, block, 1 << 24);
		var s string; for _, v := range h { s += u32(v) }; fmt.Printf("%s %%s\n", s) }
`, lib.SgoFunc, f.Name)
		case KindCurveDonna64:
			fmt.Fprintf(&b, `	{ in := []uint64{0x123456789, 0x23456789a, 0x3456789ab, 0x456789abc, 0x56789abcd};
		out := make([]uint64, 5);
		kernel.%s(out, in);
		var s string; for _, v := range out { s += u64(v) }; fmt.Printf("%s %%s\n", s) }
`, lib.SgoFunc, f.Name)
		case KindYyjsonInt:
			fmt.Fprintf(&b, `	{ buf := make([]byte, 32); n := kernel.%s(buf, %d); fmt.Printf("%s %%s\n", string(buf[:n])) }
`, lib.SgoFunc, f.Seed, f.Name)
		case KindCjsonCore:
			if len(f.Data) == 0 || len(f.Data2) == 0 {
				fmt.Fprintf(&b, "\tfmt.Printf(\"%s 0\\n\")\n", f.Name)
			} else {
				fmt.Fprintf(&b, "\tfmt.Printf(\"%s %%d\\n\", kernel.%s(d%d, e%d, uint64(len(d%d))))\n", f.Name, lib.SgoFunc, i, i, i)
			}
		case KindStbiPng:
			if len(f.Data) == 0 || len(f.Data2) == 0 {
				fmt.Fprintf(&b, "\tfmt.Printf(\"%s 00000000\\n\")\n", f.Name)
			} else {
				fmt.Fprintf(&b, `	{ recon := make([]byte, len(d%d));
		kernel.%s(recon, d%d, e%d, uint64(len(d%d)), 4, 4);
		sum := sha256.Sum256(recon); fmt.Printf("%s %%s\n", hx(sum[:])) }
`, i, lib.SgoFunc, i, i, i, f.Name)
			}
		case KindUtf8Proc:
			if len(f.Data) == 0 {
				fmt.Fprintf(&b, "\tfmt.Printf(\"%s 0 0\\n\")\n", f.Name)
			} else {
				fmt.Fprintf(&b, `	{ dst := make([]int, 1); r := kernel.%s(d%d, uint64(len(d%d)), dst); fmt.Printf("%s %%d %%d\n", r, dst[0]) }
`, lib.SgoFunc, i, i, f.Name)
			}
		case KindFastlz1:
			if len(f.Data) == 0 {
				fmt.Fprintf(&b, "\tfmt.Printf(\"%s 0\\n\")\n", f.Name)
			} else {
				fmt.Fprintf(&b, `	{ out := make([]byte, len(d%d)*4+64);
		r := kernel.%s(d%d, len(d%d), out, len(out));
		fmt.Printf("%s %%d\n", r) }
`, i, lib.SgoFunc, i, i, f.Name)
			}
		case KindMurmur128:
			fmt.Fprintf(&b, `	{ out := make([]uint64, 2); kernel.%s(d%d, uint64(len(d%d)), uint32(%d), out); fmt.Printf("%s %%s%%s\n", u64(out[0]), u64(out[1])) }
`, lib.SgoFunc, i, i, f.Seed, f.Name)
		case KindTweetHsalsa:
			fmt.Fprintf(&b, `	{ in := []byte{1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16};
		k := []byte{0x10,0x20,0x30,0x40,0x50,0x60,0x70,0x80,0x90,0xa0,0xb0,0xc0,0xd0,0xe0,0xf0,0x01,
			0x02,0x03,0x04,0x05,0x06,0x07,0x08,0x09,0x0a,0x0b,0x0c,0x0d,0x0e,0x0f,0x11,0x22};
		c := []byte("expand 32-byte k");
		out := make([]byte, 32);
		kernel.%s(out, in, k, c);
		var s string; for _, b := range out { s += fmt.Sprintf("%%02x", b) }; fmt.Printf("%s %%s\n", s) }
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
		case KindDotF32:
			if len(f.Data) == 0 || len(f.Data2) == 0 {
				fmt.Fprintf(&b, "\tfmt.Printf(\"%s %%.6e\\n\", 0.0)\n", f.Name)
			} else {
				fmt.Fprintf(&b, `	{ var res float64; n := len(d%d) / 4
		if n > 0 {
			a := make([]float32, n); b := make([]float32, n)
			for j := 0; j < n; j++ {
				a[j] = float32(d%d[j*4])
				b[j] = float32(e%d[j*4])
			}
			kernel.%s(a, b, uint64(n), &res)
		}
		fmt.Printf("%s %%.6e\n", res) }
`, i, i, i, lib.SgoFunc, f.Name)
			}
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
