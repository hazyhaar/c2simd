package c2tui

import (
	"strings"
	"testing"
	"time"
)

func TestSession_EchoLifecycle(t *testing.T) {
	sess := NewSession(80, 24, 1000)
	defer sess.Close()

	if err := sess.Start("echo", "C2VTE_SESSION_ACTIVE"); err != nil {
		t.Fatalf("Start a échoué: %v", err)
	}

	_ = sess.Wait()

	var totalDirty int
	var renderedText strings.Builder

	// Boucle d'ingestion et de drainage des trames
	for i := 0; i < 20; i++ {
		dirty, spans, _ := sess.Step(10 * time.Millisecond)
		if dirty > 0 {
			totalDirty += dirty
			rendered := sess.RenderANSI(spans)
			renderedText.Write(rendered)
		}
		time.Sleep(5 * time.Millisecond)
	}

	out := renderedText.String()
	if !strings.Contains(out, "C2VTE_SESSION_ACTIVE") {
		t.Fatalf("Sortie de session incomplète, got:\n%q", out)
	}
	if totalDirty == 0 {
		t.Fatalf("Aucune cellule modifiée signalée par DiffGrid")
	}
}

func TestSession_InteractiveShellAndResize(t *testing.T) {
	sess := NewSession(80, 24, 1000)
	defer sess.Close()

	if err := sess.Start("/bin/sh"); err != nil {
		t.Fatalf("Start(/bin/sh) a échoué: %v", err)
	}

	// Envoyer une commande avec retour à la ligne
	cmd := "echo RESULT_12345\nexit\n"
	if _, err := sess.WriteInput([]byte(cmd)); err != nil {
		t.Fatalf("WriteInput a échoué: %v", err)
	}

	// Test de redimensionnement en cours de session
	if err := sess.Resize(120, 40); err != nil {
		t.Fatalf("Resize(120, 40) a échoué: %v", err)
	}

	var renderedText strings.Builder
	for i := 0; i < 30; i++ {
		dirty, spans, err := sess.Step(10 * time.Millisecond)
		if dirty > 0 {
			rendered := sess.RenderANSI(spans)
			renderedText.Write(rendered)
		}
		if err != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	_ = sess.Wait()

	out := renderedText.String()
	if !strings.Contains(out, "RESULT_12345") {
		t.Fatalf("Sortie shell interactif manquante dans: %q", out)
	}
}
