// Package c2tuidiff compare deux grilles de cellules terminales et produit
// les segments horizontaux de cellules modifiées (spans).
//
// DiffGrid est l'entrée publique. Le scalaire est le noyau C transpilé
// C2_diff_grid_scalar ; AVX2/NEON s'activent sous goexperiment.simd.
package c2tuidiff

import "unsafe"

var (
	_ [8]struct{} = [unsafe.Sizeof(Cell{})]struct{}{}
	_ [8]struct{} = [unsafe.Sizeof(C2_cell_t{})]struct{}{}
	_ [7]struct{} = [unsafe.Offsetof(Cell{}.Width)]struct{}{}
)

// Cell représente une cellule de grille terminale alignée sur 8 octets.
type Cell struct {
	Rune  rune  // 4 octets (caractère Unicode UTF-8)
	Fg    uint8 // 1 octet (couleur de premier plan ANSI 256)
	Bg    uint8 // 1 octet (couleur de fond ANSI 256)
	Flags uint8 // 1 octet (Gras, Souligné, Inversé, Dim)
	Width uint8 // 1 octet (largeur d'affichage ; 0 = défaut 1)
}

// Span décrit un segment horizontal de cellules modifiées.
type Span struct {
	X      int
	Y      int
	Length int
}

// DiffGrid compare front et back et ajoute à *spans les segments horizontaux
// de cellules modifiées. width et height décrivent la zone utile de chaque
// grille, stride le nombre de cellules par ligne (≥ width, padding de ligne).
// Renvoie le nombre total de cellules modifiées.
//
// Les grilles doivent contenir au moins (height-1)*stride+width cellules. En
// cas de layout invalide, la fonction retourne 0 sans écrire de span.
func DiffGrid(front, back []Cell, width, height, stride int, spans *[]Span) int {
	if spans == nil {
		return 0
	}
	return diffGridFn(front, back, width, height, stride, spans)
}
