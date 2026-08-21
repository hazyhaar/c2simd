package c2tui

import (
	tuidiff "code.hazyhaar.fr/devhoros/c2simd/pkg/c2tuidiff"
	vtparser "code.hazyhaar.fr/devhoros/c2simd/pkg/c2vtparser"
)

type (
	Cell       = tuidiff.Cell
	Span       = tuidiff.Span
	Parser     = vtparser.Parser
	CursorGrid = vtparser.CursorGrid
)

func DiffGrid(front, back []Cell, width, height, stride int, spans *[]Span) int {
	return tuidiff.DiffGrid(front, back, width, height, stride, spans)
}
