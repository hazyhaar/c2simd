package c2vtparser

import (
	"unsafe"

	"code.hazyhaar.fr/devhoros/c2simd/internal/tuidiff"
)

var _ [unsafe.Sizeof(c2tuidiff.Cell{})]struct{} = [unsafe.Sizeof(Cell{})]struct{}{}

// DiffCells expose la grille comme []c2tuidiff.Cell, zéro copie, même layout 8 octets.
func (g *CursorGrid) DiffCells() []c2tuidiff.Cell {
	if len(g.Cells) == 0 {
		return nil
	}
	return unsafe.Slice((*c2tuidiff.Cell)(unsafe.Pointer(&g.Cells[0])), len(g.Cells))
}
