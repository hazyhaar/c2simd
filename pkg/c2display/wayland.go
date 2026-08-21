package c2display

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"

	c2painter "code.hazyhaar.fr/devhoros/pkg/c2painter"
)

// Constantes protocolaires Wayland (Opcodes et formats de référence).
const (
	wlDisplayID uint32 = 1

	// wl_shm formats
	wlShmFormatARGB8888 uint32 = 0
	wlShmFormatXRGB8888 uint32 = 1
	wlShmFormatRGBA8888 uint32 = 0x34324752
	wlShmFormatABGR8888 uint32 = 0x34324241

	// Masques capacités wl_seat
	wlSeatCapPointer  uint32 = 1
	wlSeatCapKeyboard uint32 = 2

	// Boutons souris standard Linux input.h
	linuxBtnLeft   uint32 = 0x110
	linuxBtnRight  uint32 = 0x111
	linuxBtnMiddle uint32 = 0x112
)

// WaylandDisplay implémente le protocole Wayland en pur Go sans bibliothèque externe (zéro CGO).
type WaylandDisplay struct {
	conn      *net.UnixConn
	mu        sync.Mutex
	writeMu   sync.Mutex
	closed    bool
	closeChan chan struct{}
	nextID    uint32
	syncDone  bool
	syncCond  *sync.Cond

	// Objets de registre
	registryID   uint32
	compositorID uint32
	shmID        uint32
	wmBaseID     uint32
	seatID       uint32
	pointerID    uint32
	keyboardID   uint32

	// Formats shm supportés
	shmFormats map[uint32]bool

	// Fenêtres actives
	windows map[uint32]*WaylandWindow

	// Callback de synchronisation actif
	syncCallbackID uint32

	// Écrans déclarés
	screens []ScreenInfo

	// Dernières coordonnées du pointeur
	lastPointerX int
	lastPointerY int
}

// NewWaylandDisplay établit la connexion au compositeur Wayland via son socket Unix.
func NewWaylandDisplay(opts DisplayOptions) (*WaylandDisplay, error) {
	socketPath, err := resolveWaylandSocket(opts)
	if err != nil {
		return nil, fmt.Errorf("c2display/wayland: chemin socket introuvable: %w", err)
	}

	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		return nil, fmt.Errorf("c2display/wayland: connexion socket impossible (%s): %w", socketPath, err)
	}

	d := &WaylandDisplay{
		conn:       conn,
		closeChan:  make(chan struct{}),
		nextID:     2,
		shmFormats: make(map[uint32]bool),
		windows:    make(map[uint32]*WaylandWindow),
	}
	d.syncCond = sync.NewCond(&d.mu)

	// Lancer la goroutine de lecture des événements serveur
	go d.readLoop()

	// 1. Obtenir le registre global wl_display.get_registry(registryID)
	d.registryID = d.allocID()
	if err := d.sendMsg(wlDisplayID, 1 /* get_registry */, d.registryID); err != nil {
		d.Close()
		return nil, fmt.Errorf("get_registry: %w", err)
	}

	// 2. Synchronisation initiale pour lister les interfaces globales
	if err := d.Roundtrip(); err != nil {
		d.Close()
		return nil, fmt.Errorf("roundtrip initial: %w", err)
	}

	// 3. Seconde synchronisation pour finaliser les bindings
	if err := d.Roundtrip(); err != nil {
		d.Close()
		return nil, fmt.Errorf("roundtrip bindings: %w", err)
	}

	if d.closed {
		d.Close()
		return nil, errors.New("connexion Wayland fermée prématurément")
	}

	if d.compositorID == 0 || d.shmID == 0 || d.wmBaseID == 0 {
		d.Close()
		return nil, errors.New("interfaces Wayland obligatoires manquantes (compositor, shm ou xdg_wm_base)")
	}

	return d, nil
}

func resolveWaylandSocket(opts DisplayOptions) (string, error) {
	if opts.SocketPath != "" {
		return opts.SocketPath, nil
	}

	sockName := opts.DisplayName
	if sockName == "" {
		sockName = os.Getenv("WAYLAND_DISPLAY")
	}
	if sockName == "" {
		sockName = "wayland-0"
	}

	if filepath.IsAbs(sockName) {
		return sockName, nil
	}

	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}

	fullPath := filepath.Join(runtimeDir, sockName)
	if _, err := os.Stat(fullPath); err != nil {
		return "", fmt.Errorf("socket %s inaccessible: %w", fullPath, err)
	}

	return fullPath, nil
}

func (d *WaylandDisplay) allocID() uint32 {
	d.mu.Lock()
	defer d.mu.Unlock()
	id := d.nextID
	d.nextID++
	return id
}

func (d *WaylandDisplay) Roundtrip() error {
	d.mu.Lock()
	d.syncDone = false
	cbID := d.nextID
	d.nextID++
	d.syncCallbackID = cbID
	d.mu.Unlock()

	if err := d.sendMsg(wlDisplayID, 0 /* sync */, cbID); err != nil {
		return err
	}

	d.mu.Lock()
	timeout := time.After(3 * time.Second)
	doneChan := make(chan struct{})
	go func() {
		d.mu.Lock()
		for !d.syncDone && !d.closed {
			d.syncCond.Wait()
		}
		d.mu.Unlock()
		close(doneChan)
	}()
	d.mu.Unlock()

	select {
	case <-doneChan:
		return nil
	case <-timeout:
		return errors.New("délai roundtrip Wayland dépassé")
	}
}

func (d *WaylandDisplay) sendMsg(objectID uint32, opcode uint16, args ...interface{}) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, objectID)
	binary.Write(buf, binary.LittleEndian, uint32(0)) // Taille et opcode insérés après

	for _, arg := range args {
		switch v := arg.(type) {
		case uint32:
			binary.Write(buf, binary.LittleEndian, v)
		case int32:
			binary.Write(buf, binary.LittleEndian, v)
		case string:
			strBytes := []byte(v)
			strLen := uint32(len(strBytes) + 1)
			binary.Write(buf, binary.LittleEndian, strLen)
			buf.Write(strBytes)
			buf.WriteByte(0)
			pad := (4 - ((len(strBytes) + 1) % 4)) % 4
			for p := 0; p < pad; p++ {
				buf.WriteByte(0)
			}
		case []byte:
			arrLen := uint32(len(v))
			binary.Write(buf, binary.LittleEndian, arrLen)
			buf.Write(v)
			pad := (4 - (len(v) % 4)) % 4
			for p := 0; p < pad; p++ {
				buf.WriteByte(0)
			}
		}
	}

	data := buf.Bytes()
	totalLen := len(data)
	sizeAndOpcode := (uint32(totalLen) << 16) | uint32(opcode)
	binary.LittleEndian.PutUint32(data[4:8], sizeAndOpcode)

	_, err := d.conn.Write(data)
	return err
}

func (d *WaylandDisplay) sendMsgWithFD(objectID uint32, opcode uint16, fd int, args ...interface{}) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, objectID)
	binary.Write(buf, binary.LittleEndian, uint32(0))

	for _, arg := range args {
		switch v := arg.(type) {
		case uint32:
			binary.Write(buf, binary.LittleEndian, v)
		case int32:
			binary.Write(buf, binary.LittleEndian, v)
		}
	}

	data := buf.Bytes()
	totalLen := len(data)
	sizeAndOpcode := (uint32(totalLen) << 16) | uint32(opcode)
	binary.LittleEndian.PutUint32(data[4:8], sizeAndOpcode)

	oob := syscall.UnixRights(fd)
	_, _, err := d.conn.WriteMsgUnix(data, oob, nil)
	return err
}

func (d *WaylandDisplay) readLoop() {
	header := make([]byte, 8)

	for {
		if _, err := io.ReadFull(d.conn, header); err != nil {
			d.Close()
			return
		}

		objectID := binary.LittleEndian.Uint32(header[0:4])
		sizeAndOpcode := binary.LittleEndian.Uint32(header[4:8])
		msgLen := int(sizeAndOpcode >> 16)
		opcode := uint16(sizeAndOpcode & 0xffff)

		payloadLen := msgLen - 8
		var payload []byte
		if payloadLen > 0 {
			payload = make([]byte, payloadLen)
			if _, err := io.ReadFull(d.conn, payload); err != nil {
				d.Close()
				return
			}
		}

		d.dispatchWaylandEvent(objectID, opcode, payload)
	}
}

func (d *WaylandDisplay) dispatchWaylandEvent(objectID uint32, opcode uint16, p []byte) {
	// 1. Erreur display
	if objectID == wlDisplayID && opcode == 0 {
		return
	}

	d.mu.Lock()
	isRegistry := objectID == d.registryID
	isShm := objectID == d.shmID
	isWmBase := objectID == d.wmBaseID
	isSeat := objectID == d.seatID
	isPointer := objectID == d.pointerID
	isKeyboard := objectID == d.keyboardID
	isSyncCb := (d.syncCallbackID != 0 && objectID == d.syncCallbackID && opcode == 0)
	d.mu.Unlock()

	// 2. Callback de synchronisation exact
	if isSyncCb {
		d.mu.Lock()
		d.syncDone = true
		d.syncCallbackID = 0
		d.syncCond.Broadcast()
		d.mu.Unlock()
		return
	}

	if isRegistry && opcode == 0 {
		// global(name uint32, interface string, version uint32)
		if len(p) >= 8 {
			name := binary.LittleEndian.Uint32(p[0:4])
			strLen := binary.LittleEndian.Uint32(p[4:8])
			if int(8+strLen) <= len(p) {
				iface := string(p[8 : 8+strLen-1])
				aligned := (strLen + 3) &^ 3
				if int(8+aligned+4) <= len(p) {
					ver := binary.LittleEndian.Uint32(p[8+aligned : 8+aligned+4])
					d.handleGlobal(name, iface, ver)
				}
			}
		}
		return
	}

	if isShm && opcode == 0 {
		// format(format uint32)
		if len(p) >= 4 {
			fmtID := binary.LittleEndian.Uint32(p[0:4])
			d.mu.Lock()
			d.shmFormats[fmtID] = true
			d.mu.Unlock()
		}
		return
	}

	if isWmBase && opcode == 0 {
		// ping(serial uint32) -> Envoyer pong(serial)
		if len(p) >= 4 {
			serial := binary.LittleEndian.Uint32(p[0:4])
			_ = d.sendMsg(d.wmBaseID, 3 /* pong */, serial)
		}
		return
	}

	if isSeat && opcode == 0 {
		// capabilities(cap uint32)
		if len(p) >= 4 {
			caps := binary.LittleEndian.Uint32(p[0:4])
			d.handleSeatCaps(caps)
		}
		return
	}

	if isPointer {
		d.handlePointerEvent(opcode, p)
		return
	}

	if isKeyboard {
		d.handleKeyboardEvent(opcode, p)
		return
	}

	// 3. Traitement des événements associés aux fenêtres actives
	d.mu.Lock()
	handled := false
	for _, win := range d.windows {
		if win.handleWaylandMsg(objectID, opcode, p) {
			handled = true
			break
		}
	}
	d.mu.Unlock()
	if handled {
		return
	}
}

func (d *WaylandDisplay) handleGlobal(name uint32, iface string, version uint32) {
	d.mu.Lock()
	defer d.mu.Unlock()

	switch iface {
	case "wl_compositor":
		if d.compositorID == 0 {
			d.compositorID = d.nextID
			d.nextID++
			v := version
			if v > 4 {
				v = 4
			}
			_ = d.sendMsg(d.registryID, 0 /* bind */, name, "wl_compositor", v, d.compositorID)
		}
	case "wl_shm":
		if d.shmID == 0 {
			d.shmID = d.nextID
			d.nextID++
			_ = d.sendMsg(d.registryID, 0 /* bind */, name, "wl_shm", uint32(1), d.shmID)
		}
	case "xdg_wm_base":
		if d.wmBaseID == 0 {
			d.wmBaseID = d.nextID
			d.nextID++
			v := version
			if v > 2 {
				v = 2
			}
			_ = d.sendMsg(d.registryID, 0 /* bind */, name, "xdg_wm_base", v, d.wmBaseID)
		}
	case "wl_seat":
		if d.seatID == 0 {
			d.seatID = d.nextID
			d.nextID++
			v := version
			if v > 5 {
				v = 5
			}
			_ = d.sendMsg(d.registryID, 0 /* bind */, name, "wl_seat", v, d.seatID)
		}
	}
}

func (d *WaylandDisplay) handleSeatCaps(caps uint32) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if (caps&wlSeatCapPointer) != 0 && d.pointerID == 0 {
		d.pointerID = d.nextID
		d.nextID++
		_ = d.sendMsg(d.seatID, 0 /* get_pointer */, d.pointerID)
	}

	if (caps&wlSeatCapKeyboard) != 0 && d.keyboardID == 0 {
		d.keyboardID = d.nextID
		d.nextID++
		_ = d.sendMsg(d.seatID, 1 /* get_keyboard */, d.keyboardID)
	}
}

func (d *WaylandDisplay) handlePointerEvent(opcode uint16, p []byte) {
	d.mu.Lock()
	var targetWin *WaylandWindow
	for _, w := range d.windows {
		targetWin = w
		break
	}
	d.mu.Unlock()

	if targetWin == nil {
		return
	}

	switch opcode {
	case 2: // motion(time uint32, surface_x fixed, surface_y fixed)
		if len(p) >= 12 {
			sx := int(int32(binary.LittleEndian.Uint32(p[4:8])) / 256)
			sy := int(int32(binary.LittleEndian.Uint32(p[8:12])) / 256)
			d.lastPointerX = sx
			d.lastPointerY = sy
			targetWin.inject(MouseEvent{
				BaseEvent: BaseEvent{Time: time.Duration(time.Now().UnixNano())},
				Action:    EventMouseMove,
				Button:    ButtonNone,
				X:         sx,
				Y:         sy,
			})
		}
	case 3: // button(serial uint32, time uint32, button uint32, state uint32)
		if len(p) >= 16 {
			btnCode := binary.LittleEndian.Uint32(p[8:12])
			btnState := binary.LittleEndian.Uint32(p[12:16])
			btn := ButtonLeft
			if btnCode == linuxBtnRight {
				btn = ButtonRight
			} else if btnCode == linuxBtnMiddle {
				btn = ButtonMiddle
			}

			act := EventMousePress
			if btnState == 0 {
				act = EventMouseRelease
			}

			targetWin.inject(MouseEvent{
				BaseEvent: BaseEvent{Time: time.Duration(time.Now().UnixNano())},
				Action:    act,
				Button:    btn,
				X:         d.lastPointerX,
				Y:         d.lastPointerY,
			})
		}
	}
}

func (d *WaylandDisplay) handleKeyboardEvent(opcode uint16, p []byte) {
	d.mu.Lock()
	var targetWin *WaylandWindow
	for _, w := range d.windows {
		targetWin = w
		break
	}
	d.mu.Unlock()

	if targetWin == nil {
		return
	}

	if opcode == 3 { // key(serial uint32, time uint32, key uint32, state uint32)
		if len(p) >= 16 {
			keyScancode := binary.LittleEndian.Uint32(p[8:12])
			state := binary.LittleEndian.Uint32(p[12:16])

			kCode, r := EvdevScancodeToKeyCode(keyScancode)
			act := EventKeyPress
			if state == 0 {
				act = EventKeyRelease
			}

			targetWin.inject(KeyEvent{
				BaseEvent: BaseEvent{Time: time.Duration(time.Now().UnixNano())},
				Action:    act,
				Key:       kCode,
				Rune:      r,
				Scancode:  keyScancode,
			})
		}
	}
}

func (d *WaylandDisplay) Type() BackendType {
	return BackendWayland
}

func (d *WaylandDisplay) Screens() []ScreenInfo {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.screens) > 0 {
		return d.screens
	}
	return nil
}

func (d *WaylandDisplay) Flush() error {
	return nil
}

func (d *WaylandDisplay) CreateWindow(opts WindowOptions) (Window, error) {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil, ErrDisplayClosed
	}
	d.mu.Unlock()

	w := opts.Width
	h := opts.Height
	if w <= 0 {
		w = 800
	}
	if h <= 0 {
		h = 600
	}

	surfaceID := d.allocID()
	xdgSurfaceID := d.allocID()
	toplevelID := d.allocID()

	// 1. wl_compositor.create_surface(surfaceID)
	if err := d.sendMsg(d.compositorID, 0, surfaceID); err != nil {
		return nil, fmt.Errorf("create_surface: %w", err)
	}

	// 2. xdg_wm_base.get_xdg_surface(xdgSurfaceID, surfaceID)
	if err := d.sendMsg(d.wmBaseID, 2, xdgSurfaceID, surfaceID); err != nil {
		return nil, fmt.Errorf("get_xdg_surface: %w", err)
	}

	// 3. xdg_surface.get_toplevel(toplevelID)
	if err := d.sendMsg(xdgSurfaceID, 1, toplevelID); err != nil {
		return nil, fmt.Errorf("get_toplevel: %w", err)
	}

	// 4. Configurer titre et app_id
	title := opts.Title
	if title == "" {
		title = "Wayland Window"
	}
	_ = d.sendMsg(toplevelID, 2 /* set_title */, title)
	_ = d.sendMsg(toplevelID, 3 /* set_app_id */, "c2display")

	// 5. wl_surface.commit
	_ = d.sendMsg(surfaceID, 6 /* commit */)

	win := &WaylandWindow{
		display:      d,
		surfaceID:    surfaceID,
		xdgSurfaceID: xdgSurfaceID,
		toplevelID:   toplevelID,
		title:        title,
		width:        w,
		height:       h,
		eventChan:    make(chan Event, 256),
		closeChan:    make(chan struct{}),
	}

	d.mu.Lock()
	d.windows[surfaceID] = win
	d.mu.Unlock()

	return win, nil
}

func (d *WaylandDisplay) removeWindow(surfaceID uint32) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.windows, surfaceID)
}

func (d *WaylandDisplay) Close() error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	d.closed = true
	close(d.closeChan)
	d.syncCond.Broadcast()

	wins := make([]*WaylandWindow, 0, len(d.windows))
	for _, w := range d.windows {
		wins = append(wins, w)
	}
	d.windows = make(map[uint32]*WaylandWindow)
	d.mu.Unlock()

	for _, w := range wins {
		w.Close()
	}

	return d.conn.Close()
}

// WaylandWindow gère le cycle de vie et le rendu d'une surface Wayland.
type WaylandWindow struct {
	mu           sync.RWMutex
	display      *WaylandDisplay
	surfaceID    uint32
	xdgSurfaceID uint32
	toplevelID   uint32
	title        string
	width        int
	height       int
	closed       bool
	eventChan    chan Event
	closeChan    chan struct{}

	// Tampons SHM
	shmPoolID   uint32
	shmBufferID uint32
	shmFile     *os.File // F9: Référence persistante interdisant la fermeture par le finalizer GC
	shmFD       int
	shmSize     int
	shmWidth    int
	shmHeight   int
	shmStride   int
	shmMmap     []byte
}

func (w *WaylandWindow) ID() uint32 {
	return w.surfaceID
}

func (w *WaylandWindow) Title() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.title
}

func (w *WaylandWindow) SetTitle(title string) error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return ErrWindowClosed
	}
	w.title = title
	w.mu.Unlock()

	return w.display.sendMsg(w.toplevelID, 2 /* set_title */, title)
}

func (w *WaylandWindow) Bounds() (width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.width, w.height
}

func (w *WaylandWindow) Resize(width, height int) error {
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
	w.mu.Unlock()

	w.inject(ResizeEvent{
		BaseEvent: BaseEvent{Time: time.Duration(time.Now().UnixNano())},
		Width:     width,
		Height:    height,
	})
	return nil
}

func (w *WaylandWindow) handleWaylandMsg(objectID uint32, opcode uint16, p []byte) bool {
	if objectID == w.xdgSurfaceID && opcode == 0 {
		// xdg_surface.configure(serial uint32) -> ack_configure + commit
		if len(p) >= 4 {
			serial := binary.LittleEndian.Uint32(p[0:4])
			_ = w.display.sendMsg(w.xdgSurfaceID, 4 /* ack_configure */, serial)
			_ = w.display.sendMsg(w.surfaceID, 6 /* commit */)
		}
		return true
	}

	if objectID == w.toplevelID {
		switch opcode {
		case 0: // xdg_toplevel.configure(width int32, height int32, states array)
			if len(p) >= 8 {
				newW := int(int32(binary.LittleEndian.Uint32(p[0:4])))
				newH := int(int32(binary.LittleEndian.Uint32(p[4:8])))
				if newW > 0 && newH > 0 {
					w.mu.Lock()
					w.width = newW
					w.height = newH
					w.mu.Unlock()
					w.inject(ResizeEvent{
						BaseEvent: BaseEvent{Time: time.Duration(time.Now().UnixNano())},
						Width:     newW,
						Height:    newH,
					})
				}
			}
			return true
		case 1: // xdg_toplevel.close
			w.inject(CloseEvent{
				BaseEvent: BaseEvent{Time: time.Duration(time.Now().UnixNano())},
			})
			return true
		}
	}

	return false
}

func (w *WaylandWindow) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	close(w.closeChan)
	w.releaseSHM()
	w.mu.Unlock()

	w.display.removeWindow(w.surfaceID)

	if w.toplevelID != 0 {
		_ = w.display.sendMsg(w.toplevelID, 0 /* destroy */)
	}
	if w.xdgSurfaceID != 0 {
		_ = w.display.sendMsg(w.xdgSurfaceID, 0 /* destroy */)
	}
	if w.surfaceID != 0 {
		_ = w.display.sendMsg(w.surfaceID, 0 /* destroy */)
	}

	return nil
}

// Present effectue le transfert de la surface graphique vers le buffer partagé wl_shm Wayland.
func (w *WaylandWindow) Present(surface *c2painter.Surface) error {
	if surface == nil {
		return errors.New("c2display/wayland: surface nulle")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return ErrWindowClosed
	}

	width := surface.Width
	height := surface.Height
	stride := width * 4
	size := stride * height

	if err := w.ensureSHM(size, width, height, stride); err != nil {
		return fmt.Errorf("ensureSHM (%dx%d): %w", width, height, err)
	}

	// Copie des pixels de c2painter.Surface vers le tampon mémoire partagé SHM
	// Les compositeurs Wayland attendent par défaut ARGB8888 (Little-Endian BGR0/BGRA)
	srcStride := surface.Stride
	if srcStride <= 0 {
		srcStride = width
	}

	for y := 0; y < height; y++ {
		dstRow := y * stride
		srcRow := y * srcStride
		for x := 0; x < width; x++ {
			p := surface.Pixels[srcRow+x]
			// Conversion RGBA Little-Endian [R, G, B, A] -> Wayland ARGB8888 [B, G, R, A]
			w.shmMmap[dstRow+x*4] = byte((p >> 16) & 0xff)   // B
			w.shmMmap[dstRow+x*4+1] = byte((p >> 8) & 0xff)  // G
			w.shmMmap[dstRow+x*4+2] = byte(p & 0xff)         // R
			w.shmMmap[dstRow+x*4+3] = byte((p >> 24) & 0xff) // A
		}
	}

	// Attacher le buffer, déclarer le dommage et commiter
	if err := w.display.sendMsg(w.surfaceID, 1 /* attach */, w.shmBufferID, int32(0), int32(0)); err != nil {
		return err
	}
	if err := w.display.sendMsg(w.surfaceID, 2 /* damage */, int32(0), int32(0), int32(width), int32(height)); err != nil {
		return err
	}
	return w.display.sendMsg(w.surfaceID, 6 /* commit */)
}

func (w *WaylandWindow) ensureSHM(size, width, height, stride int) error {
	// F8: Vérification rigoureuse des dimensions géométriques en plus de la taille
	if w.shmMmap != nil && w.shmSize >= size && w.shmWidth == width && w.shmHeight == height && w.shmStride == stride {
		return nil
	}

	w.releaseSHM()

	f, fd, err := createAnonymousMemFD("c2display_wayland_shm", size)
	if err != nil {
		return err
	}
	w.shmFile = f
	w.shmFD = fd

	mmapData, err := syscall.Mmap(fd, 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		w.releaseSHM()
		return err
	}

	poolID := w.display.allocID()
	bufferID := w.display.allocID()

	// wl_shm.create_pool(poolID, fd, size) via SCM_RIGHTS
	if err := w.display.sendMsgWithFD(w.display.shmID, 0, fd, poolID, int32(size)); err != nil {
		syscall.Munmap(mmapData)
		w.releaseSHM()
		return err
	}

	// wl_shm_pool.create_buffer(bufferID, offset, width, height, stride, format=ARGB8888)
	format := wlShmFormatARGB8888
	if err := w.display.sendMsg(poolID, 0, bufferID, int32(0), int32(width), int32(height), int32(stride), format); err != nil {
		syscall.Munmap(mmapData)
		w.releaseSHM()
		return err
	}

	w.shmSize = size
	w.shmWidth = width
	w.shmHeight = height
	w.shmStride = stride
	w.shmMmap = mmapData
	w.shmPoolID = poolID
	w.shmBufferID = bufferID

	return nil
}

func (w *WaylandWindow) releaseSHM() {
	if w.shmMmap != nil {
		_ = syscall.Munmap(w.shmMmap)
		w.shmMmap = nil
	}
	if w.shmBufferID != 0 {
		_ = w.display.sendMsg(w.shmBufferID, 0 /* destroy */)
		w.shmBufferID = 0
	}
	if w.shmPoolID != 0 {
		_ = w.display.sendMsg(w.shmPoolID, 1 /* destroy */)
		w.shmPoolID = 0
	}
	if w.shmFile != nil {
		_ = w.shmFile.Close()
		w.shmFile = nil
	} else if w.shmFD > 0 {
		_ = syscall.Close(w.shmFD)
	}
	w.shmFD = 0
	w.shmSize = 0
	w.shmWidth = 0
	w.shmHeight = 0
	w.shmStride = 0
}

func createAnonymousMemFD(name string, size int) (*os.File, int, error) {
	// Syscall memfd_create Linux avec MFD_CLOEXEC (1)
	nameBytes := append([]byte(name), 0)
	fd, _, errno := syscall.Syscall(319, uintptr(unsafe.Pointer(&nameBytes[0])), 1 /* MFD_CLOEXEC */, 0)
	if errno == 0 && int(fd) >= 0 {
		if err := syscall.Ftruncate(int(fd), int64(size)); err != nil {
			syscall.Close(int(fd))
			return nil, -1, err
		}
		f := os.NewFile(fd, name)
		return f, int(fd), nil
	}

	// Repli sur fichier temporaire dans /dev/shm
	f, err := os.CreateTemp("/dev/shm", "c2display-shm-*")
	if err != nil {
		f, err = os.CreateTemp("", "c2display-shm-*")
		if err != nil {
			return nil, -1, err
		}
	}

	// Unlink immédiat pour rendre le fichier anonyme
	_ = os.Remove(f.Name())

	if err := f.Truncate(int64(size)); err != nil {
		f.Close()
		return nil, -1, err
	}

	return f, int(f.Fd()), nil
}

func (w *WaylandWindow) inject(e Event) {
	if e == nil {
		return
	}
	// F17: Ne coalescer que les événements de déplacement de souris (MouseMove)
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

	// Pour tous les événements critiques (Button, Key, Resize, Close) : transmission garantie
	select {
	case w.eventChan <- e:
	default:
		go func(ev Event) {
			w.eventChan <- ev
		}(e)
	}
}

func (w *WaylandWindow) Events() <-chan Event {
	return w.eventChan
}

func (w *WaylandWindow) PollEvent() Event {
	select {
	case ev := <-w.eventChan:
		return ev
	default:
		return nil
	}
}

func (w *WaylandWindow) WaitEvent() Event {
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
