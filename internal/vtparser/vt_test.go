package c2vtparser

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"unsafe"

	"code.hazyhaar.fr/devhoros/c2simd/internal/tuidiff"
)

// helper de construction : une grille 10x4 et un parseur prêt.
func newFixture() (*CursorGrid, *Parser) {
	g := &CursorGrid{}
	g.Reset(10, 4)
	p := &Parser{}
	p.Reset(g)
	return g, p
}

func cellAt(g *CursorGrid, x, y int) Cell {
	return g.Cells[y*g.Width+x]
}

func TestFeedThenDiff(t *testing.T) {
	g, p := newFixture()
	p.Feed([]byte("HELLO"))
	front := make([]c2tuidiff.Cell, len(g.Cells))
	back := g.DiffCells()
	copy(front, back)
	p.Feed([]byte("\rXXXXX"))
	spans := make([]c2tuidiff.Span, 0, 16)
	n := c2tuidiff.DiffGrid(front, g.DiffCells(), g.Width, g.Height, g.Width, &spans)
	if n < 4 {
		t.Fatalf("expected dirty cells, got %d spans=%v", n, spans)
	}
}

func TestMemoryLayout(t *testing.T) {
	if unsafe.Sizeof(Cell{}) != 8 || unsafe.Sizeof(C2_cell_t{}) != 8 {
		t.Fatalf("Sizeof Cell=%d C2=%d", unsafe.Sizeof(Cell{}), unsafe.Sizeof(C2_cell_t{}))
	}
	if unsafe.Offsetof(Cell{}.Width) != 7 {
		t.Fatalf("Width offset %d", unsafe.Offsetof(Cell{}.Width))
	}
}

func TestChunkedEquivalence(t *testing.T) {
	seq := []byte("AB\x1b[31mC\x1b[0m\x1b[2;2HX")
	run := func(chunks ...[]byte) *CursorGrid {
		g := &CursorGrid{}
		g.Reset(10, 4)
		p := &Parser{}
		p.Reset(g)
		for _, c := range chunks {
			p.Feed(c)
		}
		return g
	}
	mono := run(seq)
	cuts := []int{1, len(seq) / 2, len(seq) - 1}
	for _, cut := range cuts {
		if cut < 1 || cut >= len(seq) {
			continue
		}
		got := run(seq[:cut], seq[cut:])
		if !bytes.Equal(gridBytes(mono), gridBytes(got)) {
			t.Fatalf("cut %d: grid mismatch cursor mono=(%d,%d) got=(%d,%d)", cut, mono.CursorX, mono.CursorY, got.CursorX, got.CursorY)
		}
	}
}

func gridBytes(g *CursorGrid) []byte {
	if len(g.Cells) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&g.Cells[0])), len(g.Cells)*8)
}

// TestBasicText vérifie l'écriture de texte simple et l'avancée du curseur.
func TestBasicText(t *testing.T) {
	g, p := newFixture()
	n := p.Feed([]byte("Hello"))
	if n != 5 {
		t.Fatalf("writes=%d want 5", n)
	}
	if g.CursorX != 5 || g.CursorY != 0 {
		t.Fatalf("cursor=(%d,%d) want (5,0)", g.CursorX, g.CursorY)
	}
	if cellAt(g, 0, 0).Rune != 'H' || cellAt(g, 4, 0).Rune != 'o' {
		t.Fatalf("cells wrong: %+v", g.Cells[:5])
	}
	if g.Writes != 5 {
		t.Fatalf("g.Writes=%d want 5", g.Writes)
	}
}

// TestColors256 vérifie la sélection de couleur 256 (38;5;N / 48;5;N).
func TestColors256(t *testing.T) {
	g, p := newFixture()
	p.Feed([]byte("\x1b[38;5;196mX"))
	c := cellAt(g, 0, 0)
	if c.Fg != 196 {
		t.Fatalf("Fg=%d want 196", c.Fg)
	}
	p.Feed([]byte("\x1b[48;5;21mY"))
	c = cellAt(g, 1, 0)
	if c.Bg != 21 {
		t.Fatalf("Bg=%d want 21", c.Bg)
	}
	// Truecolor 38;2;r;g;b projeté sur la palette 256.
	p.Feed([]byte("\x1b[38;2;255;0;0mZ"))
	c = cellAt(g, 2, 0)
	if c.Fg != 9 { // rouge vif {255,0,0} = index 9 (distance nulle, premier trouvé)
		t.Fatalf("truecolor Fg=%d want 9", c.Fg)
	}
	// Remise à zéro.
	p.Feed([]byte("\x1b[0mQ"))
	c = cellAt(g, 3, 0)
	if c.Fg != 0 || c.Bg != 0 {
		t.Fatalf("reset: Fg=%d Bg=%d want 0/0", c.Fg, c.Bg)
	}
}

// TestStyles vérifie les attributs gras/dim/souligné/inversé et leurs resets.
func TestStyles(t *testing.T) {
	g, p := newFixture()
	p.Feed([]byte("\x1b[1;4mA"))
	if cellAt(g, 0, 0).Flags&(AttrBold|AttrUnderline) != AttrBold|AttrUnderline {
		t.Fatalf("bold+underline flags=%d", cellAt(g, 0, 0).Flags)
	}
	p.Feed([]byte("\x1b[22mB"))
	if cellAt(g, 1, 0).Flags&AttrBold != 0 {
		t.Fatalf("bold not cleared: %d", cellAt(g, 1, 0).Flags)
	}
	p.Feed([]byte("\x1b[7;2mC"))
	f := cellAt(g, 2, 0).Flags
	if f&(AttrInverse|AttrDim) != AttrInverse|AttrDim {
		t.Fatalf("inverse+dim flags=%d", f)
	}
	p.Feed([]byte("\x1b[0mD"))
	if cellAt(g, 3, 0).Flags != 0 {
		t.Fatalf("attr not reset: %d", cellAt(g, 3, 0).Flags)
	}
}

// TestCursorMoves vérifie les déplacements de curseur CSI.
func TestCursorMoves(t *testing.T) {
	g, p := newFixture()
	p.Feed([]byte("\x1b[2;3H"))
	if g.CursorX != 2 || g.CursorY != 1 {
		t.Fatalf("CUP cursor=(%d,%d) want (2,1)", g.CursorX, g.CursorY)
	}
	p.Feed([]byte("\x1b[5A"))
	if g.CursorY != 0 {
		t.Fatalf("CUU cursorY=%d want 0 (clampé)", g.CursorY)
	}
	p.Feed([]byte("\x1b[3C"))
	if g.CursorX != 5 {
		t.Fatalf("CUF cursorX=%d want 5", g.CursorX)
	}
	p.Feed([]byte("\x1b[4D"))
	if g.CursorX != 1 {
		t.Fatalf("CUB cursorX=%d want 1", g.CursorX)
	}
	// Sauvegarde / restauration (ESC 7 / ESC 8, pas CSI).
	p.Feed([]byte("\x1b7\x1b[9;9H\x1b8"))
	if g.CursorX != 1 || g.CursorY != 0 {
		t.Fatalf("restore cursor=(%d,%d) want (1,0)", g.CursorX, g.CursorY)
	}
	// Retour chariot et nouvelle ligne.
	p.Feed([]byte("XY\r\n"))
	if g.CursorX != 0 || g.CursorY != 1 {
		t.Fatalf("CR/LF cursor=(%d,%d) want (0,1)", g.CursorX, g.CursorY)
	}
}

// TestClear vérifie les effacements d'écran et de ligne.
func TestClear(t *testing.T) {
	g, p := newFixture()
	p.Feed([]byte("abcdefghijklmnop"))
	if g.Writes != 16 {
		t.Fatalf("writes=%d want 16 (wrap après 10 cellules)", g.Writes)
	}
	p.Feed([]byte("\x1b[2J"))
	for i := range g.Cells {
		if g.Cells[i] != (Cell{}) {
			t.Fatalf("cell %d not cleared: %+v", i, g.Cells[i])
		}
	}
	// EL mode 0 : effacement de la droite du curseur uniquement.
	g.Reset(10, 4)
	p.Reset(g)
	p.Feed([]byte("abcdefghij"))
	p.Feed([]byte("\x1b[1;5H\x1b[0K"))
	if cellAt(g, 3, 0).Rune != 'd' {
		t.Fatalf("EL0 left must stay: %+v", cellAt(g, 3, 0))
	}
	if cellAt(g, 5, 0) != (Cell{}) {
		t.Fatalf("EL0 right not cleared: %+v", cellAt(g, 5, 0))
	}
}

// TestWrapAndScroll vérifie le retour à la ligne automatique et le défilement
// à la fin de la grille.
func TestWrapAndScroll(t *testing.T) {
	g, p := newFixture()
	// Remplit la ligne 0 (10 cellules) puis dépasse.
	p.Feed([]byte("0123456789"))
	if g.CursorX != 9 || !g.WrapPending {
		t.Fatalf("wrap pending: X=%d pending=%v", g.CursorX, g.WrapPending)
	}
	p.Feed([]byte("A"))
	if g.CursorY != 1 || g.CursorX != 1 {
		t.Fatalf("post-wrap cursor=(%d,%d) want (1,1)", g.CursorX, g.CursorY)
	}
	if cellAt(g, 0, 0).Rune != '0' || cellAt(g, 0, 1).Rune != 'A' {
		t.Fatalf("wrap cells wrong: %+v %+v", cellAt(g, 0, 0), cellAt(g, 0, 1))
	}
	// Remplit tout et écrit en bas : défilement.
	p.Feed([]byte("\x1b[4;1H"))
	p.Feed([]byte("0123456789"))
	p.Feed([]byte("B"))
	if g.CursorY != 3 || g.CursorX != 1 {
		t.Fatalf("scroll cursor=(%d,%d) want (1,3)", g.CursorX, g.CursorY)
	}
	// La ligne 0 a dû être défilée : la première ligne contient la ligne 1.
	if cellAt(g, 0, 0).Rune == 0 {
		t.Fatalf("scroll did not move content up")
	}
}

// TestOSC52Neutralized vérifie la neutralisation explicite du presse-papier.
func TestOSC52Neutralized(t *testing.T) {
	g, p := newFixture()
	p.Feed([]byte("AB"))
	// OSC 52 avec payload base64 : ne doit rien écrire ni stocker.
	p.Feed([]byte("\x1b]52;c;ZGF0YSBjb3BpZQ==\x07"))
	if g.Cells[0] == (Cell{}) || g.Cells[1] == (Cell{}) {
		t.Fatalf("cellules existantes écrasées: %+v", g.Cells[:2])
	}
	if g.CursorX != 2 || g.CursorY != 0 {
		t.Fatalf("OSC 52 a bougé le curseur: (%d,%d)", g.CursorX, g.CursorY)
	}
	if p.State != stateGround {
		t.Fatalf("OSC 52 non terminé: state=%d", p.State)
	}
	// L'état de sélection ne doit pas avoir changé.
	if p.CurFg != 0 || p.CurBg != 0 {
		t.Fatalf("OSC 52 a modifié la sélection: %d/%d", p.CurFg, p.CurBg)
	}
}

// TestOSCTitleNeutralized vérifie la neutralisation des réécritures de titre.
func TestOSCTitleNeutralized(t *testing.T) {
	g, p := newFixture()
	p.Feed([]byte("\x1b]0;Titre malveillant\x07"))
	if p.State != stateGround {
		t.Fatalf("OSC 0 non terminé: state=%d", p.State)
	}
	p.Feed([]byte("\x1b]2;Autre titre\x1b\\"))
	if p.State != stateGround {
		t.Fatalf("OSC 2 (ST) non terminé: state=%d", p.State)
	}
	// Aucune écriture ne doit avoir eu lieu.
	if g.Writes != 0 {
		t.Fatalf("g.Writes=%d want 0", g.Writes)
	}
}

// TestMalformedTruncated vérifie la robustesse aux séquences tronquées ou
// malformées : aucune panique, reprise propre au prochain Feed.
func TestMalformedTruncated(t *testing.T) {
	_, p := newFixture()
	// Séquence CSI coupée en deux morceaux.
	p.Feed([]byte("\x1b["))
	if p.State != stateCSIParam {
		t.Fatalf("state=%d want stateCSIParam", p.State)
	}
	p.Feed([]byte("31m"))
	if p.State != stateGround || p.CurFg != 1 {
		t.Fatalf("reprise CSI: state=%d fg=%d", p.State, p.CurFg)
	}
	// Séquence tronquée en fin de buffer (sans terminateur).
	p.Feed([]byte("\x1b]52;c;abc"))
	if p.State != stateOSCString {
		t.Fatalf("state=%d want stateOSCString (en attente)", p.State)
	}
	p.Feed([]byte("\x07")) // BEL final
	if p.State != stateGround {
		t.Fatalf("BEL n'a pas clos l'OSC: state=%d", p.State)
	}
	// Octets invalides isolés.
	p.Feed([]byte{0xff, 0xfe, 0x00, 0x1b, 0x00})
	if p.State != stateGround {
		t.Fatalf("octets invalides: state=%d", p.State)
	}
	// ESC sans suite.
	p.Feed([]byte("\x1b"))
	if p.State != stateEscape {
		t.Fatalf("ESC: state=%d want stateEscape", p.State)
	}
	p.Feed([]byte("Z")) // ESC Z = inconnu, retour au sol sans effet
	if p.State != stateGround {
		t.Fatalf("ESC inconnu: state=%d", p.State)
	}
}

// TestDCSIgnored vérifie que les séquences DCS, APC, PM et SOS sont consommées sans effet.
func TestDCSIgnored(t *testing.T) {
	g, p := newFixture()
	p.Feed([]byte("ab"))
	// DCS avec payload jusqu'à ST (ESC \).
	p.Feed([]byte("\x1bP1+rpayload-arbitraire\x1b\\"))
	if p.State != stateGround {
		t.Fatalf("DCS non terminé: state=%d", p.State)
	}
	// APC/PM/SOS avec terminateur BEL ou ST
	p.Feed([]byte("\x1b_apc-payload\x07"))
	if p.State != stateGround {
		t.Fatalf("APC non terminé: state=%d", p.State)
	}
	p.Feed([]byte("\x1b^pm-payload\x1b\\"))
	if p.State != stateGround {
		t.Fatalf("PM non terminé: state=%d", p.State)
	}
	if g.Cells[0].Rune != 'a' || g.CursorX != 2 {
		t.Fatalf("DCS a perturbé le dessin")
	}
	// DCS terminé par BEL après un CAN.
	p.Feed([]byte("\x1bP0;x"))
	p.Feed([]byte{0x1a}) // SUB : abort
	if p.State != stateGround {
		t.Fatalf("DCS SUB: state=%d", p.State)
	}
}

// TestUTF8 vérifie le décodage UTF-8 et son écriture en cellule.
func TestUTF8(t *testing.T) {
	g, p := newFixture()
	n := p.Feed([]byte("héllo"))
	if n != 5 {
		t.Fatalf("writes=%d want 5 (é = 2 octets, 1 rune)", n)
	}
	if cellAt(g, 0, 0).Rune != 'h' || cellAt(g, 1, 0).Rune != 'é' || cellAt(g, 2, 0).Rune != 'l' {
		t.Fatalf("utf8 cells wrong: %+v", g.Cells[:3])
	}
	if g.CursorX != 5 {
		t.Fatalf("cursorX=%d want 5 (1 rune par cellule)", g.CursorX)
	}

	// Test de scission UTF-8 à travers 2 Feed successifs (streaming 1 octet)
	p.Reset(g)
	g.Reset(80, 24)
	p.Feed([]byte{0xc3}) // premier octet de 'é' (0xC3 0xA9)
	p.Feed([]byte{0xa9}) // second octet
	if cellAt(g, 0, 0).Rune != 'é' {
		t.Fatalf("scission UTF-8: cell[0]=%U want U+00E9 (é)", cellAt(g, 0, 0).Rune)
	}
	if g.CursorX != 1 {
		t.Fatalf("scission UTF-8: cursorX=%d want 1", g.CursorX)
	}
}

// TestFuzzNoPanic ingère des octets aléatoires variés et vérifie l'absence de
// panique et la cohérence de la grille.
func TestFuzzNoPanic(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	charset := []byte("abcXYZ0123 \r\n\t\x1b\x00\x07\x1a[];?<>(){}1234567890#&%$@!*+/\\")
	for trial := 0; trial < 2000; trial++ {
		g, p := newFixture()
		payload := make([]byte, rng.Intn(512))
		for i := range payload {
			if rng.Intn(4) == 0 {
				payload[i] = charset[rng.Intn(len(charset))]
			} else {
				payload[i] = byte(rng.Intn(256))
			}
		}
		// Ingestion par morceaux de tailles variables.
		off := 0
		for off < len(payload) {
			step := 1 + rng.Intn(16)
			if off+step > len(payload) {
				step = len(payload) - off
			}
			p.Feed(payload[off : off+step])
			off += step
		}
		// Cohérence : chaque cellule écrite doit avoir un état valide.
		for idx, c := range g.Cells {
			if c.Flags > 0x0f {
				t.Fatalf("trial %d: cell %d flags invalides %d", trial, idx, c.Flags)
			}
		}
		_ = g
	}
}

// TestFeedZeroAlloc vérifie le contrat strict zéro allocation sur Feed.
func TestFeedZeroAlloc(t *testing.T) {
	_, p := newFixture()
	trace := buildBigTrace()
	p.Feed(trace) // warmup
	allocs := testing.AllocsPerRun(200, func() {
		p.Feed(trace)
	})
	if allocs != 0 {
		t.Fatalf("Feed allocs=%v (want 0)", allocs)
	}
}

func TestC2FeedASCII(t *testing.T) {
	var p C2_vt_parser_t
	cells := make([]C2_cell_t, 80*24)
	C2_vt_init(&p, 80, 24)
	if C2_vt_feed_byte(&p, cells, 'A') != 1 || cells[0].Rune_ != 'A' {
		t.Fatalf("put A: rune=%d cursor=%d", cells[0].Rune_, p.Cursor_x)
	}
	C2_vt_feed_byte(&p, cells, 0x1b)
	C2_vt_feed_byte(&p, cells, ']')
	C2_vt_feed_byte(&p, cells, '5')
	C2_vt_feed_byte(&p, cells, '2')
	C2_vt_feed_byte(&p, cells, 7)
	if p.State != 0 || cells[0].Rune_ != 'A' {
		t.Fatalf("OSC must not clobber: state=%d rune=%d", p.State, cells[0].Rune_)
	}
}

func TestC2CSIErase(t *testing.T) {
	var p C2_vt_parser_t
	cells := make([]C2_cell_t, 80*24)
	C2_vt_init(&p, 80, 24)
	C2_vt_feed_byte(&p, cells, 'A')
	C2_vt_feed_byte(&p, cells, 'B')
	for _, b := range []byte("\x1b[2J") {
		C2_vt_feed_byte(&p, cells, b)
	}
	if cells[0].Rune_ != 32 || cells[1].Rune_ != 32 {
		t.Fatalf("CSI 2J: %+v %+v", cells[0], cells[1])
	}
}

func TestC2CSIAndSGR(t *testing.T) {
	var p C2_vt_parser_t
	cells := make([]C2_cell_t, 80*24)
	C2_vt_init(&p, 80, 24)
	seq := []byte("\x1b[31m\x1b[2;3HX")
	for _, b := range seq {
		C2_vt_feed_byte(&p, cells, b)
	}
	if p.Cur_fg != 1 {
		t.Fatalf("SGR 31 fg=%d want 1", p.Cur_fg)
	}
	idx := 1*80 + 2
	if cells[idx].Rune_ != 'X' {
		t.Fatalf("CUP 2;3 H X at %d rune=%d cursor=(%d,%d)", idx, cells[idx].Rune_, p.Cursor_x, p.Cursor_y)
	}
}

// buildBigTrace construit une trace ANSI réaliste : un rendu plein écran de
// 40 lignes stylées (couleurs 256, truecolor, styles, déplacements de curseur,
// effacements partiels) dans une grille 160×50, sans débordement de défilement
// — qui mesure le débit de parse, non le coût des primitives de grille.
func buildBigTrace() []byte {
	var b bytes.Buffer
	colors := []string{"31", "38;5;196", "38;5;21", "1;32", "2;33", "7;35", "4;36", "38;2;255;128;0"}
	for i := 0; i < 40; i++ {
		b.WriteString("\x1b[")
		b.WriteString(colors[i%len(colors)])
		b.WriteString("m")
		b.WriteString("Ligne ")
		fmt.Fprintf(&b, "%04d", i)
		b.WriteString(" ")
		// Corps de ligne de 90 caractères ASCII imprimables.
		for j := 0; j < 90; j++ {
			b.WriteByte(byte('a' + (i*90+j)%26))
		}
		b.WriteString("\x1b[0m")
		if i%5 == 0 {
			b.WriteString("\x1b[2K")
		}
		if i%8 == 3 {
			b.WriteString("\x1b[1;2H")
			b.WriteString("\x1b[0mXY")
		}
		b.WriteString("\r\n")
	}
	b.WriteString("\x1b[K") // effacement de la ligne courante (pas de plein écran par frame)
	return b.Bytes()
}

// BenchmarkFeed mesure le débit d'ingestion d'un rendu plein écran : objectif
// > 1 Go/s, 0 B/op. Chaque itération repart d'un curseur en (0,0), comme un
// rendu frais envoyé à l'écran (pas de clear des cellules : contenu sans coût).
func BenchmarkFeed(b *testing.B) {
	g := &CursorGrid{}
	g.Reset(160, 50)
	p := &Parser{}
	p.Reset(g)
	trace := buildBigTrace()
	p.Feed(trace) // warmup
	b.SetBytes(int64(len(trace)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.CursorX, g.CursorY = 0, 0
		g.WrapPending = false
		p.Feed(trace)
	}
}

// BenchmarkFeedWide mesure le débit sur une trace à très longues lignes.
func BenchmarkFeedWide(b *testing.B) {
	g := &CursorGrid{}
	g.Reset(2000, 50)
	p := &Parser{}
	p.Reset(g)
	var sb strings.Builder
	sb.WriteString("\x1b[38;5;196m")
	for i := 0; i < 200; i++ {
		sb.WriteString("0123456789abcdefghijklmnopqrstuvwxyz\x1b[1mABCDEFGHIJKLMNOPQRSTUVWXYZ\x1b[0m")
		sb.WriteString("\x1b[31m;")
	}
	trace := []byte(sb.String())
	p.Feed(trace)
	b.SetBytes(int64(len(trace)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.CursorX, g.CursorY = 0, 0
		g.WrapPending = false
		p.Feed(trace)
	}
}

// BenchmarkFeedScroll mesure le coût du défilement de région (primitive grille)
// sur une trace qui déborde largement la hauteur de l'écran.
func BenchmarkFeedScroll(b *testing.B) {
	g := &CursorGrid{}
	g.Reset(120, 25)
	p := &Parser{}
	p.Reset(g)
	var sb strings.Builder
	for i := 0; i < 400; i++ {
		sb.WriteString("\x1b[36m")
		sb.WriteString("ligne de contenu aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\x1b[0m\r\n")
	}
	trace := []byte(sb.String())
	p.Feed(trace)
	b.SetBytes(int64(len(trace)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.CursorX, g.CursorY = 0, 0
		g.WrapPending = false
		p.Feed(trace)
	}
}
