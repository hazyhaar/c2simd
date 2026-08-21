# c2tuidiff

Compare deux grilles de cellules terminales et produit les segments horizontaux modifiés (`Span`). Consommateur : hazhar TUI (`tui55.Renderer`).

## API

`DiffGrid(front, back []Cell, width, height, stride int, spans *[]Span) int` — nombre de **cellules** changées. `stride` peut dépasser `width` (padding SIMD).

`Cell` fait 8 octets : `Rune, Fg, Bg, Flags, Width`. Les symboles `C2_*` sont le noyau C transpilé par sgoiter ; l’entrée publique est `DiffGrid`.

## Garanties

- Scalaire = `C2_diff_grid_scalar` (sgoiter, `sources/c2tuidiff.c`). SIMD AVX2/NEON si `GOEXPERIMENT=simd`.
- KAT bit-exact scalaire / SIMD, y compris `stride > width`.
- Oracle gcc -O2 (`TestC2DiffGridVsGCC`, `TestC2DiffGridVsGCCStride`).
- Zéro allocation **si** `cap(*spans)` suffit ; sinon une réallocation. Overflow C (`-1`) : un second essai avec tampon plus grand.

Toolchain : Go 1.27 (`GOTOOLCHAIN=go1.27rc3`).
