package c2display

// MouseButton identifie un bouton physique ou virtuel de souris.
type MouseButton uint8

const (
	ButtonNone MouseButton = iota
	ButtonLeft
	ButtonMiddle
	ButtonRight
	ButtonScrollUp
	ButtonScrollDown
	ButtonScrollLeft
	ButtonScrollRight
	ButtonExtra1
	ButtonExtra2
)

// KeyModifiers encode l'état des modificateurs clavier et souris.
type KeyModifiers uint16

const (
	ModShift   KeyModifiers = 1 << 0
	ModLock    KeyModifiers = 1 << 1 // CapsLock
	ModControl KeyModifiers = 1 << 2
	ModAlt     KeyModifiers = 1 << 3 // Mod1
	ModSuper   KeyModifiers = 1 << 4 // Mod4 / Windows / Super
	ModNumLock KeyModifiers = 1 << 5
)

// ConvertX11Modifiers convertit un masque d'état X11 (state) en KeyModifiers normalisé.
func ConvertX11Modifiers(state uint16) KeyModifiers {
	var m KeyModifiers
	if (state & 1) != 0 {
		m |= ModShift
	}
	if (state & 2) != 0 {
		m |= ModLock
	}
	if (state & 4) != 0 {
		m |= ModControl
	}
	if (state & 8) != 0 {
		m |= ModAlt
	}
	if (state & 64) != 0 {
		m |= ModSuper
	}
	if (state & 16) != 0 {
		m |= ModNumLock
	}
	return m
}

func x11StateToModifiers(state uint16) KeyModifiers {
	return ConvertX11Modifiers(state)
}

// KeyCode représente une touche normalisée indépendamment du protocole d'affichage.
type KeyCode uint16

const (
	KeyUnknown KeyCode = iota

	// Touches de contrôle
	KeyEscape
	KeyEnter
	KeyTab
	KeyBackspace
	KeyInsert
	KeyDelete
	KeySpace

	// Touches de navigation
	KeyLeft
	KeyUp
	KeyRight
	KeyDown
	KeyPageUp
	KeyPageDown
	KeyHome
	KeyEnd

	// Modificateurs
	KeyShiftLeft
	KeyShiftRight
	KeyControlLeft
	KeyControlRight
	KeyAltLeft
	KeyAltRight
	KeySuperLeft
	KeySuperRight
	KeyCapsLock
	KeyNumLock
	KeyScrollLock

	// Touches de fonction
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12

	// Chiffres rangée supérieure
	Key0
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9

	// Lettres
	KeyA
	KeyB
	KeyC
	KeyD
	KeyE
	KeyF
	KeyG
	KeyH
	KeyI
	KeyJ
	KeyK
	KeyL
	KeyM
	KeyN
	KeyO
	KeyP
	KeyQ
	KeyR
	KeyS
	KeyT
	KeyU
	KeyV
	KeyW
	KeyX
	KeyY
	KeyZ

	// Pavé numérique
	KeyKp0
	KeyKp1
	KeyKp2
	KeyKp3
	KeyKp4
	KeyKp5
	KeyKp6
	KeyKp7
	KeyKp8
	KeyKp9
	KeyKpDecimal
	KeyKpDivide
	KeyKpMultiply
	KeyKpSubtract
	KeyKpAdd
	KeyKpEnter
	KeyKpEqual

	// Ponctuation
	KeyMinus
	KeyEqual
	KeyBracketLeft
	KeyBracketRight
	KeyBackslash
	KeySemicolon
	KeyApostrophe
	KeyGrave
	KeyComma
	KeyPeriod
	KeySlash

	// Commandes système
	KeyPrintScreen
	KeyPause
	KeyMenu
)

// KeySymToKeyCode convertit un Keysym X11 en KeyCode normalisé et rune associée.
func KeySymToKeyCode(keysym uint32) (KeyCode, rune) {
	// ASCII standard direct
	if keysym >= 0x20 && keysym <= 0x7e {
		r := rune(keysym)
		switch {
		case keysym >= 'a' && keysym <= 'z':
			return KeyA + KeyCode(keysym-'a'), r
		case keysym >= 'A' && keysym <= 'Z':
			return KeyA + KeyCode(keysym-'A'), r
		case keysym >= '0' && keysym <= '9':
			return Key0 + KeyCode(keysym-'0'), r
		case keysym == ' ':
			return KeySpace, ' '
		case keysym == '-':
			return KeyMinus, '-'
		case keysym == '=':
			return KeyEqual, '='
		case keysym == '[':
			return KeyBracketLeft, '['
		case keysym == ']':
			return KeyBracketRight, ']'
		case keysym == '\\':
			return KeyBackslash, '\\'
		case keysym == ';':
			return KeySemicolon, ';'
		case keysym == '\'':
			return KeyApostrophe, '\''
		case keysym == '`':
			return KeyGrave, '`'
		case keysym == ',':
			return KeyComma, ','
		case keysym == '.':
			return KeyPeriod, '.'
		case keysym == '/':
			return KeySlash, '/'
		}
	}

	// Caractères Unicode directs (0x01000000 | codepoint)
	if (keysym & 0xff000000) == 0x01000000 {
		r := rune(keysym & 0x00ffffff)
		return KeyUnknown, r
	}

	// Keysyms X11 spéciaux (0xff00 .. 0xffff)
	switch keysym {
	case 0xff08: // XK_BackSpace
		return KeyBackspace, 0x08
	case 0xff09: // XK_Tab
		return KeyTab, '\t'
	case 0xff0d: // XK_Return
		return KeyEnter, '\r'
	case 0xff1b: // XK_Escape
		return KeyEscape, 0x1b
	case 0xffff: // XK_Delete
		return KeyDelete, 0x7f
	case 0xff50: // XK_Home
		return KeyHome, 0
	case 0xff51: // XK_Left
		return KeyLeft, 0
	case 0xff52: // XK_Up
		return KeyUp, 0
	case 0xff53: // XK_Right
		return KeyRight, 0
	case 0xff54: // XK_Down
		return KeyDown, 0
	case 0xff55: // XK_Prior (PageUp)
		return KeyPageUp, 0
	case 0xff56: // XK_Next (PageDown)
		return KeyPageDown, 0
	case 0xff57: // XK_End
		return KeyEnd, 0
	case 0xff63: // XK_Insert
		return KeyInsert, 0
	case 0xff61: // XK_Print
		return KeyPrintScreen, 0
	case 0xff13: // XK_Pause
		return KeyPause, 0
	case 0xffe1: // XK_Shift_L
		return KeyShiftLeft, 0
	case 0xffe2: // XK_Shift_R
		return KeyShiftRight, 0
	case 0xffe3: // XK_Control_L
		return KeyControlLeft, 0
	case 0xffe4: // XK_Control_R
		return KeyControlRight, 0
	case 0xffe9: // XK_Alt_L
		return KeyAltLeft, 0
	case 0xffea: // XK_Alt_R
		return KeyAltRight, 0
	case 0xffeb: // XK_Super_L
		return KeySuperLeft, 0
	case 0xffec: // XK_Super_R
		return KeySuperRight, 0
	case 0xffe5: // XK_Caps_Lock
		return KeyCapsLock, 0
	case 0xff7f: // XK_Num_Lock
		return KeyNumLock, 0
	case 0xff14: // XK_Scroll_Lock
		return KeyScrollLock, 0
	case 0xff67: // XK_Menu
		return KeyMenu, 0

	// Touches de fonction F1..F12
	case 0xffbe:
		return KeyF1, 0
	case 0xffbf:
		return KeyF2, 0
	case 0xffc0:
		return KeyF3, 0
	case 0xffc1:
		return KeyF4, 0
	case 0xffc2:
		return KeyF5, 0
	case 0xffc3:
		return KeyF6, 0
	case 0xffc4:
		return KeyF7, 0
	case 0xffc5:
		return KeyF8, 0
	case 0xffc6:
		return KeyF9, 0
	case 0xffc7:
		return KeyF10, 0
	case 0xffc8:
		return KeyF11, 0
	case 0xffc9:
		return KeyF12, 0

	// Pavé numérique
	case 0xffb0:
		return KeyKp0, '0'
	case 0xffb1:
		return KeyKp1, '1'
	case 0xffb2:
		return KeyKp2, '2'
	case 0xffb3:
		return KeyKp3, '3'
	case 0xffb4:
		return KeyKp4, '4'
	case 0xffb5:
		return KeyKp5, '5'
	case 0xffb6:
		return KeyKp6, '6'
	case 0xffb7:
		return KeyKp7, '7'
	case 0xffb8:
		return KeyKp8, '8'
	case 0xffb9:
		return KeyKp9, '9'
	case 0xffae:
		return KeyKpDecimal, '.'
	case 0xffaf:
		return KeyKpDivide, '/'
	case 0xffaa:
		return KeyKpMultiply, '*'
	case 0xffad:
		return KeyKpSubtract, '-'
	case 0xffab:
		return KeyKpAdd, '+'
	case 0xff8d:
		return KeyKpEnter, '\r'
	case 0xffbd:
		return KeyKpEqual, '='
	}

	return KeyUnknown, 0
}

// EvdevScancodeToKeyCode convertit un scancode matériel Linux evdev en KeyCode et rune de base.
func EvdevScancodeToKeyCode(scancode uint32) (KeyCode, rune) {
	switch scancode {
	case 1:
		return KeyEscape, 0x1b
	case 2:
		return Key1, '1'
	case 3:
		return Key2, '2'
	case 4:
		return Key3, '3'
	case 5:
		return Key4, '4'
	case 6:
		return Key5, '5'
	case 7:
		return Key6, '6'
	case 8:
		return Key7, '7'
	case 9:
		return Key8, '8'
	case 10:
		return Key9, '9'
	case 11:
		return Key0, '0'
	case 12:
		return KeyMinus, '-'
	case 13:
		return KeyEqual, '='
	case 14:
		return KeyBackspace, 0x08
	case 15:
		return KeyTab, '\t'
	case 16:
		return KeyQ, 'q'
	case 17:
		return KeyW, 'w'
	case 18:
		return KeyE, 'e'
	case 19:
		return KeyR, 'r'
	case 20:
		return KeyT, 't'
	case 21:
		return KeyY, 'y'
	case 22:
		return KeyU, 'u'
	case 23:
		return KeyI, 'i'
	case 24:
		return KeyO, 'o'
	case 25:
		return KeyP, 'p'
	case 26:
		return KeyBracketLeft, '['
	case 27:
		return KeyBracketRight, ']'
	case 28:
		return KeyEnter, '\r'
	case 29:
		return KeyControlLeft, 0
	case 30:
		return KeyA, 'a'
	case 31:
		return KeyS, 's'
	case 32:
		return KeyD, 'd'
	case 33:
		return KeyF, 'f'
	case 34:
		return KeyG, 'g'
	case 35:
		return KeyH, 'h'
	case 36:
		return KeyJ, 'j'
	case 37:
		return KeyK, 'k'
	case 38:
		return KeyL, 'l'
	case 39:
		return KeySemicolon, ';'
	case 40:
		return KeyApostrophe, '\''
	case 41:
		return KeyGrave, '`'
	case 42:
		return KeyShiftLeft, 0
	case 43:
		return KeyBackslash, '\\'
	case 44:
		return KeyZ, 'z'
	case 45:
		return KeyX, 'x'
	case 46:
		return KeyC, 'c'
	case 47:
		return KeyV, 'v'
	case 48:
		return KeyB, 'b'
	case 49:
		return KeyN, 'n'
	case 50:
		return KeyM, 'm'
	case 51:
		return KeyComma, ','
	case 52:
		return KeyPeriod, '.'
	case 53:
		return KeySlash, '/'
	case 54:
		return KeyShiftRight, 0
	case 55:
		return KeyKpMultiply, '*'
	case 56:
		return KeyAltLeft, 0
	case 57:
		return KeySpace, ' '
	case 58:
		return KeyCapsLock, 0
	case 59:
		return KeyF1, 0
	case 60:
		return KeyF2, 0
	case 61:
		return KeyF3, 0
	case 62:
		return KeyF4, 0
	case 63:
		return KeyF5, 0
	case 64:
		return KeyF6, 0
	case 65:
		return KeyF7, 0
	case 66:
		return KeyF8, 0
	case 67:
		return KeyF9, 0
	case 68:
		return KeyF10, 0
	case 71:
		return KeyKp7, '7'
	case 72:
		return KeyKp8, '8'
	case 73:
		return KeyKp9, '9'
	case 74:
		return KeyKpSubtract, '-'
	case 75:
		return KeyKp4, '4'
	case 76:
		return KeyKp5, '5'
	case 77:
		return KeyKp6, '6'
	case 78:
		return KeyKpAdd, '+'
	case 79:
		return KeyKp1, '1'
	case 80:
		return KeyKp2, '2'
	case 81:
		return KeyKp3, '3'
	case 82:
		return KeyKp0, '0'
	case 83:
		return KeyKpDecimal, '.'
	case 87:
		return KeyF11, 0
	case 88:
		return KeyF12, 0
	case 96:
		return KeyKpEnter, '\r'
	case 98:
		return KeyKpDivide, '/'
	case 97:
		return KeyControlRight, 0
	case 100:
		return KeyAltRight, 0
	case 102:
		return KeyHome, 0
	case 103:
		return KeyUp, 0
	case 104:
		return KeyPageUp, 0
	case 105:
		return KeyLeft, 0
	case 106:
		return KeyRight, 0
	case 107:
		return KeyEnd, 0
	case 108:
		return KeyDown, 0
	case 109:
		return KeyPageDown, 0
	case 110:
		return KeyInsert, 0
	case 111:
		return KeyDelete, 0
	case 125:
		return KeySuperLeft, 0
	case 126:
		return KeySuperRight, 0
	}
	return KeyUnknown, 0
}
