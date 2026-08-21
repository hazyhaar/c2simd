package c2fynedriver

import (
	"testing"

	c2display "code.hazyhaar.fr/devhoros/pkg/c2display"
	c2painter "code.hazyhaar.fr/devhoros/pkg/c2painter"
)

func TestDriver_LifecycleAndWindow(t *testing.T) {
	drv, err := NewHeadlessDriver(1024, 768)
	if err != nil {
		t.Fatalf("NewHeadlessDriver échoué: %v", err)
	}
	defer drv.Quit()

	win, err := drv.CreateWindow("Fyne CGO-Free Test", 800, 600)
	if err != nil {
		t.Fatalf("CreateWindow échoué: %v", err)
	}

	if win.Title() != "Fyne CGO-Free Test" {
		t.Fatalf("Titre inattendu: %s", win.Title())
	}

	win.SetTitle("Titre Modifié")
	if win.Title() != "Titre Modifié" {
		t.Fatalf("Titre après modification inattendu: %s", win.Title())
	}

	sz := win.Size()
	if sz.Width != 800 || sz.Height != 600 {
		t.Fatalf("Dimensions inattendues: %dx%d", sz.Width, sz.Height)
	}

	canvas := win.Canvas()
	if canvas == nil {
		t.Fatalf("Canvas est nul")
	}

	// Définition d'un contenu racine
	root := NewContainer()
	rect := NewRectangle(c2painter.PackRGBA(255, 0, 0, 255))
	rect.Move(Position{X: 10, Y: 10})
	rect.Resize(Size{Width: 100, Height: 50})
	root.Add(rect)

	win.SetContent(root)
	if win.Content() != root {
		t.Fatalf("Contenu racine non configuré")
	}

	// Rendu d'une trame
	win.RenderFrame()

	hWin, ok := win.displayWin.(*c2display.HeadlessWindow)
	if !ok {
		t.Fatalf("displayWin n'est pas un *HeadlessWindow")
	}

	if hWin.FrameCount() < 1 {
		t.Fatalf("FrameCount attendu >= 1, obtenu %d", hWin.FrameCount())
	}

	lastSurf := hWin.LastSurface()
	if lastSurf == nil || lastSurf.Width != 800 || lastSurf.Height != 600 {
		t.Fatalf("Surface présentée invalide")
	}

	// Fermeture de fenêtre
	var closedCalled bool
	win.SetOnClosed(func() {
		closedCalled = true
	})

	win.Close()
	if !closedCalled {
		t.Fatalf("Rappel OnClosed non exécuté")
	}
}

func TestCanvas_WalkAndPaint_AllPrimitives(t *testing.T) {
	drv, err := NewHeadlessDriver(960, 600)
	if err != nil {
		t.Fatalf("NewHeadlessDriver échoué: %v", err)
	}
	defer drv.Quit()

	win, err := drv.CreateWindow("Demo Primitives", 960, 600)
	if err != nil {
		t.Fatalf("CreateWindow échoué: %v", err)
	}
	defer win.Close()

	canvas := win.Canvas()

	// Construction d'un arbre riche avec toutes les primitives
	root := NewContainer()

	// 1. Rectangle plein et bordé
	r1 := NewRectangle(c2painter.PackRGBA(59, 130, 246, 255))
	r1.Move(Position{X: 20, Y: 20})
	r1.Resize(Size{Width: 120, Height: 60})
	r1.StrokeColor = c2painter.PackRGBA(255, 255, 255, 255)
	r1.StrokeWidth = 2
	root.Add(r1)

	// 2. Rectangle arrondi avec ombre
	rr := NewRoundedRectangle(c2painter.PackRGBA(16, 185, 129, 255), 10)
	rr.Move(Position{X: 160, Y: 20})
	rr.Resize(Size{Width: 120, Height: 60})
	rr.Elevation = 3
	rr.ShadowColor = c2painter.PackRGBA(0, 0, 0, 100)
	root.Add(rr)

	// 3. Cercle plein et contour
	circ := NewCircle(c2painter.PackRGBA(239, 68, 68, 255))
	circ.Move(Position{X: 300, Y: 20})
	circ.Resize(Size{Width: 60, Height: 60})
	circ.StrokeColor = c2painter.PackRGBA(255, 255, 255, 200)
	circ.StrokeWidth = 2
	root.Add(circ)

	// 4. Ligne
	line := NewLine(Position{X: 380, Y: 20}, Position{X: 480, Y: 80}, c2painter.PackRGBA(234, 179, 8, 255), 3)
	root.Add(line)

	// 5. Texte / Label
	lbl := NewLabel("CGO-FREE LINUX DRIVER")
	lbl.Move(Position{X: 500, Y: 30})
	lbl.TextSize = 16
	lbl.Color = c2painter.PackRGBA(248, 250, 252, 255)
	root.Add(lbl)

	// 6. Badge
	badge := NewBadge("STATUT ACTIF", c2painter.PackRGBA(16, 185, 129, 255), c2painter.PackRGBA(255, 255, 255, 255))
	badge.Move(Position{X: 740, Y: 25})
	badge.Resize(badge.MinSize())
	root.Add(badge)

	// 7. Bouton interactif
	var btnClicked bool
	btn := NewButton("Actionner", func() {
		btnClicked = true
	})
	btn.Move(Position{X: 20, Y: 100})
	btn.Resize(Size{Width: 140, Height: 40})
	root.Add(btn)

	// 8. Carte avec contenu composite
	cardContent := NewContainer()
	cardLbl := NewLabel("Détail interne à la carte")
	cardLbl.Move(Position{X: 10, Y: 10})
	cardContent.Add(cardLbl)

	card := NewCard("Moniteur Système", "Ressources matérielles", cardContent)
	card.Move(Position{X: 180, Y: 100})
	card.Resize(Size{Width: 280, Height: 150})
	card.HeaderBadge = "TEMPS RÉEL"
	card.BadgeBg = c2painter.PackRGBA(59, 130, 246, 255)
	card.BadgeFg = c2painter.PackRGBA(255, 255, 255, 255)
	root.Add(card)

	// 9. Widget Terminal interactif
	termComp := NewTerminalComponent(40, 10)
	termComp.Move(Position{X: 20, Y: 270})
	termComp.Resize(termComp.MinSize())
	_, _ = termComp.Term.Write([]byte("\x1b[1;32m[OK]\x1b[0m Terminal c2fyneterm démarré.\r\n"))
	root.Add(termComp)

	// 10. Compteur FPS
	fpsCounter := NewFPSCounter(drv)
	fpsCounter.Move(Position{X: 800, Y: 550})
	root.Add(fpsCounter)

	canvas.SetContent(root)

	// Validation du parcours Walk
	walkCount := 0
	canvas.Walk(func(obj CanvasObject, absPos Position, size Size) bool {
		walkCount++
		if !obj.Visible() {
			t.Errorf("Objet invisible visité: %T", obj)
		}
		return true
	})

	if walkCount < 10 {
		t.Fatalf("Nombre d'objets visités insuffisant: %d", walkCount)
	}

	// Validation du rendu Paint
	surf := c2painter.NewSurface(960, 600)
	p := c2painter.NewPainter(surf)
	canvas.Paint(p)

	// Vérifier que la surface n'est pas vierge
	hasNonZeroPixel := false
	for _, px := range surf.Pixels {
		if px != canvas.clearColor {
			hasNonZeroPixel = true
			break
		}
	}
	if !hasNonZeroPixel {
		t.Fatalf("Le rendu Paint n'a modifié aucun pixel")
	}

	// Test interaction bouton
	canvas.HandleMouseEvent(c2display.MouseEvent{
		Action: c2display.EventMouseMove,
		X:      30,
		Y:      110,
	})
	if !btn.isHovered {
		t.Errorf("Le bouton devrait être survolé")
	}

	canvas.HandleMouseEvent(c2display.MouseEvent{
		Action: c2display.EventMousePress,
		Button: c2display.ButtonLeft,
		X:      30,
		Y:      110,
	})
	if !btn.isPressed {
		t.Errorf("Le bouton devrait être enfoncé")
	}

	canvas.HandleMouseEvent(c2display.MouseEvent{
		Action: c2display.EventMouseRelease,
		Button: c2display.ButtonLeft,
		X:      30,
		Y:      110,
	})
	if !btnClicked {
		t.Errorf("Le rappel OnTapped du bouton n'a pas été déclenché")
	}

	// Test entrée clavier sur Terminal
	canvas.SetFocused(termComp)
	canvas.HandleKeyEvent(c2display.KeyEvent{
		Action: c2display.EventKeyPress,
		Key:    c2display.KeyA,
		Rune:   'A',
	})

	cell := termComp.Term.CellAt(0, 1)
	if cell.Rune != 'A' {
		t.Errorf("Caractère terminal attendu 'A', obtenu '%c'", cell.Rune)
	}
}

func TestApp_FacotryAndMetrics(t *testing.T) {
	drv, err := NewHeadlessDriver(800, 600)
	if err != nil {
		t.Fatalf("NewHeadlessDriver échoué: %v", err)
	}

	app := NewAppWithDriver(drv)
	if app.Driver() != drv {
		t.Fatalf("Driver app mismatch")
	}

	win := app.NewWindow("Fenetre 1")
	if win == nil {
		t.Fatalf("NewWindow retourné nil")
	}

	// Vérification des métriques FPS
	fps := drv.CurrentFPS()
	if fps <= 0.0 {
		t.Errorf("FPS mesuré invalide: %f", fps)
	}

	dur := drv.LastFrameDurationUs()
	if dur < 0 {
		t.Errorf("Durée de frame invalide: %d", dur)
	}

	// Arrêt propre
	app.Quit()
}

func TestFontManager_MeasureAndRaster(t *testing.T) {
	w, h := MeasureString("TEST", 16)
	if w <= 0 || h <= 0 {
		t.Fatalf("Mesure typographique invalide: %dx%d", w, h)
	}

	fm := DefaultFontManager()
	mask := fm.GetGlyphMask('A', 20)
	if mask == nil || len(mask.Mask) == 0 {
		t.Fatalf("Masque de glyphe vide pour 'A'")
	}
	if mask.Width <= 0 || mask.Height <= 0 {
		t.Fatalf("Dimensions du masque invalides: %dx%d", mask.Width, mask.Height)
	}

	// Vérifier la présence de pixels opaques dans le masque
	hasOpaque := false
	for _, b := range mask.Mask {
		if b > 0 {
			hasOpaque = true
			break
		}
	}
	if !hasOpaque {
		t.Fatalf("Le masque pour 'A' ne contient aucun pixel actif")
	}
}

func TestContainers_HBoxAndVBox(t *testing.T) {
	lbl1 := NewLabel("Label 1")
	lbl2 := NewLabel("Label 2")
	hbox := NewHBox(10, lbl1, lbl2)

	msH := hbox.MinSize()
	if msH.Width <= 0 || msH.Height <= 0 {
		t.Errorf("MinSize HBox invalide: %dx%d", msH.Width, msH.Height)
	}

	lbl3 := NewLabel("Label 3")
	lbl4 := NewLabel("Label 4")
	vbox := NewVBox(5, lbl3, lbl4)

	msV := vbox.MinSize()
	if msV.Width <= 0 || msV.Height <= 0 {
		t.Errorf("MinSize VBox invalide: %dx%d", msV.Width, msV.Height)
	}

	// Test visibilité
	lbl3.Hide()
	if lbl3.Visible() {
		t.Errorf("lbl3 devrait être masqué")
	}
	lbl3.Show()
	if !lbl3.Visible() {
		t.Errorf("lbl3 devrait être visible")
	}
}

func TestTerminalComponent_ControlKeys(t *testing.T) {
	tc := NewTerminalComponent(40, 10)
	tc.OnKeyEvent(c2display.KeyEvent{Action: c2display.EventKeyPress, Key: c2display.KeyEnter})
	tc.OnKeyEvent(c2display.KeyEvent{Action: c2display.EventKeyPress, Key: c2display.KeyBackspace})
	tc.OnKeyEvent(c2display.KeyEvent{Action: c2display.EventKeyPress, Key: c2display.KeyTab})
	tc.OnKeyEvent(c2display.KeyEvent{Action: c2display.EventKeyPress, Key: c2display.KeyUp})
	tc.OnKeyEvent(c2display.KeyEvent{Action: c2display.EventKeyPress, Key: c2display.KeyDown})
	tc.OnKeyEvent(c2display.KeyEvent{Action: c2display.EventKeyPress, Key: c2display.KeyLeft})
	tc.OnKeyEvent(c2display.KeyEvent{Action: c2display.EventKeyPress, Key: c2display.KeyRight})

	// Doit s'exécuter sans panique
	if tc.Term == nil {
		t.Errorf("Term est nul")
	}
}

func TestWindow_DynamicResize(t *testing.T) {
	drv, err := NewHeadlessDriver(800, 600)
	if err != nil {
		t.Fatalf("NewHeadlessDriver échoué: %v", err)
	}
	defer drv.Quit()

	win, err := drv.CreateWindow("Test Resize", 400, 300)
	if err != nil {
		t.Fatalf("CreateWindow échoué: %v", err)
	}
	defer win.Close()

	win.Resize(Size{Width: 1024, Height: 768})
	if win.Size().Width != 1024 || win.Size().Height != 768 {
		t.Errorf("Dimensions de fenêtre après Resize inattendues: %dx%d", win.Size().Width, win.Size().Height)
	}

	win.RenderFrame()

	hWin := win.displayWin.(*c2display.HeadlessWindow)
	last := hWin.LastSurface()
	if last.Width != 1024 || last.Height != 768 {
		t.Errorf("Dimensions de surface après Resize inattendues: %dx%d", last.Width, last.Height)
	}
}
