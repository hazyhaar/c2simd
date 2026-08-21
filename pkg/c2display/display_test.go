package c2display

import (
	"os"
	"sync"
	"testing"
	"time"

	c2painter "code.hazyhaar.fr/devhoros/pkg/c2painter"
)

func TestHeadlessDisplay(t *testing.T) {
	disp, err := NewHeadlessDisplay(DisplayOptions{
		HeadlessWidth:  1280,
		HeadlessHeight: 720,
	})
	if err != nil {
		t.Fatalf("échec d'instanciation HeadlessDisplay: %v", err)
	}
	defer disp.Close()

	if disp.Type() != BackendHeadless {
		t.Errorf("type attendu BackendHeadless, obtenu %s", disp.Type())
	}

	screens := disp.Screens()
	if len(screens) == 0 {
		t.Fatal("aucun écran retourné par HeadlessDisplay")
	}
	if screens[0].Width != 1280 || screens[0].Height != 720 {
		t.Errorf("dimensions d'écran inattendues: %dx%d", screens[0].Width, screens[0].Height)
	}

	win, err := disp.CreateWindow(WindowOptions{
		Title:  "Test Virtuel",
		Width:  640,
		Height: 480,
	})
	if err != nil {
		t.Fatalf("échec création fenêtre headless: %v", err)
	}
	defer win.Close()

	if win.Title() != "Test Virtuel" {
		t.Fatalf("titre attendu 'Test Virtuel', obtenu '%s'", win.Title())
	}

	w, h := win.Bounds()
	if w != 640 || h != 480 {
		t.Errorf("géométrie inattendue: %dx%d", w, h)
	}

	if err := win.SetTitle("Nouveau Titre"); err != nil {
		t.Fatalf("échec SetTitle: %v", err)
	}
	if win.Title() != "Nouveau Titre" {
		t.Errorf("titre après modification inattendu: %s", win.Title())
	}

	// Événement Expose initial
	ev := win.PollEvent()
	if ev == nil || ev.Type() != EventExpose {
		t.Errorf("événement Expose initial attendu, obtenu: %v", ev)
	}

	// Dessin et présentation avec c2painter
	surf := c2painter.NewSurface(640, 480)
	p := c2painter.NewPainter(surf)
	p.Clear(c2painter.PackRGBA(30, 30, 30, 255))
	p.DrawRect(50, 50, 200, 100, c2painter.PackRGBA(255, 0, 0, 255))
	p.DrawCircle(320, 240, 60, c2painter.PackRGBA(0, 255, 0, 255))

	if err := win.Present(surf); err != nil {
		t.Fatalf("échec de présentation surface: %v", err)
	}

	hw, ok := win.(*HeadlessWindow)
	if !ok {
		t.Fatal("type de fenêtre headless invalide")
	}

	if hw.FrameCount() != 1 {
		t.Errorf("nombre de frames attendu 1, obtenu %d", hw.FrameCount())
	}

	last := hw.LastSurface()
	if last == nil || last.Width != 640 || last.Height != 480 {
		t.Fatal("surface capturée invalide")
	}

	// Redimensionnement
	if err := win.Resize(800, 600); err != nil {
		t.Fatalf("échec Resize: %v", err)
	}
	rw, rh := win.Bounds()
	if rw != 800 || rh != 600 {
		t.Errorf("dimensions après redimensionnement inattendues: %dx%d", rw, rh)
	}

	resEv := win.PollEvent()
	if resEv == nil || resEv.Type() != EventResize {
		t.Errorf("événement Resize attendu, obtenu: %v", resEv)
	}

	// Injection d'événements
	hw.InjectEvent(MouseEvent{
		BaseEvent: BaseEvent{Time: 100},
		Action:    EventMousePress,
		Button:    ButtonLeft,
		X:         150,
		Y:         200,
	})

	hw.InjectEvent(KeyEvent{
		BaseEvent: BaseEvent{Time: 200},
		Action:    EventKeyPress,
		Key:       KeyA,
		Rune:      'a',
	})

	mev := win.PollEvent()
	if mev == nil || mev.Type() != EventMousePress {
		t.Errorf("événement MousePress attendu, obtenu: %v", mev)
	}

	kev := win.WaitEvent()
	if kev == nil || kev.Type() != EventKeyPress {
		t.Errorf("événement KeyPress attendu, obtenu: %v", kev)
	}
}

func TestAutoDetectionOpen(t *testing.T) {
	disp, err := Open(DisplayOptions{
		Backend: BackendAuto,
	})
	if err != nil {
		t.Fatalf("Open(BackendAuto) a échoué: %v", err)
	}
	defer disp.Close()

	if disp.Type() == "" {
		t.Fatal("type de display indéterminé")
	}

	win, err := disp.CreateWindow(WindowOptions{
		Title:  "Auto Détection Test",
		Width:  400,
		Height: 300,
	})
	if err != nil {
		t.Fatalf("CreateWindow a échoué: %v", err)
	}
	defer win.Close()

	surf := c2painter.NewSurface(400, 300)
	p := c2painter.NewPainter(surf)
	p.Clear(c2painter.PackRGBA(15, 23, 42, 255))
	p.DrawRoundedRect(40, 40, 320, 220, 16, c2painter.PackRGBA(56, 189, 248, 255))

	if err := win.Present(surf); err != nil {
		t.Fatalf("Present a échoué: %v", err)
	}
}

func TestX11DisplayLive(t *testing.T) {
	dispEnv := os.Getenv("DISPLAY")
	if dispEnv == "" {
		t.Skip("DISPLAY non configuré, saut du test X11 en direct")
	}

	disp, err := NewX11Display(DisplayOptions{
		DisplayName: dispEnv,
	})
	if err != nil {
		t.Skipf("serveur X11 inaccessible (%v), saut du test X11 en direct", err)
	}
	defer disp.Close()

	if disp.Type() != BackendX11 {
		t.Errorf("type attendu BackendX11, obtenu %s", disp.Type())
	}

	screens := disp.Screens()
	if len(screens) == 0 {
		t.Fatal("aucun écran retourné par X11Display")
	}
	if screens[0].Width <= 0 || screens[0].Height <= 0 {
		t.Errorf("dimensions d'écran X11 invalides: %dx%d", screens[0].Width, screens[0].Height)
	}

	win, err := disp.CreateWindow(WindowOptions{
		Title:  "c2display X11 Validation Test",
		Width:  500,
		Height: 400,
		X:      100,
		Y:      100,
	})
	if err != nil {
		t.Fatalf("échec de création de fenêtre X11: %v", err)
	}
	defer win.Close()

	// Dessin de validation avec c2painter
	surf := c2painter.NewSurface(500, 400)
	p := c2painter.NewPainter(surf)
	// Fond sombre
	p.Clear(c2painter.PackRGBA(18, 18, 24, 255))
	// Barre d'en-tête
	p.DrawRect(0, 0, 500, 40, c2painter.PackRGBA(30, 41, 59, 255))
	// Cartes d'affichage
	p.DrawRoundedRect(30, 60, 200, 120, 8, c2painter.PackRGBA(239, 68, 68, 255))
	p.DrawRoundedRect(260, 60, 200, 120, 8, c2painter.PackRGBA(34, 197, 94, 255))
	// Cercle central
	p.DrawCircle(250, 280, 50, c2painter.PackRGBA(59, 130, 246, 255))
	// Ligne décorative
	p.DrawLine(50, 360, 450, 360, 3, c2painter.PackRGBA(234, 179, 8, 255))

	// Présentation ZPixmap
	if err := win.Present(surf); err != nil {
		t.Fatalf("échec de présentation X11 PutImage: %v", err)
	}

	// Modification du titre
	if err := win.SetTitle("c2display X11 - Rendu Validé"); err != nil {
		t.Errorf("échec SetTitle X11: %v", err)
	}

	// Redimensionnement
	if err := win.Resize(600, 450); err != nil {
		t.Errorf("échec Resize X11: %v", err)
	}

	// Présentation sur la nouvelle taille
	surf2 := c2painter.NewSurface(600, 450)
	p2 := c2painter.NewPainter(surf2)
	p2.Clear(c2painter.PackRGBA(10, 15, 30, 255))
	p2.DrawRoundedRect(50, 50, 500, 350, 12, c2painter.PackRGBA(168, 85, 247, 255))

	if err := win.Present(surf2); err != nil {
		t.Fatalf("échec de présentation X11 après redimensionnement: %v", err)
	}

	// Dépouillement des événements en attente
	time.Sleep(50 * time.Millisecond)
	for {
		ev := win.PollEvent()
		if ev == nil {
			break
		}
	}
}

func TestX11MultiChunkPutImage(t *testing.T) {
	dispEnv := os.Getenv("DISPLAY")
	if dispEnv == "" {
		t.Skip("DISPLAY non configuré, saut du test")
	}

	disp, err := NewX11Display(DisplayOptions{
		DisplayName: dispEnv,
	})
	if err != nil {
		t.Skipf("serveur X11 inaccessible (%v)", err)
	}
	defer disp.Close()

	// Grande surface (1024x768 = 786 432 pixels = 3.14 Mo)
	// Nécessite impérativement le chunking automatique pour ne pas dépasser maxRequestBytes
	win, err := disp.CreateWindow(WindowOptions{
		Title:  "c2display X11 Grand Tampon",
		Width:  1024,
		Height: 768,
	})
	if err != nil {
		t.Fatalf("CreateWindow grand format: %v", err)
	}
	defer win.Close()

	surf := c2painter.NewSurface(1024, 768)
	p := c2painter.NewPainter(surf)
	p.Clear(c2painter.PackRGBA(20, 20, 25, 255))
	p.DrawLinearGradient(0, 0, 1024, 768, c2painter.PackRGBA(15, 23, 42, 255), c2painter.PackRGBA(88, 28, 135, 255), true)

	if err := win.Present(surf); err != nil {
		t.Fatalf("échec de présentation PutImage grand tampon segmenté: %v", err)
	}
}

func TestEventDispatchConcurrence(t *testing.T) {
	disp, err := NewHeadlessDisplay(DisplayOptions{})
	if err != nil {
		t.Fatalf("NewHeadlessDisplay: %v", err)
	}
	defer disp.Close()

	win, err := disp.CreateWindow(WindowOptions{Width: 400, Height: 300})
	if err != nil {
		t.Fatalf("CreateWindow: %v", err)
	}
	defer win.Close()

	hw := win.(*HeadlessWindow)
	const numEvents = 500
	var wg sync.WaitGroup

	// Producteur concurrent
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numEvents; i++ {
			hw.InjectEvent(MouseEvent{
				BaseEvent: BaseEvent{Time: time.Duration(i)},
				Action:    EventMouseMove,
				X:         i,
				Y:         i,
			})
		}
	}()

	// Consommateur concurrent
	wg.Add(1)
	go func() {
		defer wg.Done()
		count := 0
		for count < numEvents/2 {
			ev := win.PollEvent()
			if ev != nil {
				count++
			} else {
				time.Sleep(100 * time.Microsecond)
			}
		}
	}()

	wg.Wait()
}

func TestKeySymMapping(t *testing.T) {
	// Lettres
	k, r := KeySymToKeyCode('a')
	if k != KeyA || r != 'a' {
		t.Errorf("KeySym 'a': obtenu (%v, %c)", k, r)
	}
	k, r = KeySymToKeyCode('Z')
	if k != KeyZ || r != 'Z' {
		t.Errorf("KeySym 'Z': obtenu (%v, %c)", k, r)
	}

	// Touches spéciales X11
	k, r = KeySymToKeyCode(0xff0d) // XK_Return
	if k != KeyEnter || r != '\r' {
		t.Errorf("XK_Return: obtenu (%v, %v)", k, r)
	}

	k, _ = KeySymToKeyCode(0xff1b) // XK_Escape
	if k != KeyEscape {
		t.Errorf("XK_Escape: obtenu %v", k)
	}

	k, _ = KeySymToKeyCode(0xff52) // XK_Up
	if k != KeyUp {
		t.Errorf("XK_Up: obtenu %v", k)
	}

	// Fonctions F1..F12
	k, _ = KeySymToKeyCode(0xffbe) // F1
	if k != KeyF1 {
		t.Errorf("XK_F1: obtenu %v", k)
	}
	k, _ = KeySymToKeyCode(0xffc9) // F12
	if k != KeyF12 {
		t.Errorf("XK_F12: obtenu %v", k)
	}

	// Pavé numérique
	k, r = KeySymToKeyCode(0xffb5) // KP_5
	if k != KeyKp5 || r != '5' {
		t.Errorf("KP_5: obtenu (%v, %c)", k, r)
	}

	// Modificateurs
	k, _ = KeySymToKeyCode(0xffe1) // Shift_L
	if k != KeyShiftLeft {
		t.Errorf("Shift_L: obtenu %v", k)
	}
	k, _ = KeySymToKeyCode(0xffe3) // Control_L
	if k != KeyControlLeft {
		t.Errorf("Control_L: obtenu %v", k)
	}

	// Scancodes Linux
	k, _ = EvdevScancodeToKeyCode(1) // Escape
	if k != KeyEscape {
		t.Errorf("Scancode 1: obtenu %v", k)
	}
	k, _ = EvdevScancodeToKeyCode(28) // Enter
	if k != KeyEnter {
		t.Errorf("Scancode 28: obtenu %v", k)
	}
	k, _ = EvdevScancodeToKeyCode(57) // Space
	if k != KeySpace {
		t.Errorf("Scancode 57: obtenu %v", k)
	}
}

func TestWindowErrorHandling(t *testing.T) {
	disp, err := NewHeadlessDisplay(DisplayOptions{})
	if err != nil {
		t.Fatalf("NewHeadlessDisplay: %v", err)
	}

	win, err := disp.CreateWindow(WindowOptions{Width: 300, Height: 200})
	if err != nil {
		t.Fatalf("CreateWindow: %v", err)
	}

	// Present avec surface nulle
	if err := win.Present(nil); err != ErrNilSurface {
		t.Errorf("ErrNilSurface attendu, obtenu %v", err)
	}

	// Resize invalide
	if err := win.Resize(-10, 50); err != ErrInvalidGeometry {
		t.Errorf("ErrInvalidGeometry attendu, obtenu %v", err)
	}

	// Fermeture de la fenêtre
	if err := win.Close(); err != nil {
		t.Fatalf("Close win: %v", err)
	}

	// Opérations post-fermeture
	if err := win.SetTitle("Après Fermeture"); err != ErrWindowClosed {
		t.Errorf("ErrWindowClosed attendu sur SetTitle, obtenu %v", err)
	}
	if err := win.Resize(100, 100); err != ErrWindowClosed {
		t.Errorf("ErrWindowClosed attendu sur Resize, obtenu %v", err)
	}
	if err := win.Present(c2painter.NewSurface(100, 100)); err != ErrWindowClosed {
		t.Errorf("ErrWindowClosed attendu sur Present, obtenu %v", err)
	}

	// Fermeture du display
	if err := disp.Close(); err != nil {
		t.Fatalf("Close display: %v", err)
	}
	if _, err := disp.CreateWindow(WindowOptions{}); err != ErrDisplayClosed {
		t.Errorf("ErrDisplayClosed attendu sur CreateWindow après Close, obtenu %v", err)
	}
}

func TestAnonymousMemFDCreation(t *testing.T) {
	f, fd, err := createAnonymousMemFD("c2display_test_memfd", 4096)
	if err != nil {
		t.Fatalf("échec de création memfd: %v", err)
	}
	if fd < 0 || f == nil {
		t.Fatalf("fd invalide: %d (file: %v)", fd, f)
	}
	_ = f.Close()
}

func TestModifiersConversion(t *testing.T) {
	mods := ConvertX11Modifiers(1 | 4 | 8 | 64)
	if (mods & ModShift) == 0 {
		t.Error("ModShift manquant")
	}
	if (mods & ModControl) == 0 {
		t.Error("ModControl manquant")
	}
	if (mods & ModAlt) == 0 {
		t.Error("ModAlt manquant")
	}
	if (mods & ModSuper) == 0 {
		t.Error("ModSuper manquant")
	}
}

func TestItoaUtility(t *testing.T) {
	cases := map[int]string{
		0:      "0",
		1:      "1",
		42:     "42",
		1000:   "1000",
		-5:     "-5",
		-12345: "-12345",
	}
	for val, expected := range cases {
		if res := itoa(val); res != expected {
			t.Errorf("itoa(%d): attendu %s, obtenu %s", val, expected, res)
		}
	}
}
