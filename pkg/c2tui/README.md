# c2tui

Paquet public de la pile terminale du pôle **c2simd** (transpileur C → Go 1.27). Deux verbes : `Parser.Feed`, `DiffGrid`. Un `Cell` de 8 octets (`Rune, Fg, Bg, Flags, Width`).

`Feed` est l’automate Go (UTF-8, wrap, scroll par `copy`). Le C harvesté n’est pas le parseur. `DiffGrid` est le noyau C transpilé ; AVX2/NEON seulement si le binaire est construit avec `GOEXPERIMENT=simd`. `go test` sans cette expérience reste vert (scalaire).

Les symboles `C2_*` vivent sous `internal/` : non importables hors module. Myers n’est pas dans ce paquet. Aucune revendication « compatible VTE ».

Amont : `sources/c2_cell.h`, `c2_grid.h`, `c2tuidiff.c`, `c2vtparser.c`. Réémettre avec `bin/sgoiter` vers `internal/`. Ne pas éditer les `*_gen.go`.
