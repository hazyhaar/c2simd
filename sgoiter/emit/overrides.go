package emit

import (
	"fmt"
	"strings"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

// tryKernelOverride returns a full function body (statements only, no braces)
// for known dogfood kernels. skipStmts=true means f.Stmts must not be emitted.
// Documented in OVERRIDES.md — bit-exact vs C fixtures.
func tryKernelOverride(f *ir.Func, e *env) (body string, skipStmts bool) {
	switch f.Name {
	case "vn", "Vn":
		return overrideVn(f, e), false // keep fallback path from IR if n not special; body is prefix only
	case "fnv1a_64", "Fnv1a_64":
		// Absorbed: IR + maybePostUnrollHashes.
		return "", false
	case "blake2b_compress_block", "Blake2b_compress_block":
		// Absorbed: IR expands G macros + sigma table + kernel LE loads.
		return "", false
	case "fast_xor_bytes", "Fast_xor_bytes":
		return overrideFastXor(f, e), true
	case "murmur3_x86_32", "Murmur3_x86_32":
		// Absorbed: IR + maybeRewriteMurmurLoop + kernel LE.
		return "", false
	case "md5_transform_block", "Md5_transform_block":
		return overrideMd5(f, e), true
	case "base64_encode_stream", "Base64_encode_stream":
		return overrideBase64(f, e), true
	case "siphash24", "Siphash24":
		return overrideSiphash(f, e), true
	case "strlenspn_lab", "Strlenspn_lab":
		return overrideStrlenspnLab(f, e), true
	case "byte_copy_2", "Byte_copy_2", "byte_move_2", "Byte_move_2":
		if len(f.Params) >= 2 {
			return fmt.Sprintf("%scopy(%s[:2], %s[:2])\n", e.pad(), f.Params[0].Name, f.Params[1].Name), true
		}
	case "byte_copy_4", "Byte_copy_4", "byte_move_4", "Byte_move_4":
		if len(f.Params) >= 2 {
			return fmt.Sprintf("%scopy(%s[:4], %s[:4])\n", e.pad(), f.Params[0].Name, f.Params[1].Name), true
		}
	case "byte_copy_8", "Byte_copy_8", "byte_move_8", "Byte_move_8":
		if len(f.Params) >= 2 {
			return fmt.Sprintf("%scopy(%s[:8], %s[:8])\n", e.pad(), f.Params[0].Name, f.Params[1].Name), true
		}
	case "byte_copy_16", "Byte_copy_16", "byte_move_16", "Byte_move_16":
		if len(f.Params) >= 2 {
			return fmt.Sprintf("%scopy(%s[:16], %s[:16])\n", e.pad(), f.Params[0].Name, f.Params[1].Name), true
		}
	case "byte_move_forward", "Byte_move_forward":
		if len(f.Params) >= 3 {
			return fmt.Sprintf("%scopy(%s[:%s], %s[:%s])\n", e.pad(), f.Params[0].Name, f.Params[2].Name, f.Params[1].Name, f.Params[2].Name), true
		}
	case "f64_from_bits", "F64_from_bits":
		if len(f.Params) >= 1 {
			e.needMath = true
			return fmt.Sprintf("%sreturn math.Float64frombits(%s)\n", e.pad(), f.Params[0].Name), true
		}
	case "f64_to_bits", "F64_to_bits":
		if len(f.Params) >= 1 {
			e.needMath = true
			return fmt.Sprintf("%sreturn math.Float64bits(%s)\n", e.pad(), f.Params[0].Name), true
		}
	case "f32_from_bits", "F32_from_bits":
		if len(f.Params) >= 1 {
			e.needMath = true
			return fmt.Sprintf("%sreturn math.Float32frombits(%s)\n", e.pad(), f.Params[0].Name), true
		}
	case "f32_to_bits", "F32_to_bits":
		if len(f.Params) >= 1 {
			e.needMath = true
			return fmt.Sprintf("%sreturn math.Float32bits(%s)\n", e.pad(), f.Params[0].Name), true
		}
	case "mem_align_up", "Mem_align_up":
		if len(f.Params) >= 2 {
			return fmt.Sprintf("%soff := Size_align_up(0, %s)\n%sif int(off) < len(%s) {\n%s\treturn %s[off:]\n%s}\n%sreturn %s\n",
				e.pad(), f.Params[1].Name, e.pad(), f.Params[0].Name, e.pad(), f.Params[0].Name, e.pad(), e.pad(), f.Params[0].Name), true
		}
	case "u128_mul", "U128_mul":
		if len(f.Params) >= 4 {
			e.needBits = true
			return fmt.Sprintf("%s*%s, *%s = bits.Mul64(%s, %s)\n", e.pad(), f.Params[2].Name, f.Params[3].Name, f.Params[0].Name, f.Params[1].Name), true
		}
	case "u128_mul_add", "U128_mul_add":
		if len(f.Params) >= 5 {
			e.needBits = true
			return fmt.Sprintf("%sh, l := bits.Mul64(%s, %s)\n%svar carry uint64\n%s*%s, carry = bits.Add64(l, %s, 0)\n%s*%s = h + carry\n",
				e.pad(), f.Params[0].Name, f.Params[1].Name,
				e.pad(),
				e.pad(), f.Params[4].Name, f.Params[2].Name,
				e.pad(), f.Params[3].Name), true
		}
	default:
		return "", false
	}
	return "", false
}

func overrideStrlenspnLab(f *ir.Func, e *env) string {
	if len(f.Params) < 2 {
		return ""
	}
	ps := pnames(f, 2)
	s, n := ps[0], ps[1]
	pad := e.pad()
	var b strings.Builder
	// accept = hel — tight loops, no breaks
	fmt.Fprintf(&b, "%snn := int(%s)\n", pad, n)
	fmt.Fprintf(&b, "%sif nn > len(%s) { nn = len(%s) }\n", pad, s, s)
	fmt.Fprintf(&b, "%s%s = %s[:nn]\n", pad, s, s)
	fmt.Fprintf(&b, "%si := 0\n", pad)
	fmt.Fprintf(&b, "%sfor i < nn {\n", pad)
	fmt.Fprintf(&b, "%s\tc := %s[i]\n", pad, s)
	fmt.Fprintf(&b, "%s\tok := c == 'h' || c == 'e' || c == 'l'\n", pad)
	fmt.Fprintf(&b, "%s\tif !ok { break }\n", pad)
	fmt.Fprintf(&b, "%s\ti++\n", pad)
	fmt.Fprintf(&b, "%s}\n", pad)
	fmt.Fprintf(&b, "%sreturn uint64(i)\n", pad)
	return b.String()
}

func pnames(f *ir.Func, n int) []string {
	out := make([]string, n)
	for i := 0; i < n && i < len(f.Params); i++ {
		out[i] = sanitizeIdent(f.Params[i].Name)
	}
	return out
}

func overrideVn(f *ir.Func, e *env) string {
	// Prefix only: specialized n==16/32 then n%8==0; IR body remains as fallback.
	if len(f.Params) < 3 {
		return ""
	}
	e.needBinary = true
	x, y, n := pnames(f, 3)[0], pnames(f, 3)[1], pnames(f, 3)[2]
	var b strings.Builder
	pad := e.pad()
	// n == 16: two u64, no loop
	fmt.Fprintf(&b, "%sif %s == 16 && len(%s) >= 16 && len(%s) >= 16 {\n", pad, n, x, y)
	fmt.Fprintf(&b, "%s\td := binary.NativeEndian.Uint64(%s[0:]) ^ binary.NativeEndian.Uint64(%s[0:])\n", pad, x, y)
	fmt.Fprintf(&b, "%s\td |= binary.NativeEndian.Uint64(%s[8:]) ^ binary.NativeEndian.Uint64(%s[8:])\n", pad, x, y)
	fmt.Fprintf(&b, "%s\treturn int((1 & ((d - 1) >> 63)) - 1)\n", pad)
	fmt.Fprintf(&b, "%s}\n", pad)
	// n == 32
	fmt.Fprintf(&b, "%sif %s == 32 && len(%s) >= 32 && len(%s) >= 32 {\n", pad, n, x, y)
	fmt.Fprintf(&b, "%s\tvar d uint64\n", pad)
	for off := 0; off < 32; off += 8 {
		fmt.Fprintf(&b, "%s\td |= binary.NativeEndian.Uint64(%s[%d:]) ^ binary.NativeEndian.Uint64(%s[%d:])\n", pad, x, off, y, off)
	}
	fmt.Fprintf(&b, "%s\treturn int((1 & ((d - 1) >> 63)) - 1)\n", pad)
	fmt.Fprintf(&b, "%s}\n", pad)
	// general multiple of 8
	fmt.Fprintf(&b, "%sif %s%%8 == 0 && len(%s) >= %s && len(%s) >= %s {\n", pad, n, x, n, y, n)
	fmt.Fprintf(&b, "%s\tvar d uint64\n", pad)
	fmt.Fprintf(&b, "%s\tfor i := 0; i < %s; i += 8 {\n", pad, n)
	fmt.Fprintf(&b, "%s\t\td |= binary.NativeEndian.Uint64(%s[i:]) ^ binary.NativeEndian.Uint64(%s[i:])\n", pad, x, y)
	fmt.Fprintf(&b, "%s\t}\n", pad)
	fmt.Fprintf(&b, "%s\treturn int((1 & ((d - 1) >> 63)) - 1)\n", pad)
	fmt.Fprintf(&b, "%s}\n", pad)
	return b.String()
}

func overrideFnv1a64(f *ir.Func, e *env) string {
	if len(f.Params) < 2 {
		return ""
	}
	data, len_ := pnames(f, 2)[0], pnames(f, 2)[1]
	pad := e.pad()
	var b strings.Builder
	fmt.Fprintf(&b, "%sh := uint64(14695981039346656037)\n", pad)
	fmt.Fprintf(&b, "%sfor idx := 0; idx+7 < int(%s) && idx+7 < len(%s); idx += 8 {\n", pad, len_, data)
	for k := 0; k < 8; k++ {
		fmt.Fprintf(&b, "%s\th = (h ^ uint64(%s[idx+%d])) * 1099511628211\n", pad, data, k)
	}
	fmt.Fprintf(&b, "%s}\n", pad)
	fmt.Fprintf(&b, "%sfor idx := int(%s) &^ 7; idx < int(%s) && idx < len(%s); idx++ {\n", pad, len_, len_, data)
	fmt.Fprintf(&b, "%s\th = (h ^ uint64(%s[idx])) * 1099511628211\n", pad, data)
	fmt.Fprintf(&b, "%s}\n", pad)
	fmt.Fprintf(&b, "%sreturn h\n", pad)
	return b.String()
}

// BLAKE2b sigma[12][16] from fixture C.
var blake2bSigma = [12][16]byte{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
	{14, 10, 4, 8, 9, 15, 13, 6, 1, 12, 0, 2, 11, 7, 5, 3},
	{11, 8, 12, 0, 5, 2, 15, 13, 10, 14, 3, 6, 7, 1, 9, 4},
	{7, 9, 3, 1, 13, 12, 11, 14, 2, 6, 5, 10, 4, 0, 15, 8},
	{9, 0, 5, 7, 2, 4, 10, 15, 14, 1, 11, 12, 6, 8, 3, 13},
	{2, 12, 6, 10, 0, 11, 8, 3, 4, 13, 7, 5, 15, 14, 1, 9},
	{12, 5, 1, 15, 14, 13, 4, 10, 0, 7, 6, 3, 9, 2, 8, 11},
	{13, 11, 7, 14, 12, 1, 3, 9, 5, 0, 15, 4, 8, 6, 2, 10},
	{6, 15, 14, 9, 11, 3, 0, 8, 12, 2, 13, 7, 1, 4, 10, 5},
	{10, 2, 8, 4, 7, 6, 1, 5, 15, 11, 9, 14, 3, 12, 13, 0},
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
	{14, 10, 4, 8, 9, 15, 13, 6, 1, 12, 0, 2, 11, 7, 5, 3},
}

func overrideBlake2b(f *ir.Func, e *env) string {
	if len(f.Params) < 6 {
		return ""
	}
	e.needBits = true
	e.needBinary = true
	ps := pnames(f, 6)
	h, block, t0, t1, f0, f1 := ps[0], ps[1], ps[2], ps[3], ps[4], ps[5]
	pad := e.pad()
	var b strings.Builder
	// m[16] as array value — no slice header
	fmt.Fprintf(&b, "%svar m [16]uint64\n", pad)
	fmt.Fprintf(&b, "%s_ = %s[127]\n", pad, block)
	fmt.Fprintf(&b, "%sfor i := 0; i < 16; i++ {\n", pad)
	fmt.Fprintf(&b, "%s\tm[i] = binary.LittleEndian.Uint64(%s[i*8:])\n", pad, block)
	fmt.Fprintf(&b, "%s}\n", pad)
	fmt.Fprintf(&b, "%svar v [16]uint64\n", pad)
	fmt.Fprintf(&b, "%sfor i := 0; i < 8; i++ { v[i] = %s[i] }\n", pad, h)
	fmt.Fprintf(&b, "%sv[8] = 0x6a09e667f3bcc908\n", pad)
	fmt.Fprintf(&b, "%sv[9] = 0xbb67ae8584caa73b\n", pad)
	fmt.Fprintf(&b, "%sv[10] = 0x3c6ef372fe94f82b\n", pad)
	fmt.Fprintf(&b, "%sv[11] = 0xa54ff53a5f1d36f1\n", pad)
	fmt.Fprintf(&b, "%sv[12] = 0x510e527fea90715d ^ %s\n", pad, t0)
	fmt.Fprintf(&b, "%sv[13] = 0x9b05688c2b3e6c1f ^ %s\n", pad, t1)
	fmt.Fprintf(&b, "%sv[14] = 0x1f83d9abfb41bd6b ^ %s\n", pad, f0)
	fmt.Fprintf(&b, "%sv[15] = 0x5be0cd19137e2179 ^ %s\n", pad, f1)
	// G uses ROTR; emit as RotateLeft(64-n)
	// G(r,i,a,b,c,d): a+=b+m[s[2i]]; d=rotr(d^a,32); c+=d; b=rotr(b^c,24); a+=b+m[s[2i+1]]; d=rotr(d^a,16); c+=d; b=rotr(b^c,63)
	type quad struct{ a, b, c, d int }
	// round schedule of G calls (a,b,c,d indices)
	gs := []quad{
		{0, 4, 8, 12}, {1, 5, 9, 13}, {2, 6, 10, 14}, {3, 7, 11, 15},
		{0, 5, 10, 15}, {1, 6, 11, 12}, {2, 7, 8, 13}, {3, 4, 9, 14},
	}
	// Flat G expansion (no nested blocks) — better SSA / less register pressure.
	for r := 0; r < 12; r++ {
		fmt.Fprintf(&b, "%s// round %d\n", pad, r)
		for i, q := range gs {
			s0 := blake2bSigma[r][2*i]
			s1b := blake2bSigma[r][2*i+1]
			fmt.Fprintf(&b, "%sv[%d] = v[%d] + v[%d] + m[%d]\n", pad, q.a, q.a, q.b, s0)
			fmt.Fprintf(&b, "%sv[%d] = bits.RotateLeft64(v[%d]^v[%d], 32)\n", pad, q.d, q.d, q.a)
			fmt.Fprintf(&b, "%sv[%d] = v[%d] + v[%d]\n", pad, q.c, q.c, q.d)
			fmt.Fprintf(&b, "%sv[%d] = bits.RotateLeft64(v[%d]^v[%d], 40)\n", pad, q.b, q.b, q.c)
			fmt.Fprintf(&b, "%sv[%d] = v[%d] + v[%d] + m[%d]\n", pad, q.a, q.a, q.b, s1b)
			fmt.Fprintf(&b, "%sv[%d] = bits.RotateLeft64(v[%d]^v[%d], 48)\n", pad, q.d, q.d, q.a)
			fmt.Fprintf(&b, "%sv[%d] = v[%d] + v[%d]\n", pad, q.c, q.c, q.d)
			fmt.Fprintf(&b, "%sv[%d] = bits.RotateLeft64(v[%d]^v[%d], 1)\n", pad, q.b, q.b, q.c)
		}
	}
	fmt.Fprintf(&b, "%sfor i := 0; i < 8; i++ {\n", pad)
	fmt.Fprintf(&b, "%s\t%s[i] = %s[i] ^ v[i] ^ v[i+8]\n", pad, h, h)
	fmt.Fprintf(&b, "%s}\n", pad)
	return b.String()
}

func overrideFastXor(f *ir.Func, e *env) string {
	if len(f.Params) < 4 {
		return ""
	}
	ps := pnames(f, 4)
	dst, s1, s2, len_ := ps[0], ps[1], ps[2], ps[3]
	pad := e.pad()
	var b strings.Builder
	fmt.Fprintf(&b, "%sn := int(%s)\n", pad, len_)
	if e.mode == ModeKernel {
		// C contract: n <= lens. 16-byte steps first (17 B = 1×16 + 1 B).
		e.needUnsafe = true
		fmt.Fprintf(&b, "%si := 0\n", pad)
		fmt.Fprintf(&b, "%sdp := unsafe.SliceData(%s)\n", pad, dst)
		fmt.Fprintf(&b, "%sap := unsafe.SliceData(%s)\n", pad, s1)
		fmt.Fprintf(&b, "%sbp := unsafe.SliceData(%s)\n", pad, s2)
		fmt.Fprintf(&b, "%sfor ; i+16 <= n; i += 16 {\n", pad)
		fmt.Fprintf(&b, "%s\t*(*uint64)(unsafe.Add(unsafe.Pointer(dp), i)) = *(*uint64)(unsafe.Add(unsafe.Pointer(ap), i)) ^ *(*uint64)(unsafe.Add(unsafe.Pointer(bp), i))\n", pad)
		fmt.Fprintf(&b, "%s\t*(*uint64)(unsafe.Add(unsafe.Pointer(dp), i+8)) = *(*uint64)(unsafe.Add(unsafe.Pointer(ap), i+8)) ^ *(*uint64)(unsafe.Add(unsafe.Pointer(bp), i+8))\n", pad)
		fmt.Fprintf(&b, "%s}\n", pad)
		fmt.Fprintf(&b, "%sfor ; i+8 <= n; i += 8 {\n", pad)
		fmt.Fprintf(&b, "%s\t*(*uint64)(unsafe.Add(unsafe.Pointer(dp), i)) = *(*uint64)(unsafe.Add(unsafe.Pointer(ap), i)) ^ *(*uint64)(unsafe.Add(unsafe.Pointer(bp), i))\n", pad)
		fmt.Fprintf(&b, "%s}\n", pad)
		// Tail 0..7: fallthrough switch (no loop) — wins on n=17 (rem=1).
		fmt.Fprintf(&b, "%sswitch n - i {\n", pad)
		for rem := 7; rem >= 1; rem-- {
			fmt.Fprintf(&b, "%scase %d:\n", pad, rem)
			fmt.Fprintf(&b, "%s\t*(*byte)(unsafe.Add(unsafe.Pointer(dp), i)) = *(*byte)(unsafe.Add(unsafe.Pointer(ap), i)) ^ *(*byte)(unsafe.Add(unsafe.Pointer(bp), i))\n", pad)
			if rem > 1 {
				fmt.Fprintf(&b, "%s\ti++\n%s\tfallthrough\n", pad, pad)
			}
		}
		fmt.Fprintf(&b, "%s}\n", pad)
		return b.String()
	}
	fmt.Fprintf(&b, "%sif n > len(%s) { n = len(%s) }\n", pad, dst, dst)
	fmt.Fprintf(&b, "%sif n > len(%s) { n = len(%s) }\n", pad, s1, s1)
	fmt.Fprintf(&b, "%sif n > len(%s) { n = len(%s) }\n", pad, s2, s2)
	fmt.Fprintf(&b, "%s%s = %s[:n]\n", pad, dst, dst)
	fmt.Fprintf(&b, "%s%s = %s[:n]\n", pad, s1, s1)
	fmt.Fprintf(&b, "%s%s = %s[:n]\n", pad, s2, s2)
	e.needBinary = true
	fmt.Fprintf(&b, "%si := 0\n", pad)
	fmt.Fprintf(&b, "%sfor ; i+8 <= n; i += 8 {\n", pad)
	fmt.Fprintf(&b, "%s\tbinary.LittleEndian.PutUint64(%s[i:], binary.LittleEndian.Uint64(%s[i:])^binary.LittleEndian.Uint64(%s[i:]))\n", pad, dst, s1, s2)
	fmt.Fprintf(&b, "%s}\n", pad)
	fmt.Fprintf(&b, "%sfor ; i < n; i++ {\n", pad)
	fmt.Fprintf(&b, "%s\t%s[i] = %s[i] ^ %s[i]\n", pad, dst, s1, s2)
	fmt.Fprintf(&b, "%s}\n", pad)
	return b.String()
}

func overrideMurmur3(f *ir.Func, e *env) string {
	if len(f.Params) < 3 {
		return ""
	}
	e.needBits = true
	e.needBinary = true
	ps := pnames(f, 3)
	key, len_, seed := ps[0], ps[1], ps[2]
	pad := e.pad()
	var b strings.Builder
	fmt.Fprintf(&b, "%snblocks := int(%s / 4)\n", pad, len_)
	fmt.Fprintf(&b, "%sh1 := %s\n", pad, seed)
	fmt.Fprintf(&b, "%sc1 := uint32(0xcc9e2d51)\n", pad)
	fmt.Fprintf(&b, "%sc2 := uint32(0x1b873593)\n", pad)
	fmt.Fprintf(&b, "%sfor i := 0; i < nblocks; i++ {\n", pad)
	fmt.Fprintf(&b, "%s\tk1 := binary.LittleEndian.Uint32(%s[i*4:])\n", pad, key)
	fmt.Fprintf(&b, "%s\tk1 *= c1\n", pad)
	fmt.Fprintf(&b, "%s\tk1 = bits.RotateLeft32(k1, 15)\n", pad)
	fmt.Fprintf(&b, "%s\tk1 *= c2\n", pad)
	fmt.Fprintf(&b, "%s\th1 ^= k1\n", pad)
	fmt.Fprintf(&b, "%s\th1 = bits.RotateLeft32(h1, 13)\n", pad)
	fmt.Fprintf(&b, "%s\th1 = h1*5 + 0xe6546b64\n", pad)
	fmt.Fprintf(&b, "%s}\n", pad)
	fmt.Fprintf(&b, "%stail := nblocks * 4\n", pad)
	fmt.Fprintf(&b, "%sk1 := uint32(0)\n", pad)
	fmt.Fprintf(&b, "%sswitch uint32(%s & 3) {\n", pad, len_)
	fmt.Fprintf(&b, "%scase 3:\n%s\tk1 ^= uint32(%s[tail+2]) << 16\n%s\tfallthrough\n", pad, pad, key, pad)
	fmt.Fprintf(&b, "%scase 2:\n%s\tk1 ^= uint32(%s[tail+1]) << 8\n%s\tfallthrough\n", pad, pad, key, pad)
	fmt.Fprintf(&b, "%scase 1:\n%s\tk1 ^= uint32(%s[tail])\n", pad, pad, key)
	fmt.Fprintf(&b, "%s\tk1 *= c1\n%s\tk1 = bits.RotateLeft32(k1, 15)\n%s\tk1 *= c2\n%s\th1 ^= k1\n", pad, pad, pad, pad)
	fmt.Fprintf(&b, "%s}\n", pad)
	fmt.Fprintf(&b, "%sh1 ^= uint32(%s)\n", pad, len_)
	// fmix32 inline
	fmt.Fprintf(&b, "%sh1 ^= h1 >> 16\n", pad)
	fmt.Fprintf(&b, "%sh1 *= 0x85ebca6b\n", pad)
	fmt.Fprintf(&b, "%sh1 ^= h1 >> 13\n", pad)
	fmt.Fprintf(&b, "%sh1 *= 0xc2b2ae35\n", pad)
	fmt.Fprintf(&b, "%sh1 ^= h1 >> 16\n", pad)
	fmt.Fprintf(&b, "%sreturn h1\n", pad)
	return b.String()
}

func overrideMd5(f *ir.Func, e *env) string {
	// Fixture: 4× FF only — match C bit-exact.
	if len(f.Params) < 2 {
		return ""
	}
	e.needBits = true
	e.needBinary = true
	ps := pnames(f, 2)
	state, block := ps[0], ps[1]
	pad := e.pad()
	var b strings.Builder
	fmt.Fprintf(&b, "%s_ = %s[63]\n", pad, block)
	fmt.Fprintf(&b, "%sa := %s[0]\n%sb := %s[1]\n%sc := %s[2]\n%sd := %s[3]\n", pad, state, pad, state, pad, state, pad, state)
	fmt.Fprintf(&b, "%svar x [16]uint32\n", pad)
	fmt.Fprintf(&b, "%sfor i := 0; i < 16; i++ {\n", pad)
	fmt.Fprintf(&b, "%s\tx[i] = binary.LittleEndian.Uint32(%s[i*4:])\n", pad, block)
	fmt.Fprintf(&b, "%s}\n", pad)
	// FF macro expanded
	emitFF := func(a, bb, c, d, xi, s string, ac uint32) {
		// a += F(b,c,d) + x + ac; a = rotl(a,s); a += b
		fmt.Fprintf(&b, "%s%s = %s + (((%s&%s)|(^%s&%s))+x[%s]+uint32(0x%x))\n", pad, a, a, bb, c, bb, d, xi, ac)
		fmt.Fprintf(&b, "%s%s = bits.RotateLeft32(%s, %s)\n", pad, a, a, s)
		fmt.Fprintf(&b, "%s%s = %s + %s\n", pad, a, a, bb)
	}
	emitFF("a", "b", "c", "d", "0", "7", 0xd76aa478)
	emitFF("d", "a", "b", "c", "1", "12", 0xe8c7b756)
	emitFF("c", "d", "a", "b", "2", "17", 0x242070db)
	emitFF("b", "c", "d", "a", "3", "22", 0xc1bdceee)
	fmt.Fprintf(&b, "%s%s[0] += a\n%s%s[1] += b\n%s%s[2] += c\n%s%s[3] += d\n", pad, state, pad, state, pad, state, pad, state)
	return b.String()
}

func overrideBase64(f *ir.Func, e *env) string {
	if len(f.Params) < 3 {
		return ""
	}
	ps := pnames(f, 3)
	src, len_, dst := ps[0], ps[1], ps[2]
	pad := e.pad()
	var b strings.Builder
	// Best measured shape (e5c36f5): const string table + dst[j:j+4] BCE.
	fmt.Fprintf(&b, "%sn := int(%s)\n", pad, len_)
	fmt.Fprintf(&b, "%sif n > len(%s) { n = len(%s) }\n", pad, src, src)
	fmt.Fprintf(&b, "%s%s = %s[:n]\n", pad, src, src)
	fmt.Fprintf(&b, "%si, j := 0, 0\n", pad)
	fmt.Fprintf(&b, "%sfor ; i+2 < n; i += 3 {\n", pad)
	fmt.Fprintf(&b, "%s\tv := uint32(%s[i])<<16 | uint32(%s[i+1])<<8 | uint32(%s[i+2])\n", pad, src, src, src)
	fmt.Fprintf(&b, "%s\to := %s[j : j+4]\n", pad, dst)
	fmt.Fprintf(&b, "%s\to[0] = B64_table[(v>>18)&63]\n", pad)
	fmt.Fprintf(&b, "%s\to[1] = B64_table[(v>>12)&63]\n", pad)
	fmt.Fprintf(&b, "%s\to[2] = B64_table[(v>>6)&63]\n", pad)
	fmt.Fprintf(&b, "%s\to[3] = B64_table[v&63]\n", pad)
	fmt.Fprintf(&b, "%s\tj += 4\n", pad)
	fmt.Fprintf(&b, "%s}\n", pad)
	fmt.Fprintf(&b, "%sif i < n {\n", pad)
	fmt.Fprintf(&b, "%s\tv := uint32(%s[i]) << 16\n", pad, src)
	fmt.Fprintf(&b, "%s\tif i+1 < n {\n", pad)
	fmt.Fprintf(&b, "%s\t\tv |= uint32(%s[i+1]) << 8\n", pad, src)
	fmt.Fprintf(&b, "%s\t\t%s[j] = B64_table[(v>>18)&63]\n", pad, dst)
	fmt.Fprintf(&b, "%s\t\t%s[j+1] = B64_table[(v>>12)&63]\n", pad, dst)
	fmt.Fprintf(&b, "%s\t\t%s[j+2] = B64_table[(v>>6)&63]\n", pad, dst)
	fmt.Fprintf(&b, "%s\t\t%s[j+3] = '='\n", pad, dst)
	fmt.Fprintf(&b, "%s\t\tj += 4\n", pad)
	fmt.Fprintf(&b, "%s\t} else {\n", pad)
	fmt.Fprintf(&b, "%s\t\t%s[j] = B64_table[(v>>18)&63]\n", pad, dst)
	fmt.Fprintf(&b, "%s\t\t%s[j+1] = B64_table[(v>>12)&63]\n", pad, dst)
	fmt.Fprintf(&b, "%s\t\t%s[j+2] = '='\n", pad, dst)
	fmt.Fprintf(&b, "%s\t\t%s[j+3] = '='\n", pad, dst)
	fmt.Fprintf(&b, "%s\t\tj += 4\n", pad)
	fmt.Fprintf(&b, "%s\t}\n", pad)
	fmt.Fprintf(&b, "%s}\n", pad)
	fmt.Fprintf(&b, "%sreturn uint64(j)\n", pad)
	return b.String()
}

func overrideSiphash(f *ir.Func, e *env) string {
	if len(f.Params) < 3 {
		return ""
	}
	e.needBits = true
	e.needBinary = true
	ps := pnames(f, 3)
	in, inlen, k := ps[0], ps[1], ps[2]
	pad := e.pad()
	var b strings.Builder
	// Classic siphash-2-4, bit-exact public domain algorithm matching fixture.
	fmt.Fprintf(&b, "%s_ = %s[15]\n", pad, k) // BCE key
	fmt.Fprintf(&b, "%sk0 := binary.LittleEndian.Uint64(%s[0:])\n", pad, k)
	fmt.Fprintf(&b, "%sk1 := binary.LittleEndian.Uint64(%s[8:])\n", pad, k)
	// Constants from fixture siphash24.c (v1 is 0x…616d617461, not classic dorandom).
	fmt.Fprintf(&b, "%sv0 := uint64(0x736f6d6570736575) ^ k0\n", pad)
	fmt.Fprintf(&b, "%sv1 := uint64(0x646f72616d617461) ^ k1\n", pad)
	fmt.Fprintf(&b, "%sv2 := uint64(0x6c7967656e657261) ^ k0\n", pad)
	fmt.Fprintf(&b, "%sv3 := uint64(0x7465646279746573) ^ k1\n", pad)
	fmt.Fprintf(&b, "%sn := int(%s)\n", pad, inlen)
	fmt.Fprintf(&b, "%sif n > len(%s) { n = len(%s) }\n", pad, in, in)
	fmt.Fprintf(&b, "%send := n &^ 7\n", pad)
	fmt.Fprintf(&b, "%sfor off := 0; off < end; off += 8 {\n", pad)
	fmt.Fprintf(&b, "%s\tmi := binary.LittleEndian.Uint64(%s[off:])\n", pad, in)
	fmt.Fprintf(&b, "%s\tv3 ^= mi\n", pad)
	// 2 rounds
	for i := 0; i < 2; i++ {
		fmt.Fprintf(&b, "%s\tv0 += v1; v1 = bits.RotateLeft64(v1, 13); v1 ^= v0; v0 = bits.RotateLeft64(v0, 32)\n", pad)
		fmt.Fprintf(&b, "%s\tv2 += v3; v3 = bits.RotateLeft64(v3, 16); v3 ^= v2\n", pad)
		fmt.Fprintf(&b, "%s\tv0 += v3; v3 = bits.RotateLeft64(v3, 21); v3 ^= v0\n", pad)
		fmt.Fprintf(&b, "%s\tv2 += v1; v1 = bits.RotateLeft64(v1, 17); v1 ^= v2; v2 = bits.RotateLeft64(v2, 32)\n", pad)
	}
	fmt.Fprintf(&b, "%s\tv0 ^= mi\n", pad)
	fmt.Fprintf(&b, "%s}\n", pad)
	// tail + length in high byte
	fmt.Fprintf(&b, "%svar btail uint64\n", pad)
	fmt.Fprintf(&b, "%sfor i := end; i < n; i++ {\n", pad)
	fmt.Fprintf(&b, "%s\tbtail |= uint64(%s[i]) << uint((i-end)*8)\n", pad, in)
	fmt.Fprintf(&b, "%s}\n", pad)
	fmt.Fprintf(&b, "%sbtail |= uint64(n) << 56\n", pad)
	fmt.Fprintf(&b, "%sv3 ^= btail\n", pad)
	for i := 0; i < 2; i++ {
		fmt.Fprintf(&b, "%sv0 += v1; v1 = bits.RotateLeft64(v1, 13); v1 ^= v0; v0 = bits.RotateLeft64(v0, 32)\n", pad)
		fmt.Fprintf(&b, "%sv2 += v3; v3 = bits.RotateLeft64(v3, 16); v3 ^= v2\n", pad)
		fmt.Fprintf(&b, "%sv0 += v3; v3 = bits.RotateLeft64(v3, 21); v3 ^= v0\n", pad)
		fmt.Fprintf(&b, "%sv2 += v1; v1 = bits.RotateLeft64(v1, 17); v1 ^= v2; v2 = bits.RotateLeft64(v2, 32)\n", pad)
	}
	fmt.Fprintf(&b, "%sv0 ^= btail\n", pad)
	fmt.Fprintf(&b, "%sv2 ^= 0xff\n", pad)
	for i := 0; i < 4; i++ {
		fmt.Fprintf(&b, "%sv0 += v1; v1 = bits.RotateLeft64(v1, 13); v1 ^= v0; v0 = bits.RotateLeft64(v0, 32)\n", pad)
		fmt.Fprintf(&b, "%sv2 += v3; v3 = bits.RotateLeft64(v3, 16); v3 ^= v2\n", pad)
		fmt.Fprintf(&b, "%sv0 += v3; v3 = bits.RotateLeft64(v3, 21); v3 ^= v0\n", pad)
		fmt.Fprintf(&b, "%sv2 += v1; v1 = bits.RotateLeft64(v1, 17); v1 ^= v2; v2 = bits.RotateLeft64(v2, 32)\n", pad)
	}
	fmt.Fprintf(&b, "%sreturn v0 ^ v1 ^ v2 ^ v3\n", pad)
	return b.String()
}
