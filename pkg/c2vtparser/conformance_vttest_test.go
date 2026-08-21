package c2vtparser

import (
	"fmt"
	"testing"
)

// helper de création d'une grille propre et de son parseur associé.
func newTestVTE(w, h int) (*CursorGrid, *Parser) {
	g := &CursorGrid{}
	g.Reset(w, h)
	p := &Parser{}
	p.Reset(g)
	return g, p
}

// helper pour extraire le texte d'une ligne de la grille sous forme de chaîne.
func getGridRowText(g *CursorGrid, y int) string {
	if y < 0 || y >= g.Height {
		return ""
	}
	runes := make([]rune, g.Width)
	for x := 0; x < g.Width; x++ {
		r := g.Cells[y*g.Width+x].Rune
		if r == 0 {
			runes[x] = ' '
		} else {
			runes[x] = r
		}
	}
	return string(runes)
}

// TestConformance_CursorMovements valide l'ensemble des mouvements de curseur standards VT100/VT500 :
// CUU, CUD, CUF, CUB, CNL, CPL, CHA, CUP, HVP, VPR, HPA, VPA, HPR.
func TestConformance_CursorMovements(t *testing.T) {
	g, p := newTestVTE(20, 10)

	// 1. CUP (CSI r ; c H) et HVP (CSI r ; c f)
	p.Feed([]byte("\x1b[5;10H"))
	if g.CursorX != 9 || g.CursorY != 4 {
		t.Fatalf("CUP 5;10 failed: got (%d, %d), want (9, 4)", g.CursorX, g.CursorY)
	}

	p.Feed([]byte("\x1b[2;4f"))
	if g.CursorX != 3 || g.CursorY != 1 {
		t.Fatalf("HVP 2;4 failed: got (%d, %d), want (3, 1)", g.CursorX, g.CursorY)
	}

	// CUP sans paramètres (défaut 1;1)
	p.Feed([]byte("\x1b[H"))
	if g.CursorX != 0 || g.CursorY != 0 {
		t.Fatalf("CUP default failed: got (%d, %d), want (0, 0)", g.CursorX, g.CursorY)
	}

	// 2. CUU (CSI A), CUD (CSI B), CUF (CSI C), CUB (CSI D)
	p.Feed([]byte("\x1b[3C")) // 3 colonnes à droite -> (3, 0)
	if g.CursorX != 3 || g.CursorY != 0 {
		t.Fatalf("CUF 3 failed: got (%d, %d), want (3, 0)", g.CursorX, g.CursorY)
	}

	p.Feed([]byte("\x1b[5B")) // 5 lignes en bas -> (3, 5)
	if g.CursorX != 3 || g.CursorY != 5 {
		t.Fatalf("CUD 5 failed: got (%d, %d), want (3, 5)", g.CursorX, g.CursorY)
	}

	p.Feed([]byte("\x1b[2D")) // 2 colonnes à gauche -> (1, 5)
	if g.CursorX != 1 || g.CursorY != 5 {
		t.Fatalf("CUB 2 failed: got (%d, %d), want (1, 5)", g.CursorX, g.CursorY)
	}

	p.Feed([]byte("\x1b[3A")) // 3 lignes en haut -> (1, 2)
	if g.CursorX != 1 || g.CursorY != 2 {
		t.Fatalf("CUU 3 failed: got (%d, %d), want (1, 2)", g.CursorX, g.CursorY)
	}

	// 3. CNL (CSI E) et CPL (CSI F)
	p.Feed([]byte("\x1b[2E")) // 2 lignes en bas et colonne 0 -> (0, 4)
	if g.CursorX != 0 || g.CursorY != 4 {
		t.Fatalf("CNL 2 failed: got (%d, %d), want (0, 4)", g.CursorX, g.CursorY)
	}

	p.Feed([]byte("\x1b[5C\x1b[1F")) // Va en (5, 4) puis CPL 1 -> (0, 3)
	if g.CursorX != 0 || g.CursorY != 3 {
		t.Fatalf("CPL 1 failed: got (%d, %d), want (0, 3)", g.CursorX, g.CursorY)
	}

	// 4. CHA (CSI G) et HPA (CSI `)
	p.Feed([]byte("\x1b[15G")) // Colonne 15 (0-based 14) -> (14, 3)
	if g.CursorX != 14 || g.CursorY != 3 {
		t.Fatalf("CHA 15 failed: got (%d, %d), want (14, 3)", g.CursorX, g.CursorY)
	}

	p.Feed([]byte("\x1b[7`")) // Colonne 7 (0-based 6) -> (6, 3)
	if g.CursorX != 6 || g.CursorY != 3 {
		t.Fatalf("HPA 7 failed: got (%d, %d), want (6, 3)", g.CursorX, g.CursorY)
	}

	// 5. VPA (CSI d), VPR (CSI e) et HPR (CSI a)
	p.Feed([]byte("\x1b[8d")) // Ligne 8 (0-based 7) -> (6, 7)
	if g.CursorX != 6 || g.CursorY != 7 {
		t.Fatalf("VPA 8 failed: got (%d, %d), want (6, 7)", g.CursorX, g.CursorY)
	}

	p.Feed([]byte("\x1b[2e")) // VPR 2 -> descend de 2 -> (6, 9)
	if g.CursorX != 6 || g.CursorY != 9 {
		t.Fatalf("VPR 2 failed: got (%d, %d), want (6, 9)", g.CursorX, g.CursorY)
	}

	p.Feed([]byte("\x1b[4a")) // HPR 4 -> avance de 4 -> (10, 9)
	if g.CursorX != 10 || g.CursorY != 9 {
		t.Fatalf("HPR 4 failed: got (%d, %d), want (10, 9)", g.CursorX, g.CursorY)
	}

	// 6. Clampage aux bornes de la grille (limites haute, basse, gauche, droite)
	p.Feed([]byte("\x1b[999A\x1b[999D")) // Tout en haut à gauche
	if g.CursorX != 0 || g.CursorY != 0 {
		t.Fatalf("Clamping top-left failed: got (%d, %d), want (0, 0)", g.CursorX, g.CursorY)
	}

	p.Feed([]byte("\x1b[999B\x1b[999C")) // Tout en bas à droite
	if g.CursorX != 19 || g.CursorY != 9 {
		t.Fatalf("Clamping bottom-right failed: got (%d, %d), want (19, 9)", g.CursorX, g.CursorY)
	}
}

// TestConformance_EditingAndErasure valide les séquences d'édition de flux VT100/VT500 :
// ED (0, 1, 2, 3), EL (0, 1, 2), ICH, DCH, IL, DL, ECH.
func TestConformance_EditingAndErasure(t *testing.T) {
	g, p := newTestVTE(10, 5)

	// Remplissage initial de l'écran avec 5 lignes de chiffres
	initGrid := func() {
		p.Feed([]byte("\x1b[H\x1b[2J"))
		for y := 0; y < 5; y++ {
			p.Feed([]byte(fmt.Sprintf("0123456789\r\n")))
		}
	}

	// 1. EL (Erase in Line) : 0, 1, 2
	initGrid()
	p.Feed([]byte("\x1b[2;5H\x1b[0K")) // Ligne 2 (y=1), col 5 (x=4), efface de x=4 jusqu'à la fin
	if got := getGridRowText(g, 1); got != "0123      " {
		t.Fatalf("EL 0 failed: got %q, want %q", got, "0123      ")
	}

	initGrid()
	p.Feed([]byte("\x1b[2;5H\x1b[1K")) // Ligne 2 (y=1), col 5 (x=4), efface du début jusqu'à x=4 inclus
	if got := getGridRowText(g, 1); got != "     56789" {
		t.Fatalf("EL 1 failed: got %q, want %q", got, "     56789")
	}

	initGrid()
	p.Feed([]byte("\x1b[2;5H\x1b[2K")) // Ligne 2 (y=1), efface toute la ligne
	if got := getGridRowText(g, 1); got != "          " {
		t.Fatalf("EL 2 failed: got %q, want %q", got, "          ")
	}

	// 2. ED (Erase in Display) : 0, 1, 2, 3
	initGrid()
	p.Feed([]byte("\x1b[3;5H\x1b[0J")) // Ligne 3 (y=2), col 5 (x=4) -> efface de (4, 2) jusqu'à fin écran
	if got := getGridRowText(g, 1); got != "0123456789" {
		t.Fatalf("ED 0 row 1 altered: got %q", got)
	}
	if got := getGridRowText(g, 2); got != "0123      " {
		t.Fatalf("ED 0 row 2 failed: got %q, want %q", got, "0123      ")
	}
	if got := getGridRowText(g, 3); got != "          " {
		t.Fatalf("ED 0 row 3 failed: got %q, want blank", got)
	}
	if got := getGridRowText(g, 4); got != "          " {
		t.Fatalf("ED 0 row 4 failed: got %q, want blank", got)
	}

	initGrid()
	p.Feed([]byte("\x1b[3;5H\x1b[1J")) // Ligne 3 (y=2), col 5 (x=4) -> efface du début écran jusqu'à (4, 2) inclus
	if got := getGridRowText(g, 0); got != "          " {
		t.Fatalf("ED 1 row 0 failed: got %q, want blank", got)
	}
	if got := getGridRowText(g, 1); got != "          " {
		t.Fatalf("ED 1 row 1 failed: got %q, want blank", got)
	}
	if got := getGridRowText(g, 2); got != "     56789" {
		t.Fatalf("ED 1 row 2 failed: got %q, want %q", got, "     56789")
	}
	if got := getGridRowText(g, 3); got != "0123456789" {
		t.Fatalf("ED 1 row 3 altered: got %q", got)
	}

	initGrid()
	p.Feed([]byte("\x1b[2J")) // Efface tout l'écran
	for y := 0; y < 5; y++ {
		if got := getGridRowText(g, y); got != "          " {
			t.Fatalf("ED 2 row %d failed: got %q, want blank", y, got)
		}
	}

	// 3. ICH (Insert Character) et DCH (Delete Character)
	initGrid()
	p.Feed([]byte("\x1b[1;3H\x1b[2@")) // Ligne 1, col 3 (x=2), insère 2 espaces
	if got := getGridRowText(g, 0); got != "01  234567" {
		t.Fatalf("ICH 2 failed: got %q, want %q", got, "01  234567")
	}

	p.Feed([]byte("\x1b[1;3H\x1b[2P")) // Ligne 1, col 3 (x=2), supprime 2 caractères
	if got := getGridRowText(g, 0); got != "01234567  " {
		t.Fatalf("DCH 2 failed: got %q, want %q", got, "01234567  ")
	}

	// 4. ECH (Erase Character)
	initGrid()
	p.Feed([]byte("\x1b[1;4H\x1b[3X")) // Ligne 1, col 4 (x=3), efface 3 caractères
	if got := getGridRowText(g, 0); got != "012   6789" {
		t.Fatalf("ECH 3 failed: got %q, want %q", got, "012   6789")
	}

	// 5. IL (Insert Line) et DL (Delete Line)
	initGrid()
	p.Feed([]byte("\x1b[2;1H\x1b[1L")) // Ligne 2 (y=1), insère 1 ligne
	if got := getGridRowText(g, 0); got != "0123456789" {
		t.Fatalf("IL 1 row 0 altered: got %q", got)
	}
	if got := getGridRowText(g, 1); got != "          " {
		t.Fatalf("IL 1 row 1 failed: got %q, want blank", got)
	}
	if got := getGridRowText(g, 2); got != "0123456789" {
		t.Fatalf("IL 1 row 2 failed: got %q, want row shifted down", got)
	}

	p.Feed([]byte("\x1b[2;1H\x1b[1M")) // Ligne 2 (y=1), supprime 1 ligne
	if got := getGridRowText(g, 1); got != "0123456789" {
		t.Fatalf("DL 1 row 1 failed: got %q, want restored row", got)
	}
}

// TestConformance_DECSTBM_MarginsAndScrolling valide le respect strict des marges de scrolling VT100 :
// CSI top ; bot r, confinement du défilement, et replaçage du curseur en (1, 1).
func TestConformance_DECSTBM_MarginsAndScrolling(t *testing.T) {
	g, p := newTestVTE(10, 6)

	// Écriture de 6 lignes numérotées L1 à L6
	for y := 1; y <= 6; y++ {
		p.Feed([]byte(fmt.Sprintf("\x1b[%d;1HLINE_%d----", y, y)))
	}

	// Configuration d'une région de défilement [2, 5] (lignes 2 à 5 en 1-based, index 1 à 4 en 0-based)
	p.Feed([]byte("\x1b[2;5r"))

	// Vérification du replacement automatique du curseur en (1, 1) / index (0, 0)
	if g.CursorX != 0 || g.CursorY != 0 {
		t.Fatalf("DECSTBM must home cursor: got (%d, %d), want (0, 0)", g.CursorX, g.CursorY)
	}
	if g.Top != 1 || g.Bottom != 4 {
		t.Fatalf("DECSTBM margins mismatch: got Top=%d Bottom=%d, want Top=1 Bottom=4", g.Top, g.Bottom)
	}

	// Déplacement sur la marge inférieure de défilement (ligne 5 / index 4) et écriture de nouvelles lignes
	p.Feed([]byte("\x1b[5;1HNEW_A-----\r\n"))
	p.Feed([]byte("NEW_B-----"))

	// Vérification :
	// Ligne 1 (hors région, au-dessus de Top) : doit rester intacte ("LINE_1----")
	if got := getGridRowText(g, 0); got != "LINE_1----" {
		t.Fatalf("Row 0 above scroll region corrupted: got %q, want %q", got, "LINE_1----")
	}
	// Ligne 6 (hors région, en-dessous de Bottom) : doit rester intacte ("LINE_6----")
	if got := getGridRowText(g, 5); got != "LINE_6----" {
		t.Fatalf("Row 5 below scroll region corrupted: got %q, want %q", got, "LINE_6----")
	}
	// Lignes 4 et 5 (index 3 et 4) doivent contenir NEW_A et NEW_B
	if got := getGridRowText(g, 3); got != "NEW_A-----" {
		t.Fatalf("Row 3 in scroll region mismatch: got %q, want %q", got, "NEW_A-----")
	}
	if got := getGridRowText(g, 4); got != "NEW_B-----" {
		t.Fatalf("Row 4 in scroll region mismatch: got %q, want %q", got, "NEW_B-----")
	}

	// Test du Reverse Index (RI / ESC M) sur la marge haute de la région (Top=1)
	p.Feed([]byte("\x1b[2;1H\x1bM")) // Défilement vers le bas
	if got := getGridRowText(g, 0); got != "LINE_1----" {
		t.Fatalf("Row 0 corrupted after RI: got %q", got)
	}
	if got := getGridRowText(g, 1); got != "          " {
		t.Fatalf("Row 1 after RI must be blank: got %q", got)
	}

	// Réinitialisation des marges (CSI r sans paramètre)
	p.Feed([]byte("\x1b[r"))
	if g.Top != 0 || g.Bottom != 5 {
		t.Fatalf("DECSTBM reset failed: Top=%d Bottom=%d, want Top=0 Bottom=5", g.Top, g.Bottom)
	}
}

// TestConformance_HardwareTabStops valide le positionnement par tabulations matérielles (HT, HTS, TBC).
func TestConformance_HardwareTabStops(t *testing.T) {
	g, p := newTestVTE(80, 5)

	// Par défaut : taquets tous les 8 caractères (0, 8, 16, 24, 32...)
	p.Feed([]byte("A\tB\tC"))
	// 'A' écrit en 0 -> tab avance à 8 -> 'B' écrit en 8 -> tab avance à 16 -> 'C' écrit en 16
	if cellAt(g, 0, 0).Rune != 'A' || cellAt(g, 8, 0).Rune != 'B' || cellAt(g, 16, 0).Rune != 'C' {
		t.Fatalf("Default tab stops failed: cells not at 0, 8, 16")
	}

	// Effacement de tous les taquets (TBC 3 : CSI 3 g)
	p.Feed([]byte("\r\n\x1b[3g"))

	// Ajout de taquets personnalisés avec HTS (ESC H) aux colonnes 5 et 15 (1-based 6 et 16)
	p.Feed([]byte("\x1b[1;6H\x1bH"))  // Col 5 (0-based)
	p.Feed([]byte("\x1b[1;16H\x1bH")) // Col 15 (0-based)

	// Test de navigation sur les nouveaux taquets
	p.Feed([]byte("\x1b[2;1HX\tY\tZ"))
	if cellAt(g, 0, 1).Rune != 'X' || cellAt(g, 5, 1).Rune != 'Y' || cellAt(g, 15, 1).Rune != 'Z' {
		t.Fatalf("Custom tab stops failed: X at 0, Y at %d (want 5), Z at %d (want 15)", g.CursorX, 15)
	}

	// Effacement du taquet en colonne 5 avec TBC 0 (CSI 0 g)
	p.Feed([]byte("\x1b[1;6H\x1b[0g"))
	p.Feed([]byte("\x1b[3;1HX\tY"))
	// Maintenant le premier taquet après 0 est 15
	if cellAt(g, 0, 2).Rune != 'X' || cellAt(g, 15, 2).Rune != 'Y' {
		t.Fatalf("Tab stop clear at col 5 failed: Y at col %d (want 15)", 15)
	}
}

// TestConformance_DECPrivateModes valide les modes privés DEC :
// DECTCEM (25), DECAWM (7), DECOLM (3), DECSCNM (5).
func TestConformance_DECPrivateModes(t *testing.T) {
	g, p := newTestVTE(10, 4)

	// 1. DECTCEM (Curseur visible / invisible)
	p.Feed([]byte("\x1b[?25l"))
	if p.CursorVisible != false {
		t.Fatalf("DECTCEM ?25l failed: want CursorVisible=false")
	}
	p.Feed([]byte("\x1b[?25h"))
	if p.CursorVisible != true {
		t.Fatalf("DECTCEM ?25h failed: want CursorVisible=true")
	}

	// 2. DECAWM (Auto-Wrap Mode)
	// Désactivation de l'auto-wrap (?7l)
	p.Feed([]byte("\x1b[?7l\x1b[H"))
	p.Feed([]byte("0123456789ABCDEF")) // 16 caractères sur une ligne de 10
	// Les caractères supplémentaires doivent écraser la dernière colonne (col 9) sans passer à la ligne suivante
	if got := getGridRowText(g, 0); got != "012345678F" {
		t.Fatalf("DECAWM disabled failed: got %q, want %q", got, "012345678F")
	}
	if g.CursorY != 0 {
		t.Fatalf("DECAWM disabled caused newline: CursorY=%d, want 0", g.CursorY)
	}

	// Réactivation de l'auto-wrap (?7h)
	p.Feed([]byte("\x1b[?7h\x1b[H\x1b[2J"))
	p.Feed([]byte("0123456789X")) // 11 caractères
	if got := getGridRowText(g, 0); got != "0123456789" {
		t.Fatalf("DECAWM enabled row 0 failed: got %q", got)
	}
	if cellAt(g, 0, 1).Rune != 'X' {
		t.Fatalf("DECAWM enabled row 1 cell 0 failed: got %c, want 'X'", cellAt(g, 0, 1).Rune)
	}

	// 3. DECSCNM (Screen Inverse Video)
	p.Feed([]byte("\x1b[?5h"))
	if p.InverseScreen != true {
		t.Fatalf("DECSCNM ?5h failed: want InverseScreen=true")
	}
	p.Feed([]byte("\x1b[?5l"))
	if p.InverseScreen != false {
		t.Fatalf("DECSCNM ?5l failed: want InverseScreen=false")
	}

	// 4. DECOLM (80/132 colonnes)
	p.Feed([]byte("\x1b[?3h"))
	if g.Width != 132 || p.DECOLM132 != true {
		t.Fatalf("DECOLM ?3h failed: Width=%d, want 132", g.Width)
	}
	p.Feed([]byte("\x1b[?3l"))
	if g.Width != 80 || p.DECOLM132 != false {
		t.Fatalf("DECOLM ?3l failed: Width=%d, want 80", g.Width)
	}
}

// TestConformance_DECSpecialGraphics valide le jeu de caractères VT100 Line Drawing (ESC ( 0 / ESC ) 0 / SI / SO).
func TestConformance_DECSpecialGraphics(t *testing.T) {
	g, p := newTestVTE(10, 5)

	// Sélection du jeu DEC Special Graphics en G0 via ESC ( 0 et Shift-In (0x0F)
	p.Feed([]byte("\x1b(0\x0f"))

	// Tracé d'une boîte fermée 4x4 :
	// lqqk -> ┌──┐
	// x  x -> │  │
	// x  x -> │  │
	// mqqj -> └──┘
	p.Feed([]byte("\x1b[1;1Hlqqk"))
	p.Feed([]byte("\x1b[2;1Hx  x"))
	p.Feed([]byte("\x1b[3;1Hx  x"))
	p.Feed([]byte("\x1b[4;1Hmqqj"))

	// Vérification bit-exacte des runes de la boîte
	expectedRunes := [][]rune{
		{'┌', '─', '─', '┐'},
		{'│', ' ', ' ', '│'},
		{'│', ' ', ' ', '│'},
		{'└', '─', '─', '┘'},
	}

	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			got := cellAt(g, x, y).Rune
			want := expectedRunes[y][x]
			if got != want {
				t.Fatalf("Box drawing mismatch at (%d, %d): got %c (%U), want %c (%U)", x, y, got, got, want, want)
			}
		}
	}

	// Retour au jeu US-ASCII en G0 via ESC ( B
	p.Feed([]byte("\x1b(B"))
	p.Feed([]byte("\x1b[5;1Hlqqk"))
	for x, want := range []rune{'l', 'q', 'q', 'k'} {
		got := cellAt(g, x, 4).Rune
		if got != want {
			t.Fatalf("ASCII restore mismatch at col %d: got %c, want %c", x, got, want)
		}
	}
}
