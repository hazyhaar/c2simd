package tt55

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestVsCOracle_SystemFonts(t *testing.T) {
	// Compilation de l'oracle GCC C99
	cOracleSrc := filepath.Join("sources", "tt.c")
	mainSrc := filepath.Join("sources", "oracle_main.c")
	oracleBin := filepath.Join("sources", "tt_oracle_bin")

	buildCmd := exec.Command("gcc", "-O2", "-Wall", "-Wextra", "-Werror", "-Isources", cOracleSrc, mainSrc, "-o", oracleBin)
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Échec compilation oracle C: %v\nOutput:\n%s", err, string(out))
	}
	defer os.Remove(oracleBin)

	fontPaths, err := filepath.Glob("/usr/share/fonts/truetype/*/*.ttf")
	if err != nil || len(fontPaths) == 0 {
		t.Skip("Aucune police système TTF trouvée pour le test d'oracle")
	}

	testCPs := []uint32{
		32, 65, 97, 126, 160, 233, 246, 0x03B1, 0x03C9, 0x0410, 0x044F,
		0x20AC, 0x221E, 0x4E00, 0x9FA5, 0x1F600, 0x1F680, 0xFFFF, 0x10000,
	}

	for _, fontPath := range fontPaths {
		t.Run(filepath.Base(fontPath), func(t *testing.T) {
			data, err := os.ReadFile(fontPath)
			if err != nil {
				t.Fatalf("Lecture police impossible: %v", err)
			}

			// 1. Exécution de l'oracle C
			cCmd := exec.Command(oracleBin, fontPath)
			cOut, err := cCmd.Output()
			if err != nil {
				t.Fatalf("Exécution oracle C en erreur: %v", err)
			}

			cLines := strings.Split(strings.TrimSpace(string(cOut)), "\n")
			if len(cLines) == 0 {
				t.Fatalf("Sortie oracle C vide")
			}

			// 2. Exécution du code transpilé Go
			var font Tt55_font
			goErr := Tt55_open(data, uint64(len(data)), &font)

			// Parsing ligne OPEN oracle C
			openLine := cLines[0]
			var cOpenErr int
			var cUnits, cNumG, cNumH uint32
			var cCmapOff, cCmapLen, cCmapSub, cHmtxOff, cHmtxLen uint32
			_, err = fmt.Sscanf(openLine, "OPEN:%d units_per_em:%d num_glyphs:%d num_hmetrics:%d cmap_off:%d cmap_len:%d cmap_sub:%d hmtx_off:%d hmtx_len:%d",
				&cOpenErr, &cUnits, &cNumG, &cNumH, &cCmapOff, &cCmapLen, &cCmapSub, &cHmtxOff, &cHmtxLen)
			if err != nil {
				t.Fatalf("Parse OPEN oracle C échoué: %v\nLine: %s", err, openLine)
			}

			if goErr != cOpenErr {
				t.Fatalf("Mismatch OPEN err: Go=%d C=%d", goErr, cOpenErr)
			}
			if goErr != 0 {
				return // Les deux ont échoué identiquement
			}

			if font.Units_per_em != uint16(cUnits) {
				t.Fatalf("Mismatch Units_per_em: Go=%d C=%d", font.Units_per_em, cUnits)
			}
			if font.Num_glyphs != uint16(cNumG) {
				t.Fatalf("Mismatch Num_glyphs: Go=%d C=%d", font.Num_glyphs, cNumG)
			}
			if font.Number_of_h_metrics != uint16(cNumH) {
				t.Fatalf("Mismatch Number_of_h_metrics: Go=%d C=%d", font.Number_of_h_metrics, cNumH)
			}
			if font.Cmap_sub != cCmapSub {
				t.Fatalf("Mismatch Cmap_sub: Go=%d C=%d", font.Cmap_sub, cCmapSub)
			}
			if font.Hmtx_off != cHmtxOff {
				t.Fatalf("Mismatch Hmtx_off: Go=%d C=%d", font.Hmtx_off, cHmtxOff)
			}

			// 3. Comparaison Glyph & Advance sur chaque point de code
			cMap := make(map[uint32][4]int) // cp -> [gid, gerr, aw, aerr]
			for _, line := range cLines[1:] {
				if !strings.HasPrefix(line, "CP:") {
					continue
				}
				var cp, gid, aw uint32
				var gerr, aerr int
				_, err := fmt.Sscanf(line, "CP:%d GID:%d GERR:%d AW:%d AERR:%d", &cp, &gid, &gerr, &aw, &aerr)
				if err == nil {
					cMap[cp] = [4]int{int(gid), gerr, int(aw), aerr}
				}
			}

			for _, cp := range testCPs {
				cRes, ok := cMap[cp]
				if !ok {
					continue
				}

				var goGid uint16
				goGErr := Tt55_glyph(&font, cp, &goGid)
				var goAw uint16
				goAErr := -999
				if goGErr == 0 {
					goAErr = Tt55_advance(&font, goGid, &goAw)
				}

				if goGErr != cRes[1] || int(goGid) != cRes[0] {
					t.Fatalf("CP %d Glyph mismatch: Go=(gid=%d, err=%d) C=(gid=%d, err=%d)",
						cp, goGid, goGErr, cRes[0], cRes[1])
				}
				if goAErr != cRes[3] || int(goAw) != cRes[2] {
					t.Fatalf("CP %d Advance mismatch: Go=(aw=%d, err=%d) C=(aw=%d, err=%d)",
						cp, goAw, goAErr, cRes[2], cRes[3])
				}
			}
		})
	}
}

func TestTT55_ErrorHandling(t *testing.T) {
	var font Tt55_font

	// Paramètres nuls
	if err := Tt55_open(nil, 0, &font); err != -1 {
		t.Fatalf("Tt55_open(nil) got %d, want -1", err)
	}
	if err := Tt55_open([]byte{1, 2, 3}, 3, &font); err != -1 {
		t.Fatalf("Tt55_open(court) got %d, want -1", err)
	}

	// Buffer corrompu
	buf := make([]byte, 100)
	if err := Tt55_open(buf, uint64(len(buf)), &font); err != -2 {
		t.Fatalf("Tt55_open(bad scaler) got %d, want -2", err)
	}

	var gid, aw uint16
	if err := Tt55_glyph(nil, 65, &gid); err != -1 {
		t.Fatalf("Tt55_glyph(nil font) got %d, want -1", err)
	}
	if err := Tt55_advance(nil, 1, &aw); err != -1 {
		t.Fatalf("Tt55_advance(nil font) got %d, want -1", err)
	}
}

func BenchmarkTT55_GlyphAndAdvance(b *testing.B) {
	data, err := os.ReadFile("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf")
	if err != nil {
		b.Skip("DejaVuSans.ttf non disponible")
	}

	var font Tt55_font
	if err := Tt55_open(data, uint64(len(data)), &font); err != 0 {
		b.Fatalf("Tt55_open a échoué: %d", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	var gid, aw uint16
	for i := 0; i < b.N; i++ {
		cp := uint32(32 + (i % 1000))
		_ = Tt55_glyph(&font, cp, &gid)
		_ = Tt55_advance(&font, gid, &aw)
	}
}
