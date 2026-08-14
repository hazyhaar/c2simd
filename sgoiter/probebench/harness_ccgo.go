package probebench

import (
	"fmt"
	"strings"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/tribench"
)

// GenProbeCcgoMain times one stratum via modernc libc + ccgo symbols.
// Buffers allocated before ms0 (same contract as GenProbeGoMain).
func GenProbeCcgoMain(lib tribench.Lib, st Stratum) string {
	var b strings.Builder
	fn := lib.CcgoFunc
	if fn == "" {
		fn = lib.CFunc
	}
	b.WriteString(`package main

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"modernc.org/libc"
)

func typesize(n int) uint64 {
	if n <= 0 {
		return 1
	}
	return uint64(n)
}
func put(tls *libc.TLS, p uintptr, b []byte) {
	for i := 0; i < len(b); i++ {
		*(*byte)(unsafe.Pointer(p + uintptr(i))) = b[i]
	}
}

func main() {
	tls := libc.NewTLS()
	defer tls.Close()
	var ms0, ms1 runtime.MemStats
`)
	switch lib.Kind {
	case tribench.KindHash64, tribench.KindHash32, tribench.KindHash32Seed, tribench.KindLibInj:
		fmt.Fprintf(&b, "\tn := %d\n", st.Bytes)
		b.WriteString("\tdata := make([]byte, n)\n\tfor i := range data { data[i] = byte(i*3 + 7) }\n")
		b.WriteString("\tvar p uintptr\n\tif n > 0 { p = libc.Xmalloc(tls, typesize(n)); put(tls, p, data); defer libc.Xfree(tls, p) }\n")
		fmt.Fprintf(&b, "\titers := %d\n\tsink := uint64(0)\n", st.Iters)
		b.WriteString("\truntime.GC()\n\truntime.ReadMemStats(&ms0)\n\tt0 := time.Now()\n")
		b.WriteString("\tfor i := 0; i < iters; i++ {\n")
		switch lib.Kind {
		case tribench.KindHash64:
			fmt.Fprintf(&b, "\t\tsink ^= uint64(%s(tls, p, uint64(n)))\n", fn)
		case tribench.KindHash32:
			fmt.Fprintf(&b, "\t\tsink ^= uint64(%s(tls, p, uint64(n)))\n", fn)
		case tribench.KindHash32Seed:
			fmt.Fprintf(&b, "\t\tsink ^= uint64(%s(tls, p, uint64(n), uint32(42)))\n", fn)
		case tribench.KindLibInj:
			fmt.Fprintf(&b, "\t\tsink ^= uint64(%s(tls, p, uint64(n)))\n", fn)
		}
		b.WriteString("\t}\n\telapsed := time.Since(t0)\n\truntime.ReadMemStats(&ms1)\n")
		fmt.Fprintf(&b, "\treport(%q, %q, %q, n, iters, elapsed, ms0, ms1, sink)\n", lib.ID, st.ID, st.Phase)

	case tribench.KindSipHash:
		fmt.Fprintf(&b, "\tn := %d\n", st.Bytes)
		b.WriteString("\tdata := make([]byte, n)\n\tfor i := range data { data[i] = byte(i) }\n")
		b.WriteString("\tkey := []byte{0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15}\n")
		b.WriteString("\tvar p,k uintptr\n\tif n > 0 { p = libc.Xmalloc(tls, typesize(n)); put(tls, p, data); defer libc.Xfree(tls, p) }\n")
		b.WriteString("\tk = libc.Xmalloc(tls, 16); put(tls, k, key); defer libc.Xfree(tls, k)\n")
		fmt.Fprintf(&b, "\titers := %d\n\tsink := uint64(0)\n", st.Iters)
		b.WriteString("\truntime.GC()\n\truntime.ReadMemStats(&ms0)\n\tt0 := time.Now()\n")
		fmt.Fprintf(&b, "\tfor i := 0; i < iters; i++ { sink ^= uint64(%s(tls, p, uint64(n), k)) }\n", fn)
		b.WriteString("\telapsed := time.Since(t0)\n\truntime.ReadMemStats(&ms1)\n")
		fmt.Fprintf(&b, "\treport(%q, %q, %q, n, iters, elapsed, ms0, ms1, sink)\n", lib.ID, st.ID, st.Phase)

	case tribench.KindXor:
		fmt.Fprintf(&b, "\tn := %d\n", st.Bytes)
		b.WriteString("\ta := make([]byte, n); b2 := make([]byte, n)\n\tfor i := 0; i < n; i++ { a[i] = byte(i); b2[i] = byte(i * 5) }\n")
		b.WriteString("\tp1 := libc.Xmalloc(tls, typesize(n)); p2 := libc.Xmalloc(tls, typesize(n)); pd := libc.Xmalloc(tls, typesize(n))\n")
		b.WriteString("\tput(tls, p1, a); put(tls, p2, b2)\n\tdefer libc.Xfree(tls, p1); defer libc.Xfree(tls, p2); defer libc.Xfree(tls, pd)\n")
		fmt.Fprintf(&b, "\titers := %d\n", st.Iters)
		b.WriteString("\truntime.GC()\n\truntime.ReadMemStats(&ms0)\n\tt0 := time.Now()\n")
		fmt.Fprintf(&b, "\tfor i := 0; i < iters; i++ { %s(tls, pd, p1, p2, uint64(n)) }\n", fn)
		b.WriteString("\telapsed := time.Since(t0)\n\truntime.ReadMemStats(&ms1)\n")
		b.WriteString("\tsink := uint64(*(*byte)(unsafe.Pointer(pd)))\n")
		fmt.Fprintf(&b, "\treport(%q, %q, %q, n, iters, elapsed, ms0, ms1, sink)\n", lib.ID, st.ID, st.Phase)

	case tribench.KindBase64:
		fmt.Fprintf(&b, "\tn := %d\n", st.Bytes)
		b.WriteString("\tsrc := make([]byte, n)\n\tfor i := range src { src[i] = byte(i) }\n")
		b.WriteString("\tdn := uint64((n+2)/3*4 + 8)\n")
		b.WriteString("\tp := libc.Xmalloc(tls, typesize(n)); dst := libc.Xmalloc(tls, dn)\n")
		b.WriteString("\tput(tls, p, src)\n\tdefer libc.Xfree(tls, p); defer libc.Xfree(tls, dst)\n")
		fmt.Fprintf(&b, "\titers := %d\n\tsink := uint64(0)\n", st.Iters)
		b.WriteString("\truntime.GC()\n\truntime.ReadMemStats(&ms0)\n\tt0 := time.Now()\n")
		fmt.Fprintf(&b, "\tfor i := 0; i < iters; i++ { sink += uint64(%s(tls, p, uint64(n), dst)) }\n", fn)
		b.WriteString("\telapsed := time.Since(t0)\n\truntime.ReadMemStats(&ms1)\n")
		fmt.Fprintf(&b, "\treport(%q, %q, %q, n, iters, elapsed, ms0, ms1, sink)\n", lib.ID, st.ID, st.Phase)

	case tribench.KindBlake2b:
		b.WriteString("\th := libc.Xmalloc(tls, 64); block := libc.Xmalloc(tls, 128)\n")
		b.WriteString("\tfor j := 0; j < 8; j++ { *(*uint64)(unsafe.Pointer(h + uintptr(j*8))) = 0 }\n")
		b.WriteString("\tfor j := 0; j < 128; j++ { *(*byte)(unsafe.Pointer(block + uintptr(j))) = byte(j) }\n")
		b.WriteString("\tdefer libc.Xfree(tls, h); defer libc.Xfree(tls, block)\n")
		fmt.Fprintf(&b, "\titers := %d\n", st.Iters)
		b.WriteString("\truntime.GC()\n\truntime.ReadMemStats(&ms0)\n\tt0 := time.Now()\n")
		fmt.Fprintf(&b, "\tfor i := 0; i < iters; i++ { %s(tls, h, block, 0, 0, 0xffffffffffffffff, 0xffffffffffffffff) }\n", fn)
		b.WriteString("\telapsed := time.Since(t0)\n\truntime.ReadMemStats(&ms1)\n")
		b.WriteString("\tsink := *(*uint64)(unsafe.Pointer(h))\n")
		fmt.Fprintf(&b, "\treport(%q, %q, %q, 128, iters, elapsed, ms0, ms1, sink)\n", lib.ID, st.ID, st.Phase)

	case tribench.KindMD5:
		b.WriteString("\tst := libc.Xmalloc(tls, 16); block := libc.Xmalloc(tls, 64)\n")
		b.WriteString("\tfor j := 0; j < 4; j++ { *(*uint32)(unsafe.Pointer(st + uintptr(j*4))) = 0 }\n")
		b.WriteString("\tfor j := 0; j < 64; j++ { *(*byte)(unsafe.Pointer(block + uintptr(j))) = byte(j) }\n")
		b.WriteString("\tdefer libc.Xfree(tls, st); defer libc.Xfree(tls, block)\n")
		fmt.Fprintf(&b, "\titers := %d\n", st.Iters)
		b.WriteString("\truntime.GC()\n\truntime.ReadMemStats(&ms0)\n\tt0 := time.Now()\n")
		fmt.Fprintf(&b, "\tfor i := 0; i < iters; i++ { %s(tls, st, block) }\n", fn)
		b.WriteString("\telapsed := time.Since(t0)\n\truntime.ReadMemStats(&ms1)\n")
		b.WriteString("\tsink := uint64(*(*uint32)(unsafe.Pointer(st)))\n")
		fmt.Fprintf(&b, "\treport(%q, %q, %q, 64, iters, elapsed, ms0, ms1, sink)\n", lib.ID, st.ID, st.Phase)

	case tribench.KindChaChaQR:
		// ccgo: chacha20_quarter_round(tls, a, b, c, d) with *uint32 as uintptr
		b.WriteString("\ta := libc.Xmalloc(tls, 4); b2 := libc.Xmalloc(tls, 4); c := libc.Xmalloc(tls, 4); d := libc.Xmalloc(tls, 4)\n")
		b.WriteString("\t*(*uint32)(unsafe.Pointer(a))=1; *(*uint32)(unsafe.Pointer(b2))=2; *(*uint32)(unsafe.Pointer(c))=3; *(*uint32)(unsafe.Pointer(d))=4\n")
		b.WriteString("\tdefer libc.Xfree(tls,a); defer libc.Xfree(tls,b2); defer libc.Xfree(tls,c); defer libc.Xfree(tls,d)\n")
		fmt.Fprintf(&b, "\titers := %d\n", st.Iters)
		b.WriteString("\truntime.GC()\n\truntime.ReadMemStats(&ms0)\n\tt0 := time.Now()\n")
		fmt.Fprintf(&b, "\tfor i := 0; i < iters; i++ { %s(tls, a, b2, c, d) }\n", fn)
		b.WriteString("\telapsed := time.Since(t0)\n\truntime.ReadMemStats(&ms1)\n")
		b.WriteString("\tsink := uint64(*(*uint32)(unsafe.Pointer(a))) ^ uint64(*(*uint32)(unsafe.Pointer(d)))\n")
		fmt.Fprintf(&b, "\treport(%q, %q, %q, 16, iters, elapsed, ms0, ms1, sink)\n", lib.ID, st.ID, st.Phase)

	case tribench.KindPoly5:
		b.WriteString("\th := libc.Xmalloc(tls, 20); r := libc.Xmalloc(tls, 20); m := libc.Xmalloc(tls, 16)\n")
		b.WriteString("\tfor j := 0; j < 5; j++ { *(*uint32)(unsafe.Pointer(h+uintptr(j*4)))=uint32(j); *(*uint32)(unsafe.Pointer(r+uintptr(j*4)))=uint32(j+3) }\n")
		b.WriteString("\tfor j := 0; j < 4; j++ { *(*uint32)(unsafe.Pointer(m+uintptr(j*4)))=uint32(j*7) }\n")
		b.WriteString("\tdefer libc.Xfree(tls,h); defer libc.Xfree(tls,r); defer libc.Xfree(tls,m)\n")
		fmt.Fprintf(&b, "\titers := %d\n", st.Iters)
		b.WriteString("\truntime.GC()\n\truntime.ReadMemStats(&ms0)\n\tt0 := time.Now()\n")
		fmt.Fprintf(&b, "\tfor i := 0; i < iters; i++ { %s(tls, h, r, m) }\n", fn)
		b.WriteString("\telapsed := time.Since(t0)\n\truntime.ReadMemStats(&ms1)\n")
		b.WriteString("\tsink := uint64(*(*uint32)(unsafe.Pointer(h)))\n")
		fmt.Fprintf(&b, "\treport(%q, %q, %q, 16, iters, elapsed, ms0, ms1, sink)\n", lib.ID, st.ID, st.Phase)

	case tribench.KindTweetVer:
		eq := st.ID == "ver_eq"
		b.WriteString("\tx := libc.Xmalloc(tls, 16); y := libc.Xmalloc(tls, 16)\n")
		if eq {
			b.WriteString("\tfor j := 0; j < 16; j++ { *(*byte)(unsafe.Pointer(x+uintptr(j)))=1; *(*byte)(unsafe.Pointer(y+uintptr(j)))=1 }\n")
		} else {
			b.WriteString("\tfor j := 0; j < 16; j++ { *(*byte)(unsafe.Pointer(x+uintptr(j)))=1; *(*byte)(unsafe.Pointer(y+uintptr(j)))=2 }\n")
		}
		b.WriteString("\tdefer libc.Xfree(tls,x); defer libc.Xfree(tls,y)\n")
		fmt.Fprintf(&b, "\titers := %d\n\tsink := 0\n", st.Iters)
		b.WriteString("\truntime.GC()\n\truntime.ReadMemStats(&ms0)\n\tt0 := time.Now()\n")
		fmt.Fprintf(&b, "\tfor i := 0; i < iters; i++ { sink += int(%s(tls, x, y)) }\n", fn)
		b.WriteString("\telapsed := time.Since(t0)\n\truntime.ReadMemStats(&ms1)\n")
		fmt.Fprintf(&b, "\treport(%q, %q, %q, 16, iters, elapsed, ms0, ms1, uint64(sink))\n", lib.ID, st.ID, st.Phase)

	default:
		b.WriteString("\tfmt.Println(`PROBE{\"error\":\"ccgo kind\"}`)\n")
	}
	b.WriteString("}\n\n")
	b.WriteString(probeReportCcgo)
	return b.String()
}

const probeReportCcgo = `
func report(lib, stratum, phase string, payload int, iters int, elapsed time.Duration, ms0, ms1 runtime.MemStats, sink uint64) {
	if iters < 1 {
		iters = 1
	}
	ns := float64(elapsed.Nanoseconds()) / float64(iters)
	bytesTotal := float64(payload) * float64(iters)
	mbs := 0.0
	if elapsed.Seconds() > 0 {
		mbs = bytesTotal / elapsed.Seconds() / (1024 * 1024)
	}
	allocs := int64(ms1.Mallocs - ms0.Mallocs)
	abytes := int64(ms1.TotalAlloc - ms0.TotalAlloc)
	if sink == 0xdead {
		fmt.Print("")
	}
	fmt.Printf("PROBE{\"lib\":%q,\"stratum\":%q,\"phase\":%q,\"backend\":\"ccgo\",\"payload_bytes\":%d,\"iters\":%d,\"ns_per_op\":%.3f,\"mb_s\":%.3f,\"allocs\":%d,\"alloc_bytes\":%d,\"heap_inuse\":%d,\"max_rss_kib\":0,\"minflt\":0,\"majflt\":0,\"sink\":%d}\n",
		lib, stratum, phase, payload, iters, ns, mbs, allocs, abytes, ms1.HeapInuse, sink)
	_ = unsafe.Sizeof(0)
	_ = binary.LittleEndian
}
`
