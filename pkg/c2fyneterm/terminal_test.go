package c2fyneterm

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	c2painter "code.hazyhaar.fr/devhoros/pkg/c2painter"
	tt55 "code.hazyhaar.fr/devhoros/pkg/tt55"
)

// Test 1 : Simulation fidèle d'une session de htop
// (Alternate Screen Buffer, positionnement curseur, barres CPU colorées, cadres de boîte, barre de touches fonction)
func TestTerminal_HtopSession(t *testing.T) {
	term := NewTerminalWidget(80, 24)

	var htopSeq bytes.Buffer
	htopSeq.WriteString("\x1b[?1049h\x1b[?25l\x1b[H\x1b[2J")

	// Ligne 1 : CPU 1 [||||||||||||||||||          48.5%]
	htopSeq.WriteString("\x1b[1;1H\x1b[1;32m1  \x1b[0;37m[\x1b[1;32m||||||||||\x1b[1;34m||||||||\x1b[0;37m          48.5%]")

	// Ligne 2 : CPU 2 [||||||||                    22.1%]
	htopSeq.WriteString("\x1b[2;1H\x1b[1;32m2  \x1b[0;37m[\x1b[1;32m||||||||  \x1b[0;37m                    22.1%]")

	// Ligne 3 : Mem   [|||||||||||||||||||| 4.12G/15.5G]
	htopSeq.WriteString("\x1b[3;1H\x1b[1;36mMem\x1b[0;37m[\x1b[1;32m||||||||||||||||||||\x1b[0;37m 4.12G/15.5G]")

	// Ligne 5 : En-tête de table des processus en vidéo inverse
	htopSeq.WriteString("\x1b[5;1H\x1b[7m  PID USER      PRI  NI  VIRT   RES   SHR S CPU% MEM%   TIME+  Command         \x1b[0m")

	// Lignes 6 à 8 : Liste des processus
	htopSeq.WriteString("\x1b[6;1H 1234 root       20   0  1.2G  240M   80M S 12.5  1.5  0:14.22 /usr/bin/dockerd")
	htopSeq.WriteString("\x1b[7;1H 5678 cl-ment    20   0  850M  120M   45M S  8.2  0.8  0:05.18 /bin/go test    ")
	htopSeq.WriteString("\x1b[8;1H 9012 cl-ment    20   0  420M   65M   30M R  4.1  0.4  0:01.02 htop            ")

	// Ligne 24 : Barre d'aide des touches F1 à F10 en vidéo inverse
	htopSeq.WriteString("\x1b[24;1H\x1b[7mF1\x1b[0mHelp  \x1b[7mF2\x1b[0mSetup \x1b[7mF3\x1b[0mSearch\x1b[7mF4\x1b[0mFilter\x1b[7mF5\x1b[0mTree  \x1b[7mF6\x1b[0mSortBy\x1b[7mF9\x1b[0mKill  \x1b[7mF10\x1b[0mQuit")

	n, err := term.Write(htopSeq.Bytes())
	if err != nil || n != htopSeq.Len() {
		t.Fatalf("Write htop a échoué: %v (n=%d, want %d)", err, n, htopSeq.Len())
	}

	// 1. Vérification de l'activation de l'Alternate Screen
	if !term.IsAlternateScreen() {
		t.Fatalf("Le mode Alternate Screen aurait dû être activé par \\x1b[?1049h")
	}

	// 2. Vérification de l'état du curseur (masqué)
	if term.cursorVisible {
		t.Fatalf("Le curseur aurait dû être masqué par \\x1b[?25l")
	}

	// 3. Vérification du contenu des lignes
	c1 := term.CellAt(0, 0)
	if c1.Rune != '1' {
		t.Fatalf("Ligne 1 Col 0 attendue '1', obtenu '%c'", c1.Rune)
	}

	// Vérification de la vidéo inverse sur l'en-tête de la ligne 4 (row index 4 = ligne 5)
	cHeader := term.CellAt(5, 4)
	if (cHeader.Flags & AttrInverse) == 0 {
		t.Fatalf("L'en-tête de table de htop aurait dû comporter AttrInverse")
	}

	// 4. Rendu de la trame htop sur une Surface c2painter
	surface := c2painter.NewSurface(term.cols*term.cellWidth, term.rows*term.cellHeight)
	painter := c2painter.NewPainter(surface)
	term.Render(painter)

	if len(surface.Pixels) != surface.Width*surface.Height {
		t.Fatalf("Taille de surface incorrecte: %d pixels", len(surface.Pixels))
	}

	// 5. Quitter htop : retour au buffer principal
	term.Write([]byte("\x1b[?1049l\x1b[?25h"))
	if term.IsAlternateScreen() {
		t.Fatalf("Le mode Alternate Screen aurait dû être désactivé par \\x1b[?1049l")
	}
	if !term.cursorVisible {
		t.Fatalf("Le curseur aurait dû être réactivé par \\x1b[?25h")
	}
}

// Test 2 : Simulation fidèle de la sortie de neofetch
// (Logo ANSI multicolore, couleurs 24-bit TrueColor, styles Gras / Souligné / Italique / Barré)
func TestTerminal_NeofetchOutput(t *testing.T) {
	term := NewTerminalWidget(80, 24)

	var neoSeq bytes.Buffer
	// Titre en gras et cyan
	neoSeq.WriteString("\x1b[1;36muser\x1b[0m@\x1b[1;36mhoros55\x1b[0m\r\n")
	neoSeq.WriteString("-------------\r\n")

	// OS et Kernel avec couleurs TrueColor 24-bit
	neoSeq.WriteString("\x1b[38;2;255;128;0mOS\x1b[0m: Horos Linux x86_64\r\n")
	neoSeq.WriteString("\x1b[38;2;0;200;255mKernel\x1b[0m: 6.8.0-custom-simd\r\n")

	// Attributs typographiques variés
	neoSeq.WriteString("\x1b[1mGras\x1b[0m \x1b[3mItalique\x1b[0m \x1b[4mSouligné\x1b[0m \x1b[9mBarré\x1b[0m \x1b[2mFaible\x1b[0m\r\n")

	// Palette de 8 blocs de couleurs de fond
	neoSeq.WriteString("\x1b[40m   \x1b[41m   \x1b[42m   \x1b[43m   \x1b[44m   \x1b[45m   \x1b[46m   \x1b[47m   \x1b[0m\r\n")

	term.Write(neoSeq.Bytes())

	// Vérification de la ligne de styles (ligne index 4)
	cBold := term.CellAt(0, 4)
	if (cBold.Flags & AttrBold) == 0 {
		t.Fatalf("Cellule 'Gras' attendue avec AttrBold, obtenu flags=%d", cBold.Flags)
	}

	cItalic := term.CellAt(5, 4)
	if (cItalic.Flags & AttrItalic) == 0 {
		t.Fatalf("Cellule 'Italique' attendue avec AttrItalic, obtenu flags=%d", cItalic.Flags)
	}

	cUnder := term.CellAt(14, 4)
	if (cUnder.Flags & AttrUnderline) == 0 {
		t.Fatalf("Cellule 'Souligné' attendue avec AttrUnderline, obtenu flags=%d", cUnder.Flags)
	}

	cStrike := term.CellAt(23, 4)
	if (cStrike.Flags & AttrStrikethrough) == 0 {
		t.Fatalf("Cellule 'Barré' attendue avec AttrStrikethrough, obtenu flags=%d", cStrike.Flags)
	}

	cDim := term.CellAt(30, 4)
	if (cDim.Flags & AttrDim) == 0 {
		t.Fatalf("Cellule 'Faible' attendue avec AttrDim, obtenu flags=%d", cDim.Flags)
	}

	// Rendu sur tampon brut de pixels RGBA
	surface := c2painter.NewSurface(term.cols*term.cellWidth, term.rows*term.cellHeight)
	term.RenderToSurface(surface)
}

// Test 3 : Simulation de `ls --color=auto`
// (Runs de noms de fichiers avec attributs de couleurs réels)
func TestTerminal_LsColorOutput(t *testing.T) {
	term := NewTerminalWidget(80, 24)

	var lsSeq bytes.Buffer
	// Dossier en bleu gras
	lsSeq.WriteString("\x1b[1;34mcmd\x1b[0m  ")
	// Dossier en bleu gras
	lsSeq.WriteString("\x1b[1;34mpkg\x1b[0m  ")
	// Binaire exécutable en vert gras
	lsSeq.WriteString("\x1b[1;32mmain.bin\x1b[0m*  ")
	// Archive en rouge gras
	lsSeq.WriteString("\x1b[1;31marchive.tar.gz\x1b[0m  ")
	// Lien symbolique en cyan gras
	lsSeq.WriteString("\x1b[1;36mlink.so\x1b[0m@  ")
	// Fichier régulier sans couleur
	lsSeq.WriteString("README.md\r\n")

	term.Write(lsSeq.Bytes())

	// Col 0: 'c' de "cmd" en bleu gras
	cDir := term.CellAt(0, 0)
	if cDir.Rune != 'c' || (cDir.Flags&AttrBold) == 0 || cDir.Fg != 4 {
		t.Fatalf("Dossier attendu 'c' bleu gras (fg=4, bold), obtenu rune='%c' fg=%d flags=%d", cDir.Rune, cDir.Fg, cDir.Flags)
	}

	// Col 10: 'm' de "main.bin" en vert gras (fg=2)
	cExec := term.CellAt(10, 0)
	if cExec.Rune != 'm' || (cExec.Flags&AttrBold) == 0 || cExec.Fg != 2 {
		t.Fatalf("Exécutable attendu 'm' vert gras (fg=2, bold), obtenu rune='%c' fg=%d flags=%d", cExec.Rune, cExec.Fg, cExec.Flags)
	}
}

// Test 4 : Défilement massif de 100 000 lignes dans le scrollback ring
// (Performance, rétention en mémoire tampon circulaire, pas de fuite ni de panic)
func TestTerminal_ScrollbackMassive100kLines(t *testing.T) {
	term := NewTerminalWidget(80, 24)

	var chunk bytes.Buffer
	for i := 1; i <= 1000; i++ {
		fmt.Fprintf(&chunk, "Ligne de journal d'audit mecanique numero %06d : ok\r\n", i)
	}
	chunkBytes := chunk.Bytes()

	// Ingestion de 100 paquets de 1 000 lignes = 100 000 lignes
	for p := 0; p < 100; p++ {
		term.Write(chunkBytes)
	}

	count := term.ScrollbackCount()
	if count <= 0 {
		t.Fatalf("Le scrollback count devrait etre > 0, obtenu %d", count)
	}
	if count > 10000 {
		t.Fatalf("Le scrollback count depasse la capacite maximale de 10000, obtenu %d", count)
	}

	// Test de défilement vers le haut
	term.ScrollUp(50)
	if term.ScrollOffset() != 50 {
		t.Fatalf("ScrollOffset attendu 50, obtenu %d", term.ScrollOffset())
	}

	term.ScrollToTop()
	if term.ScrollOffset() != count {
		t.Fatalf("ScrollToTop attendu %d, obtenu %d", count, term.ScrollOffset())
	}

	term.ScrollDown(20)
	if term.ScrollOffset() != count-20 {
		t.Fatalf("ScrollDown attendu %d, obtenu %d", count-20, term.ScrollOffset())
	}

	term.ScrollToBottom()
	if term.ScrollOffset() != 0 {
		t.Fatalf("ScrollToBottom attendu 0, obtenu %d", term.ScrollOffset())
	}
}

// Test 5 : Caractères graphiques DEC VT100 et Box Drawing
func TestTerminal_DEC_VT100Graphics(t *testing.T) {
	term := NewTerminalWidget(80, 24)

	// Bascule en mode graphique DEC (\x1b(0), écriture d'une boîte, retour en ASCII (\x1b(B))
	// l = ┌, q = ─, k = ┐, x = │, m = └, j = ┘
	var decSeq bytes.Buffer
	decSeq.WriteString("\x1b(0")
	decSeq.WriteString("lqqqqk\r\n")
	decSeq.WriteString("x    x\r\n")
	decSeq.WriteString("mqqqqj\r\n")
	decSeq.WriteString("\x1b(B")
	decSeq.WriteString("Texte ASCII standard")

	term.Write(decSeq.Bytes())

	// Ligne 0 : ┌────┐
	if r := term.CellAt(0, 0).Rune; r != '┌' {
		t.Fatalf("Attendu '┌', obtenu '%c'", r)
	}
	if r := term.CellAt(1, 0).Rune; r != '─' {
		t.Fatalf("Attendu '─', obtenu '%c'", r)
	}
	if r := term.CellAt(5, 0).Rune; r != '┐' {
		t.Fatalf("Attendu '┐', obtenu '%c'", r)
	}

	// Ligne 1 : │    │
	if r := term.CellAt(0, 1).Rune; r != '│' {
		t.Fatalf("Attendu '│', obtenu '%c'", r)
	}
	if r := term.CellAt(5, 1).Rune; r != '│' {
		t.Fatalf("Attendu '│', obtenu '%c'", r)
	}

	// Ligne 2 : └────┘
	if r := term.CellAt(0, 2).Rune; r != '└' {
		t.Fatalf("Attendu '└', obtenu '%c'", r)
	}
	if r := term.CellAt(5, 2).Rune; r != '┘' {
		t.Fatalf("Attendu '┘', obtenu '%c'", r)
	}

	// Ligne 3 : Texte ASCII standard
	if r := term.CellAt(0, 3).Rune; r != 'T' {
		t.Fatalf("Attendu 'T', obtenu '%c'", r)
	}
}

// Test 6 : Sélection textuelle souris et Presse-papier OSC 52
func TestTerminal_SelectionAndClipboard(t *testing.T) {
	term := NewTerminalWidget(80, 24)

	term.Write([]byte("Premiere ligne de test\r\nDeuxieme ligne de test\r\nTroisieme ligne"))

	// Sélection des colonnes 0..7 sur les lignes 0..1
	term.StartSelection(0, 0)
	term.UpdateSelection(7, 1)
	term.EndSelection()

	if !term.HasSelection() {
		t.Fatalf("Une sélection active devrait être présente")
	}

	// Vérification de la sélection
	selected := term.GetSelectedText()
	if !strings.HasPrefix(selected, "Premiere") {
		t.Fatalf("Texte sélectionné inattendu: %q", selected)
	}

	// Copie de sélection
	var copiedViaCallback string
	term.OnClipboardCopy = func(text string) {
		copiedViaCallback = text
	}
	copied := term.CopySelection()
	if copied != selected {
		t.Fatalf("Mismatch CopySelection: %q vs %q", copied, selected)
	}
	if copiedViaCallback != selected {
		t.Fatalf("Callback OnClipboardCopy non déclenché ou valeur erronée: %q", copiedViaCallback)
	}

	// Test OSC 52 Ingestion directe (base64 de "Hello Antigravity!")
	// "Hello Antigravity!" en base64 = "SGVsbG8gQW50aWdyYXZpdHkh"
	osc52Payload := "\x1b]52;c;SGVsbG8gQW50aWdyYXZpdHkh\x07"
	term.Write([]byte(osc52Payload))

	if term.ClipboardText() != "Hello Antigravity!" {
		t.Fatalf("Presse-papier OSC 52 attendu 'Hello Antigravity!', obtenu %q", term.ClipboardText())
	}
}

// Test 7 : Intégration Police TrueType tt55 réelle (zéro mock)
func TestTerminal_TrueTypeFontIntegration(t *testing.T) {
	term := NewTerminalWidget(80, 24)

	// Recherche d'une police système TTF réelle
	fontPath := "/usr/share/fonts/truetype/ubuntu/UbuntuMono-B.ttf"
	if _, err := os.Stat(fontPath); err != nil {
		fontPath = "/usr/share/fonts/truetype/ubuntu/Ubuntu-R.ttf"
	}
	fontData, err := os.ReadFile(fontPath)
	if err != nil {
		t.Skipf("Police système TTF absente pour le test: %v", err)
	}

	var font tt55.Tt55_font
	ret := tt55.Tt55_open(fontData, uint64(len(fontData)), &font)
	if ret != 0 {
		t.Fatalf("Tt55_open a échoué avec le code %d", ret)
	}

	term.SetFont(&font, 9, 18)

	if term.Font() != &font {
		t.Fatalf("La police tt55 n'a pas été enregistrée")
	}

	w, h := term.CellSize()
	if w != 9 || h != 18 {
		t.Fatalf("Dimensions de cellule attendues (9, 18), obtenu (%d, %d)", w, h)
	}

	// Vérification de la résolution de glyphes via tt55
	var gid uint16
	if tt55.Tt55_glyph(&font, uint32('A'), &gid) == 0 {
		var adv uint16
		tt55.Tt55_advance(&font, gid, &adv)
		if adv == 0 {
			t.Fatalf("Advance width pour 'A' devrait être non nulle")
		}
	}
}

// Benchmark 1 : Débit d'ingestion texte ASCII brut (prouvant 0 alloc/op)
func BenchmarkTerminalFeed_AsciiText(b *testing.B) {
	term := NewTerminalWidget(80, 24)
	data := []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor.\r\n")

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		term.Feed(data)
	}
}

// Benchmark 2 : Débit d'ingestion avec séquences de couleurs ANSI et styles
func BenchmarkTerminalFeed_AnsiColorRuns(b *testing.B) {
	term := NewTerminalWidget(80, 24)
	data := []byte("\x1b[1;31m[ERROR]\x1b[0m \x1b[38;2;100;200;255mTraitement du paquet #4294967295\x1b[0m termine avec succes.\r\n")

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		term.Feed(data)
	}
}

// Benchmark 3 : Débit de rafraîchissement d'une trame complète htop
func BenchmarkTerminalFeed_HtopSnapshot(b *testing.B) {
	term := NewTerminalWidget(80, 24)
	var htopSeq bytes.Buffer
	htopSeq.WriteString("\x1b[H\x1b[2J")
	for row := 1; row <= 24; row++ {
		fmt.Fprintf(&htopSeq, "\x1b[%d;1H\x1b[1;32mCPU %02d\x1b[0m [\x1b[38;5;46m||||||||||||||||\x1b[0m 64.2%%] \x1b[7m Proc %04d \x1b[0m", row, row, row*100)
	}
	snapshot := htopSeq.Bytes()

	b.SetBytes(int64(len(snapshot)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		term.Feed(snapshot)
	}
}

// Benchmark 4 : Débit de rendu d'une trame pleine (Full Frame Render FPS et 0 B/op)
func BenchmarkTerminalRender_FullFrame(b *testing.B) {
	term := NewTerminalWidget(80, 24)
	term.Write([]byte("\x1b[1;32mInitialisation du terminal matriciel haute cadence.\x1b[0m\r\n"))
	for i := 0; i < 22; i++ {
		term.Write([]byte("\x1b[38;2;120;180;240m[SYSTEM]\x1b[0m Ligne de rendu dense avec caracteres graphiques DEC ┌───┐ et styles.\r\n"))
	}

	w, h := term.cols*term.cellWidth, term.rows*term.cellHeight
	surface := c2painter.NewSurface(w, h)
	painter := c2painter.NewPainter(surface)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		term.Render(painter)
	}
}

// Benchmark 5 : Ingestion et défilement intensif
func BenchmarkTerminalScroll_100kLines(b *testing.B) {
	term := NewTerminalWidget(80, 24)
	line := []byte("Message d'evenement d'audit horos55 en streaming haute cadence.\r\n")

	b.SetBytes(int64(len(line)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		term.Feed(line)
	}
}
