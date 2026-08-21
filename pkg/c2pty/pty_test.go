package c2pty

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestOpenPTY(t *testing.T) {
	ws := &Winsize{
		Rows: 30,
		Cols: 100,
	}
	master, slave, err := OpenPTY(ws)
	if err != nil {
		t.Fatalf("OpenPTY a échoué: %v", err)
	}
	defer master.Close()
	defer slave.Close()

	ptsName, err := Ptsname(master)
	if err != nil {
		t.Fatalf("Ptsname a échoué: %v", err)
	}
	if !strings.HasPrefix(ptsName, "/dev/pts/") {
		t.Errorf("ptsName inattendu: %s", ptsName)
	}

	gotWs, err := GetWinsize(master)
	if err != nil {
		t.Fatalf("GetWinsize a échoué: %v", err)
	}
	if gotWs.Rows != 30 || gotWs.Cols != 100 {
		t.Errorf("Winsize incorrect: attendu 30x100, obtenu %dx%d", gotWs.Rows, gotWs.Cols)
	}

	newWs := &Winsize{Rows: 40, Cols: 120}
	if err := SetWinsize(master, newWs); err != nil {
		t.Fatalf("SetWinsize a échoué: %v", err)
	}

	gotWs2, err := GetWinsize(master)
	if err != nil {
		t.Fatalf("GetWinsize après modification a échoué: %v", err)
	}
	if gotWs2.Rows != 40 || gotWs2.Cols != 120 {
		t.Errorf("Winsize après modification incorrect: attendu 40x120, obtenu %dx%d", gotWs2.Rows, gotWs2.Cols)
	}
}

func TestPTYEcho(t *testing.T) {
	pty, err := StartCommand("echo", "C2PTY_ECHO_SUCCESS")
	if err != nil {
		t.Fatalf("StartCommand a échoué: %v", err)
	}
	defer pty.Close()

	var buf bytes.Buffer
	readBuf := make([]byte, 512)
	for {
		n, rerr := pty.Read(readBuf)
		if n > 0 {
			buf.Write(readBuf[:n])
		}
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				break
			}
			t.Fatalf("Erreur inattendue lors de la lecture: %v", rerr)
		}
	}

	out := buf.String()
	if !strings.Contains(out, "C2PTY_ECHO_SUCCESS") {
		t.Fatalf("Sortie echo manquante dans: %q", out)
	}

	if err := pty.Wait(); err != nil {
		t.Fatalf("pty.Wait a échoué: %v", err)
	}
}

func TestPTYInteractiveShell(t *testing.T) {
	pty, err := Open(&Winsize{Rows: 24, Cols: 80})
	if err != nil {
		t.Fatalf("Open a échoué: %v", err)
	}
	defer pty.Close()

	if err := pty.Start("/bin/sh"); err != nil {
		t.Fatalf("Start(/bin/sh) a échoué: %v", err)
	}

	// Écriture d'une commande dans le shell interactif
	cmdInput := "echo C2PTY_PROMPT_VAL_1337\n"
	if _, err := pty.Write([]byte(cmdInput)); err != nil {
		t.Fatalf("pty.Write a échoué: %v", err)
	}

	var output bytes.Buffer
	readBuf := make([]byte, 512)
	target := "C2PTY_PROMPT_VAL_1337"
	found := false

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		n, err := pty.Read(readBuf)
		if n > 0 {
			output.Write(readBuf[:n])
			if strings.Contains(output.String(), target) {
				found = true
				break
			}
		}
		if err != nil && !errors.Is(err, io.EOF) {
			t.Fatalf("Erreur lecture: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !found {
		t.Fatalf("Chaîne cible %q non trouvée dans la sortie shell: %q", target, output.String())
	}

	// Fermeture propre via exit 0
	if _, err := pty.Write([]byte("exit 0\n")); err != nil {
		t.Fatalf("pty.Write(exit) a échoué: %v", err)
	}

	if err := pty.Wait(); err != nil {
		t.Fatalf("pty.Wait après exit 0 a échoué: %v", err)
	}
}

func TestPTYResizeAndSigwinch(t *testing.T) {
	script := `
trap "echo WINCH_NOTIFICATION_RECEIVED" WINCH
echo INITIAL_READY_FOR_RESIZE
while true; do
    sleep 0.05
done
`
	pty, err := StartWithWinsize(&Winsize{Rows: 24, Cols: 80}, "/bin/sh", "-c", script)
	if err != nil {
		t.Fatalf("StartWithWinsize a échoué: %v", err)
	}
	defer pty.Close()

	var output bytes.Buffer
	readBuf := make([]byte, 512)

	// Attente de l'état INITIAL_READY
	ready := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		n, err := pty.Read(readBuf)
		if n > 0 {
			output.Write(readBuf[:n])
			if strings.Contains(output.String(), "INITIAL_READY_FOR_RESIZE") {
				ready = true
				break
			}
		}
		if err != nil && !errors.Is(err, io.EOF) {
			t.Fatalf("Erreur lecture initiale: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !ready {
		t.Fatalf("Le sous-processus n'a pas atteint l'état INITIAL_READY: %s", output.String())
	}

	// Déclenchement du redimensionnement
	if err := pty.Resize(132, 43); err != nil {
		t.Fatalf("pty.Resize a échoué: %v", err)
	}

	ws, err := pty.GetWinsize()
	if err != nil {
		t.Fatalf("pty.GetWinsize a échoué: %v", err)
	}
	if ws.Cols != 132 || ws.Rows != 43 {
		t.Fatalf("Géométrie non synchronisée: attendu 132x43, obtenu %dx%d", ws.Cols, ws.Rows)
	}

	// Vérification de la réception du signal SIGWINCH par le shell
	winchCaught := false
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		n, err := pty.Read(readBuf)
		if n > 0 {
			output.Write(readBuf[:n])
			if strings.Contains(output.String(), "WINCH_NOTIFICATION_RECEIVED") {
				winchCaught = true
				break
			}
		}
		if err != nil && !errors.Is(err, io.EOF) {
			t.Fatalf("Erreur lecture après resize: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !winchCaught {
		t.Fatalf("Le signal SIGWINCH n'a pas été capté par le sous-shell: %s", output.String())
	}

	// Arrêt propre
	if err := pty.Kill(); err != nil {
		t.Fatalf("pty.Kill a échoué: %v", err)
	}
}

func TestPTYClose(t *testing.T) {
	pty, err := StartCommand("sleep", "30")
	if err != nil {
		t.Fatalf("StartCommand a échoué: %v", err)
	}

	if err := pty.Close(); err != nil {
		t.Fatalf("Premier Close() a échoué: %v", err)
	}

	// Idempotence du second Close()
	if err := pty.Close(); err != nil {
		t.Fatalf("Second Close() doit être idempotent et sans erreur: %v", err)
	}

	buf := make([]byte, 16)
	if _, err := pty.Read(buf); !errors.Is(err, ErrClosed) {
		t.Errorf("Read sur PTY fermé doit renvoyer ErrClosed, obtenu: %v", err)
	}

	if _, err := pty.Write([]byte("test")); !errors.Is(err, ErrClosed) {
		t.Errorf("Write sur PTY fermé doit renvoyer ErrClosed, obtenu: %v", err)
	}
}

func TestForkPTY(t *testing.T) {
	cmd := exec.Command("echo", "FORKPTY_DIRECT_TEST")
	master, err := ForkPTY(&Winsize{Rows: 25, Cols: 80}, cmd)
	if err != nil {
		t.Fatalf("ForkPTY a échoué: %v", err)
	}
	defer master.Close()

	out, err := io.ReadAll(master)
	if err != nil && !errors.Is(err, io.EOF) {
		// Sous Linux io.ReadAll peut recevoir EIO sur master nu sans wrapper pty.Read
		if !strings.Contains(err.Error(), "input/output error") {
			t.Fatalf("io.ReadAll sur master ForkPTY a échoué: %v", err)
		}
	}

	if !strings.Contains(string(out), "FORKPTY_DIRECT_TEST") {
		t.Fatalf("Sortie attendue manquante: %s", string(out))
	}

	if err := cmd.Wait(); err != nil {
		t.Fatalf("cmd.Wait a échoué: %v", err)
	}
}

func TestRingBuffer_BasicOperations(t *testing.T) {
	rb := NewRingBuffer(16)
	if rb.Cap() != 16 {
		t.Fatalf("Capacité attendue 16, obtenu: %d", rb.Cap())
	}
	if !rb.IsEmpty() {
		t.Fatalf("Le tampon neuf devrait être vide")
	}
	if rb.IsFull() {
		t.Fatalf("Le tampon neuf ne devrait pas être plein")
	}
	if rb.Free() != 16 {
		t.Fatalf("Espace libre attendu 16, obtenu: %d", rb.Free())
	}

	// Écriture partielle
	n, err := rb.Write([]byte("hello"))
	if err != nil || n != 5 {
		t.Fatalf("Écriture 'hello' a échoué: n=%d, err=%v", n, err)
	}
	if rb.Len() != 5 {
		t.Fatalf("Longueur attendue 5, obtenu: %d", rb.Len())
	}
	if rb.Free() != 11 {
		t.Fatalf("Espace libre attendu 11, obtenu: %d", rb.Free())
	}

	// Lecture partielle
	readBuf := make([]byte, 3)
	n, err = rb.Read(readBuf)
	if err != nil || n != 3 || string(readBuf) != "hel" {
		t.Fatalf("Lecture attendue 'hel', obtenu: %q, n=%d, err=%v", string(readBuf[:n]), n, err)
	}
	if rb.Len() != 2 {
		t.Fatalf("Longueur attendue 2, obtenu: %d", rb.Len())
	}

	// Vidage du reste
	readBuf = make([]byte, 10)
	n, err = rb.Read(readBuf)
	if err != nil || n != 2 || string(readBuf[:n]) != "lo" {
		t.Fatalf("Lecture attendue 'lo', obtenu: %q, n=%d, err=%v", string(readBuf[:n]), n, err)
	}
	if !rb.IsEmpty() {
		t.Fatalf("Le tampon devrait être vide après vidage")
	}

	// Lecture sur tampon vide
	n, err = rb.Read(readBuf)
	if !errors.Is(err, ErrBufferEmpty) || n != 0 {
		t.Fatalf("Lecture sur tampon vide doit renvoyer ErrBufferEmpty, obtenu: n=%d, err=%v", n, err)
	}
}

func TestRingBuffer_WrapAround(t *testing.T) {
	rb := NewRingBuffer(8)

	for i := 0; i < 20; i++ {
		payload := []byte(fmt.Sprintf("%04d", i))
		n, err := rb.Write(payload)
		if err != nil || n != 4 {
			t.Fatalf("Itération %d: écriture échouée: n=%d, err=%v", i, n, err)
		}

		out := make([]byte, 4)
		n, err = rb.Read(out)
		if err != nil || n != 4 || !bytes.Equal(out, payload) {
			t.Fatalf("Itération %d: lecture attendue %s, obtenu %s", i, string(payload), string(out))
		}
	}
}

func TestRingBuffer_WriteOverflow(t *testing.T) {
	rb := NewRingBuffer(8)

	n, err := rb.Write([]byte("1234567890"))
	if !errors.Is(err, ErrBufferFull) {
		t.Fatalf("Attendu ErrBufferFull, obtenu: %v", err)
	}
	if n != 8 {
		t.Fatalf("Nombre d'octets écrits attendu 8, obtenu: %d", n)
	}
	if !rb.IsFull() {
		t.Fatalf("Le tampon devrait être plein")
	}

	out := rb.Bytes()
	if string(out) != "12345678" {
		t.Fatalf("Contenu attendu '12345678', obtenu: %q", string(out))
	}
}

func TestRingBuffer_WriteOverwrite(t *testing.T) {
	rb := NewRingBuffer(6)

	rb.WriteOverwrite([]byte("ABCDEF"))
	if rb.String() != "ABCDEF" {
		t.Fatalf("Attendu 'ABCDEF', obtenu: %q", rb.String())
	}

	// Écrasement partiel
	rb.WriteOverwrite([]byte("12"))
	if rb.String() != "CDEF12" {
		t.Fatalf("Attendu 'CDEF12', obtenu: %q", rb.String())
	}

	// Écrasement total supérieur à la capacité
	rb.WriteOverwrite([]byte("XYZ_0123456789"))
	if rb.String() != "456789" {
		t.Fatalf("Attendu '456789' (derniers 6 octets), obtenu: %q", rb.String())
	}
}

func TestRingBuffer_PeekAndDiscard(t *testing.T) {
	rb := NewRingBuffer(16)
	rb.Write([]byte("HELLO_WORLD"))

	peeked := rb.Peek(5)
	if string(peeked) != "HELLO" {
		t.Fatalf("Peek attendu 'HELLO', obtenu: %q", string(peeked))
	}
	if rb.Len() != 11 {
		t.Fatalf("Peek ne doit pas modifier la longueur, attendu 11, obtenu: %d", rb.Len())
	}

	discarded := rb.Discard(6)
	if discarded != 6 {
		t.Fatalf("Discard attendu 6, obtenu: %d", discarded)
	}
	if rb.String() != "WORLD" {
		t.Fatalf("Contenu restant attendu 'WORLD', obtenu: %q", rb.String())
	}

	rb.Reset()
	if !rb.IsEmpty() || rb.Len() != 0 {
		t.Fatalf("Reset doit vider le tampon")
	}
}

func TestRingBuffer_CloseAndDrain(t *testing.T) {
	rb := NewRingBuffer(16)
	rb.Write([]byte("DATA_LEFT"))

	if err := rb.Close(); err != nil {
		t.Fatalf("rb.Close a échoué: %v", err)
	}

	// Écriture impossible après fermeture
	_, err := rb.Write([]byte("NEW"))
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("Write après fermeture doit renvoyer io.ErrClosedPipe, obtenu: %v", err)
	}

	// Vidage des données résiduelles
	buf := make([]byte, 16)
	n, err := rb.Read(buf)
	if err != nil || string(buf[:n]) != "DATA_LEFT" {
		t.Fatalf("Lecture résiduelle attendue 'DATA_LEFT', obtenu: %q, err=%v", string(buf[:n]), err)
	}

	// Tampon vide et fermé -> io.EOF
	n, err = rb.Read(buf)
	if !errors.Is(err, io.EOF) || n != 0 {
		t.Fatalf("Lecture sur tampon vidé et fermé doit renvoyer io.EOF, obtenu: n=%d, err=%v", n, err)
	}
}

func TestRingBuffer_Concurrent(t *testing.T) {
	rb := NewRingBuffer(128)
	var wg sync.WaitGroup

	numProducers := 4
	itemsPerProducer := 250

	for p := 0; p < numProducers; p++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < itemsPerProducer; i++ {
				data := []byte{byte(id), byte(i % 256)}
				for {
					_, err := rb.Write(data)
					if err == nil {
						break
					}
					time.Sleep(50 * time.Microsecond)
				}
			}
		}(p)
	}

	var consumedCount int
	var mu sync.Mutex
	done := make(chan struct{})

	go func() {
		readBuf := make([]byte, 32)
		for {
			select {
			case <-done:
				return
			default:
				n, err := rb.Read(readBuf)
				if n > 0 {
					mu.Lock()
					consumedCount += n
					mu.Unlock()
				}
				if err != nil {
					time.Sleep(50 * time.Microsecond)
				}
			}
		}
	}()

	wg.Wait()

	// Drainer ce qui reste
	time.Sleep(50 * time.Millisecond)
	rb.Close()

	readBuf := make([]byte, 64)
	for {
		n, err := rb.Read(readBuf)
		if n > 0 {
			mu.Lock()
			consumedCount += n
			mu.Unlock()
		}
		if errors.Is(err, io.EOF) {
			break
		}
	}
	close(done)

	expected := numProducers * itemsPerProducer * 2
	if consumedCount != expected {
		t.Fatalf("Total consommé incorrect: attendu %d, obtenu %d", expected, consumedCount)
	}
}
