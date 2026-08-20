// Package c2vtparser ingère un flux ANSI et mute une grille de cellules
// in-place. Parser.Feed est le chemin de production (UTF-8, wrap, scroll).
// Les symboles C2_vt_* sont un sous-ensemble C transpilé, pas un remplacement
// de Feed. OSC 52 et titres de fenêtre sont consommés sans être exécutés.
package c2vtparser

import (
	"sync/atomic"
	"unicode/utf8"
	"unsafe"
)

var (
	_ [8]struct{} = [unsafe.Sizeof(Cell{})]struct{}{}
	_ [8]struct{} = [unsafe.Sizeof(C2_cell_t{})]struct{}{}
	_ [7]struct{} = [unsafe.Offsetof(Cell{}.Width)]struct{}{}
)

// Attributs de style (CurAttr / Flags).
const (
	AttrBold      uint8 = 1 << iota // Gras
	AttrDim                         // Dim
	AttrUnderline                   // Souligné
	AttrInverse                     // Inversé
)

// Cell est une cellule de la grille terminale.
type Cell struct {
	Rune  rune
	Fg    uint8
	Bg    uint8
	Flags uint8
	Width uint8
}

// CursorGrid est la cible de dessin concrète, mutée in-place par l'automate.
// La grille est un tampon linéaire de width*height cellules, ligne par ligne.
type CursorGrid struct {
	Cells       []Cell
	Width       int
	Height      int
	CursorX     int
	CursorY     int
	SavedX      int
	SavedY      int
	Top         int
	Bottom      int
	Fg          uint8
	Bg          uint8
	Attr        uint8
	WrapPending bool
	Writes      int
}

// Reset (ré)initialise la grille pour width×height cellules. La mémoire du
// tampon Cells est réutilisée quand sa capacité le permet (zéro allocation).
func (g *CursorGrid) Reset(width, height int) {
	g.CursorX, g.CursorY = 0, 0
	g.SavedX, g.SavedY = 0, 0
	g.Fg, g.Bg, g.Attr = 0, 0, 0
	g.WrapPending = false
	g.Writes = 0
	if width < 1 || height < 1 || width > (int(^uint(0)>>1))/height {
		g.Width, g.Height = 0, 0
		g.Top, g.Bottom = 0, -1
		g.Cells = g.Cells[:0]
		return
	}
	n := width * height
	g.Width = width
	g.Height = height
	g.Top, g.Bottom = 0, height-1
	if cap(g.Cells) < n {
		g.Cells = make([]Cell, n)
	} else {
		g.Cells = g.Cells[:n]
		clear(g.Cells)
	}
}

// États de l'automate VT500 (modèle Paul Flo Williams simplifié).
const (
	stateGround uint8 = iota
	stateEscape
	stateCSIParam
	stateCSIInt
	stateCSIIgnore
	stateOSCEntry
	stateOSCString
	stateOSCEscape
	stateDCSEntry
	stateDCSInt
	stateDCSIgnore
	stateDCSStr
	stateDCSEscape
)

// Parser maintient l'état courant de l'automate VT500. Les champs exportés
// décrivent la sélection active (couleurs, attributs) et les paramètres de la
// séquence CSI en cours. Le pointeur de grille est concret : aucune interface.
type Parser struct {
	State     uint8
	CurFg     uint8
	CurBg     uint8
	CurAttr   uint8
	Params    [16]int
	NumParams int

	grid        *CursorGrid
	prm         int
	prmSet      bool // le paramètre courant a-t-il reçu au moins un chiffre ?
	oscNum      int
	private     uint8
	pendingUTF8 [4]byte
	pendingLen  uint8
	busy        uint32
}

// Reset prépare le parseur pour dessiner sur la grille g et remet l'automate
// à l'état GROUND.
func (p *Parser) Reset(g *CursorGrid) {
	p.State = stateGround
	p.CurFg, p.CurBg, p.CurAttr = 0, 0, 0
	p.NumParams = 0
	p.prm = 0
	p.prmSet = false
	p.oscNum = 0
	p.private = 0
	p.pendingLen = 0
	p.grid = g
}

// Feed ingère un buffer d'octets et fait évoluer l'automate in-place. Aucune
// allocation n'est réalisée. Renvoie le nombre de cellules écrites.
func (p *Parser) Feed(data []byte) int {
	g := p.grid
	if g == nil || g.Width < 1 || g.Height < 1 {
		return 0
	}
	if !atomic.CompareAndSwapUint32(&p.busy, 0, 1) {
		panic("c2vtparser: Feed concurrent")
	}
	defer atomic.StoreUint32(&p.busy, 0)
	writes := 0

	// Traiter d'abord les octets UTF-8 en suspens du Feed précédent
	if p.pendingLen > 0 && len(data) > 0 {
		for len(data) > 0 && p.pendingLen < 4 {
			p.pendingUTF8[p.pendingLen] = data[0]
			p.pendingLen++
			data = data[1:]
			r, size := utf8.DecodeRune(p.pendingUTF8[:p.pendingLen])
			if r != utf8.RuneError || size > 1 {
				g.put(r, p.CurFg, p.CurBg, p.CurAttr)
				writes++
				p.pendingLen = 0
				break
			}
			if !utf8.RuneStart(p.pendingUTF8[0]) || p.pendingLen == 4 {
				// Octet invalide
				g.put(r, p.CurFg, p.CurBg, p.CurAttr)
				writes++
				p.pendingLen = 0
				break
			}
		}
	}

	n := len(data)
	for i := 0; i < n; i++ {
		// Chemin rapide : régulier GROUND, consommation par lots d'un run de
		// texte ASCII imprimable. Les octets de contrôle interrompent le run.
		if p.State == stateGround {
			j := i
			// Scan accéléré par mot de 8 octets : (b - 0x20) < (0x80 - 0x20)
			for j+8 <= n {
				w := *(*uint64)(unsafe.Pointer(&data[j]))
				hasLow := (w - 0x2020202020202020) & 0x8080808080808080
				hasHigh := w & 0x8080808080808080
				if (hasLow | hasHigh) != 0 {
					break
				}
				j += 8
			}
			for j < n && data[j] >= 0x20 && data[j] < 0x80 {
				j++
			}
			if j > i {
				writes += g.putRun(data[i:j], p.CurFg, p.CurBg, p.CurAttr)
				i = j - 1
				continue
			}
		}
		// Chemin rapide CSI : boucle serrée sur les paramètres (chiffres,
		// séparateurs, marqueurs privés). Tout autre octet (final, CAN…)
		// sort de la boucle et est traité par le chemin lent.
		if p.State == stateCSIParam {
			for i < n {
				b := data[i]
				switch {
				case b >= '0' && b <= '9':
					if p.prm < 65535 {
						p.prm = p.prm*10 + int(b-'0')
						if p.prm > 65535 {
							p.prm = 65535
						}
					}
					p.prmSet = true
					i++
				case b == ';':
					p.addParam()
					i++
				case b == '?' || b == '>' || b == '<' || b == '=':
					p.private = b
					i++
				default:
					goto csiFastDone
				}
			}
		csiFastDone:
			if i >= n {
				break // fin de buffer en cours de séquence : état préservé
			}
		}
		b := data[i]
		switch p.State {
		case stateGround:
			switch {
			case b == 0x1b:
				p.State = stateEscape
			case b == 0x0d:
				g.CursorX = 0
				g.WrapPending = false
			case b == 0x0a || b == 0x0b || b == 0x0c:
				g.newline()
			case b == 0x08:
				if g.CursorX > 0 {
					g.CursorX--
				}
				g.WrapPending = false
			case b == 0x09:
				g.tab()
			case b == 0x07, b == 0x00:
				// BEL et NUL : ignorés (pas d'alarme audible ni d'action).
			case b == 0x18 || b == 0x1a:
				p.State = stateGround // CAN/SUB : retour à l'état GROUND
			case b < 0x20:
				// Autres C0 : ignorés.
			default:
				// Séquence UTF-8 : si la séquence est tronquée en fin de buffer,
				// stocker les octets dans pendingUTF8 pour le prochain Feed.
				if !utf8.FullRune(data[i:]) && utf8.RuneStart(data[i]) {
					p.pendingLen = uint8(copy(p.pendingUTF8[:], data[i:]))
					i = n // fin de buffer consommée
					break
				}
				r, size := utf8.DecodeRune(data[i:])
				g.put(r, p.CurFg, p.CurBg, p.CurAttr)
				writes++
				i += size - 1
			}
		case stateEscape:
			switch {
			case b == '[':
				// '[' est consommé comme introducteur CSI ; le prochain octet
				// est le premier paramètre de la séquence.
				p.State = stateCSIParam
				p.NumParams = 0
				p.prm = 0
				p.prmSet = false
				p.private = 0
			case b == ']':
				p.State = stateOSCEntry
				p.oscNum = 0
			case b == 'P' || b == '_' || b == '^' || b == 'X':
				// 'P' (DCS), '_' (APC), '^' (PM), 'X' (SOS) : consommés et neutralisés
				// jusqu'au terminateur ST (\x1b\) ou BEL (0x07).
				p.State = stateDCSStr
				p.NumParams = 0
				p.prm = 0
				p.prmSet = false
				p.private = 0
			case b == '7':
				g.SavedX, g.SavedY = g.CursorX, g.CursorY
				p.State = stateGround
			case b == '8':
				g.CursorX, g.CursorY = g.SavedX, g.SavedY
				p.State = stateGround
			case b == 'D': // IND : index
				g.newline()
				p.State = stateGround
			case b == 'M': // RI : reverse index
				g.reverseIndex()
				p.State = stateGround
			case b == 'E': // NEL : next line
				g.CursorX = 0
				g.newline()
				p.State = stateGround
			case b == 'c': // RIS : reset
				p.CurFg, p.CurBg, p.CurAttr = 0, 0, 0
				p.State = stateGround
			case b == 0x18 || b == 0x1a:
				p.State = stateGround
			default:
				// Séquences ESC inconnues : ignorées sans action.
				p.State = stateGround
			}
		case stateCSIParam:
			switch {
			case b >= '0' && b <= '9':
				if p.prm < 65535 {
					p.prm = p.prm*10 + int(b-'0')
					if p.prm > 65535 {
						p.prm = 65535
					}
				}
				p.prmSet = true
			case b == ';':
				p.addParam()
			case b == '?' || b == '>' || b == '<' || b == '=':
				p.private = b
			case b >= 0x20 && b <= 0x2f:
				p.State = stateCSIInt
			case b >= 0x40 && b <= 0x7e:
				p.finishParam()
				p.dispatchCSI(b)
				p.State = stateGround
			case b == 0x18 || b == 0x1a:
				p.State = stateGround
			default:
				p.State = stateCSIIgnore
			}
		case stateCSIInt:
			if b >= 0x40 && b <= 0x7e {
				p.finishParam()
				p.dispatchCSI(b)
				p.State = stateGround
			} else if b == 0x18 || b == 0x1a {
				p.State = stateGround
			}
		case stateCSIIgnore:
			if b >= 0x40 && b <= 0x7e || b == 0x18 || b == 0x1a {
				p.State = stateGround
			}
		case stateOSCEntry:
			switch {
			case b >= '0' && b <= '9':
				p.oscNum = p.oscNum*10 + int(b-'0')
			case b == ';':
				p.State = stateOSCString
			case b == 0x07: // BEL : fin d'OSC, neutralisé
				p.State = stateGround
			case b == 0x1b:
				p.State = stateOSCEscape
			case b == 0x18 || b == 0x1a:
				p.State = stateGround
			default:
				// Caractère inattendu : on consomme (neutralisation).
				p.State = stateOSCString
			}
		case stateOSCString:
			switch {
			case b == 0x07: // BEL : fin d'OSC, neutralisé
				p.State = stateGround
			case b == 0x1b:
				p.State = stateOSCEscape
			case b == 0x18 || b == 0x1a:
				p.State = stateGround
			}
			// Tout autre octet : payload consommé sans interprétation.
			// Neutralisation explicite d'OSC 52 (presse-papier) et des
			// réécritures de titres (OSC 0/1/2) : rien n'est stocké, rien
			// n'est exécuté.
		case stateOSCEscape:
			switch {
			case b == '\\' || b == 0x07: // ST ou BEL : fin d'OSC
				p.State = stateGround
			case b == 0x1b:
				// ESC répété : on reste en attente.
			default:
				p.State = stateOSCString
			}
		case stateDCSEntry:
			switch {
			case b >= 0x20 && b <= 0x2f, b >= 0x30 && b <= 0x3f:
				p.State = stateDCSInt
			case b >= 0x40 && b <= 0x7e:
				p.State = stateDCSStr
			case b == 0x18 || b == 0x1a:
				p.State = stateGround
			default:
				p.State = stateDCSIgnore
			}
		case stateDCSInt:
			switch {
			case b >= 0x20 && b <= 0x2f, b >= 0x30 && b <= 0x3f:
				// Paramètres/intermédiaires DCS collectés (inutiles ici).
			case b >= 0x40 && b <= 0x7e:
				p.State = stateDCSStr
			case b == 0x18 || b == 0x1a:
				p.State = stateGround
			default:
				p.State = stateDCSIgnore
			}
		case stateDCSIgnore:
			if b >= 0x40 && b <= 0x7e || b == 0x18 || b == 0x1a {
				p.State = stateGround
			}
		case stateDCSStr:
			switch {
			case b == 0x07: // BEL : fin de séquence
				p.State = stateGround
			case b == 0x1b:
				p.State = stateDCSEscape
			case b == 0x18 || b == 0x1a:
				p.State = stateGround
			}
		case stateDCSEscape:
			switch {
			case b == '\\' || b == 0x07: // ST ou BEL : fin de DCS/APC/PM/SOS
				p.State = stateGround
			case b == 0x1b:
				// ESC répété.
			default:
				p.State = stateDCSStr
			}
		}
	}
	g.Writes += writes
	return writes
}

// addParam enregistre le paramètre en cours et repart de zéro.
func (p *Parser) addParam() {
	if p.NumParams < len(p.Params) {
		p.Params[p.NumParams] = p.prm
	}
	if p.NumParams < len(p.Params) {
		p.NumParams++
	}
	p.prm = 0
	p.prmSet = false
}

// finishParam clôt la liste de paramètres d'une séquence CSI : le paramètre
// courant est enregistré s'il a reçu des chiffres ; une séquence sans aucun
// paramètre produit le défaut 0.
func (p *Parser) finishParam() {
	if p.prmSet || p.NumParams == 0 {
		p.addParam()
	}
}

// param renvoie le paramètre d'index i avec défaut 0 (paramètres vides).
func (p *Parser) param(i int) int {
	if i < p.NumParams {
		return p.Params[i]
	}
	return 0
}

// dispatchCSI applique l'effet de la séquence CSI finale.
func (p *Parser) dispatchCSI(final byte) {
	g := p.grid
	if p.NumParams == 0 {
		p.addParam()
	}
	switch final {
	case 'm':
		p.applySGR()
	case 'H', 'f':
		g.setCursor(p.param(1), p.param(0))
	case 'A':
		g.move(0, -max(1, p.param(0)))
	case 'B':
		g.move(0, max(1, p.param(0)))
	case 'C':
		g.move(max(1, p.param(0)), 0)
	case 'D':
		g.move(-max(1, p.param(0)), 0)
	case 'G', '`':
		g.setCursor(p.param(0), g.CursorY+1)
	case 'd':
		g.setCursor(g.CursorX+1, p.param(0))
	case 'J':
		g.clearScreen(p.param(0))
	case 'K':
		g.clearLine(p.param(0))
	case 's':
		g.SavedX, g.SavedY = g.CursorX, g.CursorY
	case 'u':
		g.CursorX, g.CursorY = g.SavedX, g.SavedY
	case 'r':
		if p.NumParams >= 2 {
			top, bot := p.param(0), p.param(1)
			if top >= 1 && top <= bot && bot <= g.Height {
				g.Top, g.Bottom = top-1, bot-1
			}
		}
	case 'S':
		g.scrollUp(max(1, p.param(0)))
	case 'T':
		g.scrollDown(max(1, p.param(0)))
	case 'h', 'l', 'n', 'c', 'q', '@', 'P', 'X':
		// Modes, DSR, DA, styles de curseur, insertions : ignorés.
		// Aucune réponse n'est émise (pas de sortie terminale).
	default:
		// Séquence inconnue : ignorée sans effet.
	}
}

// applySGR applique la sélection graphique (couleurs, attributs, remise à zéro).
func (p *Parser) applySGR() {
	i := 0
	for i < p.NumParams {
		v := p.Params[i]
		switch {
		case v == 0:
			p.CurFg, p.CurBg, p.CurAttr = 0, 0, 0
		case v == 1:
			p.CurAttr |= AttrBold
		case v == 2:
			p.CurAttr |= AttrDim
		case v == 4:
			p.CurAttr |= AttrUnderline
		case v == 7:
			p.CurAttr |= AttrInverse
		case v == 22:
			p.CurAttr &^= AttrBold | AttrDim
		case v == 23:
			p.CurAttr &^= AttrDim
		case v == 24:
			p.CurAttr &^= AttrUnderline
		case v == 27:
			p.CurAttr &^= AttrInverse
		case v == 39:
			p.CurFg = 0
		case v == 49:
			p.CurBg = 0
		case v >= 30 && v <= 37:
			p.CurFg = uint8(v - 30)
		case v == 38:
			switch {
			case p.NumParams-i >= 3 && p.Params[i+1] == 5:
				p.CurFg = uint8(p.Params[i+2])
				i += 2
			case p.NumParams-i >= 5 && p.Params[i+1] == 2:
				p.CurFg = nearestColor(p.Params[i+2], p.Params[i+3], p.Params[i+4])
				i += 4
			}
		case v >= 40 && v <= 47:
			p.CurBg = uint8(v - 40)
		case v == 48:
			switch {
			case p.NumParams-i >= 3 && p.Params[i+1] == 5:
				p.CurBg = uint8(p.Params[i+2])
				i += 2
			case p.NumParams-i >= 5 && p.Params[i+1] == 2:
				p.CurBg = nearestColor(p.Params[i+2], p.Params[i+3], p.Params[i+4])
				i += 4
			}
		case v >= 90 && v <= 97:
			p.CurFg = uint8(v - 90 + 8)
		case v >= 100 && v <= 107:
			p.CurBg = uint8(v - 100 + 8)
		}
		i++
	}
}

// put écrit une cellule à la position du curseur avec retour à la ligne et
// défilement automatiques.
func (g *CursorGrid) put(r rune, fg, bg, flags uint8) {
	g.putRunByte(r, fg, bg, flags)
}

// putRunByte écrit une cellule unique (partagé avec putRun).
func (g *CursorGrid) putRunByte(r rune, fg, bg, flags uint8) {
	if g.WrapPending {
		g.WrapPending = false
		g.CursorX = 0
		g.CursorY++
		if g.CursorY > g.Bottom {
			g.scrollUp(1)
			g.CursorY = g.Bottom
		}
	}
	idx := g.CursorY*g.Width + g.CursorX
	if idx >= 0 && idx < len(g.Cells) {
		g.Cells[idx] = Cell{Rune: r, Fg: fg, Bg: bg, Flags: flags, Width: 1}
	}
	if g.CursorX+1 >= g.Width {
		g.WrapPending = true
	} else {
		g.CursorX++
	}
}

// putRun écrit un run de texte ASCII imprimable en une seule passe, avec le
// même comportement de retour à la ligne et de défilement que put. Renvoie le
// nombre de cellules écrites.
//
// Chemin chaud : une boucle interne serrée écrit les cellules d'une ligne sans
// branche de wrap ni check de bornes (le curseur est toujours dans la grille) ;
// le retour à la ligne et le défilement sont traités entre les lignes.
func (g *CursorGrid) putRun(text []byte, fg, bg, flags uint8) int {
	width := g.Width
	if width < 1 || len(g.Cells) == 0 {
		return 0
	}
	bottom := g.Bottom
	cells := g.Cells
	cx, cy := g.CursorX, g.CursorY
	wp := g.WrapPending
	written := 0
	end := len(text)

	// Attributs combinés précalculés pour l'alignement Little-Endian de Cell (Rune: 4 octets, Fg: octet 4, Bg: octet 5, Flags: octet 6) :
	attrPrefix := uint64(fg)<<32 | uint64(bg)<<40 | uint64(flags)<<48 | uint64(1)<<56

	for written < end {
		if wp {
			wp = false
			cx = 0
			cy++
			if cy > bottom {
				g.scrollUp(1)
				cy = bottom
				cells = g.Cells
			}
		}
		row := cy * width
		maxLine := width - cx
		if maxLine <= 0 {
			break
		}
		toWrite := end - written
		if toWrite > maxLine {
			toWrite = maxLine
		}
		if row < 0 || row+cx+toWrite > len(cells) {
			break
		}

		// Boucle interne serrée déroulée x4 : injection uint64 directe
		rowOffset := row + cx
		k := 0
		for k+4 <= toWrite {
			*(*uint64)(unsafe.Pointer(&cells[rowOffset+k])) = attrPrefix | uint64(text[written+k])
			*(*uint64)(unsafe.Pointer(&cells[rowOffset+k+1])) = attrPrefix | uint64(text[written+k+1])
			*(*uint64)(unsafe.Pointer(&cells[rowOffset+k+2])) = attrPrefix | uint64(text[written+k+2])
			*(*uint64)(unsafe.Pointer(&cells[rowOffset+k+3])) = attrPrefix | uint64(text[written+k+3])
			k += 4
		}
		for k < toWrite {
			*(*uint64)(unsafe.Pointer(&cells[rowOffset+k])) = attrPrefix | uint64(text[written+k])
			k++
		}
		cx += toWrite
		written += toWrite

		if cx >= width {
			cx = width - 1
			wp = true
		}
	}
	g.CursorX, g.CursorY = cx, cy
	g.WrapPending = wp
	return written
}

// newline avance le curseur d'une ligne, avec défilement si nécessaire.
func (g *CursorGrid) newline() {
	g.CursorY++
	if g.CursorY > g.Bottom {
		g.scrollUp(1)
		g.CursorY = g.Bottom
	}
}

// reverseIndex recule le curseur d'une ligne, avec défilement vers le bas.
func (g *CursorGrid) reverseIndex() {
	if g.CursorY == g.Top {
		g.scrollDown(1)
	} else if g.CursorY > 0 {
		g.CursorY--
	}
}

// tab avance le curseur à la prochaine tabulation (pas de 8).
func (g *CursorGrid) tab() {
	next := (g.CursorX/8 + 1) * 8
	if next < g.Width {
		g.CursorX = next
	} else {
		g.CursorX = g.Width - 1
	}
}

// setCursor positionne le curseur en coordonnées 1-based (clampées).
func (g *CursorGrid) setCursor(x, y int) {
	g.WrapPending = false
	if x < 1 {
		x = 1
	}
	if y < 1 {
		y = 1
	}
	if x > g.Width {
		x = g.Width
	}
	if y > g.Height {
		y = g.Height
	}
	g.CursorX, g.CursorY = x-1, y-1
}

// move déplace le curseur relativement (clampé à la grille).
func (g *CursorGrid) move(dx, dy int) {
	g.WrapPending = false
	nx, ny := g.CursorX+dx, g.CursorY+dy
	if nx < 0 {
		nx = 0
	}
	if ny < 0 {
		ny = 0
	}
	if nx >= g.Width {
		nx = g.Width - 1
	}
	if ny >= g.Height {
		ny = g.Height - 1
	}
	g.CursorX, g.CursorY = nx, ny
}

// clearScreen efface l'écran selon le mode (0 : dessous, 1 : dessus, 2/3 : tout).
func (g *CursorGrid) clearScreen(mode int) {
	switch mode {
	case 0:
		g.clearRowPart(g.CursorY, g.CursorX, g.Width)
		for y := g.CursorY + 1; y < g.Height; y++ {
			g.clearRowPart(y, 0, g.Width)
		}
	case 1:
		for y := 0; y < g.CursorY; y++ {
			g.clearRowPart(y, 0, g.Width)
		}
		g.clearRowPart(g.CursorY, 0, g.CursorX+1)
	case 2, 3:
		for y := 0; y < g.Height; y++ {
			g.clearRowPart(y, 0, g.Width)
		}
	}
}

// clearLine efface la ligne selon le mode (0 : droite, 1 : gauche, 2 : tout).
func (g *CursorGrid) clearLine(mode int) {
	switch mode {
	case 0:
		g.clearRowPart(g.CursorY, g.CursorX, g.Width)
	case 1:
		g.clearRowPart(g.CursorY, 0, g.CursorX+1)
	case 2:
		g.clearRowPart(g.CursorY, 0, g.Width)
	}
}

// clearRowPart efface une portion de ligne avec des cellules vides.
func (g *CursorGrid) clearRowPart(y, x0, x1 int) {
	if y < 0 || y >= g.Height {
		return
	}
	row := y * g.Width
	if x0 < 0 {
		x0 = 0
	}
	if x1 > g.Width {
		x1 = g.Width
	}
	if x0 < x1 {
		start := row + x0
		end := row + x1
		if start >= 0 && end <= len(g.Cells) {
			clear(g.Cells[start:end])
		}
	}
}

var (
	scrollUseC bool
	scrollHook func(n, cells int)
)

func (g *CursorGrid) c2cells() []C2_cell_t {
	if len(g.Cells) == 0 {
		return nil
	}
	return unsafe.Slice((*C2_cell_t)(unsafe.Pointer(&g.Cells[0])), len(g.Cells))
}

func (g *CursorGrid) scrollUp(n int) {
	top, bot := g.Top, g.Bottom
	if top >= bot || n <= 0 {
		return
	}
	line := bot - top + 1
	if n > line {
		n = line
	}
	if scrollHook != nil {
		scrollHook(n, line*g.Width)
	}
	if scrollUseC && top == 0 && bot == g.Height-1 {
		C2_grid_scroll_up(g.c2cells(), g.Width, g.Height, g.Width, n)
		return
	}
	move := (line - n) * g.Width
	dst := top * g.Width
	src := (top + n) * g.Width
	copy(g.Cells[dst:dst+move], g.Cells[src:src+move])
	for y := bot - n + 1; y <= bot; y++ {
		g.clearRowPart(y, 0, g.Width)
	}
}

// scrollDown fait défiler la région de défilement vers le bas de n lignes,
// les lignes du haut étant effacées.
func (g *CursorGrid) scrollDown(n int) {
	top, bot := g.Top, g.Bottom
	if top >= bot || n <= 0 {
		return
	}
	line := bot - top + 1
	if n > line {
		n = line
	}
	move := (line - n) * g.Width
	dst := (top + n) * g.Width
	src := top * g.Width
	copy(g.Cells[dst:dst+move], g.Cells[src:src+move])
	for y := top; y < top+n; y++ {
		g.clearRowPart(y, 0, g.Width)
	}
}

// palette256 est la table précalculée des 256 couleurs xterm (thésaurisation
// archtime : calculée une fois avant l'exécution, jamais au runtime des
// parses).
var palette256 [256][3]uint8

func init() {
	sys := [16][3]uint8{
		{0, 0, 0}, {128, 0, 0}, {0, 128, 0}, {128, 128, 0},
		{0, 0, 128}, {128, 0, 128}, {0, 128, 128}, {192, 192, 192},
		{128, 128, 128}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0},
		{0, 0, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
	}
	copy(palette256[:16], sys[:])
	cube := [6]uint8{0, 95, 135, 175, 215, 255}
	for r := 0; r < 6; r++ {
		for g := 0; g < 6; g++ {
			for b := 0; b < 6; b++ {
				palette256[16+r*36+g*6+b] = [3]uint8{cube[r], cube[g], cube[b]}
			}
		}
	}
	for i := 0; i < 24; i++ {
		v := uint8(8 + i*10)
		palette256[232+i] = [3]uint8{v, v, v}
	}
}

// nearestColor projette une couleur RGB 24-bit sur l'index 256 couleurs le
// plus proche (distance euclidienne).
//
// La palette est structurée : 16 couleurs système, un cube 6×6×6 et 24 gris.
// Le cube étant une grille séparable, son plus proche voisin est obtenu par
// arrondi canal par canal ; le plus proche global est le minimum sur les
// 16 système ∪ le cube arrondi ∪ 24 gris. Exact et sans allocation : 41
// distances au lieu de 256 (thésaurisation archtime).
func nearestColor(r, g, b int) uint8 {
	rr, gg, bb := uint8(r), uint8(g), uint8(b)
	best := uint8(0)
	bestD := int32(1<<30 - 1)
	// Couleurs système (0..15).
	for i := 0; i < 16; i++ {
		c := palette256[i]
		dr := int32(c[0]) - int32(rr)
		dg := int32(c[1]) - int32(gg)
		db := int32(c[2]) - int32(bb)
		if d := dr*dr + dg*dg + db*db; d < bestD {
			bestD, best = d, uint8(i)
		}
	}
	// Cube 6×6×6 : arrondi canal par canal sur {0,95,135,175,215,255}.
	cr := cubeIndex(r)
	cg := cubeIndex(g)
	cb := cubeIndex(b)
	c := palette256[16+cr*36+cg*6+cb]
	dr := int32(c[0]) - int32(rr)
	dg := int32(c[1]) - int32(gg)
	db := int32(c[2]) - int32(bb)
	if d := dr*dr + dg*dg + db*db; d < bestD {
		bestD, best = d, uint8(16+cr*36+cg*6+cb)
	}
	// Gris (232..255).
	for i := 232; i < 256; i++ {
		c := palette256[i]
		dr := int32(c[0]) - int32(rr)
		dg := int32(c[1]) - int32(gg)
		db := int32(c[2]) - int32(bb)
		if d := dr*dr + dg*dg + db*db; d < bestD {
			bestD, best = d, uint8(i)
		}
	}
	return best
}

// cubeIndex retourne l'indice du niveau de cube le plus proche pour un canal.
func cubeIndex(v int) int {
	switch {
	case v < 47:
		return 0
	case v < 115:
		return 1
	case v < 155:
		return 2
	case v < 195:
		return 3
	case v < 235:
		return 4
	default:
		return 5
	}
}
