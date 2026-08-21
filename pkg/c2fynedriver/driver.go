package c2fynedriver

import (
	"sync"
	"sync/atomic"
	"time"

	c2display "code.hazyhaar.fr/devhoros/pkg/c2display"
	c2painter "code.hazyhaar.fr/devhoros/pkg/c2painter"
)

// Driver implémente le cycle de vie de l'application Fyne et la gestion des fenêtres natives sans CGO.
type Driver struct {
	mu            sync.RWMutex
	display       c2display.Display
	windows       map[uint32]*Window
	nextWinID     uint32
	running       bool
	quitChan      chan struct{}
	targetFPS     int
	fpsCurrent    float64
	frameDurUs    int
	totalFrames   uint64
	fpsMutex      sync.RWMutex
}

var (
	defaultDriver     *Driver
	defaultDriverOnce sync.Once
)

// NewDriver initialise une nouvelle instance de Driver connectée au sous-système c2display.
func NewDriver() (*Driver, error) {
	disp, err := c2display.Open(c2display.DisplayOptions{
		Backend: c2display.BackendAuto,
	})
	if err != nil {
		return nil, err
	}

	d := &Driver{
		display:    disp,
		windows:    make(map[uint32]*Window),
		nextWinID:  1,
		quitChan:   make(chan struct{}),
		targetFPS:  60,
		fpsCurrent: 60.0,
	}
	return d, nil
}

// NewHeadlessDriver instancie un Driver dédié aux tests en environnement sans écran.
func NewHeadlessDriver(w, h int) (*Driver, error) {
	disp, err := c2display.Open(c2display.DisplayOptions{
		Backend:        c2display.BackendHeadless,
		HeadlessWidth:  w,
		HeadlessHeight: h,
	})
	if err != nil {
		return nil, err
	}

	return &Driver{
		display:    disp,
		windows:    make(map[uint32]*Window),
		nextWinID:  1,
		quitChan:   make(chan struct{}),
		targetFPS:  60,
		fpsCurrent: 60.0,
	}, nil
}

// GetDefaultDriver retourne l'instance partagée du Driver.
func GetDefaultDriver() *Driver {
	defaultDriverOnce.Do(func() {
		d, err := NewDriver()
		if err != nil {
			panic("c2fynedriver: échec d'initialisation du Display par défaut: " + err.Error())
		}
		defaultDriver = d
	})
	return defaultDriver
}

// CreateWindow instancie une nouvelle fenêtre Fyne avec le titre et les dimensions indiqués.
func (d *Driver) CreateWindow(title string, width, height int) (*Window, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}

	winID := d.nextWinID
	d.nextWinID++

	dispWin, err := d.display.CreateWindow(c2display.WindowOptions{
		Title:  title,
		Width:  width,
		Height: height,
	})
	if err != nil {
		return nil, err
	}

	surf := c2painter.NewSurface(width, height)
	paint := c2painter.NewPainter(surf)

	win := &Window{
		id:         winID,
		title:      title,
		driver:     d,
		displayWin: dispWin,
		surface:    surf,
		painter:    paint,
		size:       Size{Width: width, Height: height},
		closeChan:  make(chan struct{}),
		visible:    true,
	}

	canvas := NewCanvas(win, width, height)
	canvas.driver = d
	win.canvas = canvas

	d.windows[winID] = win

	// Démarrage de la boucle de traitement des événements de la fenêtre
	go win.eventLoop()

	return win, nil
}

// Run démarre la boucle principale d'application et de rafraîchissement d'affichage.
func (d *Driver) Run() {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return
	}
	d.running = true
	d.mu.Unlock()

	ticker := time.NewTicker(time.Second / time.Duration(d.targetFPS))
	defer ticker.Stop()

	var frameCount int
	lastFPSTime := time.Now()

	for {
		select {
		case <-d.quitChan:
			return

		case now := <-ticker.C:
			start := time.Now()

			// Rafraîchissement de toutes les fenêtres actives
			d.mu.RLock()
			wins := make([]*Window, 0, len(d.windows))
			for _, w := range d.windows {
				wins = append(wins, w)
			}
			d.mu.RUnlock()

			if len(wins) == 0 {
				return // Toutes les fenêtres sont fermées
			}

			for _, w := range wins {
				if w.IsVisible() && w.IsDirty() {
					w.RenderFrame()
				}
			}

			dur := time.Since(start)
			durUs := int(dur.Microseconds())

			frameCount++
			atomic.AddUint64(&d.totalFrames, 1)

			if now.Sub(lastFPSTime) >= time.Second {
				fps := float64(frameCount) / now.Sub(lastFPSTime).Seconds()
				d.fpsMutex.Lock()
				d.fpsCurrent = fps
				d.frameDurUs = durUs
				d.fpsMutex.Unlock()

				frameCount = 0
				lastFPSTime = now
			}
		}
	}
}

// Quit demande l'arrêt propre du Driver et la fermeture de toutes les fenêtres.
func (d *Driver) Quit() {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return
	}
	d.running = false
	close(d.quitChan)
	wins := make([]*Window, 0, len(d.windows))
	for _, w := range d.windows {
		wins = append(wins, w)
	}
	d.mu.Unlock()

	for _, w := range wins {
		w.Close()
	}

	if d.display != nil {
		_ = d.display.Close()
	}
}

// CurrentFPS retourne la cadence de rafraîchissement mesurée en temps réel.
func (d *Driver) CurrentFPS() float64 {
	d.fpsMutex.RLock()
	defer d.fpsMutex.RUnlock()
	return d.fpsCurrent
}

// LastFrameDurationUs retourne la durée de la dernière passe de rendu en microsecondes.
func (d *Driver) LastFrameDurationUs() int {
	d.fpsMutex.RLock()
	defer d.fpsMutex.RUnlock()
	return d.frameDurUs
}

// TotalFrames retourne le compteur cumulé de trames rendues.
func (d *Driver) TotalFrames() uint64 {
	return atomic.LoadUint64(&d.totalFrames)
}

func (d *Driver) removeWindow(id uint32) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.windows, id)
	if len(d.windows) == 0 && d.running {
		select {
		case d.quitChan <- struct{}{}:
		default:
		}
	}
}

// ==========================================
// Fenêtre d'Application (Window)
// ==========================================

// Window représente une fenêtre native Fyne gérée par c2fynedriver.
type Window struct {
	mu         sync.RWMutex
	id         uint32
	title      string
	driver     *Driver
	displayWin c2display.Window
	canvas     *Canvas
	surface    *c2painter.Surface
	painter    *c2painter.Painter
	size       Size
	visible    bool
	closed     bool
	closeChan  chan struct{}
	onClosed   func()
}

// Canvas retourne le canevas graphique associé à la fenêtre.
func (w *Window) Canvas() *Canvas {
	return w.canvas
}

// Title retourne le titre de la fenêtre.
func (w *Window) Title() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.title
}

// SetTitle modifie le titre de la fenêtre.
func (w *Window) SetTitle(title string) {
	w.mu.Lock()
	w.title = title
	if w.displayWin != nil {
		_ = w.displayWin.SetTitle(title)
	}
	w.mu.Unlock()
}

// Size retourne les dimensions de la fenêtre.
func (w *Window) Size() Size {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.size
}

// Resize redimensionne la fenêtre et réalloue la surface de rendu c2painter.
func (w *Window) Resize(size Size) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if size.Width <= 0 || size.Height <= 0 {
		return
	}

	w.size = size
	w.surface = c2painter.NewSurface(size.Width, size.Height)
	w.painter = c2painter.NewPainter(w.surface)
	w.canvas.Resize(size)

	if w.displayWin != nil {
		_ = w.displayWin.Resize(size.Width, size.Height)
	}
}

// SetContent définit le widget racine de la fenêtre.
func (w *Window) SetContent(content CanvasObject) {
	w.canvas.SetContent(content)
}

// Content retourne le widget racine de la fenêtre.
func (w *Window) Content() CanvasObject {
	return w.canvas.Content()
}

// IsVisible retourne vrai si la fenêtre est actuellement visible sous protection de verrou.
func (w *Window) IsVisible() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.visible && !w.closed
}

// IsDirty retourne vrai si le canevas de la fenêtre a des changements à restituer.
func (w *Window) IsDirty() bool {
	if w.canvas == nil {
		return false
	}
	return w.canvas.IsDirty()
}

// Show rend la fenêtre visible.
func (w *Window) Show() {
	w.mu.Lock()
	w.visible = true
	w.mu.Unlock()
}

// Hide masque la fenêtre.
func (w *Window) Hide() {
	w.mu.Lock()
	w.visible = false
	w.mu.Unlock()
}

// Close ferme la fenêtre et libère ses ressources graphiques.
func (w *Window) Close() {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.closed = true
	close(w.closeChan)
	w.mu.Unlock()

	if w.displayWin != nil {
		_ = w.displayWin.Close()
	}

	if w.onClosed != nil {
		w.onClosed()
	}

	w.driver.removeWindow(w.id)
}

// SetOnClosed configure la fonction de rappel exécutée à la fermeture de la fenêtre.
func (w *Window) SetOnClosed(fn func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onClosed = fn
}

// ShowAndRun affiche la fenêtre et démarre la boucle principale du Driver.
func (w *Window) ShowAndRun() {
	w.Show()
	w.driver.Run()
}

// RenderFrame exécute une passe de rendu complète (Canvas.Paint) et la projection native (Present).
func (w *Window) RenderFrame() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed || !w.visible || w.surface == nil || w.painter == nil {
		return
	}

	// 1. Rendu de l'arbre graphique via c2painter
	w.canvas.Paint(w.painter)

	// 2. Présentation au serveur d'affichage c2display (X11 / Wayland / Headless)
	if w.displayWin != nil {
		_ = w.displayWin.Present(w.surface)
	}
}

// eventLoop dépouille les événements natifs de la fenêtre d'affichage.
func (w *Window) eventLoop() {
	if w.displayWin == nil {
		return
	}

	events := w.displayWin.Events()
	for {
		select {
		case <-w.closeChan:
			return

		case ev, ok := <-events:
			if !ok || ev == nil {
				return
			}

			switch e := ev.(type) {
			case c2display.MouseEvent:
				w.canvas.HandleMouseEvent(e)

			case c2display.KeyEvent:
				w.canvas.HandleKeyEvent(e)

			case c2display.ResizeEvent:
				w.Resize(Size{Width: e.Width, Height: e.Height})

			case c2display.ExposeEvent:
				w.RenderFrame()

			case c2display.CloseEvent:
				w.Close()
				return
			}
		}
	}
}

// ==========================================
// Facade Application Fyne (App)
// ==========================================

// App représente l'instance d'application Fyne de haut niveau.
type App struct {
	driver *Driver
}

// NewApp instancie une nouvelle application Fyne branchée sur c2fynedriver.
func NewApp() *App {
	return &App{
		driver: GetDefaultDriver(),
	}
}

// NewAppWithDriver instancie une application avec un pilote explicite (ex: headless).
func NewAppWithDriver(d *Driver) *App {
	return &App{
		driver: d,
	}
}

// NewWindow crée une nouvelle fenêtre d'application.
func (a *App) NewWindow(title string) *Window {
	win, err := a.driver.CreateWindow(title, 800, 600)
	if err != nil {
		panic("c2fynedriver: échec de création de fenêtre: " + err.Error())
	}
	return win
}

// Run démarre l'application.
func (a *App) Run() {
	a.driver.Run()
}

// Quit quitte l'application.
func (a *App) Quit() {
	a.driver.Quit()
}

// Driver retourne le pilote sous-jacent.
func (a *App) Driver() *Driver {
	return a.driver
}
