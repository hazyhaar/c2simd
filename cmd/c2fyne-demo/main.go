package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	c2fynedriver "code.hazyhaar.fr/devhoros/pkg/c2fynedriver"
	c2painter "code.hazyhaar.fr/devhoros/pkg/c2painter"
)

func main() {
	headlessFlag := flag.Bool("headless", false, "Exécute la démonstration en mode virtuel headless (sans affichage)")
	benchFrames := flag.Int("bench-frames", 0, "Nombre de trames à rendre avant arrêt automatique (pour benchmarks/tests)")
	flag.Parse()

	var drv *c2fynedriver.Driver
	var err error

	if *headlessFlag {
		drv, err = c2fynedriver.NewHeadlessDriver(960, 600)
	} else {
		drv, err = c2fynedriver.NewDriver()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur initialisation Driver: %v\n", err)
		os.Exit(1)
	}

	win, err := drv.CreateWindow("c2fyne — Démonstration Pilote Fyne CGO-Free", 960, 600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur création fenêtre: %v\n", err)
		os.Exit(1)
	}

	// Arbre graphique principal
	root := c2fynedriver.NewContainer()

	// 1. Barre d'en-tête (Header)
	headerBg := c2fynedriver.NewRectangle(c2painter.PackRGBA(15, 23, 42, 255)) // Slate 900
	headerBg.Move(c2fynedriver.Position{X: 0, Y: 0})
	headerBg.Resize(c2fynedriver.Size{Width: 960, Height: 64})
	root.Add(headerBg)

	headerLine := c2fynedriver.NewLine(
		c2fynedriver.Position{X: 0, Y: 64},
		c2fynedriver.Position{X: 960, Y: 64},
		c2painter.PackRGBA(30, 41, 59, 255),
		1,
	)
	root.Add(headerLine)

	titleLbl := c2fynedriver.NewLabel("HOROS // FYNE-SGOITER")
	titleLbl.Move(c2fynedriver.Position{X: 20, Y: 14})
	titleLbl.TextSize = 16
	titleLbl.Color = c2painter.PackRGBA(248, 250, 252, 255)
	root.Add(titleLbl)

	subtitleLbl := c2fynedriver.NewLabel("Moteur de rendu vectoriel SIMD en pur Go (Zéro CGO)")
	subtitleLbl.Move(c2fynedriver.Position{X: 20, Y: 36})
	subtitleLbl.TextSize = 11
	subtitleLbl.Color = c2painter.PackRGBA(148, 163, 184, 255)
	root.Add(subtitleLbl)

	// Badge de statut "CGO-FREE LINUX DRIVER"
	statusBadge := c2fynedriver.NewBadge(
		"CGO-FREE LINUX DRIVER",
		c2painter.PackRGBA(16, 185, 129, 255), // Emerald 500
		c2painter.PackRGBA(255, 255, 255, 255),
	)
	badgeMin := statusBadge.MinSize()
	statusBadge.Move(c2fynedriver.Position{X: 240, Y: 12})
	statusBadge.Resize(badgeMin)
	root.Add(statusBadge)

	// Compteur de FPS en direct dans l'en-tête
	fpsWidget := c2fynedriver.NewFPSCounter(drv)
	fpsWidget.Move(c2fynedriver.Position{X: 810, Y: 18})
	root.Add(fpsWidget)

	// 2. Panneau latéral gauche : Cartes & Contrôles
	leftCardContent := c2fynedriver.NewContainer()

	// Widget Terminal (instancié tôt pour liaisons des boutons)
	termComp := c2fynedriver.NewTerminalComponent(68, 18)
	termComp.Move(c2fynedriver.Position{X: 0, Y: 0})

	// Message de bienvenue initial dans le terminal
	initBanner := "\x1b[1;36m┌──────────────────────────────────────────────────────────────────┐\x1b[0m\r\n" +
		"\x1b[1;36m│\x1b[0m  \x1b[1;32mHOROS SYSTEM\x1b[0m — Pilote Graphique Fyne Souverain CGO-Free         \x1b[1;36m│\x1b[0m\r\n" +
		"\x1b[1;36m│\x1b[0m  Pile: \x1b[1;33mc2display\x1b[0m (X11/Wayland/Mem) + \x1b[1;33mc2painter\x1b[0m (SIMD) + \x1b[1;33mtt55\x1b[0m    \x1b[1;36m│\x1b[0m\r\n" +
		"\x1b[1;36m│\x1b[0m  Vitesse de rendu trame : \x1b[1;35m< 250 µs\x1b[0m | Débit streaming : \x1b[1;35m220 Mo/s\x1b[0m   \x1b[1;36m│\x1b[0m\r\n" +
		"\x1b[1;36m└──────────────────────────────────────────────────────────────────┘\x1b[0m\r\n\r\n" +
		"\x1b[1;34mroot@horos55\x1b[0m:\x1b[1;32m~/worktrees/fyne-sgoiter\x1b[0m# uname -a\r\n" +
		"Linux horos-c2simd 6.8.0 #1 SMP PREEMPT_DYNAMIC x86_64 Go1.27 CGO_ENABLED=0\r\n\r\n" +
		"\x1b[1;34mroot@horos55\x1b[0m:\x1b[1;32m~/worktrees/fyne-sgoiter\x1b[0m# "

	_, _ = termComp.Term.Write([]byte(initBanner))

	// Bouton 1: Dégradé TrueColor ANSI 24-bit
	btnTrueColor := c2fynedriver.NewButton("TrueColor ANSI", func() {
		msg := "\r\n\x1b[1m[TEST TRUECOLOR 24-BIT]\x1b[0m\r\n"
		for i := 0; i < 32; i++ {
			r := uint8((i * 255) / 31)
			g := uint8(255 - (i*255)/31)
			b := uint8((i * 128) / 31)
			msg += fmt.Sprintf("\x1b[48;2;%d;%d;%dm \x1b[0m", r, g, b)
		}
		msg += "\r\n\x1b[1;34mroot@horos55\x1b[0m:\x1b[1;32m~/worktrees/fyne-sgoiter\x1b[0m# "
		_, _ = termComp.Term.Write([]byte(msg))
	})
	btnTrueColor.Move(c2fynedriver.Position{X: 0, Y: 0})
	btnTrueColor.Resize(c2fynedriver.Size{Width: 200, Height: 36})
	leftCardContent.Add(btnTrueColor)

	// Bouton 2: Test Cadres DEC VT100
	btnDEC := c2fynedriver.NewButton("DEC VT100 Box", func() {
		msg := "\r\n\x1b[1m[TEST GRAPHISME DEC VT100]\x1b[0m\r\n" +
			"\x1b[1;33m┌───────────────┬───────────────┐\x1b[0m\r\n" +
			"\x1b[1;33m│\x1b[0m Composant      \x1b[1;33m│\x1b[0m Statut        \x1b[1;33m│\x1b[0m\r\n" +
			"\x1b[1;33m├───────────────┼───────────────┤\x1b[0m\r\n" +
			"\x1b[1;33m│\x1b[0m c2fynedriver   \x1b[1;33m│\x1b[0m \x1b[32mPARITÉ CGO=0\x1b[0m  \x1b[1;33m│\x1b[0m\r\n" +
			"\x1b[1;33m│\x1b[0m c2painter      \x1b[1;33m│\x1b[0m \x1b[32mSIMD 4000 FPS\x1b[0m \x1b[1;33m│\x1b[0m\r\n" +
			"\x1b[1;33m│\x1b[0m c2fyneterm     \x1b[1;33m│\x1b[0m \x1b[32mZÉRO ALLOC\x1b[0m    \x1b[1;33m│\x1b[0m\r\n" +
			"\x1b[1;33m└───────────────┴───────────────┘\x1b[0m\r\n" +
			"\x1b[1;34mroot@horos55\x1b[0m:\x1b[1;32m~/worktrees/fyne-sgoiter\x1b[0m# "
		_, _ = termComp.Term.Write([]byte(msg))
	})
	btnDEC.Move(c2fynedriver.Position{X: 0, Y: 46})
	btnDEC.Resize(c2fynedriver.Size{Width: 200, Height: 36})
	leftCardContent.Add(btnDEC)

	// Bouton 3: Effacer Terminal
	btnClear := c2fynedriver.NewButton("Effacer Écran", func() {
		_, _ = termComp.Term.Write([]byte("\x1b[2J\x1b[H\x1b[1;34mroot@horos55\x1b[0m:\x1b[1;32m~/worktrees/fyne-sgoiter\x1b[0m# "))
	})
	btnClear.Move(c2fynedriver.Position{X: 0, Y: 92})
	btnClear.Resize(c2fynedriver.Size{Width: 200, Height: 36})
	btnClear.BgColor = c2painter.PackRGBA(71, 85, 105, 255)
	btnClear.HoverColor = c2painter.PackRGBA(100, 116, 139, 255)
	leftCardContent.Add(btnClear)

	// Carte gauche des Contrôles
	controlsCard := c2fynedriver.NewCard("Actions & Commandes", "Interactions directes", leftCardContent)
	controlsCard.Move(c2fynedriver.Position{X: 20, Y: 80})
	controlsCard.Resize(c2fynedriver.Size{Width: 232, Height: 210})
	controlsCard.Elevation = 4
	root.Add(controlsCard)

	// Carte gauche inférieure : Diagnostic des sous-systèmes
	diagContent := c2fynedriver.NewContainer()

	diagItems := []struct {
		name   string
		status string
		col    uint32
	}{
		{"c2display (IPC)", "Opérationnel", c2painter.PackRGBA(16, 185, 129, 255)},
		{"c2painter (2D)", "SIMD Pur Go", c2painter.PackRGBA(59, 130, 246, 255)},
		{"c2fyneterm", "VT500 / 0-Alloc", c2painter.PackRGBA(168, 85, 247, 255)},
		{"tt55 (Typo)", "TrueType Scaler", c2painter.PackRGBA(234, 179, 8, 255)},
	}

	for idx, item := range diagItems {
		itemY := idx * 24
		lblN := c2fynedriver.NewLabel(item.name)
		lblN.Move(c2fynedriver.Position{X: 0, Y: itemY})
		lblN.TextSize = 11
		lblN.Color = c2painter.PackRGBA(203, 213, 225, 255)
		diagContent.Add(lblN)

		lblS := c2fynedriver.NewLabel(item.status)
		lblS.Move(c2fynedriver.Position{X: 100, Y: itemY})
		lblS.TextSize = 11
		lblS.Color = item.col
		diagContent.Add(lblS)
	}

	diagCard := c2fynedriver.NewCard("Architecture CGO=0", "Statut des moteurs", diagContent)
	diagCard.Move(c2fynedriver.Position{X: 20, Y: 304})
	diagCard.Resize(c2fynedriver.Size{Width: 232, Height: 180})
	diagCard.Elevation = 4
	root.Add(diagCard)

	// 3. Carte centrale hébergeant le Widget Terminal interactif
	termCardContent := c2fynedriver.NewContainer()
	termCardContent.Add(termComp)

	termCard := c2fynedriver.NewCard("Console Émulateur VTE", "c2fyneterm — Séquences ANSI & 24-bit TrueColor", termCardContent)
	termCard.Move(c2fynedriver.Position{X: 268, Y: 80})
	termCard.Resize(c2fynedriver.Size{Width: 672, Height: 404})
	termCard.Elevation = 5
	termCard.HeaderBadge = "INTERACTIF"
	termCard.BadgeBg = c2painter.PackRGBA(37, 99, 235, 255)
	termCard.BadgeFg = c2painter.PackRGBA(255, 255, 255, 255)
	root.Add(termCard)

	// 4. Barre de pied de page (Footer)
	footerBg := c2fynedriver.NewRectangle(c2painter.PackRGBA(15, 23, 42, 255))
	footerBg.Move(c2fynedriver.Position{X: 0, Y: 560})
	footerBg.Resize(c2fynedriver.Size{Width: 960, Height: 40})
	root.Add(footerBg)

	footerLine := c2fynedriver.NewLine(
		c2fynedriver.Position{X: 0, Y: 560},
		c2fynedriver.Position{X: 960, Y: 560},
		c2painter.PackRGBA(30, 41, 59, 255),
		1,
	)
	root.Add(footerLine)

	footerLbl := c2fynedriver.NewLabel("HOROS55 — PILOTE FYNE SANS CGO — ARCHITECTURE MATRICIELLE SGOITER 2026")
	footerLbl.Move(c2fynedriver.Position{X: 20, Y: 574})
	footerLbl.TextSize = 11
	footerLbl.Color = c2painter.PackRGBA(100, 116, 139, 255)
	root.Add(footerLbl)

	// Pastille pulsation active
	pulseDot := c2fynedriver.NewCircle(c2painter.PackRGBA(34, 197, 94, 255))
	pulseDot.Move(c2fynedriver.Position{X: 930, Y: 574})
	pulseDot.Resize(c2fynedriver.Size{Width: 10, Height: 10})
	root.Add(pulseDot)

	win.SetContent(root)
	win.Canvas().SetFocused(termComp)

	// Si un nombre limité de frames est requis (ex: bench ou test CI)
	if *benchFrames > 0 {
		fmt.Printf("Exécution du banc de rendu sur %d trames...\n", *benchFrames)
		start := time.Now()
		for i := 0; i < *benchFrames; i++ {
			win.RenderFrame()
		}
		elapsed := time.Since(start)
		fps := float64(*benchFrames) / elapsed.Seconds()
		fmt.Printf("Terminé en %v (%.1f FPS moyen, %.1f µs/trame)\n", elapsed, fps, float64(elapsed.Microseconds())/float64(*benchFrames))
		return
	}

	// Lancement en mode interactif continu
	fmt.Println("Démarrage de c2fyne-demo (960x600 px, CGO_ENABLED=0)...")
	win.ShowAndRun()
}
