package c2fyneterm

import (
	"encoding/base64"
	"io"
	"sync"
	"time"
	"unsafe"

	c2grid "code.hazyhaar.fr/devhoros/c2simd/pkg/c2grid"
	c2vtparser "code.hazyhaar.fr/devhoros/c2simd/pkg/c2vtparser"
	c2painter "code.hazyhaar.fr/devhoros/pkg/c2painter"
	tt55 "code.hazyhaar.fr/devhoros/pkg/tt55"
)

// Style de curseur du terminal.
type CursorStyle uint8

const (
	CursorBlock CursorStyle = iota
	CursorBar
	CursorUnderline
)

// Attributs typographiques et styles étendus pour les cellules.
const (
	AttrBold          uint8 = 1 << 0 // Gras
	AttrDim           uint8 = 1 << 1 // Faible luminosité
	AttrUnderline     uint8 = 1 << 2 // Souligné
	AttrInverse       uint8 = 1 << 3 // Vidéo inverse
	AttrItalic        uint8 = 1 << 4 // Italique
	AttrStrikethrough uint8 = 1 << 5 // Barré
	AttrBlink         uint8 = 1 << 6 // Clignotant
	AttrDEC           uint8 = 1 << 7 // Jeu graphique DEC VT100 actif
)

// Cell représente une cellule de grille terminale alignée sur 8 octets.
type Cell struct {
	Rune  rune  // 4 octets
	Fg    uint8 // 1 octet (index de palette 0..255)
	Bg    uint8 // 1 octet (index de palette 0..255)
	Flags uint8 // 1 octet (attributs de style)
	Width uint8 // 1 octet (largeur de cellule, typiquement 1 ou 2)
}

var (
	_ [8]struct{} = [unsafe.Sizeof(Cell{})]struct{}{}
	_ [8]struct{} = [unsafe.Sizeof(c2vtparser.Cell{})]struct{}{}
	_ [8]struct{} = [unsafe.Sizeof(c2grid.C2_cell_t{})]struct{}{}
)

// Table pré-calculée des 256 couleurs xterm au format RGBA 32-bit Little Endian (thésaurisation archtime).
var XtermPalette [256]uint32

func init() {
	sys := [16][3]uint8{
		{0, 0, 0}, {170, 0, 0}, {0, 170, 0}, {170, 85, 0},
		{0, 0, 170}, {170, 0, 170}, {0, 170, 170}, {170, 170, 170},
		{85, 85, 85}, {255, 85, 85}, {85, 255, 85}, {255, 255, 85},
		{85, 85, 255}, {255, 85, 255}, {85, 255, 255}, {255, 255, 255},
	}
	for i := 0; i < 16; i++ {
		XtermPalette[i] = c2painter.PackRGBA(sys[i][0], sys[i][1], sys[i][2], 255)
	}
	cube := [6]uint8{0, 95, 135, 175, 215, 255}
	for r := 0; r < 6; r++ {
		for g := 0; g < 6; g++ {
			for b := 0; b < 6; b++ {
				idx := 16 + r*36 + g*6 + b
				XtermPalette[idx] = c2painter.PackRGBA(cube[r], cube[g], cube[b], 255)
			}
		}
	}
	for i := 0; i < 24; i++ {
		v := uint8(8 + i*10)
		XtermPalette[232+i] = c2painter.PackRGBA(v, v, v, 255)
	}
}

// Matrices bitmap 8x8 pour les caractères ASCII 32 (' ') à 126 ('~') (thésaurisation archtime).
var font8x8 = [96][8]uint8{
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, // 32 ' '
	{0x18, 0x18, 0x18, 0x18, 0x18, 0x00, 0x18, 0x00}, // 33 '!'
	{0x66, 0x66, 0x24, 0x00, 0x00, 0x00, 0x00, 0x00}, // 34 '"'
	{0x24, 0x7E, 0x24, 0x24, 0x7E, 0x24, 0x00, 0x00}, // 35 '#'
	{0x18, 0x3E, 0x60, 0x3C, 0x06, 0x7C, 0x18, 0x00}, // 36 '$'
	{0x62, 0x64, 0x08, 0x10, 0x20, 0x26, 0x46, 0x00}, // 37 '%'
	{0x38, 0x6C, 0x38, 0x76, 0xDC, 0xCC, 0x76, 0x00}, // 38 '&'
	{0x18, 0x18, 0x30, 0x00, 0x00, 0x00, 0x00, 0x00}, // 39 '\''
	{0x0C, 0x18, 0x30, 0x30, 0x30, 0x18, 0x0C, 0x00}, // 40 '('
	{0x30, 0x18, 0x0C, 0x0C, 0x0C, 0x18, 0x30, 0x00}, // 41 ')'
	{0x00, 0x66, 0x3C, 0xFF, 0x3C, 0x66, 0x00, 0x00}, // 42 '*'
	{0x00, 0x18, 0x18, 0x7E, 0x18, 0x18, 0x00, 0x00}, // 43 '+'
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x18, 0x18, 0x30}, // 44 ','
	{0x00, 0x00, 0x00, 0x7E, 0x00, 0x00, 0x00, 0x00}, // 45 '-'
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x18, 0x18, 0x00}, // 46 '.'
	{0x02, 0x06, 0x0C, 0x18, 0x30, 0x60, 0x40, 0x00}, // 47 '/'
	{0x3C, 0x66, 0x6E, 0x76, 0x66, 0x66, 0x3C, 0x00}, // 48 '0'
	{0x18, 0x38, 0x18, 0x18, 0x18, 0x18, 0x7E, 0x00}, // 49 '1'
	{0x3C, 0x66, 0x06, 0x0C, 0x18, 0x30, 0x7E, 0x00}, // 50 '2'
	{0x3C, 0x66, 0x06, 0x1C, 0x06, 0x66, 0x3C, 0x00}, // 51 '3'
	{0x0C, 0x1C, 0x34, 0x64, 0x7E, 0x04, 0x04, 0x00}, // 52 '4'
	{0x7E, 0x60, 0x7C, 0x06, 0x06, 0x66, 0x3C, 0x00}, // 53 '5'
	{0x1C, 0x30, 0x60, 0x7C, 0x66, 0x66, 0x3C, 0x00}, // 54 '6'
	{0x7E, 0x06, 0x0C, 0x18, 0x30, 0x30, 0x30, 0x00}, // 55 '7'
	{0x3C, 0x66, 0x66, 0x3C, 0x66, 0x66, 0x3C, 0x00}, // 56 '8'
	{0x3C, 0x66, 0x66, 0x3E, 0x06, 0x0C, 0x38, 0x00}, // 57 '9'
	{0x00, 0x18, 0x18, 0x00, 0x18, 0x18, 0x00, 0x00}, // 58 ':'
	{0x00, 0x18, 0x18, 0x00, 0x18, 0x18, 0x30, 0x00}, // 59 ';'
	{0x06, 0x0C, 0x18, 0x30, 0x18, 0x0C, 0x06, 0x00}, // 60 '<'
	{0x00, 0x7E, 0x00, 0x7E, 0x00, 0x00, 0x00, 0x00}, // 61 '='
	{0x60, 0x30, 0x18, 0x0C, 0x18, 0x30, 0x60, 0x00}, // 62 '>'
	{0x3C, 0x66, 0x06, 0x0C, 0x18, 0x00, 0x18, 0x00}, // 63 '?'
	{0x3C, 0x66, 0x6E, 0x6E, 0x60, 0x62, 0x3C, 0x00}, // 64 '@'
	{0x18, 0x3C, 0x66, 0x66, 0x7E, 0x66, 0x66, 0x00}, // 65 'A'
	{0x7C, 0x66, 0x66, 0x7C, 0x66, 0x66, 0x7C, 0x00}, // 66 'B'
	{0x3C, 0x66, 0x60, 0x60, 0x60, 0x66, 0x3C, 0x00}, // 67 'C'
	{0x78, 0x6C, 0x66, 0x66, 0x66, 0x6C, 0x78, 0x00}, // 68 'D'
	{0x7E, 0x60, 0x60, 0x7C, 0x60, 0x60, 0x7E, 0x00}, // 69 'E'
	{0x7E, 0x60, 0x60, 0x7C, 0x60, 0x60, 0x60, 0x00}, // 70 'F'
	{0x3C, 0x66, 0x60, 0x6E, 0x66, 0x66, 0x3C, 0x00}, // 71 'G'
	{0x66, 0x66, 0x66, 0x7E, 0x66, 0x66, 0x66, 0x00}, // 72 'H'
	{0x3C, 0x18, 0x18, 0x18, 0x18, 0x18, 0x3C, 0x00}, // 73 'I'
	{0x0E, 0x06, 0x06, 0x06, 0x66, 0x66, 0x3C, 0x00}, // 74 'J'
	{0x66, 0x6C, 0x78, 0x70, 0x78, 0x6C, 0x66, 0x00}, // 75 'K'
	{0x60, 0x60, 0x60, 0x60, 0x60, 0x60, 0x7E, 0x00}, // 76 'L'
	{0x63, 0x77, 0x7F, 0x6B, 0x63, 0x63, 0x63, 0x00}, // 77 'M'
	{0x66, 0x76, 0x7E, 0x7E, 0x6E, 0x66, 0x66, 0x00}, // 78 'N'
	{0x3C, 0x66, 0x66, 0x66, 0x66, 0x66, 0x3C, 0x00}, // 79 'O'
	{0x7C, 0x66, 0x66, 0x7C, 0x60, 0x60, 0x60, 0x00}, // 80 'P'
	{0x3C, 0x66, 0x66, 0x66, 0x66, 0x3C, 0x0E, 0x00}, // 81 'Q'
	{0x7C, 0x66, 0x66, 0x7C, 0x78, 0x6C, 0x66, 0x00}, // 82 'R'
	{0x3C, 0x66, 0x60, 0x3C, 0x06, 0x66, 0x3C, 0x00}, // 83 'S'
	{0x7E, 0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0x00}, // 84 'T'
	{0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x3C, 0x00}, // 85 'U'
	{0x66, 0x66, 0x66, 0x66, 0x66, 0x3C, 0x18, 0x00}, // 86 'V'
	{0x63, 0x63, 0x63, 0x6B, 0x7F, 0x77, 0x63, 0x00}, // 87 'W'
	{0x66, 0x66, 0x3C, 0x18, 0x3C, 0x66, 0x66, 0x00}, // 88 'X'
	{0x66, 0x66, 0x66, 0x3C, 0x18, 0x18, 0x18, 0x00}, // 89 'Y'
	{0x7E, 0x06, 0x0C, 0x18, 0x30, 0x60, 0x7E, 0x00}, // 90 'Z'
	{0x3C, 0x30, 0x30, 0x30, 0x30, 0x30, 0x3C, 0x00}, // 91 '['
	{0x40, 0x60, 0x30, 0x18, 0x0C, 0x06, 0x02, 0x00}, // 92 '\'
	{0x3C, 0x0C, 0x0C, 0x0C, 0x0C, 0x0C, 0x3C, 0x00}, // 93 ']'
	{0x18, 0x3C, 0x66, 0x00, 0x00, 0x00, 0x00, 0x00}, // 94 '^'
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0x00}, // 95 '_'
	{0x30, 0x18, 0x0C, 0x00, 0x00, 0x00, 0x00, 0x00}, // 96 '`'
	{0x00, 0x00, 0x3C, 0x06, 0x3E, 0x66, 0x3E, 0x00}, // 97 'a'
	{0x60, 0x60, 0x7C, 0x66, 0x66, 0x66, 0x7C, 0x00}, // 98 'b'
	{0x00, 0x00, 0x3C, 0x66, 0x60, 0x66, 0x3C, 0x00}, // 99 'c'
	{0x06, 0x06, 0x3E, 0x66, 0x66, 0x66, 0x3E, 0x00}, // 100 'd'
	{0x00, 0x00, 0x3C, 0x66, 0x7E, 0x60, 0x3C, 0x00}, // 101 'e'
	{0x1C, 0x30, 0x7C, 0x30, 0x30, 0x30, 0x30, 0x00}, // 102 'f'
	{0x00, 0x00, 0x3E, 0x66, 0x66, 0x3E, 0x06, 0x3C}, // 103 'g'
	{0x60, 0x60, 0x7C, 0x66, 0x66, 0x66, 0x66, 0x00}, // 104 'h'
	{0x18, 0x00, 0x38, 0x18, 0x18, 0x18, 0x3C, 0x00}, // 105 'i'
	{0x0C, 0x00, 0x0C, 0x0C, 0x0C, 0x0C, 0x6C, 0x38}, // 106 'j'
	{0x60, 0x60, 0x66, 0x6C, 0x78, 0x6C, 0x66, 0x00}, // 107 'k'
	{0x38, 0x18, 0x18, 0x18, 0x18, 0x18, 0x3C, 0x00}, // 108 'l'
	{0x00, 0x00, 0x66, 0x7F, 0x7F, 0x6B, 0x63, 0x00}, // 109 'm'
	{0x00, 0x00, 0x7C, 0x66, 0x66, 0x66, 0x66, 0x00}, // 110 'n'
	{0x00, 0x00, 0x3C, 0x66, 0x66, 0x66, 0x3C, 0x00}, // 111 'o'
	{0x00, 0x00, 0x7C, 0x66, 0x66, 0x7C, 0x60, 0x60}, // 112 'p'
	{0x00, 0x00, 0x3E, 0x66, 0x66, 0x3E, 0x06, 0x06}, // 113 'q'
	{0x00, 0x00, 0x7C, 0x66, 0x60, 0x60, 0x60, 0x00}, // 114 'r'
	{0x00, 0x00, 0x3E, 0x60, 0x3C, 0x06, 0x7C, 0x00}, // 115 's'
	{0x30, 0x30, 0x7C, 0x30, 0x30, 0x30, 0x1C, 0x00}, // 116 't'
	{0x00, 0x00, 0x66, 0x66, 0x66, 0x66, 0x3E, 0x00}, // 117 'u'
	{0x00, 0x00, 0x66, 0x66, 0x66, 0x3C, 0x18, 0x00}, // 118 'v'
	{0x00, 0x00, 0x63, 0x6B, 0x7F, 0x77, 0x63, 0x00}, // 119 'w'
	{0x00, 0x00, 0x66, 0x3C, 0x18, 0x3C, 0x66, 0x00}, // 120 'x'
	{0x00, 0x00, 0x66, 0x66, 0x66, 0x3E, 0x06, 0x3C}, // 121 'y'
	{0x00, 0x00, 0x7E, 0x0C, 0x18, 0x30, 0x7E, 0x00}, // 122 'z'
	{0x0E, 0x18, 0x18, 0x70, 0x18, 0x18, 0x0E, 0x00}, // 123 '{'
	{0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0x00}, // 124 '|'
	{0x70, 0x18, 0x18, 0x0E, 0x18, 0x18, 0x70, 0x00}, // 125 '}'
	{0x76, 0xDC, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, // 126 '~'
}

// TerminalWidget est le widget terminal matriciel autonome haute performance.
type TerminalWidget struct {
	mu sync.RWMutex

	// Géométrie
	cols int
	rows int

	// Grilles matricielles (tampons principal et alterné)
	primaryGrid   c2vtparser.CursorGrid
	alternateGrid c2vtparser.CursorGrid
	activeGrid    *c2vtparser.CursorGrid
	isAlternate   bool

	// Automate VT500
	parser c2vtparser.Parser

	// Tampon circulaire de défilement (scrollback)
	scrollback   *c2grid.ScrollbackRing
	scrollOffset int // 0 = bas (temps réel), >0 = remontée dans l'historique

	// Curseur
	cursorStyle   CursorStyle
	cursorVisible bool
	blinkEnabled  bool
	blinkVisible  bool
	lastBlink     time.Time
	blinkInterval time.Duration

	// Sélection souris
	selecting    bool
	hasSelection bool
	selStartX    int
	selStartY    int
	selEndX      int
	selEndY      int

	// Presse-papier
	clipboardText   string
	OnClipboardCopy func(text string)
	OnPTYWrite      func(data []byte)

	// Mode DEC VT100
	decGraphicsMode bool

	// Police TrueType tt55
	ttFont     *tt55.Tt55_font
	cellWidth  int
	cellHeight int

	// Couleurs par défaut (format RGBA 32-bit uint32)
	DefaultFg   uint32
	DefaultBg   uint32
	CursorColor uint32
	SelectionBg uint32
	SelectionFg uint32

	// Tampon temporaire de ligne pour archivage scrollback (zéro alloc)
	lineBuf []c2grid.C2_cell_t
}

// NewTerminalWidget initialise une nouvelle instance de TerminalWidget.
func NewTerminalWidget(cols, rows int) *TerminalWidget {
	if cols < 1 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}

	w := &TerminalWidget{
		cols:          cols,
		rows:          rows,
		cursorStyle:   CursorBlock,
		cursorVisible: true,
		blinkEnabled:  true,
		blinkVisible:  true,
		blinkInterval: 500 * time.Millisecond,
		cellWidth:     8,
		cellHeight:    16,
		scrollback:    c2grid.NewScrollbackRingFixed(10000, cols),
		DefaultFg:     c2painter.PackRGBA(204, 204, 204, 255), // #CCCCCC
		DefaultBg:     c2painter.PackRGBA(12, 12, 12, 255),    // #0C0C0C
		CursorColor:   c2painter.PackRGBA(255, 255, 255, 200),
		SelectionBg:   c2painter.PackRGBA(38, 79, 120, 220),   // Bleu sélection
		SelectionFg:   c2painter.PackRGBA(255, 255, 255, 255),
	}

	w.primaryGrid.Reset(cols, rows)
	w.primaryGrid.OnScrollUp = func(line []c2vtparser.Cell) {
		if len(line) > 0 && w.scrollback != nil {
			c2Line := unsafe.Slice((*c2grid.C2_cell_t)(unsafe.Pointer(&line[0])), len(line))
			w.scrollback.Push(c2Line)
		}
	}
	w.alternateGrid.Reset(cols, rows)
	w.activeGrid = &w.primaryGrid
	w.parser.Reset(w.activeGrid)

	return w
}

// Ensure io.Writer interface.
var _ io.Writer = (*TerminalWidget)(nil)

// Write implémente io.Writer pour le streaming direct depuis un PTY Unix sans blocage.
func (t *TerminalWidget) Write(p []byte) (n int, err error) {
	t.Feed(p)
	return len(p), nil
}

// Feed ingère un flux d'octets ANSI/VT500 à zéro allocation mémoire en régime permanent.
func (t *TerminalWidget) Feed(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	var clipCopy string
	t.mu.Lock()

	totalWrites := 0
	n := len(data)
	i := 0

	for i < n {
		// 1. Détection des séquences de contrôle d'état (OSC 52, DEC private modes)
		if data[i] == 0x1b && i+1 < n {
			// Séquence OSC (ex: OSC 52 Presse-papier)
			if data[i+1] == ']' {
				j := i + 2
				for j < n && data[j] != 0x07 && !(data[j] == 0x1b && j+1 < n && data[j+1] == '\\') {
					j++
				}
				if j < n {
					oscPayload := data[i+2 : j]
					copied := t.handleOSC(oscPayload)
					if copied != "" {
						clipCopy = copied
					}
					if data[j] == 0x07 {
						i = j + 1
					} else {
						i = j + 2
					}
					continue
				}
			}

			// Séquence CSI DEC Private Mode (ex: Alternate Screen \x1b[?1049h / \x1b[?1049l)
			if data[i+1] == '[' && i+2 < n && data[i+2] == '?' {
				j := i + 3
				prm := 0
				for j < n && data[j] >= '0' && data[j] <= '9' {
					prm = prm*10 + int(data[j]-'0')
					j++
				}
				if j < n && (data[j] == 'h' || data[j] == 'l') {
					modeSet := data[j] == 'h'
					t.handleDECMode(prm, modeSet)
					i = j + 1
					continue
				}
			}
		}

		// 2. Découpage et délégation au parseur VT500 direct
		nextSpec := n
		for k := i + 1; k < n; k++ {
			if data[k] == 0x1b && k+1 < n {
				if data[k+1] == ']' || (data[k+1] == '[' && k+2 < n && data[k+2] == '?') {
					nextSpec = k
					break
				}
			}
		}

		chunk := data[i:nextSpec]
		if len(chunk) > 0 {
			writes := t.parser.Feed(chunk)
			totalWrites += writes
		}
		i = nextSpec
	}

	cb := t.OnClipboardCopy
	t.mu.Unlock()

	// F12: Appel du rappel presse-papier hors de la section critique
	if clipCopy != "" && cb != nil {
		cb(clipCopy)
	}

	return totalWrites
}

// captureScrollback archive les lignes défilées sans allocation mémoire.
func (t *TerminalWidget) captureScrollback() {
	if t.scrollback == nil || t.cols <= 0 {
		return
	}
	if len(t.lineBuf) < t.cols {
		t.lineBuf = make([]c2grid.C2_cell_t, t.cols)
	}
	cells := t.activeGrid.Cells
	if len(cells) >= t.cols {
		src := unsafe.Slice((*c2grid.C2_cell_t)(unsafe.Pointer(&cells[0])), t.cols)
		copy(t.lineBuf, src)
		t.scrollback.Push(t.lineBuf)
	}
}

// handleDECMode traite les bascules de modes privés DEC (Alternate Screen, Curseur, etc.).
func (t *TerminalWidget) handleDECMode(mode int, enable bool) {
	switch mode {
	case 1049, 1047, 47: // Alternate Screen Buffer
		if enable && !t.isAlternate {
			t.isAlternate = true
			t.alternateGrid.Reset(t.cols, t.rows)
			t.activeGrid = &t.alternateGrid
			t.parser.Reset(t.activeGrid)
			t.scrollOffset = 0
		} else if !enable && t.isAlternate {
			t.isAlternate = false
			t.activeGrid = &t.primaryGrid
			t.parser.Reset(t.activeGrid)
			t.scrollOffset = 0
		}
	case 25: // Visibilité du curseur
		t.cursorVisible = enable
	case 12: // Clignotement du curseur
		t.blinkEnabled = enable
	}
}

// handleOSC traite les séquences de commande du système d'exploitation (OSC 52, OSC 0/2).
func (t *TerminalWidget) handleOSC(payload []byte) string {
	if len(payload) < 3 {
		return ""
	}
	// OSC 52 : Presse-papier (ex: 52;c;<base64>)
	if payload[0] == '5' && payload[1] == '2' && payload[2] == ';' {
		parts := payload[3:]
		if len(parts) >= 2 && (parts[0] == 'c' || parts[0] == 'p') && parts[1] == ';' {
			parts = parts[2:]
		}
		if len(parts) > 0 {
			decoded, err := base64.StdEncoding.DecodeString(string(parts))
			if err == nil {
				t.clipboardText = string(decoded)
				return t.clipboardText
			}
		}
	}
	return ""
}

// putRuneDEC écrit une rune DEC directement dans la grille sans allocation.
func putRuneDEC(g *c2vtparser.CursorGrid, r rune, fg, bg, flags uint8) {
	if g.WrapPending {
		g.WrapPending = false
		g.CursorX = 0
		g.CursorY++
		if g.CursorY > g.Bottom {
			move := (g.Bottom - g.Top) * g.Width
			dst := g.Top * g.Width
			src := (g.Top + 1) * g.Width
			copy(g.Cells[dst:dst+move], g.Cells[src:src+move])
			g.CursorY = g.Bottom
		}
	}
	idx := g.CursorY*g.Width + g.CursorX
	if idx >= 0 && idx < len(g.Cells) {
		g.Cells[idx] = c2vtparser.Cell{Rune: r, Fg: fg, Bg: bg, Flags: flags, Width: 1}
	}
	if g.CursorX+1 >= g.Width {
		g.WrapPending = true
	} else {
		g.CursorX++
	}
}

// Resize redimensionne la géométrie matricielle du terminal.
func (t *TerminalWidget) Resize(newCols, newRows int) {
	if newCols <= 0 || newRows <= 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	if newCols == t.cols && newRows == t.rows {
		return
	}

	t.cols = newCols
	t.rows = newRows
	t.primaryGrid.Reset(newCols, newRows)
	t.primaryGrid.OnScrollUp = func(line []c2vtparser.Cell) {
		if len(line) > 0 && t.scrollback != nil {
			c2Line := unsafe.Slice((*c2grid.C2_cell_t)(unsafe.Pointer(&line[0])), len(line))
			t.scrollback.Push(c2Line)
		}
	}
	t.alternateGrid.Reset(newCols, newRows)
	if t.isAlternate {
		t.activeGrid = &t.alternateGrid
	} else {
		t.activeGrid = &t.primaryGrid
	}
	t.parser.Reset(t.activeGrid)
	t.lineBuf = make([]c2grid.C2_cell_t, newCols)
	t.clearSelectionLocked()
}

// SetFont configure la police TrueType tt55 et calcule les métriques de cellule.
func (t *TerminalWidget) SetFont(font *tt55.Tt55_font, cellW, cellH int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ttFont = font
	if cellW > 0 {
		t.cellWidth = cellW
	}
	if cellH > 0 {
		t.cellHeight = cellH
	}
}

// SetCursorStyle modifie le style visuel du curseur.
func (t *TerminalWidget) SetCursorStyle(style CursorStyle) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cursorStyle = style
}

// SetCursorVisible active ou masque la visibilité du curseur.
func (t *TerminalWidget) SetCursorVisible(visible bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cursorVisible = visible
}

// SetBlink active ou désactive le clignotement du curseur.
func (t *TerminalWidget) SetBlink(enabled bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.blinkEnabled = enabled
}

// ToggleBlink bascule la phase de clignotement pour les animations de trame.
func (t *TerminalWidget) ToggleBlink() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.blinkVisible = !t.blinkVisible
}

// ScrollUp remonte dans l'historique du scrollback.
func (t *TerminalWidget) ScrollUp(lines int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.isAlternate || t.scrollback == nil {
		return
	}
	maxOffset := t.scrollback.Count()
	t.scrollOffset += lines
	if t.scrollOffset > maxOffset {
		t.scrollOffset = maxOffset
	}
}

// ScrollDown redescend vers le bas de l'écran.
func (t *TerminalWidget) ScrollDown(lines int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.isAlternate {
		return
	}
	t.scrollOffset -= lines
	if t.scrollOffset < 0 {
		t.scrollOffset = 0
	}
}

// ScrollToTop remonte au tout début de l'historique.
func (t *TerminalWidget) ScrollToTop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.isAlternate || t.scrollback == nil {
		return
	}
	t.scrollOffset = t.scrollback.Count()
}

// ScrollToBottom revient au temps réel en bas d'écran.
func (t *TerminalWidget) ScrollToBottom() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.scrollOffset = 0
}

// StartSelection démarre une sélection textuelle à la coordonnée (col, row).
func (t *TerminalWidget) StartSelection(col, row int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.selecting = true
	t.hasSelection = true
	t.selStartX, t.selStartY = col, row
	t.selEndX, t.selEndY = col, row
}

// UpdateSelection étend la sélection textuelle en cours.
func (t *TerminalWidget) UpdateSelection(col, row int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.selecting {
		t.selEndX, t.selEndY = col, row
	}
}

// EndSelection finalise la sélection textuelle.
func (t *TerminalWidget) EndSelection() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.selecting = false
	if t.selStartX == t.selEndX && t.selStartY == t.selEndY {
		t.hasSelection = false
	}
}

func (t *TerminalWidget) clearSelectionLocked() {
	t.selecting = false
	t.hasSelection = false
}

// ClearSelection réinitialise la sélection active de manière thread-safe.
func (t *TerminalWidget) ClearSelection() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.clearSelectionLocked()
}

// HasSelection indique si une sélection textuelle non vide existe.
func (t *TerminalWidget) HasSelection() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.hasSelection
}

// isCellSelected vérifie sans allocation si une cellule (col, row) est dans la sélection.
func (t *TerminalWidget) isCellSelected(col, row int) bool {
	if !t.hasSelection {
		return false
	}
	minY, maxY := t.selStartY, t.selEndY
	minX, maxX := t.selStartX, t.selEndX
	if minY > maxY || (minY == maxY && minX > maxX) {
		minY, maxY = t.selEndY, t.selStartY
		minX, maxX = t.selEndX, t.selStartX
	}

	if row < minY || row > maxY {
		return false
	}
	if row > minY && row < maxY {
		return true
	}
	if minY == maxY {
		return col >= minX && col <= maxX
	}
	if row == minY {
		return col >= minX
	}
	if row == maxY {
		return col <= maxX
	}
	return false
}

// GetSelectedText extrait le texte de la sélection active avec découpage des espaces trailing.
func (t *TerminalWidget) GetSelectedText() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.hasSelection || t.cols <= 0 {
		return ""
	}

	minY, maxY := t.selStartY, t.selEndY
	minX, maxX := t.selStartX, t.selEndX
	if minY > maxY || (minY == maxY && minX > maxX) {
		minY, maxY = t.selEndY, t.selStartY
		minX, maxX = t.selEndX, t.selStartX
	}

	var res []rune
	cells := t.activeGrid.Cells

	for y := minY; y <= maxY; y++ {
		if y < 0 || y >= t.rows {
			continue
		}
		startX := 0
		endX := t.cols - 1
		if y == minY {
			startX = minX
		}
		if y == maxY {
			endX = maxX
		}
		if startX < 0 {
			startX = 0
		}
		if endX >= t.cols {
			endX = t.cols - 1
		}

		lineRunes := make([]rune, 0, endX-startX+1)
		for x := startX; x <= endX; x++ {
			idx := y*t.cols + x
			if idx >= 0 && idx < len(cells) {
				r := cells[idx].Rune
				if r == 0 {
					r = ' '
				}
				lineRunes = append(lineRunes, r)
			}
		}

		// Rognage des espaces de fin de ligne
		lastNonSpace := len(lineRunes) - 1
		for lastNonSpace >= 0 && lineRunes[lastNonSpace] == ' ' {
			lastNonSpace--
		}
		if lastNonSpace >= 0 {
			res = append(res, lineRunes[:lastNonSpace+1]...)
		}
		if y < maxY {
			res = append(res, '\n')
		}
	}

	return string(res)
}

// CopySelection copie le texte sélectionné dans le presse-papier interne et déclenche le callback.
func (t *TerminalWidget) CopySelection() string {
	txt := t.GetSelectedText()
	if len(txt) > 0 {
		t.clipboardText = txt
		if t.OnClipboardCopy != nil {
			t.OnClipboardCopy(txt)
		}
	}
	return txt
}

// Paste envoie un texte au processus terminal via le callback PTY.
func (t *TerminalWidget) Paste(text string) {
	if t.OnPTYWrite != nil && len(text) > 0 {
		t.OnPTYWrite([]byte(text))
	}
}

// RenderToSurface effectue le rendu direct sur une Surface c2painter.
func (t *TerminalWidget) RenderToSurface(s *c2painter.Surface) {
	if s == nil || s.Width <= 0 || s.Height <= 0 || len(s.Pixels) == 0 {
		return
	}
	t.RenderToPixels(s.Pixels, s.Width, s.Height, s.Stride)
}

// Render effectue le rendu direct sur un Painter c2painter.
func (t *TerminalWidget) Render(p *c2painter.Painter) {
	if p == nil {
		return
	}
	t.RenderToPixels(p.Pixels(), p.Width(), p.Height(), p.Stride())
}

// RenderToPixels effectue le tracé direct dans un tampon []uint32 linéaire.
// RÈGLE DURE : ZÉRO allocation sur le chemin chaud en régime permanent.
func (t *TerminalWidget) RenderToPixels(pixels []uint32, w, h, stride int) {
	if len(pixels) == 0 || w <= 0 || h <= 0 || stride <= 0 {
		return
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	cols := t.cols
	rows := t.rows
	cellW := t.cellWidth
	cellH := t.cellHeight
	if cols <= 0 || rows <= 0 || cellW <= 0 || cellH <= 0 {
		return
	}

	defBg := t.DefaultBg
	defFg := t.DefaultFg
	cursorCol := t.CursorColor
	selBg := t.SelectionBg
	selFg := t.SelectionFg

	curX := t.activeGrid.CursorX
	curY := t.activeGrid.CursorY
	drawCursor := t.cursorVisible && (!t.blinkEnabled || t.blinkVisible) && t.scrollOffset == 0

	cells := t.activeGrid.Cells
	gridLen := len(cells)

	for row := 0; row < rows; row++ {
		rowY := row * cellH
		if rowY >= h {
			break
		}
		maxH := cellH
		if rowY+maxH > h {
			maxH = h - rowY
		}

		var lineCells []c2vtparser.Cell
		var isScrollbackLine bool

		if t.scrollOffset > 0 && t.scrollback != nil {
			sbCount := t.scrollback.Count()
			sbIdx := sbCount - t.scrollOffset + row
			if sbIdx >= 0 && sbIdx < sbCount {
				sbLine := t.scrollback.Get(sbIdx)
				if len(sbLine) > 0 {
					isScrollbackLine = true
					lineCells = unsafe.Slice((*c2vtparser.Cell)(unsafe.Pointer(&sbLine[0])), len(sbLine))
				}
			}
		}

		if !isScrollbackLine {
			startIdx := row * cols
			if startIdx+cols <= gridLen {
				lineCells = cells[startIdx : startIdx+cols]
			}
		}

		for col := 0; col < cols; col++ {
			colX := col * cellW
			if colX >= w {
				break
			}
			maxW := cellW
			if colX+maxW > w {
				maxW = w - colX
			}

			var c c2vtparser.Cell
			if col < len(lineCells) {
				c = lineCells[col]
			}

			fgCol := defFg
			bgCol := defBg

			if c.Fg > 0 && int(c.Fg) < len(XtermPalette) {
				fgCol = XtermPalette[c.Fg]
			}
			if c.Bg > 0 && int(c.Bg) < len(XtermPalette) {
				bgCol = XtermPalette[c.Bg]
			}

			if (c.Flags & AttrDim) != 0 {
				fgCol = dimColor(fgCol)
			}

			if (c.Flags & AttrInverse) != 0 {
				fgCol, bgCol = bgCol, fgCol
			}

			isSelected := !isScrollbackLine && t.isCellSelected(col, row)
			if isSelected {
				bgCol = selBg
				fgCol = selFg
			}

			// 1. Remplissage du fond
			for py := 0; py < maxH; py++ {
				dstOff := (rowY+py)*stride + colX
				for px := 0; px < maxW; px++ {
					pixels[dstOff+px] = bgCol
				}
			}

			// 2. Tracé du glyphe
			r := c.Rune
			flags := c.Flags

			if r != 0 && r != ' ' {
				if isBoxDrawingRune(r) {
					drawBoxGlyphPixels(pixels, colX, rowY, maxW, maxH, stride, r, fgCol, flags)
				} else if r >= 32 && r <= 126 {
					drawAsciiGlyphPixels(pixels, colX, rowY, maxW, maxH, stride, uint8(r), fgCol, flags)
				} else {
					drawGenericRunePixels(pixels, colX, rowY, maxW, maxH, stride, r, fgCol, flags)
				}
			}

			// 3. Tracé des styles de souligné et barré
			if (flags & AttrUnderline) != 0 {
				underY := rowY + maxH - 2
				if underY >= 0 && underY < h {
					dstOff := underY*stride + colX
					for px := 0; px < maxW; px++ {
						pixels[dstOff+px] = fgCol
					}
				}
			}

			if (flags & AttrStrikethrough) != 0 {
				strikeY := rowY + maxH/2
				if strikeY >= 0 && strikeY < h {
					dstOff := strikeY*stride + colX
					for px := 0; px < maxW; px++ {
						pixels[dstOff+px] = fgCol
					}
				}
			}

			// 4. Tracé du curseur clignotant
			if drawCursor && col == curX && row == curY {
				t.drawCursorPixels(pixels, colX, rowY, maxW, maxH, stride, cursorCol)
			}
		}
	}
}

// RenderToBuffer effectue le tracé direct dans un tampon []byte linéaire (stride en octets).
func (t *TerminalWidget) RenderToBuffer(pix []byte, w, h, byteStride int) {
	if len(pix) == 0 || w <= 0 || h <= 0 || byteStride < w*4 {
		return
	}
	uint32Slice := unsafe.Slice((*uint32)(unsafe.Pointer(&pix[0])), len(pix)/4)
	t.RenderToPixels(uint32Slice, w, h, byteStride/4)
}

func dimColor(c uint32) uint32 {
	r := (c & 0xFF) >> 1
	g := ((c >> 8) & 0xFF) >> 1
	b := ((c >> 16) & 0xFF) >> 1
	a := (c >> 24) & 0xFF
	return r | (g << 8) | (b << 16) | (a << 24)
}

// drawCursorPixels trace le curseur selon son style géométrique dans un tampon pixels uint32.
// F5: Rendu idempotent et constant éliminant l'inversion XOR non reproductible sous concurrence.
func (t *TerminalWidget) drawCursorPixels(pixels []uint32, x, y, w, h, stride int, col uint32) {
	if col == 0 {
		col = 0xFFCCCCCC
	}
	switch t.cursorStyle {
	case CursorBlock:
		for cy := 0; cy < h; cy++ {
			off := (y+cy)*stride + x
			for cx := 0; cx < w; cx++ {
				pixels[off+cx] = col
			}
		}
	case CursorBar:
		barW := 2
		if barW > w {
			barW = w
		}
		for cy := 0; cy < h; cy++ {
			off := (y+cy)*stride + x
			for cx := 0; cx < barW; cx++ {
				pixels[off+cx] = col
			}
		}
	case CursorUnderline:
		underH := 2
		if underH > h {
			underH = h
		}
		startY := y + h - underH
		for cy := 0; cy < underH; cy++ {
			off := (startY+cy)*stride + x
			for cx := 0; cx < w; cx++ {
				pixels[off+cx] = col
			}
		}
	}
}

// isBoxDrawingRune identifie les caractères graphiques vectoriels de boîte DEC / Unicode.
func isBoxDrawingRune(r rune) bool {
	switch r {
	case '─', '━', '═', '│', '┃', '║',
		'┌', '┏', '╔', '┐', '┓', '╗',
		'└', '┗', '╚', '┘', '┛', '╝',
		'├', '┣', '╠', '┤', '┫', '╣',
		'┬', '┳', '╦', '┴', '┻', '╩',
		'┼', '╋', '╬',
		'█', '▀', '▄', '▌', '▐', '░', '▒', '▓', '■', '◆', '·', '°', '±':
		return true
	default:
		return false
	}
}

// drawBoxGlyphPixels effectue le tracé vectoriel direct des lignes et boîtes dans pixels uint32.
func drawBoxGlyphPixels(pixels []uint32, x, y, w, h, stride int, r rune, col uint32, flags uint8) {
	midX := x + w/2
	midY := y + h/2

	switch r {
	case '─', '━', '═': // Ligne horizontale
		for px := 0; px < w; px++ {
			pixels[midY*stride+(x+px)] = col
		}
	case '│', '┃', '║': // Ligne verticale
		for py := 0; py < h; py++ {
			pixels[(y+py)*stride+midX] = col
		}
	case '┌', '┏', '╔': // Coin haut-gauche
		for px := midX; px < x+w; px++ {
			pixels[midY*stride+px] = col
		}
		for py := midY; py < y+h; py++ {
			pixels[py*stride+midX] = col
		}
	case '┐', '┓', '╗': // Coin haut-droit
		for px := x; px <= midX; px++ {
			pixels[midY*stride+px] = col
		}
		for py := midY; py < y+h; py++ {
			pixels[py*stride+midX] = col
		}
	case '└', '┗', '╚': // Coin bas-gauche
		for px := midX; px < x+w; px++ {
			pixels[midY*stride+px] = col
		}
		for py := y; py <= midY; py++ {
			pixels[py*stride+midX] = col
		}
	case '┘', '┛', '╝': // Coin bas-droit
		for px := x; px <= midX; px++ {
			pixels[midY*stride+px] = col
		}
		for py := y; py <= midY; py++ {
			pixels[py*stride+midX] = col
		}
	case '├', '┣', '╠': // Té gauche
		for py := y; py < y+h; py++ {
			pixels[py*stride+midX] = col
		}
		for px := midX; px < x+w; px++ {
			pixels[midY*stride+px] = col
		}
	case '┤', '┫', '╣': // Té droit
		for py := y; py < y+h; py++ {
			pixels[py*stride+midX] = col
		}
		for px := x; px <= midX; px++ {
			pixels[midY*stride+px] = col
		}
	case '┬', '┳', '╦': // Té supérieur
		for px := x; px < x+w; px++ {
			pixels[midY*stride+px] = col
		}
		for py := midY; py < y+h; py++ {
			pixels[py*stride+midX] = col
		}
	case '┴', '┻', '╩': // Té inférieur
		for px := x; px < x+w; px++ {
			pixels[midY*stride+px] = col
		}
		for py := y; py <= midY; py++ {
			pixels[py*stride+midX] = col
		}
	case '┼', '╋', '╬': // Croisement
		for px := x; px < x+w; px++ {
			pixels[midY*stride+px] = col
		}
		for py := y; py < y+h; py++ {
			pixels[py*stride+midX] = col
		}
	case '█': // Bloc plein
		for py := 0; py < h; py++ {
			off := (y+py)*stride + x
			for px := 0; px < w; px++ {
				pixels[off+px] = col
			}
		}
	case '▀': // Demi-bloc supérieur
		for py := 0; py < h/2; py++ {
			off := (y+py)*stride + x
			for px := 0; px < w; px++ {
				pixels[off+px] = col
			}
		}
	case '▄': // Demi-bloc inférieur
		for py := h / 2; py < h; py++ {
			off := (y+py)*stride + x
			for px := 0; px < w; px++ {
				pixels[off+px] = col
			}
		}
	case '▌': // Demi-bloc gauche
		for py := 0; py < h; py++ {
			off := (y+py)*stride + x
			for px := 0; px < w/2; px++ {
				pixels[off+px] = col
			}
		}
	case '▐': // Demi-bloc droit
		for py := 0; py < h; py++ {
			off := (y+py)*stride + x
			for px := w / 2; px < w; px++ {
				pixels[off+px] = col
			}
		}
	case '▒', '░', '▓': // Damier
		for py := 0; py < h; py++ {
			off := (y+py)*stride + x
			for px := 0; px < w; px++ {
				if (px+py)%2 == 0 {
					pixels[off+px] = col
				}
			}
		}
	default:
		drawAsciiGlyphPixels(pixels, x, y, w, h, stride, '?', col, flags)
	}
}

// drawAsciiGlyphPixels rasterise un caractère ASCII avec styles (gras, italique) dans pixels uint32.
func drawAsciiGlyphPixels(pixels []uint32, x, y, w, h, stride int, ch uint8, col uint32, flags uint8) {
	if ch < 32 || ch > 126 {
		ch = '?'
	}
	bitmap := font8x8[ch-32]
	isBold := (flags & AttrBold) != 0
	isItalic := (flags & AttrItalic) != 0

	scaleY := h / 8
	if scaleY < 1 {
		scaleY = 1
	}

	for row := 0; row < 8; row++ {
		bits := bitmap[row]
		if bits == 0 {
			continue
		}

		italicOffset := 0
		if isItalic {
			italicOffset = (7 - row) / 4
		}

		for sy := 0; sy < scaleY; sy++ {
			curY := y + row*scaleY + sy
			if curY >= y+h {
				break
			}
			rowOff := curY * stride

			for colIdx := 0; colIdx < 8; colIdx++ {
				if (bits & (1 << (7 - colIdx))) != 0 {
					curX := x + colIdx + italicOffset
					if curX < x+w {
						pixels[rowOff+curX] = col
					}
					if isBold && (curX+1 < x+w) {
						pixels[rowOff+curX+1] = col
					}
				}
			}
		}
	}
}

// drawGenericRunePixels dessine un point de code générique Unicode UTF-8 dans pixels uint32.
func drawGenericRunePixels(pixels []uint32, x, y, w, h, stride int, r rune, col uint32, flags uint8) {
	if r >= 32 && r <= 126 {
		drawAsciiGlyphPixels(pixels, x, y, w, h, stride, uint8(r), col, flags)
		return
	}
	drawAsciiGlyphPixels(pixels, x, y, w, h, stride, '?', col, flags)
}

// Cols retourne le nombre de colonnes de la grille.
func (t *TerminalWidget) Cols() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.cols
}

// Rows retourne le nombre de lignes de la grille.
func (t *TerminalWidget) Rows() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.rows
}

// CursorPos retourne la position courante (col, row) du curseur.
func (t *TerminalWidget) CursorPos() (int, int) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.activeGrid.CursorX, t.activeGrid.CursorY
}

// IsAlternateScreen indique si le tampon alterné (alternate screen) est actif.
func (t *TerminalWidget) IsAlternateScreen() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.isAlternate
}

// ScrollOffset retourne l'offset de défilement actif dans l'historique.
func (t *TerminalWidget) ScrollOffset() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.scrollOffset
}

// ScrollbackCount retourne le nombre total de lignes archivées dans le scrollback ring.
func (t *TerminalWidget) ScrollbackCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.scrollback == nil {
		return 0
	}
	return t.scrollback.Count()
}

// ClipboardText retourne le contenu textuel actuel du presse-papier interne.
func (t *TerminalWidget) ClipboardText() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.clipboardText
}

// CellSize retourne la largeur et la hauteur en pixels d'une cellule.
func (t *TerminalWidget) CellSize() (int, int) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.cellWidth, t.cellHeight
}

// Font retourne la police TrueType tt55 configurée.
func (t *TerminalWidget) Font() *tt55.Tt55_font {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.ttFont
}

// RawGrid retourne un pointeur direct vers la grille active sous-jacente.
func (t *TerminalWidget) RawGrid() *c2vtparser.CursorGrid {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.activeGrid
}

// SetSelection définit directement les coordonnées d'une sélection.
func (t *TerminalWidget) SetSelection(startX, startY, endX, endY int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.hasSelection = true
	t.selecting = false
	t.selStartX, t.selStartY = startX, startY
	t.selEndX, t.selEndY = endX, endY
}

// CellAt extrait la cellule (col, row) de la grille active avec synchronisation.
func (t *TerminalWidget) CellAt(col, row int) Cell {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if col < 0 || col >= t.cols || row < 0 || row >= t.rows {
		return Cell{}
	}
	idx := row*t.cols + col
	cells := t.activeGrid.Cells
	if idx >= 0 && idx < len(cells) {
		c := cells[idx]
		return Cell{
			Rune:  c.Rune,
			Fg:    c.Fg,
			Bg:    c.Bg,
			Flags: c.Flags,
			Width: c.Width,
		}
	}
	return Cell{}
}

