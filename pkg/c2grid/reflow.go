package c2grid

// Reflow resizes the grid and reflows text. Basic implementation.
func (g *Grid) Reflow(newCols, newRows int) {
	if newCols == g.cols && newRows == g.rows {
		return
	}
	
	newPrimary := make([]C2_cell_t, newCols*newRows)
	newAlternate := make([]C2_cell_t, newCols*newRows)
	
	minCols := g.cols
	if newCols < minCols {
		minCols = newCols
	}
	minRows := g.rows
	if newRows < minRows {
		minRows = newRows
	}
	
	for y := 0; y < minRows; y++ {
		srcStart := y * g.cols
		dstStart := y * newCols
		copy(newPrimary[dstStart:dstStart+minCols], g.primary[srcStart:srcStart+minCols])
		copy(newAlternate[dstStart:dstStart+minCols], g.alternate[srcStart:srcStart+minCols])
	}
	
	g.cols = newCols
	g.rows = newRows
	g.primary = newPrimary
	g.alternate = newAlternate
	
	if g.active == &g.primary {
		g.active = &g.primary
	} else {
		g.active = &g.alternate
	}
	
	if g.cursorX >= g.cols {
		g.cursorX = g.cols - 1
	}
	if g.cursorY >= g.rows {
		g.cursorY = g.rows - 1
	}
}
