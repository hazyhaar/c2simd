package c2display

import (
	"sync"
	"sync/atomic"

	c2painter "code.hazyhaar.fr/devhoros/pkg/c2painter"
)

// HeadlessDisplay représente un serveur d'affichage virtuel en mémoire pour tests et environnements sans écran.
type HeadlessDisplay struct {
	mu        sync.RWMutex
	closed    bool
	screens   []ScreenInfo
	windows   map[uint32]*HeadlessWindow
	nextWinID uint32
}

// NewHeadlessDisplay initialise une nouvelle instance de HeadlessDisplay.
func NewHeadlessDisplay(opts DisplayOptions) (*HeadlessDisplay, error) {
	w := opts.HeadlessWidth
	h := opts.HeadlessHeight
	if w <= 0 {
		w = 1920
	}
	if h <= 0 {
		h = 1080
	}

	return &HeadlessDisplay{
		screens: []ScreenInfo{
			{
				ID:          0,
				Width:       w,
				Height:      h,
				WidthMM:     508,
				HeightMM:    285,
				Depth:       32,
				ScaleFactor: 1.0,
				Primary:     true,
			},
		},
		windows:   make(map[uint32]*HeadlessWindow),
		nextWinID: 1,
	}, nil
}

func (d *HeadlessDisplay) Type() BackendType {
	return BackendHeadless
}

func (d *HeadlessDisplay) Screens() []ScreenInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	res := make([]ScreenInfo, len(d.screens))
	copy(res, d.screens)
	return res
}

func (d *HeadlessDisplay) Flush() error {
	return nil
}

func (d *HeadlessDisplay) CreateWindow(opts WindowOptions) (Window, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil, ErrDisplayClosed
	}

	w := opts.Width
	h := opts.Height
	if w <= 0 {
		w = 800
	}
	if h <= 0 {
		h = 600
	}

	winID := d.nextWinID
	d.nextWinID++

	title := opts.Title
	if title == "" {
		title = "Headless Window"
	}

	win := &HeadlessWindow{
		id:         winID,
		title:      title,
		width:      w,
		height:     h,
		x:          opts.X,
		y:          opts.Y,
		display:    d,
		eventChan:  make(chan Event, 256),
		closeChan:  make(chan struct{}),
		lastBuffer: c2painter.NewSurface(w, h),
	}

	d.windows[winID] = win

	// Émettre un Expose initial
	win.InjectEvent(ExposeEvent{
		BaseEvent: BaseEvent{Time: 0},
		X:         0,
		Y:         0,
		Width:     w,
		Height:    h,
	})

	return win, nil
}

func (d *HeadlessDisplay) removeWindow(id uint32) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.windows, id)
}

func (d *HeadlessDisplay) Close() error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	d.closed = true
	wins := make([]*HeadlessWindow, 0, len(d.windows))
	for _, w := range d.windows {
		wins = append(wins, w)
	}
	d.windows = make(map[uint32]*HeadlessWindow)
	d.mu.Unlock()

	for _, w := range wins {
		w.Close()
	}
	return nil
}

// HeadlessWindow représente une fenêtre virtuelle en mémoire.
type HeadlessWindow struct {
	mu         sync.RWMutex
	id         uint32
	title      string
	width      int
	height     int
	x          int
	y          int
	closed     bool
	frameCount uint64
	display    *HeadlessDisplay
	eventChan  chan Event
	closeChan  chan struct{}
	lastBuffer *c2painter.Surface
}

func (w *HeadlessWindow) ID() uint32 {
	return w.id
}

func (w *HeadlessWindow) Title() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.title
}

func (w *HeadlessWindow) SetTitle(title string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return ErrWindowClosed
	}
	w.title = title
	return nil
}

func (w *HeadlessWindow) Bounds() (width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.width, w.height
}

func (w *HeadlessWindow) Resize(width, height int) error {
	if width <= 0 || height <= 0 {
		return ErrInvalidGeometry
	}
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return ErrWindowClosed
	}
	w.width = width
	w.height = height
	w.lastBuffer = c2painter.NewSurface(width, height)
	w.mu.Unlock()

	w.InjectEvent(ResizeEvent{
		BaseEvent: BaseEvent{Time: 0},
		Width:     width,
		Height:    height,
	})
	return nil
}

// Present met à jour le tampon interne de la fenêtre virtuelle de manière thread-safe.
func (w *HeadlessWindow) Present(surface *c2painter.Surface) error {
	if surface == nil || surface.Pixels == nil {
		return ErrNilSurface
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return ErrWindowClosed
	}

	if w.lastBuffer == nil || w.lastBuffer.Width != surface.Width || w.lastBuffer.Height != surface.Height {
		w.lastBuffer = c2painter.NewSurface(surface.Width, surface.Height)
	}

	copy(w.lastBuffer.Pixels, surface.Pixels)
	atomic.AddUint64(&w.frameCount, 1)
	return nil
}

// LastSurface retourne une copie de consultation de la dernière surface présentée.
func (w *HeadlessWindow) LastSurface() *c2painter.Surface {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.lastBuffer == nil {
		return nil
	}
	s := c2painter.NewSurface(w.lastBuffer.Width, w.lastBuffer.Height)
	copy(s.Pixels, w.lastBuffer.Pixels)
	return s
}

// FrameCount retourne le nombre total de trames présentées.
func (w *HeadlessWindow) FrameCount() uint64 {
	return atomic.LoadUint64(&w.frameCount)
}

func (w *HeadlessWindow) Events() <-chan Event {
	return w.eventChan
}

func (w *HeadlessWindow) PollEvent() Event {
	select {
	case ev := <-w.eventChan:
		return ev
	default:
		return nil
	}
}

func (w *HeadlessWindow) WaitEvent() Event {
	select {
	case ev, ok := <-w.eventChan:
		if !ok {
			return nil
		}
		return ev
	case <-w.closeChan:
		return nil
	}
}

// InjectEvent insère un événement dans la file d'attente de la fenêtre (pour tests et simulations).
func (w *HeadlessWindow) InjectEvent(e Event) {
	if e == nil {
		return
	}
	if mm, ok := e.(MouseEvent); ok && mm.Action == EventMouseMove {
		select {
		case w.eventChan <- e:
		default:
			select {
			case old := <-w.eventChan:
				if _, isMM := old.(MouseEvent); !isMM {
					select {
					case w.eventChan <- old:
					default:
					}
				}
			default:
			}
			select {
			case w.eventChan <- e:
			default:
			}
		}
		return
	}

	select {
	case w.eventChan <- e:
	default:
		go func(ev Event) {
			w.eventChan <- ev
		}(e)
	}
}

func (w *HeadlessWindow) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	close(w.closeChan)
	w.mu.Unlock()

	w.display.removeWindow(w.id)
	return nil
}
