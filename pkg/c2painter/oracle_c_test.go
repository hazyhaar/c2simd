package c2painter

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type primitiveTest struct {
	name   string
	cCode  string
	goCall func(p *Painter)
}

func TestPainterPrimitivesVsCOracle(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not available")
	}

	srcC, err := filepath.Abs(filepath.Join("sources", "c2_painter.c"))
	if err != nil {
		t.Fatal(err)
	}
	srcH, err := filepath.Abs(filepath.Join("sources", "c2_painter.h"))
	if err != nil {
		t.Fatal(err)
	}

	tests := []primitiveTest{
		{
			name: "Clear",
			cCode: `c2_clear(&p, c2_rgba(20, 30, 40, 255));`,
			goCall: func(p *Painter) {
				p.Clear(RGBA(20, 30, 40, 255))
			},
		},
		{
			name: "FillRect_Solid_And_Blend",
			cCode: `
				c2_clear(&p, c2_rgba(10, 10, 10, 255));
				c2_fill_rect(&p, 10, 10, 40, 30, c2_rgba(255, 0, 0, 255));
				c2_fill_rect(&p, 25, 20, 40, 30, c2_rgba(0, 255, 0, 128));
			`,
			goCall: func(p *Painter) {
				p.Clear(RGBA(10, 10, 10, 255))
				p.DrawRect(10, 10, 40, 30, RGBA(255, 0, 0, 255))
				p.DrawRect(25, 20, 40, 30, RGBA(0, 255, 0, 128))
			},
		},
		{
			name: "StrokeRect",
			cCode: `
				c2_clear(&p, c2_rgba(0, 0, 0, 255));
				c2_stroke_rect(&p, 10, 10, 50, 40, 3, c2_rgba(255, 255, 255, 255));
			`,
			goCall: func(p *Painter) {
				p.Clear(RGBA(0, 0, 0, 255))
				p.StrokeRect(10, 10, 50, 40, 3, RGBA(255, 255, 255, 255))
			},
		},
		{
			name: "FillRoundedRect",
			cCode: `
				c2_clear(&p, c2_rgba(0, 0, 0, 255));
				c2_fill_rounded_rect(&p, 20, 20, 60, 50, 12, c2_rgba(0, 128, 255, 220));
			`,
			goCall: func(p *Painter) {
				p.Clear(RGBA(0, 0, 0, 255))
				p.DrawRoundedRect(20, 20, 60, 50, 12, RGBA(0, 128, 255, 220))
			},
		},
		{
			name: "StrokeRoundedRect",
			cCode: `
				c2_clear(&p, c2_rgba(0, 0, 0, 255));
				c2_stroke_rounded_rect(&p, 20, 20, 60, 50, 12, 3, c2_rgba(255, 200, 0, 255));
			`,
			goCall: func(p *Painter) {
				p.Clear(RGBA(0, 0, 0, 255))
				p.StrokeRoundedRect(20, 20, 60, 50, 12, 3, RGBA(255, 200, 0, 255))
			},
		},
		{
			name: "FillCircle",
			cCode: `
				c2_clear(&p, c2_rgba(0, 0, 0, 255));
				c2_fill_circle(&p, 64, 64, 30, c2_rgba(255, 50, 150, 220));
			`,
			goCall: func(p *Painter) {
				p.Clear(RGBA(0, 0, 0, 255))
				p.DrawCircle(64, 64, 30, RGBA(255, 50, 150, 220))
			},
		},
		{
			name: "StrokeCircle",
			cCode: `
				c2_clear(&p, c2_rgba(0, 0, 0, 255));
				c2_stroke_circle(&p, 64, 64, 30, 4, c2_rgba(50, 255, 200, 255));
			`,
			goCall: func(p *Painter) {
				p.Clear(RGBA(0, 0, 0, 255))
				p.StrokeCircle(64, 64, 30, 4, RGBA(50, 255, 200, 255))
			},
		},
		{
			name: "FillEllipse",
			cCode: `
				c2_clear(&p, c2_rgba(0, 0, 0, 255));
				c2_fill_ellipse(&p, 64, 64, 40, 25, c2_rgba(200, 100, 255, 180));
			`,
			goCall: func(p *Painter) {
				p.Clear(RGBA(0, 0, 0, 255))
				p.DrawEllipse(64, 64, 40, 25, RGBA(200, 100, 255, 180))
			},
		},
		{
			name: "StrokeEllipse",
			cCode: `
				c2_clear(&p, c2_rgba(0, 0, 0, 255));
				c2_stroke_ellipse(&p, 64, 64, 40, 25, 3, c2_rgba(255, 150, 50, 255));
			`,
			goCall: func(p *Painter) {
				p.Clear(RGBA(0, 0, 0, 255))
				p.StrokeEllipse(64, 64, 40, 25, 3, RGBA(255, 150, 50, 255))
			},
		},
		{
			name: "DrawLine",
			cCode: `
				c2_clear(&p, c2_rgba(0, 0, 0, 255));
				c2_draw_line(&p, 10, 10, 110, 110, 4, c2_rgba(255, 255, 0, 200));
				c2_draw_line(&p, 110, 10, 10, 110, 2, c2_rgba(0, 255, 255, 255));
			`,
			goCall: func(p *Painter) {
				p.Clear(RGBA(0, 0, 0, 255))
				p.DrawLine(10, 10, 110, 110, 4, RGBA(255, 255, 0, 200))
				p.DrawLine(110, 10, 10, 110, 2, RGBA(0, 255, 255, 255))
			},
		},
		{
			name: "FillLinearGradient",
			cCode: `
				c2_clear(&p, c2_rgba(0, 0, 0, 255));
				c2_fill_linear_gradient(&p, 10, 10, 100, 100, c2_rgba(255, 0, 0, 255), c2_rgba(0, 0, 255, 255), 1);
			`,
			goCall: func(p *Painter) {
				p.Clear(RGBA(0, 0, 0, 255))
				p.DrawLinearGradient(10, 10, 100, 100, RGBA(255, 0, 0, 255), RGBA(0, 0, 255, 255), true)
			},
		},
		{
			name: "BlitMask",
			cCode: `
				c2_clear(&p, c2_rgba(0, 0, 0, 255));
				uint8_t mask[16] = {
					0, 255, 255, 0,
					255, 255, 255, 255,
					255, 255, 255, 255,
					0, 255, 255, 0
				};
				c2_blit_mask(&p, 20, 20, 4, 4, mask, 4, c2_rgba(255, 128, 0, 255));
			`,
			goCall: func(p *Painter) {
				p.Clear(RGBA(0, 0, 0, 255))
				mask := []byte{
					0, 255, 255, 0,
					255, 255, 255, 255,
					255, 255, 255, 255,
					0, 255, 255, 0,
				}
				p.DrawTextGlyph(20, 20, 4, 4, mask, 4, RGBA(255, 128, 0, 255))
			},
		},
		{
			name: "BlitRGBA",
			cCode: `
				c2_clear(&p, c2_rgba(0, 0, 0, 255));
				uint32_t icon[4] = {
					c2_rgba(255, 0, 0, 255), c2_rgba(0, 255, 0, 255),
					c2_rgba(0, 0, 255, 255), c2_rgba(255, 255, 0, 255)
				};
				c2_blit_rgba(&p, 30, 30, 2, 2, icon, 2);
			`,
			goCall: func(p *Painter) {
				p.Clear(RGBA(0, 0, 0, 255))
				icon := []uint32{
					RGBA(255, 0, 0, 255), RGBA(0, 255, 0, 255),
					RGBA(0, 0, 255, 255), RGBA(255, 255, 0, 255),
				}
				p.Blit(30, 30, 2, 2, icon, 2)
			},
		},
		{
			name: "ClippingAndNegativeCoords",
			cCode: `
				c2_clear(&p, c2_rgba(20, 20, 30, 255));
				c2_painter_set_clip(&p, 20, 20, 80, 80);
				c2_fill_rect(&p, -10, -10, 60, 60, c2_rgba(255, 0, 0, 255));
				c2_fill_circle(&p, 100, 100, 40, c2_rgba(0, 255, 0, 255));
				c2_draw_line(&p, -20, 50, 150, 50, 4, c2_rgba(255, 255, 0, 255));
				c2_painter_reset_clip(&p);
			`,
			goCall: func(p *Painter) {
				p.Clear(RGBA(20, 20, 30, 255))
				p.SetClip(20, 20, 80, 80)
				p.DrawRect(-10, -10, 60, 60, RGBA(255, 0, 0, 255))
				p.DrawCircle(100, 100, 40, RGBA(0, 255, 0, 255))
				p.DrawLine(-20, 50, 150, 50, 4, RGBA(255, 255, 0, 255))
				p.ResetClip()
			},
		},
		{
			name: "DrawCursorStyles",
			cCode: `
				c2_clear(&p, c2_rgba(0, 0, 0, 255));
				c2_draw_cursor(&p, 10, 10, 12, 20, 0, c2_rgba(255, 255, 255, 255));
				c2_draw_cursor(&p, 30, 10, 12, 20, 1, c2_rgba(0, 255, 255, 255));
				c2_draw_cursor(&p, 50, 10, 12, 20, 2, c2_rgba(255, 0, 255, 255));
				c2_draw_cursor(&p, 70, 10, 12, 20, 3, c2_rgba(255, 255, 0, 255));
			`,
			goCall: func(p *Painter) {
				p.Clear(RGBA(0, 0, 0, 255))
				p.DrawCursor(10, 10, 12, 20, 0, RGBA(255, 255, 255, 255))
				p.DrawCursor(30, 10, 12, 20, 1, RGBA(0, 255, 255, 255))
				p.DrawCursor(50, 10, 12, 20, 2, RGBA(255, 0, 255, 255))
				p.DrawCursor(70, 10, 12, 20, 3, RGBA(255, 255, 0, 255))
			},
		},
	}

	const w = 128
	const h = 128
	const stride = 128

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			mainC := fmt.Sprintf(`#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include "%s"
#include "%s"

int main(int argc, char **argv) {
    if (argc < 2) return 1;
    const char *out_path = argv[1];

    int w = %d;
    int h = %d;
    int stride = %d;
    uint32_t *pix = (uint32_t *)calloc(w * h, sizeof(uint32_t));
    if (!pix) return 1;

    c2_painter_t p;
    c2_painter_init(&p, pix, w, h, stride);

    %s

    FILE *f = fopen(out_path, "wb");
    if (!f) return 1;
    fwrite(pix, sizeof(uint32_t), w * h, f);
    fclose(f);
    free(pix);
    return 0;
}
`, srcH, srcC, w, h, stride, tc.cCode)

			mainPath := filepath.Join(dir, "oracle_main.c")
			if err := os.WriteFile(mainPath, []byte(mainC), 0o644); err != nil {
				t.Fatal(err)
			}

			bin := filepath.Join(dir, "oracle_bin")
			cmd := exec.Command("gcc", "-std=c99", "-Wall", "-Wextra", "-Werror", "-O2", "-o", bin, mainPath)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("gcc compilation: %v\n%s", err, out)
			}

			cOutPath := filepath.Join(dir, "oracle_pixels.bin")
			if out, err := exec.Command(bin, cOutPath).CombinedOutput(); err != nil {
				t.Fatalf("oracle exec: %v\n%s", err, out)
			}

			cBytes, err := os.ReadFile(cOutPath)
			if err != nil {
				t.Fatal(err)
			}

			cPts := make([]uint32, w*h)
			for i := 0; i < len(cPts); i++ {
				cPts[i] = binary.LittleEndian.Uint32(cBytes[i*4 : (i+1)*4])
			}

			goSurface := NewSurface(w, h)
			p := NewPainter(goSurface)
			tc.goCall(p)

			diffCount := 0
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					idx := y*stride + x
					if p.Pixels()[idx] != cPts[idx] {
						if diffCount < 5 {
							t.Errorf("Différence bit-exacte à (%d, %d): Go=0x%08X vs C=0x%08X", x, y, p.Pixels()[idx], cPts[idx])
						}
						diffCount++
					}
				}
			}

			if diffCount > 0 {
				t.Fatalf("Échec de parité: %d pixels divergents sur %d", diffCount, w*h)
			}
		})
	}
}

func TestPainterCompositeSceneVsCOracle(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not available")
	}

	srcC, err := filepath.Abs(filepath.Join("sources", "c2_painter.c"))
	if err != nil {
		t.Fatal(err)
	}
	srcH, err := filepath.Abs(filepath.Join("sources", "c2_painter.h"))
	if err != nil {
		t.Fatal(err)
	}

	const w = 128
	const h = 128
	const stride = 128

	dir := t.TempDir()
	mainC := fmt.Sprintf(`#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include "%s"
#include "%s"

int main(int argc, char **argv) {
    if (argc < 2) return 1;
    const char *out_path = argv[1];

    int w = %d;
    int h = %d;
    int stride = %d;
    uint32_t *pix = (uint32_t *)calloc(w * h, sizeof(uint32_t));
    if (!pix) return 1;

    c2_painter_t p;
    c2_painter_init(&p, pix, w, h, stride);

    c2_clear(&p, c2_rgba(20, 24, 30, 255));
    c2_fill_rect(&p, 10, 10, 40, 30, c2_rgba(255, 0, 0, 255));
    c2_fill_rect(&p, 25, 20, 40, 30, c2_rgba(0, 255, 0, 128));
    c2_fill_rounded_rect(&p, 60, 10, 50, 40, 8, c2_rgba(0, 128, 255, 200));
    c2_stroke_rounded_rect(&p, 60, 60, 50, 40, 10, 3, c2_rgba(255, 200, 0, 255));
    c2_stroke_rect(&p, 10, 50, 40, 30, 2, c2_rgba(255, 255, 255, 255));
    c2_fill_circle(&p, 30, 100, 15, c2_rgba(255, 50, 150, 220));
    c2_stroke_circle(&p, 70, 100, 14, 2, c2_rgba(50, 255, 200, 255));
    c2_fill_ellipse(&p, 105, 100, 18, 10, c2_rgba(200, 100, 255, 180));
    c2_stroke_ellipse(&p, 105, 50, 15, 8, 2, c2_rgba(255, 150, 50, 255));
    c2_draw_line(&p, 5, 5, 120, 120, 2, c2_rgba(255, 255, 0, 160));
    c2_draw_line(&p, 120, 10, 10, 120, 1, c2_rgba(0, 255, 255, 200));
    c2_fill_linear_gradient(&p, 5, 80, 20, 40, c2_rgba(255, 0, 0, 255), c2_rgba(0, 0, 255, 255), 1);

    uint8_t mask[16] = {
        0, 255, 255, 0,
        255, 255, 255, 255,
        255, 255, 255, 255,
        0, 255, 255, 0
    };
    c2_blit_mask(&p, 50, 45, 4, 4, mask, 4, c2_rgba(255, 255, 255, 255));

    uint32_t icon[4] = {
        c2_rgba(255, 0, 0, 255), c2_rgba(0, 255, 0, 255),
        c2_rgba(0, 0, 255, 255), c2_rgba(255, 255, 0, 255)
    };
    c2_blit_rgba(&p, 80, 80, 2, 2, icon, 2);

    FILE *f = fopen(out_path, "wb");
    if (!f) return 1;
    fwrite(pix, sizeof(uint32_t), w * h, f);
    fclose(f);
    free(pix);
    return 0;
}
`, srcH, srcC, w, h, stride)

	mainPath := filepath.Join(dir, "composite_main.c")
	if err := os.WriteFile(mainPath, []byte(mainC), 0o644); err != nil {
		t.Fatal(err)
	}

	bin := filepath.Join(dir, "composite_bin")
	cmd := exec.Command("gcc", "-std=c99", "-Wall", "-Wextra", "-Werror", "-O2", "-o", bin, mainPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gcc compilation: %v\n%s", err, out)
	}

	cOutPath := filepath.Join(dir, "composite_pixels.bin")
	if out, err := exec.Command(bin, cOutPath).CombinedOutput(); err != nil {
		t.Fatalf("oracle exec: %v\n%s", err, out)
	}

	cBytes, err := os.ReadFile(cOutPath)
	if err != nil {
		t.Fatal(err)
	}

	cPts := make([]uint32, w*h)
	for i := 0; i < len(cPts); i++ {
		cPts[i] = binary.LittleEndian.Uint32(cBytes[i*4 : (i+1)*4])
	}

	goSurface := NewSurface(w, h)
	p := NewPainter(goSurface)

	p.Clear(RGBA(20, 24, 30, 255))
	p.DrawRect(10, 10, 40, 30, RGBA(255, 0, 0, 255))
	p.DrawRect(25, 20, 40, 30, RGBA(0, 255, 0, 128))
	p.DrawRoundedRect(60, 10, 50, 40, 8, RGBA(0, 128, 255, 200))
	p.StrokeRoundedRect(60, 60, 50, 40, 10, 3, RGBA(255, 200, 0, 255))
	p.StrokeRect(10, 50, 40, 30, 2, RGBA(255, 255, 255, 255))
	p.DrawCircle(30, 100, 15, RGBA(255, 50, 150, 220))
	p.StrokeCircle(70, 100, 14, 2, RGBA(50, 255, 200, 255))
	p.DrawEllipse(105, 100, 18, 10, RGBA(200, 100, 255, 180))
	p.StrokeEllipse(105, 50, 15, 8, 2, RGBA(255, 150, 50, 255))
	p.DrawLine(5, 5, 120, 120, 2, RGBA(255, 255, 0, 160))
	p.DrawLine(120, 10, 10, 120, 1, RGBA(0, 255, 255, 200))
	p.DrawLinearGradient(5, 80, 20, 40, RGBA(255, 0, 0, 255), RGBA(0, 0, 255, 255), true)

	mask := []byte{
		0, 255, 255, 0,
		255, 255, 255, 255,
		255, 255, 255, 255,
		0, 255, 255, 0,
	}
	p.DrawTextGlyph(50, 45, 4, 4, mask, 4, RGBA(255, 255, 255, 255))

	icon := []uint32{
		RGBA(255, 0, 0, 255), RGBA(0, 255, 0, 255),
		RGBA(0, 0, 255, 255), RGBA(255, 255, 0, 255),
	}
	p.Blit(80, 80, 2, 2, icon, 2)

	diffCount := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*stride + x
			if p.Pixels()[idx] != cPts[idx] {
				if diffCount < 5 {
					t.Errorf("Différence composite à (%d, %d): Go=0x%08X vs C=0x%08X", x, y, p.Pixels()[idx], cPts[idx])
				}
				diffCount++
			}
		}
	}

	if diffCount > 0 {
		t.Fatalf("Échec composite: %d pixels divergents sur %d", diffCount, w*h)
	}
}
