package c2vtparser

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// Modes de reporting souris (DECSET 1000 / 1002 / 1003).
const (
	MouseModeNone   = 0
	MouseModeX10    = 1000 // Clics uniquement
	MouseModeButton = 1002 // Clics et mouvements avec bouton pressé (drag)
	MouseModeAny    = 1003 // Tous les mouvements souris
)

// Constantes de modes DECSET / DECRST étendus.
const (
	ModeDECOLM         = 3 // 80/132 colonnes
	ModeDECSCNM        = 5 // Écran inversé (Screen Inverse)
	ModeDECAWM         = 7 // Auto-wrap à la marge droite
	ModeCursorVisible  = 25
	ModeAltScreen47    = 47
	ModeAltScreen1049  = 1049
	ModeBracketedPaste = 2004
	ModeSyncUpdate     = 2026 // Synchronized Output (DEC 2026 / anti-tearing)
	ModeMouseX10       = 1000
	ModeMouseButton    = 1002
	ModeMouseAny       = 1003
	ModeMouseSGR       = 1006 // Format étendu <btn;x;yM
	ModeMouseSGRPixels = 1016 // Format pixels <btn;x;yM
)

// appendOSC ajoute un octet au buffer OSC courant avec protection de borne (1 Mo max).
func (p *Parser) appendOSC(b byte) {
	if p.oscLen < len(p.oscStatic) {
		p.oscStatic[p.oscLen] = b
		p.oscLen++
		return
	}
	if p.oscBuf == nil {
		p.oscBuf = make([]byte, len(p.oscStatic), 8192)
		copy(p.oscBuf, p.oscStatic[:])
	}
	if len(p.oscBuf) < 1024*1024 { // Cap à 1 Mo max
		p.oscBuf = append(p.oscBuf, b)
		p.oscLen++
	}
}

// getOSCPayload renvoie le payload accumulé sous forme de slice d'octets.
func (p *Parser) getOSCPayload() []byte {
	if p.oscBuf != nil {
		return p.oscBuf[:p.oscLen]
	}
	return p.oscStatic[:p.oscLen]
}

// resetOSC remet à zéro le buffer OSC.
func (p *Parser) resetOSC() {
	p.oscNum = 0
	p.oscLen = 0
	if p.oscBuf != nil {
		p.oscBuf = p.oscBuf[:0]
	}
}

// dispatchOSC traite l'exécution de la commande OSC accumulée.
func (p *Parser) dispatchOSC() {
	payload := string(p.getOSCPayload())
	num := p.oscNum
	p.resetOSC()

	switch num {
	case 52: // OSC 52 : Presse-papier (Clipboard)
		p.handleOSC52(payload)
	case 8: // OSC 8 : Hyperliens cliquables
		p.handleOSC8(payload)
	case 0, 1, 2: // OSC 0, 1, 2 : Titres de fenêtre et d'icône
		p.handleOSCTitle(payload)
	}
}

// handleOSC52 extrait la cible et décode le payload Base64 de manière sécurisée.
func (p *Parser) handleOSC52(payload string) {
	semi := strings.IndexByte(payload, ';')
	if semi < 0 {
		return
	}
	target := payload[:semi]
	dataStr := payload[semi+1:]

	if dataStr == "?" {
		if p.OnClipboardQuery != nil {
			p.OnClipboardQuery(target)
		}
		return
	}

	// Nettoyage des retours à la ligne pouvant polluer le flux base64
	cleanB64 := strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, dataStr)

	decoded, err := base64.StdEncoding.DecodeString(cleanB64)
	if err != nil {
		// Essai avec l'encodage RawStd si padding absent
		decoded, err = base64.RawStdEncoding.DecodeString(cleanB64)
		if err != nil {
			return
		}
	}

	if p.OnClipboardWrite != nil {
		p.OnClipboardWrite(target, decoded)
	}
}

// handleOSC8 gère l'ouverture et la fermeture des hyperliens (OSC 8 ; params ; url ST).
func (p *Parser) handleOSC8(payload string) {
	semi := strings.IndexByte(payload, ';')
	if semi < 0 {
		// Pas de séparateur : fermeture par défaut
		p.CurLinkID = ""
		p.CurLinkURL = ""
		if p.OnHyperlink != nil {
			p.OnHyperlink("", "", false)
		}
		return
	}
	params := payload[:semi]
	url := payload[semi+1:]

	var linkID string
	if len(params) > 0 {
		for _, part := range strings.Split(params, ":") {
			if strings.HasPrefix(part, "id=") {
				linkID = strings.TrimPrefix(part, "id=")
				break
			}
		}
	}

	if len(url) > 0 {
		p.CurLinkID = linkID
		p.CurLinkURL = url
		if p.OnHyperlink != nil {
			p.OnHyperlink(linkID, url, true)
		}
	} else {
		p.CurLinkID = ""
		p.CurLinkURL = ""
		if p.OnHyperlink != nil {
			p.OnHyperlink("", "", false)
		}
	}
}

// handleOSCTitle filtre les métacaractères et notifie du changement de titre.
func (p *Parser) handleOSCTitle(payload string) {
	// Filtrage strict des caractères de contrôle C0 et métacaractères
	sanitized := strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7F {
			return -1
		}
		return r
	}, payload)

	if p.OnTitleChange != nil {
		p.OnTitleChange(sanitized)
	}
}

// dispatchDECModes gère l'activation (final == 'h') ou désactivation (final == 'l') des modes DEC.
func (p *Parser) dispatchDECModes(final byte) {
	active := (final == 'h')
	for i := 0; i < p.NumParams; i++ {
		mode := p.Params[i]
		switch mode {
		case ModeSyncUpdate:
			p.SyncUpdate = active
			if p.OnSyncUpdate != nil {
				p.OnSyncUpdate(active)
			}
		case ModeDECAWM:
			if p.grid != nil {
				p.grid.AutoWrap = active
			}
		case ModeDECOLM:
			p.DECOLM132 = active
			if p.grid != nil {
				cols := 80
				if active {
					cols = 132
				}
				p.grid.Reset(cols, p.grid.Height)
			}
			if p.OnDECOLM != nil {
				p.OnDECOLM(active)
			}
		case ModeDECSCNM:
			p.InverseScreen = active
			if p.OnScreenInverse != nil {
				p.OnScreenInverse(active)
			}
		case ModeMouseX10:
			if active {
				p.MouseMode = MouseModeX10
			} else if p.MouseMode == MouseModeX10 {
				p.MouseMode = MouseModeNone
			}
		case ModeMouseButton:
			if active {
				p.MouseMode = MouseModeButton
			} else if p.MouseMode == MouseModeButton {
				p.MouseMode = MouseModeNone
			}
		case ModeMouseAny:
			if active {
				p.MouseMode = MouseModeAny
			} else if p.MouseMode == MouseModeAny {
				p.MouseMode = MouseModeNone
			}
		case ModeMouseSGR:
			p.MouseSGR = active
		case ModeMouseSGRPixels:
			p.MouseSGRPixels = active
		case ModeAltScreen47, ModeAltScreen1049:
			p.AltScreen = active
			if p.OnAltScreen != nil {
				p.OnAltScreen(active)
			}
		case ModeCursorVisible:
			p.CursorVisible = active
			if p.OnCursorVisible != nil {
				p.OnCursorVisible(active)
			}
		case ModeBracketedPaste:
			p.BracketedPaste = active
			if p.OnBracketedPaste != nil {
				p.OnBracketedPaste(active)
			}
		}
	}
}

// EncodeMouseEvent encode un événement souris au format X10 ou SGR étendu (<btn;x;yM ou <btn;xym).
func EncodeMouseEvent(btn, x, y int, release bool, sgr bool) []byte {
	if sgr {
		action := byte('M')
		if release {
			action = 'm'
		}
		return []byte(fmt.Sprintf("\x1b[<%d;%d;%d%c", btn, x+1, y+1, action))
	}
	// Format X10 standard avec décalage +32
	cb := byte(32 + btn)
	if release {
		cb = byte(32 + 3) // Bouton 3 = release en X10 standard
	}
	cx := byte(32 + x + 1)
	cy := byte(32 + y + 1)
	return []byte{0x1b, '[', 'M', cb, cx, cy}
}

// EncodeBracketedPaste enveloppe le contenu dans les balises de paste bracketed (\x1b[200~ ... \x1b[201~).
func EncodeBracketedPaste(content []byte) []byte {
	res := make([]byte, 0, len(content)+12)
	res = append(res, "\x1b[200~"...)
	res = append(res, content...)
	res = append(res, "\x1b[201~"...)
	return res
}

// EncodeSyncUpdate émet la séquence CSI ? 2026 h/l de début ou fin de trame synchronisée.
func EncodeSyncUpdate(start bool) []byte {
	if start {
		return []byte("\x1b[?2026h")
	}
	return []byte("\x1b[?2026l")
}
