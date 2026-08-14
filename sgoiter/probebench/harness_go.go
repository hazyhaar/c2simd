package probebench

import (
	"fmt"
	"strings"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/tribench"
)

// GenProbeGoMain generates package main that probes one stratum in-process.
// Prints one JSON line: PROBE{...}
func GenProbeGoMain(lib tribench.Lib, st Stratum) string {
	var b strings.Builder
	b.WriteString(`package main

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"trib/kernel"
)

func main() {
	var ms0, ms1 runtime.MemStats

`)
	// payload
	switch lib.Kind {
	case tribench.KindHash64, tribench.KindHash32, tribench.KindHash32Seed, tribench.KindLibInj:
		fmt.Fprintf(&b, "\tdata := make([]byte, %d)\n\tfor i := range data { data[i] = byte(i * 3 + 7) }\n", st.Bytes)
		fmt.Fprintf(&b, "\titers := %d\n\tsink := uint64(0)\n", st.Iters)
		b.WriteString("\truntime.GC()\n\truntime.ReadMemStats(&ms0)\n")
		fmt.Fprintf(&b, "\tt0 := time.Now()\n\tfor i := 0; i < iters; i++ {\n")
		switch lib.Kind {
		case tribench.KindHash64:
			fmt.Fprintf(&b, "\t\tsink ^= kernel.%s(data, uint64(len(data)))\n", lib.SgoFunc)
		case tribench.KindHash32:
			fmt.Fprintf(&b, "\t\tsink ^= uint64(kernel.%s(data, uint64(len(data))))\n", lib.SgoFunc)
		case tribench.KindHash32Seed:
			fmt.Fprintf(&b, "\t\tsink ^= uint64(kernel.%s(data, uint64(len(data)), 42))\n", lib.SgoFunc)
		case tribench.KindLibInj:
			fmt.Fprintf(&b, "\t\tsink ^= uint64(kernel.%s(data, uint64(len(data))))\n", lib.SgoFunc)
		}
		b.WriteString("\t}\n\telapsed := time.Since(t0)\n")
		b.WriteString("\truntime.ReadMemStats(&ms1)\n")
		fmt.Fprintf(&b, "\treport(%q, %q, %q, %d, iters, elapsed, ms0, ms1, sink)\n", lib.ID, st.ID, st.Phase, st.Bytes)

	case tribench.KindSipHash:
		fmt.Fprintf(&b, "\tdata := make([]byte, %d)\n\tfor i := range data { data[i] = byte(i) }\n", st.Bytes)
		b.WriteString("\tkey := []byte{0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15}\n")
		fmt.Fprintf(&b, "\titers := %d\n\tsink := uint64(0)\n", st.Iters)
		b.WriteString("\truntime.GC()\n\truntime.ReadMemStats(&ms0)\n\tt0 := time.Now()\n")
		fmt.Fprintf(&b, "\tfor i := 0; i < iters; i++ { sink ^= kernel.%s(data, uint64(len(data)), key) }\n", lib.SgoFunc)
		b.WriteString("\telapsed := time.Since(t0)\n\truntime.ReadMemStats(&ms1)\n")
		fmt.Fprintf(&b, "\treport(%q, %q, %q, %d, iters, elapsed, ms0, ms1, sink)\n", lib.ID, st.ID, st.Phase, st.Bytes)

	case tribench.KindXor:
		fmt.Fprintf(&b, "\tn := %d\n\ta := make([]byte, n); b2 := make([]byte, n); dst := make([]byte, n)\n", st.Bytes)
		b.WriteString("\tfor i := 0; i < n; i++ { a[i] = byte(i); b2[i] = byte(i * 5) }\n")
		fmt.Fprintf(&b, "\titers := %d\n", st.Iters)
		b.WriteString("\truntime.GC()\n\truntime.ReadMemStats(&ms0)\n\tt0 := time.Now()\n")
		fmt.Fprintf(&b, "\tfor i := 0; i < iters; i++ { kernel.%s(dst, a, b2, uint64(n)) }\n", lib.SgoFunc)
		b.WriteString("\telapsed := time.Since(t0)\n\truntime.ReadMemStats(&ms1)\n\tsink := uint64(dst[0])\n")
		fmt.Fprintf(&b, "\treport(%q, %q, %q, %d, iters, elapsed, ms0, ms1, sink)\n", lib.ID, st.ID, st.Phase, st.Bytes)

	case tribench.KindBase64:
		fmt.Fprintf(&b, "\tn := %d\n\tsrc := make([]byte, n); dst := make([]byte, (n+2)/3*4+8)\n", st.Bytes)
		b.WriteString("\tfor i := range src { src[i] = byte(i) }\n")
		fmt.Fprintf(&b, "\titers := %d\n\tsink := uint64(0)\n", st.Iters)
		b.WriteString("\truntime.GC()\n\truntime.ReadMemStats(&ms0)\n\tt0 := time.Now()\n")
		fmt.Fprintf(&b, "\tfor i := 0; i < iters; i++ { sink += kernel.%s(src, uint64(n), dst) }\n", lib.SgoFunc)
		b.WriteString("\telapsed := time.Since(t0)\n\truntime.ReadMemStats(&ms1)\n")
		fmt.Fprintf(&b, "\treport(%q, %q, %q, %d, iters, elapsed, ms0, ms1, sink)\n", lib.ID, st.ID, st.Phase, st.Bytes)

	case tribench.KindBlake2b:
		b.WriteString("\th := make([]uint64, 8); block := make([]byte, 128)\n\tfor i := range block { block[i] = byte(i) }\n")
		fmt.Fprintf(&b, "\titers := %d\n", st.Iters)
		b.WriteString("\truntime.GC()\n\truntime.ReadMemStats(&ms0)\n\tt0 := time.Now()\n")
		fmt.Fprintf(&b, "\tfor i := 0; i < iters; i++ { kernel.%s(h, block, 0, 0, 0xffffffffffffffff, 0xffffffffffffffff) }\n", lib.SgoFunc)
		b.WriteString("\telapsed := time.Since(t0)\n\truntime.ReadMemStats(&ms1)\n\tsink := h[0]\n")
		fmt.Fprintf(&b, "\treport(%q, %q, %q, 128, iters, elapsed, ms0, ms1, sink)\n", lib.ID, st.ID, st.Phase)

	case tribench.KindMD5:
		b.WriteString("\tst := make([]uint32, 4); block := make([]byte, 64)\n\tfor i := range block { block[i] = byte(i) }\n")
		fmt.Fprintf(&b, "\titers := %d\n", st.Iters)
		b.WriteString("\truntime.GC()\n\truntime.ReadMemStats(&ms0)\n\tt0 := time.Now()\n")
		fmt.Fprintf(&b, "\tfor i := 0; i < iters; i++ { kernel.%s(st, block) }\n", lib.SgoFunc)
		b.WriteString("\telapsed := time.Since(t0)\n\truntime.ReadMemStats(&ms1)\n\tsink := uint64(st[0])\n")
		fmt.Fprintf(&b, "\treport(%q, %q, %q, 64, iters, elapsed, ms0, ms1, sink)\n", lib.ID, st.ID, st.Phase)

	case tribench.KindChaChaQR:
		b.WriteString("\tvar a,b2,c,d uint32 = 1,2,3,4\n")
		fmt.Fprintf(&b, "\titers := %d\n", st.Iters)
		b.WriteString("\truntime.GC()\n\truntime.ReadMemStats(&ms0)\n\tt0 := time.Now()\n")
		fmt.Fprintf(&b, "\tfor i := 0; i < iters; i++ { kernel.%s(&a,&b2,&c,&d) }\n", lib.SgoFunc)
		b.WriteString("\telapsed := time.Since(t0)\n\truntime.ReadMemStats(&ms1)\n\tsink := uint64(a)^uint64(d)\n")
		fmt.Fprintf(&b, "\treport(%q, %q, %q, 16, iters, elapsed, ms0, ms1, sink)\n", lib.ID, st.ID, st.Phase)

	case tribench.KindPoly5:
		b.WriteString("\th := make([]uint32, 5); r := make([]uint32, 5); m := make([]uint32, 4)\n")
		b.WriteString("\tfor i := 0; i < 5; i++ { h[i]=uint32(i); r[i]=uint32(i+3) }; for i:=0;i<4;i++{m[i]=uint32(i*7)}\n")
		fmt.Fprintf(&b, "\titers := %d\n", st.Iters)
		b.WriteString("\truntime.GC()\n\truntime.ReadMemStats(&ms0)\n\tt0 := time.Now()\n")
		fmt.Fprintf(&b, "\tfor i := 0; i < iters; i++ { kernel.%s(h, r, m) }\n", lib.SgoFunc)
		b.WriteString("\telapsed := time.Since(t0)\n\truntime.ReadMemStats(&ms1)\n\tsink := uint64(h[0])\n")
		fmt.Fprintf(&b, "\treport(%q, %q, %q, 16, iters, elapsed, ms0, ms1, sink)\n", lib.ID, st.ID, st.Phase)

	case tribench.KindTweetVer:
		eq := st.ID == "ver_eq"
		b.WriteString("\tx := make([]byte, 16); y := make([]byte, 16)\n")
		if eq {
			b.WriteString("\tfor i := range x { x[i]=1; y[i]=1 }\n")
		} else {
			b.WriteString("\tfor i := range x { x[i]=1; y[i]=2 }\n")
		}
		// harvest exposes Vn; crypto_verify_16 is thin wrapper (n=16)
		fmt.Fprintf(&b, "\titers := %d\n\tsink := 0\n", st.Iters)
		b.WriteString("\truntime.GC()\n\truntime.ReadMemStats(&ms0)\n\tt0 := time.Now()\n")
		b.WriteString("\tfor i := 0; i < iters; i++ { sink += kernel.Vn(x, y, 16) }\n")
		b.WriteString("\telapsed := time.Since(t0)\n\truntime.ReadMemStats(&ms1)\n")
		fmt.Fprintf(&b, "\treport(%q, %q, %q, 16, iters, elapsed, ms0, ms1, uint64(sink))\n", lib.ID, st.ID, st.Phase)

	default:
		b.WriteString("\t_ = binary.LittleEndian\n\t_ = unsafe.Sizeof(0)\n\tfmt.Println(`PROBE{\"error\":\"kind\"}`)\n")
	}
	b.WriteString("}\n\n")
	b.WriteString(probeReportGo)
	return b.String()
}

const probeReportGo = `
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
	fmt.Printf("PROBE{\"lib\":%q,\"stratum\":%q,\"phase\":%q,\"backend\":\"sgoiter\",\"payload_bytes\":%d,\"iters\":%d,\"ns_per_op\":%.3f,\"mb_s\":%.3f,\"allocs\":%d,\"alloc_bytes\":%d,\"heap_inuse\":%d,\"max_rss_kib\":0,\"minflt\":0,\"majflt\":0,\"sink\":%d}\n",
		lib, stratum, phase, payload, iters, ns, mbs, allocs, abytes, ms1.HeapInuse, sink)
	_ = unsafe.Sizeof(0)
	_ = binary.LittleEndian
}
`
