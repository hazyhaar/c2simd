package c2tui

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"

	"code.hazyhaar.fr/devhoros/c2simd/pkg/c2grid"
	"code.hazyhaar.fr/devhoros/c2simd/pkg/c2pty"
	tuidiff "code.hazyhaar.fr/devhoros/c2simd/pkg/c2tuidiff"
	vtparser "code.hazyhaar.fr/devhoros/c2simd/pkg/c2vtparser"
)

// Session orchestre une session de terminal interactive complète :
// PTY Unix (c2pty) <-> Parser ANSI (c2vtparser) <-> Grille 2D (c2grid) <-> Diff SIMD (c2tuidiff).
type Session struct {
	mu         sync.Mutex
	cols       int
	rows       int
	pty        *c2pty.PTY
	grid       vtparser.CursorGrid
	parser     vtparser.Parser
	front      []Cell
	spans      []Span
	scrollback *c2grid.ScrollbackRing
	inRing     *c2pty.RingBuffer
	readBuf    [8192]byte
	renderBuf  bytes.Buffer
	done       chan struct{}
	closed     bool
}

// NewSession instancie une nouvelle session terminale aux dimensions spécifiées.
func NewSession(cols, rows int, scrollbackCap int) *Session {
	if cols < 1 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}
	if scrollbackCap <= 0 {
		scrollbackCap = 10000
	}

	s := &Session{
		cols:       cols,
		rows:       rows,
		front:      make([]Cell, cols*rows),
		spans:      make([]Span, 0, cols*rows),
		scrollback: c2grid.NewScrollbackRing(scrollbackCap),
		inRing:     c2pty.NewRingBuffer(256 * 1024), // 256 Ko de tampon non-bloquant
		done:       make(chan struct{}),
	}
	s.grid.Reset(cols, rows)
	s.parser.Reset(&s.grid)
	return s
}

// Start démarre l'exécution d'un processus dans le pseudo-terminal esclave associé.
func (s *Session) Start(command string, args ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pty != nil {
		return fmt.Errorf("c2tui: session déjà démarrée")
	}

	ws := &c2pty.Winsize{
		Rows: uint16(s.rows),
		Cols: uint16(s.cols),
	}
	p, err := c2pty.Open(ws)
	if err != nil {
		return fmt.Errorf("c2tui: échec ouverture PTY: %w", err)
	}
	s.pty = p

	if err := s.pty.Start(command, args...); err != nil {
		_ = s.pty.Close()
		s.pty = nil
		return fmt.Errorf("c2tui: échec démarrage processus: %w", err)
	}

	// Goroutine de drainage PTY asynchrone non-bloquante vers le RingBuffer
	go func() {
		buf := make([]byte, 8192)
		for {
			n, rerr := s.pty.Read(buf)
			if n > 0 {
				_ = s.inRing.WriteOverwrite(buf[:n])
			}
			if rerr != nil {
				s.inRing.Close()
				return
			}
		}
	}()

	return nil
}

// Resize propage un redimensionnement de fenêtre au PTY et à la grille matricielle.
func (s *Session) Resize(cols, rows int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cols < 1 || rows < 1 {
		return fmt.Errorf("c2tui: dimensions invalides %dx%d", cols, rows)
	}
	s.cols = cols
	s.rows = rows

	s.grid.Reset(cols, rows)
	s.parser.Reset(&s.grid)
	if cap(s.front) < cols*rows {
		s.front = make([]Cell, cols*rows)
	} else {
		s.front = s.front[:cols*rows]
		clear(s.front)
	}

	if s.pty != nil {
		return s.pty.Resize(uint16(cols), uint16(rows))
	}
	return nil
}

// WriteInput injecte des données utilisateur vers l'entrée du sous-processus PTY.
func (s *Session) WriteInput(data []byte) (int, error) {
	s.mu.Lock()
	p := s.pty
	s.mu.Unlock()

	if p == nil {
		return 0, fmt.Errorf("c2tui: PTY non initialisé")
	}
	return p.Write(data)
}

// Step ingère les octets disponibles sur le tampon circulaire, met à jour la grille et calcule le diff.
func (s *Session) Step(timeout time.Duration) (int, []Span, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pty == nil {
		return 0, nil, fmt.Errorf("c2tui: PTY non initialisé")
	}

	// Lecture non-bloquante sur le RingBuffer interne
	n, err := s.inRing.Read(s.readBuf[:])
	if n > 0 {
		s.parser.Feed(s.readBuf[:n])
	}
	if err == c2pty.ErrBufferEmpty {
		err = nil
	}
	if err != nil && err != io.EOF {
		return 0, nil, err
	}

	// Calcul du différentiel matriciel in-place (zéro allocation)
	s.spans = s.spans[:0]
	dirty := tuidiff.DiffGrid(s.front, s.grid.DiffCells(), s.cols, s.rows, s.cols, &s.spans)
	if dirty > 0 {
		copy(s.front, s.grid.DiffCells())
	}

	return dirty, s.spans, err
}

// RenderANSI génère un flux ANSI optimal pour projeter les cellules modifiées sur le terminal hôte.
func (s *Session) RenderANSI(spans []tuidiff.Span) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.renderBuf.Reset()
	for _, span := range spans {
		// Positionnement du curseur (1-based ANSI)
		fmt.Fprintf(&s.renderBuf, "\x1b[%d;%dH", span.Y+1, span.X+1)
		for i := 0; i < span.Length; i++ {
			idx := span.Y*s.cols + span.X + i
			if idx < len(s.front) {
				c := s.front[idx]
				r := c.Rune
				if r == 0 {
					r = ' '
				}
				s.renderBuf.WriteRune(r)
			}
		}
	}
	return s.renderBuf.Bytes()
}

// Close termine la session et ferme les descripteurs de fichiers associés.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true
	close(s.done)

	if s.pty != nil {
		return s.pty.Close()
	}
	return nil
}

// Wait attend la fin d'exécution du processus enfant dans le PTY.
func (s *Session) Wait() error {
	s.mu.Lock()
	p := s.pty
	s.mu.Unlock()

	if p == nil {
		return nil
	}
	return p.Wait()
}
