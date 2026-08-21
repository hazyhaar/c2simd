package c2display

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	c2painter "code.hazyhaar.fr/devhoros/pkg/c2painter"
)

// X11Display représente une connexion native au serveur X11 sans dépendance CGO.
type X11Display struct {
	mu          sync.RWMutex
	conn        net.Conn
	closed      bool
	resourceID  uint32
	idBase      uint32
	idMask      uint32
	rootWindow  uint32
	rootVisual  uint32
	whitePixel  uint32
	blackPixel  uint32
	screens     []ScreenInfo
	windows     map[uint32]*X11Window
	wmDeleteWin uint32
	wmProtocols uint32
}

// NewX11Display établit la connexion au socket X11.
func NewX11Display(opts DisplayOptions) (*X11Display, error) {
	displayName := opts.DisplayName
	if displayName == "" {
		displayName = os.Getenv("DISPLAY")
	}
	if displayName == "" {
		displayName = ":0"
	}

	displayNum := "0"
	if strings.HasPrefix(displayName, ":") {
		parts := strings.Split(strings.TrimPrefix(displayName, ":"), ".")
		displayNum = parts[0]
	}

	sockPath := opts.SocketPath
	if sockPath == "" {
		sockPath = "/tmp/.X11-unix/X" + displayNum
	}

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		// Tentative via TCP localhost si Unix socket échoue
		tcpAddr := fmt.Sprintf("127.0.0.1:%d", 6000)
		conn, err = net.Dial("tcp", tcpAddr)
		if err != nil {
			return nil, fmt.Errorf("%w: impossible de joindre X11 sur %s: %v", ErrNoServerFound, sockPath, err)
		}
	}

	// Handshake de connexion X11 (Little Endian 'l' = 0x6c)
	handshake := make([]byte, 12)
	handshake[0] = 0x6c                               // Little-endian
	binary.LittleEndian.PutUint16(handshake[2:4], 11) // Version majeure
	binary.LittleEndian.PutUint16(handshake[4:6], 0)  // Version mineure

	if _, err := conn.Write(handshake); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("%w: échec écriture poignée de main: %v", ErrHandshakeFailed, err)
	}

	// Lecture de l'en-tête de réponse (8 octets)
	header := make([]byte, 8)
	if _, err := io.ReadFull(conn, header); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("%w: échec lecture en-tête: %v", ErrHandshakeFailed, err)
	}

	status := header[0]
	if status != 1 {
		_ = conn.Close()
		return nil, fmt.Errorf("%w: refus du serveur X11 (statut %d)", ErrHandshakeFailed, status)
	}

	extraLen := int(binary.LittleEndian.Uint16(header[6:8])) * 4
	extraData := make([]byte, extraLen)
	if _, err := io.ReadFull(conn, extraData); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("%w: échec lecture corps réponse: %v", ErrHandshakeFailed, err)
	}

	// Parsing des métadonnées du serveur
	idBase := binary.LittleEndian.Uint32(extraData[4:8])
	idMask := binary.LittleEndian.Uint32(extraData[8:12])
	vendorLen := int(binary.LittleEndian.Uint16(extraData[16:18]))
	numScreens := int(extraData[20])
	numFormats := int(extraData[21])

	offset := 32 + ((vendorLen + 3) & ^3) + numFormats*8

	var rootWin, rootVis, whitePix, blackPix uint32
	var widthPx, heightPx, widthMM, heightMM int

	screens := make([]ScreenInfo, 0, numScreens)
	for i := 0; i < numScreens && offset+40 <= len(extraData); i++ {
		rootWin = binary.LittleEndian.Uint32(extraData[offset : offset+4])
		whitePix = binary.LittleEndian.Uint32(extraData[offset+8 : offset+12])
		blackPix = binary.LittleEndian.Uint32(extraData[offset+12 : offset+16])
		widthPx = int(binary.LittleEndian.Uint16(extraData[offset+20 : offset+22]))
		heightPx = int(binary.LittleEndian.Uint16(extraData[offset+22 : offset+24]))
		widthMM = int(binary.LittleEndian.Uint16(extraData[offset+24 : offset+26]))
		heightMM = int(binary.LittleEndian.Uint16(extraData[offset+26 : offset+28]))
		rootVis = binary.LittleEndian.Uint32(extraData[offset+32 : offset+36])
		numDepths := int(extraData[offset+39])

		screens = append(screens, ScreenInfo{
			ID:          i,
			Width:       widthPx,
			Height:      heightPx,
			WidthMM:     widthMM,
			HeightMM:    heightMM,
			Depth:       24,
			ScaleFactor: 1.0,
			Primary:     i == 0,
		})

		// Sauter les profondeurs
		offset += 40
		for d := 0; d < numDepths && offset+8 <= len(extraData); d++ {
			numVisuals := int(binary.LittleEndian.Uint16(extraData[offset+2 : offset+4]))
			offset += 8 + numVisuals*24
		}
	}

	disp := &X11Display{
		conn:       conn,
		resourceID: 0,
		idBase:     idBase,
		idMask:     idMask,
		rootWindow: rootWin,
		rootVisual: rootVis,
		whitePixel: whitePix,
		blackPixel: blackPix,
		screens:    screens,
		windows:    make(map[uint32]*X11Window),
	}

	// Résolution synchrone des atomes WM_PROTOCOLS et WM_DELETE_WINDOW
	if wmProto, err := disp.internAtom("WM_PROTOCOLS"); err == nil {
		disp.wmProtocols = wmProto
	}
	if wmDel, err := disp.internAtom("WM_DELETE_WINDOW"); err == nil {
		disp.wmDeleteWin = wmDel
	}

	// Démarrage de la boucle asynchrone de lecture des événements X11
	go disp.eventLoop()

	return disp, nil
}

// internAtom résout un identifiant d'atome X11 par son nom via la requête InternAtom (Opcode 16).
func (d *X11Display) internAtom(name string) (uint32, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	nameLen := len(name)
	pad := (4 - (nameLen % 4)) % 4
	reqLenWords := (8 + nameLen + pad) / 4

	req := make([]byte, reqLenWords*4)
	req[0] = 16 // InternAtom
	req[1] = 0  // onlyIfExists = false
	binary.LittleEndian.PutUint16(req[2:4], uint16(reqLenWords))
	binary.LittleEndian.PutUint16(req[4:6], uint16(nameLen))
	copy(req[8:], name)

	if _, err := d.conn.Write(req); err != nil {
		return 0, fmt.Errorf("échec écriture InternAtom(%s): %w", name, err)
	}

	var reply [32]byte
	if _, err := io.ReadFull(d.conn, reply[:]); err != nil {
		return 0, fmt.Errorf("échec lecture réponse InternAtom(%s): %w", name, err)
	}

	if reply[0] != 1 {
		return 0, fmt.Errorf("erreur réponse InternAtom(%s): status=%d", name, reply[0])
	}

	atom := binary.LittleEndian.Uint32(reply[8:12])
	return atom, nil
}

func (d *X11Display) allocID() uint32 {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.resourceID++
	return d.idBase | (d.resourceID & d.idMask)
}

func (d *X11Display) Type() BackendType {
	return BackendX11
}

func (d *X11Display) Screens() []ScreenInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	res := make([]ScreenInfo, len(d.screens))
	copy(res, d.screens)
	return res
}

func (d *X11Display) Flush() error {
	return nil
}

func (d *X11Display) CreateWindow(opts WindowOptions) (Window, error) {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil, ErrDisplayClosed
	}
	d.mu.Unlock()

	w := opts.Width
	if w <= 0 {
		w = 800
	}
	h := opts.Height
	if h <= 0 {
		h = 600
	}

	winID := d.allocID()
	gcID := d.allocID()

	// 1. CreateWindow (Opcode 1)
	createWinReq := make([]byte, 32+8)
	createWinReq[0] = 1  // CreateWindow
	createWinReq[1] = 24 // Depth = 24
	binary.LittleEndian.PutUint16(createWinReq[2:4], 10)
	binary.LittleEndian.PutUint32(createWinReq[4:8], winID)
	binary.LittleEndian.PutUint32(createWinReq[8:12], d.rootWindow)
	binary.LittleEndian.PutUint16(createWinReq[12:14], uint16(opts.X))
	binary.LittleEndian.PutUint16(createWinReq[14:16], uint16(opts.Y))
	binary.LittleEndian.PutUint16(createWinReq[16:18], uint16(w))
	binary.LittleEndian.PutUint16(createWinReq[18:20], uint16(h))
	binary.LittleEndian.PutUint16(createWinReq[20:22], 0) // Border width
	binary.LittleEndian.PutUint16(createWinReq[22:24], 1) // InputOutput
	binary.LittleEndian.PutUint32(createWinReq[24:28], d.rootVisual)

	// Masque d'attributs (BackgroundPixel + EventMask)
	valueMask := uint32(0x00000002 | 0x00000800)
	binary.LittleEndian.PutUint32(createWinReq[28:32], valueMask)
	binary.LittleEndian.PutUint32(createWinReq[32:36], d.blackPixel)

	// EventMask : KeyPress(1), KeyRelease(2), ButtonPress(4), ButtonRelease(8),
	// PointerMotion(64), Exposure(32768), StructureNotify(131072)
	eventMask := uint32(1 | 2 | 4 | 8 | 64 | 32768 | 131072)
	binary.LittleEndian.PutUint32(createWinReq[36:40], eventMask)

	if _, err := d.conn.Write(createWinReq); err != nil {
		return nil, fmt.Errorf("échec CreateWindow: %w", err)
	}

	// 2. CreateGC (Opcode 55)
	createGCReq := make([]byte, 16)
	createGCReq[0] = 55 // CreateGC
	binary.LittleEndian.PutUint16(createGCReq[2:4], 4)
	binary.LittleEndian.PutUint32(createGCReq[4:8], gcID)
	binary.LittleEndian.PutUint32(createGCReq[8:12], winID)
	binary.LittleEndian.PutUint32(createGCReq[12:16], 0)

	if _, err := d.conn.Write(createGCReq); err != nil {
		return nil, fmt.Errorf("échec CreateGC: %w", err)
	}

	// 3. F10: Enregistrement de WM_PROTOCOLS -> WM_DELETE_WINDOW (Opcode 18 ChangeProperty)
	if d.wmProtocols != 0 && d.wmDeleteWin != 0 {
		propReq := make([]byte, 28)
		propReq[0] = 18 // ChangeProperty
		propReq[1] = 0  // ModeReplace
		binary.LittleEndian.PutUint16(propReq[2:4], 7)
		binary.LittleEndian.PutUint32(propReq[4:8], winID)
		binary.LittleEndian.PutUint32(propReq[8:12], d.wmProtocols)
		binary.LittleEndian.PutUint32(propReq[12:16], 4) // ATOM = 4
		propReq[16] = 32                               // Format 32-bit
		binary.LittleEndian.PutUint32(propReq[20:24], 1)
		binary.LittleEndian.PutUint32(propReq[24:28], d.wmDeleteWin)
		_, _ = d.conn.Write(propReq)
	}

	// 4. MapWindow (Opcode 8)
	mapReq := make([]byte, 8)
	mapReq[0] = 8 // MapWindow
	binary.LittleEndian.PutUint16(mapReq[2:4], 2)
	binary.LittleEndian.PutUint32(mapReq[4:8], winID)

	if _, err := d.conn.Write(mapReq); err != nil {
		return nil, fmt.Errorf("échec MapWindow: %w", err)
	}

	win := &X11Window{
		display:   d,
		id:        winID,
		gcID:      gcID,
		title:     opts.Title,
		width:     w,
		height:    h,
		eventChan: make(chan Event, 256),
		closeChan: make(chan struct{}),
		bgraBuf:   make([]byte, w*h*4),
	}

	d.mu.Lock()
	d.windows[winID] = win
	d.mu.Unlock()

	return win, nil
}

func (d *X11Display) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}

// X11Window gère le rendu et les interactions d'une fenêtre X11 en pur Go.
type X11Window struct {
	mu         sync.RWMutex
	display    *X11Display
	id         uint32
	gcID       uint32
	title      string
	width      int
	height     int
	closed     bool
	frameCount uint64
	eventChan  chan Event
	closeChan  chan struct{}
	bgraBuf    []byte
}

func (w *X11Window) ID() uint32 {
	return w.id
}

func (w *X11Window) Title() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.title
}

func (w *X11Window) SetTitle(title string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.title = title
	// Opcode 18: ChangeProperty WM_NAME
	titleBytes := []byte(title)
	pad := (4 - (len(titleBytes) % 4)) % 4
	reqLen := 6 + (len(titleBytes)+pad)/4
	req := make([]byte, reqLen*4)
	req[0] = 18 // ChangeProperty
	req[1] = 0  // Replace
	binary.LittleEndian.PutUint16(req[2:4], uint16(reqLen))
	binary.LittleEndian.PutUint32(req[4:8], w.id)
	binary.LittleEndian.PutUint32(req[8:12], 39)  // Atom WM_NAME (39)
	binary.LittleEndian.PutUint32(req[12:16], 31) // Atom STRING (31)
	req[16] = 8                                   // Format 8-bit
	binary.LittleEndian.PutUint32(req[20:24], uint32(len(titleBytes)))
	copy(req[24:], titleBytes)

	w.display.mu.Lock()
	_, err := w.display.conn.Write(req)
	w.display.mu.Unlock()
	return err
}

func (w *X11Window) Bounds() (int, int) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.width, w.height
}

func (w *X11Window) Resize(width, height int) error {
	if width <= 0 || height <= 0 {
		return ErrInvalidGeometry
	}
	w.mu.Lock()
	w.width = width
	w.height = height
	w.bgraBuf = make([]byte, width*height*4)
	w.mu.Unlock()
	return nil
}

// Present transmet la surface c2painter à la fenêtre X11 via PutImage ZPixmap.
func (w *X11Window) Present(surface *c2painter.Surface) error {
	if surface == nil || surface.Pixels == nil {
		return ErrNilSurface
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return ErrWindowClosed
	}

	width := surface.Width
	height := surface.Height

	if len(w.bgraBuf) < width*height*4 {
		w.bgraBuf = make([]byte, width*height*4)
	}

	// Conversion RGBA -> BGRA / 32-bit pour serveur X11
	srcPix := surface.Pixels
	dstBuf := w.bgraBuf
	total := width * height

	for i := 0; i < total; i++ {
		p := srcPix[i]
		r := byte(p)
		g := byte(p >> 8)
		b := byte(p >> 16)
		a := byte(p >> 24)

		off := i * 4
		dstBuf[off+0] = b
		dstBuf[off+1] = g
		dstBuf[off+2] = r
		dstBuf[off+3] = a
	}

	// Opcode 72: PutImage (ZPixmap = 2, Depth = 24)
	// Découpage dynamique en tranches pour respecter rigoureusement la limite CARD16 (65535 mots)
	// Opcode 72: PutImage (ZPixmap = 2, Depth = 24)
	// Découpage dynamique en tranches pour respecter rigoureusement la limite CARD16 (65535 mots)
	// F7: Borne stricte maxLinesPerChunk = 65529 / width pour garantir reqLen = 6 + chunkH*width <= 65535
	maxLinesPerChunk := 65529 / width
	if maxLinesPerChunk < 1 {
		maxLinesPerChunk = 1
	} else if maxLinesPerChunk > 128 {
		maxLinesPerChunk = 128
	}
	for y := 0; y < height; y += maxLinesPerChunk {
		chunkH := maxLinesPerChunk
		if y+chunkH > height {
			chunkH = height - y
		}

		dataLen := width * chunkH * 4
		reqLen := 6 + dataLen/4
		header := make([]byte, 24)
		header[0] = 72 // PutImage
		header[1] = 2  // ZPixmap
		binary.LittleEndian.PutUint16(header[2:4], uint16(reqLen))
		binary.LittleEndian.PutUint32(header[4:8], w.id)
		binary.LittleEndian.PutUint32(header[8:12], w.gcID)
		binary.LittleEndian.PutUint16(header[12:14], uint16(width))
		binary.LittleEndian.PutUint16(header[14:16], uint16(chunkH))
		binary.LittleEndian.PutUint16(header[16:18], 0)         // Dst X
		binary.LittleEndian.PutUint16(header[18:20], uint16(y)) // Dst Y
		header[20] = 0                                          // Left pad
		header[21] = 24                                         // Depth

		w.display.mu.Lock()
		if _, err := w.display.conn.Write(header); err != nil {
			w.display.mu.Unlock()
			return err
		}
		startOffset := y * width * 4
		endOffset := startOffset + dataLen
		if _, err := w.display.conn.Write(dstBuf[startOffset:endOffset]); err != nil {
			w.display.mu.Unlock()
			return err
		}
		w.display.mu.Unlock()
	}

	atomic.AddUint64(&w.frameCount, 1)
	return nil
}

func (w *X11Window) Events() <-chan Event {
	return w.eventChan
}

func (w *X11Window) PollEvent() Event {
	select {
	case ev := <-w.eventChan:
		return ev
	default:
		return nil
	}
}

func (w *X11Window) WaitEvent() Event {
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

func (w *X11Window) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	w.closed = true
	close(w.closeChan)
	return nil
}

// eventLoop centralisé sur X11Display lisant en continu les paquets d'événements X11 de 32 octets.
func (d *X11Display) eventLoop() {
	buf := make([]byte, 32)
	for {
		if _, err := io.ReadFull(d.conn, buf); err != nil {
			return
		}

		eventType := buf[0] & 0x7F

		d.mu.RLock()
		var targetWin *X11Window
		wID1 := binary.LittleEndian.Uint32(buf[4:8])
		wID2 := binary.LittleEndian.Uint32(buf[8:12])
		if w, ok := d.windows[wID1]; ok {
			targetWin = w
		} else if w, ok := d.windows[wID2]; ok {
			targetWin = w
		} else {
			for _, w := range d.windows {
				targetWin = w
				break
			}
		}
		d.mu.RUnlock()

		if targetWin != nil {
			targetWin.handleX11Event(eventType, buf)
		}
	}
}

func (w *X11Window) handleX11Event(eventType byte, buf []byte) {
	switch eventType {
	case 2: // KeyPress
		scancode := uint32(buf[1])
		detail := binary.LittleEndian.Uint16(buf[28:30]) // State modifiers
		kCode, r := EvdevScancodeToKeyCode(scancode - 8)
		w.sendEvent(KeyEvent{
			Action:    EventKeyPress,
			Key:       kCode,
			Rune:      r,
			Scancode:  scancode,
			Modifiers: KeyModifiers(detail),
		})

	case 3: // KeyRelease
		scancode := uint32(buf[1])
		detail := binary.LittleEndian.Uint16(buf[28:30])
		kCode, r := EvdevScancodeToKeyCode(scancode - 8)
		w.sendEvent(KeyEvent{
			Action:    EventKeyRelease,
			Key:       kCode,
			Rune:      r,
			Scancode:  scancode,
			Modifiers: KeyModifiers(detail),
		})

	case 4: // ButtonPress
		btn := MouseButton(buf[1])
		x := int(int16(binary.LittleEndian.Uint16(buf[24:26])))
		y := int(int16(binary.LittleEndian.Uint16(buf[26:28])))
		w.sendEvent(MouseEvent{
			Action: EventMousePress,
			Button: btn,
			X:      x,
			Y:      y,
		})

	case 5: // ButtonRelease
		btn := MouseButton(buf[1])
		x := int(int16(binary.LittleEndian.Uint16(buf[24:26])))
		y := int(int16(binary.LittleEndian.Uint16(buf[26:28])))
		w.sendEvent(MouseEvent{
			Action: EventMouseRelease,
			Button: btn,
			X:      x,
			Y:      y,
		})

	case 6: // MotionNotify
		x := int(int16(binary.LittleEndian.Uint16(buf[24:26])))
		y := int(int16(binary.LittleEndian.Uint16(buf[26:28])))
		w.sendEvent(MouseEvent{
			Action: EventMouseMove,
			X:      x,
			Y:      y,
		})

	case 12: // Expose
		x := int(binary.LittleEndian.Uint16(buf[8:10]))
		y := int(binary.LittleEndian.Uint16(buf[10:12]))
		ew := int(binary.LittleEndian.Uint16(buf[12:14]))
		eh := int(binary.LittleEndian.Uint16(buf[14:16]))
		w.sendEvent(ExposeEvent{
			X:      x,
			Y:      y,
			Width:  ew,
			Height: eh,
		})

	case 22: // ConfigureNotify
		cw := int(binary.LittleEndian.Uint16(buf[20:22]))
		ch := int(binary.LittleEndian.Uint16(buf[22:24]))
		w.mu.Lock()
		w.width = cw
		w.height = ch
		w.mu.Unlock()
		w.sendEvent(ResizeEvent{
			Width:  cw,
			Height: ch,
		})

	case 33: // ClientMessage (ex. WM_DELETE_WINDOW)
		if len(buf) >= 20 {
			atom := binary.LittleEndian.Uint32(buf[16:20])
			if w.display.wmDeleteWin != 0 && atom == w.display.wmDeleteWin {
				w.sendEvent(CloseEvent{})
			}
		}
	}
}

func (w *X11Window) sendEvent(e Event) {
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

	// Pour tous les événements d'état critiques : envoi garanti sans perte
	select {
	case w.eventChan <- e:
	default:
		go func(ev Event) {
			w.eventChan <- ev
		}(e)
	}
}
