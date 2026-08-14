package sgoiter_test

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// upstreamMonocypher returns the vendored monocypher sources, or skips: they are
// gitignored, so a clean checkout has no C oracle to run.
func upstreamMonocypher(t *testing.T) (string, string) {
	t.Helper()
	dir := filepath.Join("..", "spec", "c_sources", "upstream", "monocypher", "4.0.2")
	c, err := filepath.Abs(filepath.Join(dir, "monocypher.c"))
	if err != nil {
		t.Fatal(err)
	}
	h, err := filepath.Abs(filepath.Join(dir, "monocypher.h"))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{c, h} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("monocypher upstream absent (%s): no C oracle to run", p)
		}
	}
	return c, filepath.Dir(h)
}

// The pack is checked against golang.org/x/crypto in TestMonocypherRFCPack. That
// proves the vectors match the RFCs, not that monocypher agrees with them. This
// test runs the C library itself on the algorithms 4.0.2 exposes directly:
// crypto_poly1305 and crypto_blake2b. The IETF AEAD vectors are out of reach —
// crypto_aead_lock takes a 24-byte XChaCha20 nonce, the pack carries the 12-byte
// IETF nonce — and the chacha20 vectors have no standalone entry point.
func TestMonocypherCOracleOnPack(t *testing.T) {
	if _, err := exec.LookPath("cc"); err != nil {
		t.Skip("no C compiler")
	}
	csrc, incDir := upstreamMonocypher(t)

	type vec struct {
		ID     string `json:"id"`
		Alg    string `json:"alg"`
		KeyHex string `json:"key_hex"`
		PTHex  string `json:"pt_hex"`
		MACHex string `json:"mac_hex"`
	}
	f, err := os.Open(filepath.Join("testdata", "monocypher_rfc", "vectors.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var covered []vec
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var v vec
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			t.Fatalf("bad vector line: %v", err)
		}
		if v.Alg == "poly1305" || v.Alg == "blake2b" {
			covered = append(covered, v)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(covered) == 0 {
		t.Fatal("no poly1305 or blake2b vector in the pack")
	}

	dir := t.TempDir()
	var prog strings.Builder
	prog.WriteString(`#include <stdio.h>
#include <string.h>
#include <stdint.h>
#include "monocypher.h"

static void put(const uint8_t *b, size_t n) {
    for (size_t i = 0; i < n; i++) printf("%02x", b[i]);
    printf("\n");
}

int main(void) {
`)
	for _, v := range covered {
		key, err := hex.DecodeString(v.KeyHex)
		if err != nil {
			t.Fatalf("[%s] key_hex: %v", v.ID, err)
		}
		pt, err := hex.DecodeString(v.PTHex)
		if err != nil {
			t.Fatalf("[%s] pt_hex: %v", v.ID, err)
		}
		mac, err := hex.DecodeString(v.MACHex)
		if err != nil {
			t.Fatalf("[%s] mac_hex: %v", v.ID, err)
		}
		switch v.Alg {
		case "poly1305":
			if len(key) != 32 {
				t.Fatalf("[%s] poly1305 key is %d bytes, want 32", v.ID, len(key))
			}
			fmt.Fprintf(&prog, "    { %s %s uint8_t out[16]; crypto_poly1305(out, msg, %d, key); put(out, 16); }\n",
				cArray("key", key), cArray("msg", pt), len(pt))
		case "blake2b":
			if len(key) > 0 {
				fmt.Fprintf(&prog, "    { %s %s uint8_t out[%d]; crypto_blake2b_keyed(out, %d, key, %d, msg, %d); put(out, %d); }\n",
					cArray("key", key), cArray("msg", pt), len(mac), len(mac), len(key), len(pt), len(mac))
				break
			}
			fmt.Fprintf(&prog, "    { %s uint8_t out[%d]; crypto_blake2b(out, %d, msg, %d); put(out, %d); }\n",
				cArray("msg", pt), len(mac), len(mac), len(pt), len(mac))
		}
	}
	prog.WriteString("    return 0;\n}\n")

	src := filepath.Join(dir, "oracle.c")
	if err := os.WriteFile(src, []byte(prog.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "oracle")
	build := exec.Command("cc", "-O2", "-I", incDir, "-o", bin, src, csrc)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("cc: %v\n%s", err, out)
	}
	out, err := exec.Command(bin).Output()
	if err != nil {
		t.Fatalf("oracle run: %v", err)
	}
	got := strings.Fields(strings.TrimSpace(string(out)))
	if len(got) != len(covered) {
		t.Fatalf("oracle printed %d lines for %d vectors", len(got), len(covered))
	}
	confirmed := 0
	for i, v := range covered {
		if got[i] != strings.ToLower(v.MACHex) {
			t.Errorf("[%s] monocypher C = %s, pack says %s", v.ID, got[i], v.MACHex)
			continue
		}
		confirmed++
	}
	t.Logf("monocypher 4.0.2 C confirms %d of %d pack vectors (poly1305, blake2b)", confirmed, len(covered))
}

// cArray renders a byte slice as a C array declaration; an empty slice still needs
// one element to be valid C.
func cArray(name string, b []byte) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "uint8_t %s[] = {", name)
	if len(b) == 0 {
		sb.WriteString("0")
	}
	for i, c := range b {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, "%d", c)
	}
	sb.WriteString("};")
	return sb.String()
}
