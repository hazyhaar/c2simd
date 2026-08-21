package c2display

import (
	"errors"
	"io"
	"os"
	"time"

	c2painter "code.hazyhaar.fr/devhoros/pkg/c2painter"
)

// Erreurs protocolaires et opérationnelles.
var (
	ErrDisplayClosed   = errors.New("c2display: display fermé")
	ErrWindowClosed    = errors.New("c2display: fenêtre fermée")
	ErrUnsupported     = errors.New("c2display: backend non supporté")
	ErrHandshakeFailed = errors.New("c2display: échec de négociation protocolaire")
	ErrNoServerFound   = errors.New("c2display: aucun serveur d'affichage disponible")
	ErrNilSurface      = errors.New("c2display: surface nulle")
	ErrInvalidGeometry = errors.New("c2display: géométrie invalide")
)

// BackendType identifie la technologie de serveur d'affichage.
type BackendType string

const (
	BackendAuto     BackendType = "auto"
	BackendX11      BackendType = "x11"
	BackendWayland  BackendType = "wayland"
	BackendHeadless BackendType = "headless"
)

// EventType spécifie le type d'un événement d'affichage.
type EventType uint8

const (
	EventNone EventType = iota
	EventMousePress
	EventMouseRelease
	EventMouseMove
	EventMouseScroll
	EventKeyPress
	EventKeyRelease
	EventResize
	EventExpose
	EventFocusIn
	EventFocusOut
	EventClose
)

// Event est l'interface commune de tous les événements d'entrée et de cycle de vie.
type Event interface {
	Type() EventType
	Timestamp() time.Duration
}

// BaseEvent fournit l'horodatage commun aux événements.
type BaseEvent struct {
	Time time.Duration
}

// Timestamp retourne l'horodatage relatif de l'événement.
func (b BaseEvent) Timestamp() time.Duration {
	return b.Time
}

// MouseEvent représente une interaction pointeur (déplacement, clic, molette).
type MouseEvent struct {
	BaseEvent
	Action    EventType
	Button    MouseButton
	X         int
	Y         int
	DeltaX    float64
	DeltaY    float64
	Modifiers KeyModifiers
}

func (m MouseEvent) Type() EventType {
	return m.Action
}

// KeyEvent représente une frappe ou un relâchement de touche clavier.
type KeyEvent struct {
	BaseEvent
	Action    EventType
	Key       KeyCode
	Rune      rune
	Scancode  uint32
	Modifiers KeyModifiers
}

func (k KeyEvent) Type() EventType {
	return k.Action
}

// ResizeEvent signale la modification des dimensions de la fenêtre.
type ResizeEvent struct {
	BaseEvent
	Width  int
	Height int
}

func (r ResizeEvent) Type() EventType {
	return EventResize
}

// ExposeEvent signale qu'une zone de la surface doit être rafraîchie.
type ExposeEvent struct {
	BaseEvent
	X      int
	Y      int
	Width  int
	Height int
}

func (e ExposeEvent) Type() EventType {
	return EventExpose
}

// FocusEvent signale la prise ou perte du focus clavier/fenêtre.
type FocusEvent struct {
	BaseEvent
	Focused bool
}

func (f FocusEvent) Type() EventType {
	if f.Focused {
		return EventFocusIn
	}
	return EventFocusOut
}

// CloseEvent signale une demande de fermeture émise par l'utilisateur ou le gestionnaire de fenêtres.
type CloseEvent struct {
	BaseEvent
}

func (c CloseEvent) Type() EventType {
	return EventClose
}

// ScreenInfo synthétise les propriétés d'un écran physique ou virtuel.
type ScreenInfo struct {
	ID          int
	Width       int
	Height      int
	WidthMM     int
	HeightMM    int
	Depth       int
	ScaleFactor float64
	Primary     bool
}

// WindowOptions paramètre la création d'une fenêtre.
type WindowOptions struct {
	Title       string
	Width       int
	Height      int
	X           int
	Y           int
	Decorated   bool
	Resizable   bool
	Transparent bool
}

// DisplayOptions paramètre la connexion au serveur d'affichage.
type DisplayOptions struct {
	Backend        BackendType
	SocketPath     string
	DisplayName    string
	HeadlessWidth  int
	HeadlessHeight int
}

// Window représente une fenêtre d'affichage manipulable.
type Window interface {
	io.Closer
	ID() uint32
	Title() string
	SetTitle(title string) error
	Bounds() (width, height int)
	Resize(width, height int) error
	Present(surface *c2painter.Surface) error
	Events() <-chan Event
	PollEvent() Event
	WaitEvent() Event
}

// Display représente l'instance de connexion au sous-système graphique.
type Display interface {
	io.Closer
	Type() BackendType
	CreateWindow(opts WindowOptions) (Window, error)
	Screens() []ScreenInfo
	Flush() error
}

// Open établit une connexion au serveur d'affichage selon les options fournies.
// En mode BackendAuto, l'ordre de détection automatique est :
// 1. Wayland (si $WAYLAND_DISPLAY ou socket /run/user/$UID/wayland-* est accessible).
// 2. X11 (si $DISPLAY ou socket /tmp/.X11-unix/X* est accessible).
// 3. Headless (si aucun serveur natif n'est disponible).
func Open(opts DisplayOptions) (Display, error) {
	switch opts.Backend {
	case BackendHeadless:
		return NewHeadlessDisplay(opts)

	case BackendX11:
		return NewX11Display(opts)

	case BackendWayland:
		return NewWaylandDisplay(opts)

	case "", BackendAuto:
		// 1. Tenter Wayland si variable ou socket présent
		if isWaylandAvailable(opts) {
			if disp, err := NewWaylandDisplay(opts); err == nil {
				return disp, nil
			}
		}

		// 2. Tenter X11 si variable ou socket présent
		if isX11Available(opts) {
			if disp, err := NewX11Display(opts); err == nil {
				return disp, nil
			}
		}

		// 3. Repli automatique en mode Headless
		return NewHeadlessDisplay(opts)

	default:
		return nil, ErrUnsupported
	}
}

// NewDisplay est un alias idiomatique pour Open.
func NewDisplay(opts DisplayOptions) (Display, error) {
	return Open(opts)
}

func isWaylandAvailable(opts DisplayOptions) bool {
	if opts.SocketPath != "" && opts.Backend == BackendWayland {
		return true
	}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return true
	}
	uid := os.Getuid()
	sock := "/run/user/" + itoa(uid) + "/wayland-0"
	if _, err := os.Stat(sock); err == nil {
		return true
	}
	return false
}

func isX11Available(opts DisplayOptions) bool {
	if opts.SocketPath != "" && opts.Backend == BackendX11 {
		return true
	}
	if os.Getenv("DISPLAY") != "" {
		return true
	}
	if _, err := os.Stat("/tmp/.X11-unix/X0"); err == nil {
		return true
	}
	return false
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b [32]byte
	i := len(b)
	neg := false
	if v < 0 {
		neg = true
		v = -v
	}
	for v > 0 {
		i--
		b[i] = byte('0' + (v % 10))
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
