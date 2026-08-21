package front_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/emit"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/front"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/rules"
)

func TestHarvestStructsMono(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "spec", "c_sources", "upstream", "monocypher", "4.0.2", "monocypher_amalg.c"))
	if err != nil {
		t.Fatal(err)
	}
	res, err := front.ParsePartial(string(src), "mono")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("structs count: %d", len(res.Module.Structs))
	for _, st := range res.Module.Structs {
		t.Logf("struct %s: %+v", st.Name, st.Fields)
	}
	t.Logf("skipped count: %d", len(res.Skipped))
	for _, s := range res.Skipped {
		t.Logf("skipped: %s", s)
	}
	m, err := rules.ApplyAll(res.Module)
	if err != nil {
		t.Fatal(err)
	}
	emit.FillStubs(m)
	for _, fn := range m.Funcs {
		if fn.Name == "fe_sub" {
			for i, st := range fn.Stmts {
				t.Logf("stmt %d kind=%v for_body_len=%d", i, st.Kind, len(st.ForBody))
				for j, s := range st.ForBody {
					t.Logf("  body[%d] kind=%v ins=%+v", j, s.Kind, s.Ins)
				}
			}
		}
	}
}

func TestParseAdd(t *testing.T) {
	path := filepath.Join("..", "testdata", "c", "add.c")
	m, err := front.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Funcs) != 1 || m.Funcs[0].Name != "add" {
		t.Fatalf("funcs: %+v", m.Funcs)
	}
	src, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	if src == "" {
		t.Fatal("empty emit")
	}
}

func TestRejectAsm(t *testing.T) {
	_, err := front.Parse(`int f(void) { asm("nop"); return 0; }`, "x")
	if err == nil {
		t.Fatal("expected error")
	}
	fe, ok := err.(*front.Error)
	if !ok || fe.Code != front.ErrAsm {
		t.Fatalf("got %v", err)
	}
}

func TestMulOk(t *testing.T) {
	m, err := front.Parse(`int mul(int a, int b) { return a * b; }`, "mul")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ins := range m.Funcs[0].Body {
		if ins.Op == ir.OpMul {
			found = true
		}
	}
	if !found {
		t.Fatalf("no mul: %+v", m.Funcs[0].Body)
	}
}

func TestHarvestMurmurFull(t *testing.T) {
	path := filepath.Join("..", "testdata", "c", "murmur3_lab.c")
	if _, err := os.Stat(path); err != nil {
		t.Skip(err)
	}
	res, err := front.ParsePartial(mustRead(t, path), "murmur3_lab")
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, f := range res.Module.Funcs {
		names[f.Name] = true
	}
	if !names["rotl32"] || !names["fmix32"] || !names["murmur3_x86_32"] {
		t.Fatalf("want rotl32+fmix32+murmur3, got %v skipped=%v", names, res.Skipped)
	}
	src, err := emit.Emit(res.Module, emit.ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	if src == "" {
		t.Fatal("empty")
	}
}

// TestIncludeLocal checks S3: a local #include "x.h" is folded inline before
// analysis (its function is harvestable), and a missing local header yields
// err_include instead of being silently dropped.
func TestIncludeLocal(t *testing.T) {
	dir := t.TempDir()
	hdr := filepath.Join(dir, "inc.h")
	if err := os.WriteFile(hdr, []byte("#pragma once\nstatic uint32_t double_it(uint32_t v) { return v + v; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "main.c")
	body := "#include \"inc.h\"\n#include <stdint.h>\nuint32_t run(uint32_t x) { return double_it(x); }\n"
	if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := front.ParseFile(src)
	if err != nil {
		t.Fatalf("local include not folded: %v", err)
	}
	if !funcExists(m, "run") {
		t.Fatalf("run not harvested: %+v", m.Funcs)
	}

	// missing local header -> err_include
	bad := filepath.Join(dir, "bad.c")
	if err := os.WriteFile(bad, []byte("#include \"nope.h\"\nuint32_t f(void) { return 1; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := front.ParseFile(bad); err == nil {
		t.Fatal("expected err_include for missing local header, got nil")
	} else if !strings.Contains(err.Error(), "err_include") {
		t.Fatalf("expected err_include, got: %v", err)
	}
}

func funcExists(m *ir.Module, name string) bool {
	for i := range m.Funcs {
		if m.Funcs[i].Name == name {
			return true
		}
	}
	return false
}

func TestStripPreprocess(t *testing.T) {
	src := "#include <stdint.h>\nstatic uint32_t fmix32(uint32_t h) {\n h ^= h >> 16;\n return h;\n}\n"
	m, err := front.Parse(src, "t")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Funcs) != 1 || m.Funcs[0].Name != "fmix32" {
		t.Fatalf("%+v", m.Funcs)
	}
}

func mustRead(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestMonoAEAD_MultiBlock_1KB(t *testing.T) {
	// front/ → ../../spec/c_sources/... (module c2simd)
	path := filepath.Join("..", "..", "spec", "c_sources", "upstream", "monocypher", "4.0.2", "monocypher_amalg.c")
	if _, err := os.Stat(path); err != nil {
		t.Skip("monocypher_amalg.c missing: ", path)
	}
	m, err := front.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err = rules.ApplyAll(m)
	if err != nil {
		t.Fatal(err)
	}
	emit.FillStubs(m)
	src, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "mono.go"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	testSrc := `package monocypher_amalg

import (
	"bytes"
	"testing"
)

func TestMonoAEAD_MultiBlock_1KB_Generated(t *testing.T) {
	key := make([]byte, 32)
	for i := range key { key[i] = byte(i + 1) }
	nonce := make([]byte, 24)
	for i := range nonce { nonce[i] = byte(i + 10) }
	ad := []byte("HEADER 1KB MULTI-BLOCK AEAD TEST")
	pt := make([]byte, 1024)
	for i := range pt { pt[i] = byte((i * 17 + 3) % 251) }
	ct := make([]byte, len(pt))
	mac := make([]byte, 16)

	Crypto_aead_lock(ct, mac, key, nonce, ad, uint64(len(ad)), pt, uint64(len(pt)))

	ptOut := make([]byte, len(pt))
	res := Crypto_aead_unlock(ptOut, mac, key, nonce, ad, uint64(len(ad)), ct, uint64(len(ct)))

	if res != 0 {
		t.Fatalf("Crypto_aead_unlock returned MAC verification error %d", res)
	}
	if !bytes.Equal(pt, ptOut) {
		t.Fatalf("Decrypted plaintext does not match original for 1KB!")
	}
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "mono_test.go"), []byte(testSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module mono\n\ngo 1.26\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "-v", ".")
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test 1KB failed: %v\nOutput: %s", err, string(out))
	}
}

func TestBug2PArrowMulPlus(t *testing.T) {
	src := `
typedef struct {
    int prm;
} c2_vt_parser_t;

int c2_test_add_digit(c2_vt_parser_t *p, unsigned char b) {
    int v = p->prm * 10 + ((int)b - 48);
    return v;
}
`
	res, err := front.ParsePartial(src, "test_bug2")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skipped) > 0 {
		t.Fatalf("unexpected skipped: %v", res.Skipped)
	}
	if len(res.Module.Funcs) != 1 {
		t.Fatalf("expected 1 func, got %d", len(res.Module.Funcs))
	}
}

func TestProbeCryptoChacha20H(t *testing.T) {
	src := `
typedef unsigned char u8;
typedef unsigned int u32;
static const u8 *chacha20_constant = (const u8*)"expand 32-byte k";
void crypto_wipe(void *secret, unsigned long size);
#define WIPE_BUFFER(buffer) crypto_wipe(buffer, sizeof(buffer))
void load32_le_buf (u32 *dst, const u8 *src, unsigned long size);
void store32_le_buf(u8 *dst, const u32 *src, unsigned long size);
void chacha20_rounds(u32 out[16], const u32 in[16]);

void crypto_chacha20_h(u8 out[32], const u8 key[32], const u8 in [16])
{
	u32 block[16];
	load32_le_buf(block     , chacha20_constant, 4);
	load32_le_buf(block +  4, key              , 8);
	load32_le_buf(block + 12, in               , 4);

	chacha20_rounds(block, block);

	store32_le_buf(out   , block   , 4);
	store32_le_buf(out+16, block+12, 4);
	WIPE_BUFFER(block);
}
`
	res, err := front.ParsePartial(src, "probe_h")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skipped) > 0 {
		t.Fatalf("unexpected skipped: %v", res.Skipped)
	}
}
