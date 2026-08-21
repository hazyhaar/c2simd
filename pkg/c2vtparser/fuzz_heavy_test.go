package c2vtparser

import (
	"math/rand"
	"testing"
)

// TestFuzzMassiveCorruptedStream soumet le parseur à 100 000 blocs de flux ANSI hautement corrompus.
func TestFuzzMassiveCorruptedStream(t *testing.T) {
	g := &CursorGrid{}
	g.Reset(80, 24)
	p := &Parser{}
	p.Reset(g)

	rng := rand.New(rand.NewSource(42))

	primitives := [][]byte{
		[]byte("\x1b[H"),
		[]byte("\x1b[2J"),
		[]byte("\x1b[38;2;255;128;64m"),
		[]byte("\x1b[48;5;196m"),
		[]byte("\x1b[1;4;7m"),
		[]byte("\x1b[0m"),
		[]byte("\x1b]52;c;SGVsbG8=\x07"),
		[]byte("\x1b]8;id=x;https://example.com\x1b\\"),
		[]byte("\x1b(0lqqk\x1b(B"),
		[]byte("\x1b[?2026h"),
		[]byte("\x1b[?2026l"),
		[]byte("\x1b[?1000;1006h"),
		[]byte("\x1b[2;20r"),
		[]byte("\r\n\t\x00\x08\x0b\x0c"),
		[]byte("Texte en UTF-8 : accentué éàç, emoji 🚀🌟, kanji 日本語"),
		[]byte{0xFF, 0xFE, 0xC0, 0x80, 0x1B, 0x5B, 0x3F}, // Octets malformés
	}

	buf := make([]byte, 1024)
	iterations := 100000

	for i := 0; i < iterations; i++ {
		// Générer un bloc hybride
		bufLen := 0
		numFragments := rng.Intn(10) + 1
		for f := 0; f < numFragments; f++ {
			prim := primitives[rng.Intn(len(primitives))]
			// Mutation aléatoire d'un octet
			if rng.Intn(4) == 0 && len(prim) > 0 {
				mutated := append([]byte(nil), prim...)
				mutIdx := rng.Intn(len(mutated))
				mutated[mutIdx] = byte(rng.Intn(256))
				if bufLen+len(mutated) < len(buf) {
					copy(buf[bufLen:], mutated)
					bufLen += len(mutated)
				}
			} else {
				if bufLen+len(prim) < len(buf) {
					copy(buf[bufLen:], prim)
					bufLen += len(prim)
				}
			}
		}

		if bufLen > 0 {
			// Ingestion sans panic
			p.Feed(buf[:bufLen])
		}

		// Réinitialisation périodique de grille pour varier les dimensions
		if i%10000 == 0 {
			w := rng.Intn(150) + 10
			h := rng.Intn(60) + 5
			g.Reset(w, h)
			p.Reset(g)
		}
	}
}

// TestFuzzHighCadence_RapidModeSwitching teste 50 000 itérations de bascules de modes en rafale.
func TestFuzzHighCadence_RapidModeSwitching(t *testing.T) {
	g := &CursorGrid{}
	g.Reset(80, 24)
	p := &Parser{}
	p.Reset(g)

	rng := rand.New(rand.NewSource(1337))
	modeSequences := [][]byte{
		[]byte("\x1b[?2026h"),
		[]byte("\x1b[?2026l"),
		[]byte("\x1b[?1000h"),
		[]byte("\x1b[?1000l"),
		[]byte("\x1b[?1002h"),
		[]byte("\x1b[?1003h"),
		[]byte("\x1b[?1006h"),
		[]byte("\x1b[?1049h"),
		[]byte("\x1b[?1049l"),
		[]byte("\x1b[?25h"),
		[]byte("\x1b[?25l"),
		[]byte("\x1b[?2004h"),
		[]byte("\x1b[?2004l"),
		[]byte("\x1b[?3h"),
		[]byte("\x1b[?5h"),
		[]byte("\x1b[?7h"),
		[]byte("\x1b[1;24r"),
		[]byte("\x1b[5;15r"),
	}

	for i := 0; i < 50000; i++ {
		seq := modeSequences[rng.Intn(len(modeSequences))]
		p.Feed(seq)
		if rng.Intn(5) == 0 {
			p.Feed([]byte("RND_STR"))
		}
	}
}
