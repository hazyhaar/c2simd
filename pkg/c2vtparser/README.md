# c2vtparser

Parseur ANSI in-place pour une grille de cellules 8 octets (`Rune, Fg, Bg, Flags, Width`). Consommateur : hazhar TUI (`AgentTab` PTY).

## API

- `Parser.Feed([]byte) int` — automate Go (UTF-8, wrap, scroll, CSI étendu, OSC 52 neutralisé). C’est le chemin de production.
- `CursorGrid.DiffCells() []c2tuidiff.Cell` — vue zéro-copie vers le moteur de diff.
- Les symboles `C2_vt_*` sont un **sous-ensemble C** transpilé (ASCII, SGR 0/1/4/7/30–37/40–47, CUP, CSI J/K). Ils ne remplacent pas `Feed`.

## Garanties

- OSC 52 / titres : consommés, jamais exécutés.
- `TestChunkedEquivalence` : coupure de flux vs monobloc.
- `TestFeedZeroAlloc` : 0 alloc si la grille est déjà dimensionnée.
- Le débit « 1 Go/s » n’est pas un contrat CI ; voir `BenchmarkFeed*`.

Toolchain : Go 1.27.
