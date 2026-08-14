package tribench

import (
	"fmt"
	"strings"
)

// GenHarnessCcgoMain produces main using modernc libc TLS calling ccgo symbols.
func GenHarnessCcgoMain(lib Lib) string {
	var b strings.Builder
	b.WriteString("package main\n\nimport (\n")
	if lib.Kind == KindXor {
		b.WriteString("\t\"crypto/sha256\"\n\t\"encoding/hex\"\n")
	}
	b.WriteString("\t\"fmt\"\n\t\"unsafe\"\n\n\t\"modernc.org/libc\"\n)\n\n")
	if lib.Kind == KindXor {
		b.WriteString("func hx(b []byte) string { return hex.EncodeToString(b) }\n")
	}
	b.WriteString(`func fmtU64(v uint64) string { return fmt.Sprintf("%016x", v) }
func fmtU32(v uint32) string { return fmt.Sprintf("%08x", v) }

func put(tls *libc.TLS, p uintptr, b []byte) {
	for i := 0; i < len(b); i++ {
		*(*byte)(unsafe.Pointer(p + uintptr(i))) = b[i]
	}
}
func get(tls *libc.TLS, p uintptr, n int) []byte {
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = *(*byte)(unsafe.Pointer(p + uintptr(i)))
	}
	return out
}

func main() {
	tls := libc.NewTLS()
	defer tls.Close()
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
		n := len(f.Data)
		switch lib.Kind {
		case KindHash64:
			fmt.Fprintf(&b, `	{ var p uintptr; if len(d%d)>0 { p = libc.Xmalloc(tls, typesize(len(d%d))); put(tls, p, d%d); defer libc.Xfree(tls, p) }
		h := %s(tls, p, uint64(len(d%d))); fmt.Printf("%s %%s\n", fmtU64(uint64(h))) }
`, i, i, i, lib.CcgoFunc, i, f.Name)
		case KindHash32:
			fmt.Fprintf(&b, `	{ var p uintptr; if len(d%d)>0 { p = libc.Xmalloc(tls, typesize(len(d%d))); put(tls, p, d%d); defer libc.Xfree(tls, p) }
		h := %s(tls, p, uint64(len(d%d))); fmt.Printf("%s %%s\n", fmtU32(uint32(h))) }
`, i, i, i, lib.CcgoFunc, i, f.Name)
		case KindHash32Seed:
			fmt.Fprintf(&b, `	{ var p uintptr; if len(d%d)>0 { p = libc.Xmalloc(tls, typesize(len(d%d))); put(tls, p, d%d); defer libc.Xfree(tls, p) }
		h := %s(tls, p, uint64(len(d%d)), uint32(%d)); fmt.Printf("%s %%s\n", fmtU32(uint32(h))) }
`, i, i, i, lib.CcgoFunc, i, f.Seed, f.Name)
		case KindSipHash:
			fmt.Fprintf(&b, `	{ var p,k uintptr
		if len(d%d)>0 { p = libc.Xmalloc(tls, typesize(len(d%d))); put(tls, p, d%d); defer libc.Xfree(tls, p) }
		k = libc.Xmalloc(tls, 16); put(tls, k, e%d); defer libc.Xfree(tls, k)
		h := %s(tls, p, uint64(len(d%d)), k); fmt.Printf("%s %%s\n", fmtU64(uint64(h))) }
`, i, i, i, i, lib.CcgoFunc, i, f.Name)
		case KindXor:
			fmt.Fprintf(&b, `	{ n := len(d%d); p1 := libc.Xmalloc(tls, typesize(n)); p2 := libc.Xmalloc(tls, typesize(n)); pd := libc.Xmalloc(tls, typesize(n))
		put(tls, p1, d%d); put(tls, p2, e%d); defer libc.Xfree(tls, p1); defer libc.Xfree(tls, p2); defer libc.Xfree(tls, pd)
		%s(tls, pd, p1, p2, uint64(n)); out := get(tls, pd, n); sum := sha256.Sum256(out); fmt.Printf("%s %%s\n", hx(sum[:])) }
`, i, i, i, lib.CcgoFunc, f.Name)
		case KindBlake2b:
			fmt.Fprintf(&b, `	{ h := libc.Xmalloc(tls, 8*8); block := libc.Xmalloc(tls, 128)
		for j:=0;j<8;j++ { *(*uint64)(unsafe.Pointer(h+uintptr(j*8))) = 0 }
		for j:=0;j<128;j++ { *(*byte)(unsafe.Pointer(block+uintptr(j))) = 0 }
		put(tls, block, d%d)
		%s(tls, h, block, 0, 0, 0xffffffffffffffff, 0xffffffffffffffff)
		var s string; for j:=0;j<8;j++ { s += fmtU64(*(*uint64)(unsafe.Pointer(h+uintptr(j*8)))) }
		fmt.Printf("%s %%s\n", s); libc.Xfree(tls,h); libc.Xfree(tls,block) }
`, i, lib.CcgoFunc, f.Name)
		case KindChaChaQR:
			fmt.Fprintf(&b, `	{ a:=libc.Xmalloc(tls,4); b:=libc.Xmalloc(tls,4); c:=libc.Xmalloc(tls,4); d:=libc.Xmalloc(tls,4)
		*(*uint32)(unsafe.Pointer(a))=0x11111111; *(*uint32)(unsafe.Pointer(b))=0x22222222
		*(*uint32)(unsafe.Pointer(c))=0x33333333; *(*uint32)(unsafe.Pointer(d))=0x44444444
		%s(tls,a,b,c,d)
		fmt.Printf("%s %%s%%s%%s%%s\n", fmtU32(*(*uint32)(unsafe.Pointer(a))), fmtU32(*(*uint32)(unsafe.Pointer(b))),
			fmtU32(*(*uint32)(unsafe.Pointer(c))), fmtU32(*(*uint32)(unsafe.Pointer(d))))
		libc.Xfree(tls,a);libc.Xfree(tls,b);libc.Xfree(tls,c);libc.Xfree(tls,d) }
`, lib.CcgoFunc, f.Name)
		case KindMD5:
			fmt.Fprintf(&b, `	{ st:=libc.Xmalloc(tls,16); block:=libc.Xmalloc(tls,64)
		iv:=[]uint32{0x67452301,0xefcdab89,0x98badcfe,0x10325476}
		for j:=0;j<4;j++ { *(*uint32)(unsafe.Pointer(st+uintptr(j*4)))=iv[j] }
		for j:=0;j<64;j++ { *(*byte)(unsafe.Pointer(block+uintptr(j)))=0 }
		put(tls, block, d%d)
		%s(tls, st, block)
		var s string; for j:=0;j<4;j++ { s += fmtU32(*(*uint32)(unsafe.Pointer(st+uintptr(j*4)))) }
		fmt.Printf("%s %%s\n", s); libc.Xfree(tls,st); libc.Xfree(tls,block) }
`, i, lib.CcgoFunc, f.Name)
		case KindPoly5:
			fmt.Fprintf(&b, `	{ h:=libc.Xmalloc(tls,20); r:=libc.Xmalloc(tls,20); m:=libc.Xmalloc(tls,16)
		rv:=[]uint32{1,2,3,4,5}; mv:=[]uint32{9,8,7,6}
		for j:=0;j<5;j++ { *(*uint32)(unsafe.Pointer(h+uintptr(j*4)))=0; *(*uint32)(unsafe.Pointer(r+uintptr(j*4)))=rv[j] }
		for j:=0;j<4;j++ { *(*uint32)(unsafe.Pointer(m+uintptr(j*4)))=mv[j] }
		%s(tls,h,r,m)
		var s string; for j:=0;j<5;j++ { s += fmtU32(*(*uint32)(unsafe.Pointer(h+uintptr(j*4)))) }
		fmt.Printf("%s %%s\n", s); libc.Xfree(tls,h);libc.Xfree(tls,r);libc.Xfree(tls,m) }
`, lib.CcgoFunc, f.Name)
		case KindBase64:
			fmt.Fprintf(&b, `	{ n:=len(d%d); p:=libc.Xmalloc(tls, typesize(n)); dn:=uint64((n+2)/3*4+8); dst:=libc.Xmalloc(tls, dn)
		put(tls,p,d%d); nn := %s(tls,p,uint64(n),dst); out:=get(tls,dst,int(nn))
		fmt.Printf("%s %%s\n", string(out)); libc.Xfree(tls,p); libc.Xfree(tls,dst) }
`, i, i, lib.CcgoFunc, f.Name)
		case KindTweetVer:
			fmt.Fprintf(&b, `	{ x:=libc.Xmalloc(tls,16); y:=libc.Xmalloc(tls,16); put(tls,x,d%d); put(tls,y,e%d)
		r:=%s(tls,x,y); fmt.Printf("%s %%d\n", int(r)); libc.Xfree(tls,x); libc.Xfree(tls,y) }
`, i, i, lib.CcgoFunc, f.Name)
		case KindLibInj:
			fmt.Fprintf(&b, `	{ n:=len(d%d); p:=libc.Xmalloc(tls, typesize(n+1)); acc:=libc.Xmalloc(tls, 4)
		put(tls,p,d%d); put(tls,acc,[]byte{'h','e','l',0})
		r:=%s(tls,p,uint64(n),acc); fmt.Printf("%s %%s\n", fmtU64(uint64(r)))
		libc.Xfree(tls,p); libc.Xfree(tls,acc) }
`, i, i, lib.CcgoFunc, f.Name)
		default:
			_ = n
		}
	}
	b.WriteString("}\n\nfunc typesize(n int) uint64 {\n\tif n <= 0 { return 1 }\n\treturn uint64(n)\n}\n")
	return b.String()
}
