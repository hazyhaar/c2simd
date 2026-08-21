package c2vtparser

import (
	"fmt"
	"testing"
)

// TestEscTest_CursorBoundaryMatrix teste la robustesse de l'émulateur sur les cas limites
// de coordonnées (0, 1, dépassement positif, négatif, overflow).
func TestEscTest_CursorBoundaryMatrix(t *testing.T) {
	g, p := newTestVTE(80, 24)

	// Cas 1 : Paramètre 0 dans CUP/HVP doit être traité comme 1 (1-based -> index 0)
	p.Feed([]byte("\x1b[0;0H"))
	if g.CursorX != 0 || g.CursorY != 0 {
		t.Fatalf("CUP 0;0 failed: got (%d, %d), want (0, 0)", g.CursorX, g.CursorY)
	}

	p.Feed([]byte("\x1b[0;0f"))
	if g.CursorX != 0 || g.CursorY != 0 {
		t.Fatalf("HVP 0;0 failed: got (%d, %d), want (0, 0)", g.CursorX, g.CursorY)
	}

	// Cas 2 : Coordonnées astronomiques dans CUP (clamping à Width et Height)
	p.Feed([]byte("\x1b[9999;9999H"))
	if g.CursorX != 79 || g.CursorY != 23 {
		t.Fatalf("CUP clamp failed: got (%d, %d), want (79, 23)", g.CursorX, g.CursorY)
	}

	// Cas 3 : CHA et VPA avec paramètre 0 (doit aller en col 1 / row 1 -> index 0)
	p.Feed([]byte("\x1b[0G"))
	if g.CursorX != 0 {
		t.Fatalf("CHA 0 failed: got CursorX=%d, want 0", g.CursorX)
	}
	p.Feed([]byte("\x1b[0d"))
	if g.CursorY != 0 {
		t.Fatalf("VPA 0 failed: got CursorY=%d, want 0", g.CursorY)
	}

	// Cas 4 : Mouvements relatifs au-delà des bornes
	p.Feed([]byte("\x1b[100A\x1b[100D")) // Tout en haut à gauche
	if g.CursorX != 0 || g.CursorY != 0 {
		t.Fatalf("Move beyond top-left failed: got (%d, %d), want (0, 0)", g.CursorX, g.CursorY)
	}

	p.Feed([]byte("\x1b[100B\x1b[100C")) // Tout en bas à droite
	if g.CursorX != 79 || g.CursorY != 23 {
		t.Fatalf("Move beyond bottom-right failed: got (%d, %d), want (79, 23)", g.CursorX, g.CursorY)
	}
}

// TestEscTest_EditOperations_Interleaved valide les séquences d'édition combinées
// (ICH, DCH, ECH, IL, DL) et l'intégrité de la mémoire de la grille.
func TestEscTest_EditOperations_Interleaved(t *testing.T) {
	g, p := newTestVTE(20, 10)

	// Remplissage initial
	for y := 1; y <= 10; y++ {
		p.Feed([]byte(fmt.Sprintf("\x1b[%d;1HABCDEFGHIJKLMNOPQRST", y)))
	}

	// Ligne 1 : Insertion de 3 caractères en colonne 5 (index 4)
	// Original : ABCDEFGHIJKLMNOPQRST
	// Après ICH 3 : ABCD   EFGHIJKLMNOPQ (RST perdus à droite)
	p.Feed([]byte("\x1b[1;5H\x1b[3@"))
	if got := getGridRowText(g, 0); got != "ABCD   EFGHIJKLMNOPQ" {
		t.Fatalf("ICH 3 failed: got %q, want %q", got, "ABCD   EFGHIJKLMNOPQ")
	}

	// Ligne 1 : Suppression de 3 caractères en colonne 5 (index 4)
	// Après DCH 3 : ABCDEFGHIJKLMNOPQ   (3 espaces à la fin)
	p.Feed([]byte("\x1b[1;5H\x1b[3P"))
	if got := getGridRowText(g, 0); got != "ABCDEFGHIJKLMNOPQ   " {
		t.Fatalf("DCH 3 failed: got %q, want %q", got, "ABCDEFGHIJKLMNOPQ   ")
	}

	// Ligne 1 : Effacement de 4 caractères en colonne 5 (index 4)
	// Après ECH 4 : ABCD    IJKLMNOPQ   (in-place, pas de décalage)
	p.Feed([]byte("\x1b[1;5H\x1b[4X"))
	if got := getGridRowText(g, 0); got != "ABCD    IJKLMNOPQ   " {
		t.Fatalf("ECH 4 failed: got %q, want %q", got, "ABCD    IJKLMNOPQ   ")
	}

	// Insertion de ligne (IL 2) à la ligne 5 (index 4)
	// Lignes 5 et 6 deviennent vierges, les lignes inférieures sont poussées vers le bas
	p.Feed([]byte("\x1b[5;1H\x1b[2L"))
	if got := getGridRowText(g, 4); got != "                    " {
		t.Fatalf("IL 2 row 4 failed: got %q, want blank", got)
	}
	if got := getGridRowText(g, 5); got != "                    " {
		t.Fatalf("IL 2 row 5 failed: got %q, want blank", got)
	}
	if got := getGridRowText(g, 6); got != "ABCDEFGHIJKLMNOPQRST" {
		t.Fatalf("IL 2 row 6 failed: got %q, want shifted row", got)
	}

	// Suppression de ligne (DL 2) à la ligne 5 (index 4)
	// Restaure la continuité des lignes
	p.Feed([]byte("\x1b[5;1H\x1b[2M"))
	if got := getGridRowText(g, 4); got != "ABCDEFGHIJKLMNOPQRST" {
		t.Fatalf("DL 2 row 4 failed: got %q, want restored row", got)
	}
}

// TestEscTest_DECSTBM_BoundaryScenarios valide les scénarios limites de marges DECSTBM :
// marges invalides ignorées, curseur hors région de défilement, et scrolling strict.
func TestEscTest_DECSTBM_BoundaryScenarios(t *testing.T) {
	g, p := newTestVTE(10, 8)

	// 1. Marges invalides : top > bot (ex: \x1b[6;3r) ou bot > Height (ex: \x1b[2;20r)
	// Les marges courantes [0, 7] doivent être conservées.
	p.Feed([]byte("\x1b[6;3r"))
	if g.Top != 0 || g.Bottom != 7 {
		t.Fatalf("Invalid DECSTBM (top > bot) must be ignored: Top=%d Bottom=%d", g.Top, g.Bottom)
	}

	p.Feed([]byte("\x1b[2;20r"))
	if g.Top != 0 || g.Bottom != 7 {
		t.Fatalf("Invalid DECSTBM (bot > Height) must be ignored: Top=%d Bottom=%d", g.Top, g.Bottom)
	}

	// 2. Marges valides [3, 6] (1-based -> index [2, 5])
	p.Feed([]byte("\x1b[3;6r"))
	if g.Top != 2 || g.Bottom != 5 {
		t.Fatalf("Valid DECSTBM mismatch: Top=%d Bottom=%d, want Top=2 Bottom=5", g.Top, g.Bottom)
	}

	// Remplissage des lignes 1 à 8
	for y := 1; y <= 8; y++ {
		p.Feed([]byte(fmt.Sprintf("\x1b[%d;1HROW_%d-----", y, y)))
	}

	// 3. Curseur situé EN DESSOUS de la région de scroll (ex: ligne 7 / index 6)
	// Un Line Feed (\n) ne doit PAS faire défiler la région de scroll interne [2, 5] !
	p.Feed([]byte("\x1b[7;1H\n"))
	if got := getGridRowText(g, 2); got != "ROW_3-----" {
		t.Fatalf("Row 2 inside scroll region corrupted by LF outside region: got %q", got)
	}
	if got := getGridRowText(g, 5); got != "ROW_6-----" {
		t.Fatalf("Row 5 inside scroll region corrupted by LF outside region: got %q", got)
	}

	// 4. Curseur situé AU-DESSUS de la région de scroll (ex: ligne 1 / index 0)
	// Un Reverse Index (\x1bM) ne doit PAS faire défiler la région [2, 5]
	p.Feed([]byte("\x1b[1;1H\x1bM"))
	if got := getGridRowText(g, 2); got != "ROW_3-----" {
		t.Fatalf("Row 2 corrupted by RI outside region: got %q", got)
	}
	if got := getGridRowText(g, 0); got != "ROW_1-----" {
		t.Fatalf("Row 0 corrupted by RI at top: got %q", got)
	}
}

// TestEscTest_TabulationMatrix teste une disposition complexe et non-uniforme de taquets de tabulation.
func TestEscTest_TabulationMatrix(t *testing.T) {
	g, p := newTestVTE(80, 5)

	// Efface tous les taquets par défaut
	p.Feed([]byte("\x1b[3g"))

	// Pose de taquets aux colonnes : 3, 7, 13, 25, 50 (1-based : 4, 8, 14, 26, 51)
	stops := []int{3, 7, 13, 25, 50}
	for _, s := range stops {
		p.Feed([]byte(fmt.Sprintf("\x1b[1;%dH\x1bH", s+1)))
	}

	// Navigation séquentielle avec \t
	p.Feed([]byte("\x1b[1;1H"))
	for _, expectedCol := range stops {
		p.Feed([]byte("\t"))
		if g.CursorX != expectedCol {
			t.Fatalf("Tab navigation failed: got CursorX=%d, want %d", g.CursorX, expectedCol)
		}
	}

	// Tabulation au-delà du dernier taquet : doit atteindre la marge droite (Width-1 = 79)
	p.Feed([]byte("\t"))
	if g.CursorX != 79 {
		t.Fatalf("Tab beyond last stop failed: got CursorX=%d, want 79", g.CursorX)
	}
}

// TestEscTest_DECAWM_WrapScenarios teste les subtilités d'armement et désarmement de WrapPending.
func TestEscTest_DECAWM_WrapScenarios(t *testing.T) {
	g, p := newTestVTE(10, 3)

	// Cas 1 : Remplissage exact d'une ligne de 10 caractères -> WrapPending = true
	p.Feed([]byte("\x1b[1;1H0123456789"))
	if !g.WrapPending {
		t.Fatalf("WrapPending must be true after writing 10 chars on 10-wide grid")
	}
	if g.CursorX != 9 || g.CursorY != 0 {
		t.Fatalf("Cursor must be at (9, 0), got (%d, %d)", g.CursorX, g.CursorY)
	}

	// Un Backspace (\b) doit désarmer WrapPending et reculer en colonne 8
	p.Feed([]byte("\b"))
	if g.WrapPending {
		t.Fatalf("Backspace must clear WrapPending")
	}
	if g.CursorX != 8 || g.CursorY != 0 {
		t.Fatalf("Cursor after BS must be (8, 0), got (%d, %d)", g.CursorX, g.CursorY)
	}

	// Cas 2 : Réarmement de WrapPending par écriture
	p.Feed([]byte("XY")) // Écrit 'X' en col 8, 'Y' en col 9 -> WrapPending = true
	if !g.WrapPending {
		t.Fatalf("WrapPending must be true")
	}

	// Un déplacement CSI D (CUB 1) doit désarmer WrapPending
	p.Feed([]byte("\x1b[1D"))
	if g.WrapPending {
		t.Fatalf("CUB must clear WrapPending")
	}
	if g.CursorX != 8 {
		t.Fatalf("CursorX after CUB 1 must be 8, got %d", g.CursorX)
	}

	// Cas 3 : Mode DECAWM inactif (?7l)
	p.Feed([]byte("\x1b[?7l\x1b[2;1H"))
	p.Feed([]byte("ABCDEFGHIJKLMN")) // 14 caractères sur ligne 2
	// Doit contenir "ABCDEFGHIN" sur ligne 2 et CursorY doit rester 1
	if got := getGridRowText(g, 1); got != "ABCDEFGHIN" {
		t.Fatalf("DECAWM disabled wrap failed: got %q, want %q", got, "ABCDEFGHIN")
	}
	if g.CursorY != 1 {
		t.Fatalf("DECAWM disabled must not advance CursorY: got %d", g.CursorY)
	}
}

// TestEscTest_ZeroAlloc_HotPaths vérifie qu'aucune allocation mémoire n'a lieu
// lors de l'exécution de toutes les opérations du terminal.
func TestEscTest_ZeroAlloc_HotPaths(t *testing.T) {
	_, p := newTestVTE(80, 24)

	// Corpus de test varié combinant mouvements, édition, DEC graphics, tabulations et texte
	seq := []byte("\x1b[H\x1b[2J" +
		"Hello World\t\x1b[31mRed\x1b[0m\r\n" +
		"\x1b(0lqqk\x1b(B\r\n" +
		"\x1b[3@\x1b[2P\x1b[1X" +
		"\x1b[5;10H\x1b[2A\x1b[3C\x1b[1D" +
		"\x1b[2;5r\x1b[H\r\n\x1b[r")

	// Échauffement
	p.Feed(seq)

	// Mesure des allocations
	allocs := testing.AllocsPerRun(1000, func() {
		p.Feed(seq)
	})

	if allocs > 0 {
		t.Fatalf("Zero-alloc violation on hot paths: got %f allocs/op, want 0", allocs)
	}
}
