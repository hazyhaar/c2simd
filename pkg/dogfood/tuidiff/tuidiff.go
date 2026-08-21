// Package tuidiff fournit l'implémentation dogfoodée issue du retour d'expérience sgoiter
// pour la comparaison vectorisée de cellules de grilles terminales.
package tuidiff

import (
	c2tuidiff "code.hazyhaar.fr/devhoros/c2simd/pkg/c2tuidiff"
)

// DiffWrapper expose le diffing sous forme d'utilitaire haut niveau zéro-alloc.
type DiffWrapper struct {
	Spans []c2tuidiff.Span
}

func NewDiffWrapper(initialCapacity int) *DiffWrapper {
	return &DiffWrapper{
		Spans: make([]c2tuidiff.Span, 0, initialCapacity),
	}
}

// Diff réinitialise et calcule les segments de cellules modifiées entre front et back.
func (d *DiffWrapper) Diff(front, back []c2tuidiff.Cell, width, height, stride int) int {
	d.Spans = d.Spans[:0]
	return c2tuidiff.DiffGrid(front, back, width, height, stride, &d.Spans)
}
