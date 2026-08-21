package c2grid

type Grid struct {
	cols    int
	rows    int
	cursorX int
	cursorY int
	
	primary   []C2_cell_t
	alternate []C2_cell_t
	active    *[]C2_cell_t

	currentFg    uint8
	currentBg    uint8
	currentFlags uint8
}

func NewGrid(cols, rows int) *Grid {
	g := &Grid{
		cols:      cols,
		rows:      rows,
		primary:   make([]C2_cell_t, cols*rows),
		alternate: make([]C2_cell_t, cols*rows),
	}
	g.active = &g.primary
	return g
}

func (g *Grid) PutCell(r uint32, width uint8) {
	if g.cursorX >= g.cols || g.cursorY >= g.rows {
		return
	}
	idx := g.cursorY*g.cols + g.cursorX
	(*g.active)[idx] = C2_cell_t{
		Rune:  r,
		Fg:    g.currentFg,
		Bg:    g.currentBg,
		Flags: g.currentFlags,
		Width: width,
	}
	g.cursorX++
	if g.cursorX >= g.cols {
		g.cursorX = 0
		g.cursorY++
		if g.cursorY >= g.rows {
			g.cursorY = g.rows - 1
			g.ScrollUp(1)
		}
	}
}

func (g *Grid) Clear() {
	buf := *g.active
	for i := range buf {
		buf[i] = C2_cell_t{Bg: g.currentBg}
	}
	g.cursorX = 0
	g.cursorY = 0
}

func (g *Grid) ScrollUp(n int) {
	if n <= 0 || n >= g.rows {
		g.Clear()
		return
	}
	buf := *g.active
	copy(buf, buf[n*g.cols:])
	clearStart := (g.rows - n) * g.cols
	for i := clearStart; i < len(buf); i++ {
		buf[i] = C2_cell_t{Bg: g.currentBg}
	}
}

func (g *Grid) ScrollDown(n int) {
	if n <= 0 || n >= g.rows {
		g.Clear()
		return
	}
	buf := *g.active
	copy(buf[n*g.cols:], buf[:(g.rows-n)*g.cols])
	clearEnd := n * g.cols
	for i := 0; i < clearEnd; i++ {
		buf[i] = C2_cell_t{Bg: g.currentBg}
	}
}

func (g *Grid) SetSGR(fg, bg, flags uint8) {
	g.currentFg = fg
	g.currentBg = bg
	g.currentFlags = flags
}

func (g *Grid) SwapScreen() {
	if g.active == &g.primary {
		g.active = &g.alternate
	} else {
		g.active = &g.primary
	}
}
