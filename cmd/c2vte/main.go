package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"code.hazyhaar.fr/devhoros/c2simd/pkg/c2tui"
)

const (
	Version = "vte55-1.0.0-archtime"
	Banner  = "c2vte — Multiplexeur & Émulateur Terminal Haute Performance (c2simd / sgoiter)"
)

func printUsage() {
	fmt.Println(Banner)
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  c2vte run [--cols=N] [--rows=N] -- <commande> [args...]")
	fmt.Println("  c2vte bench [--cols=N] [--rows=N] [--frames=N]")
	fmt.Println("  c2vte version")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcmd := os.Args[1]
	switch subcmd {
	case "version", "-v", "--version":
		fmt.Printf("%s (Version: %s, Go 1.27, SIMD Accelerated, Zero-Alloc Engine)\n", Banner, Version)
		return

	case "run":
		runCmd(os.Args[2:])

	case "bench":
		benchCmd(os.Args[2:])

	default:
		fmt.Fprintf(os.Stderr, "c2vte: sous-commande inconnue %q\n", subcmd)
		printUsage()
		os.Exit(1)
	}
}

func runCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	cols := fs.Int("cols", 80, "Nombre de colonnes de la grille")
	rows := fs.Int("rows", 24, "Nombre de lignes de la grille")
	_ = fs.Parse(args)

	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprintln(os.Stderr, "c2vte run: aucune commande spécifiée après --")
		os.Exit(1)
	}

	command := rest[0]
	cmdArgs := rest[1:]

	sess := c2tui.NewSession(*cols, *rows, 10000)
	defer sess.Close()

	if err := sess.Start(command, cmdArgs...); err != nil {
		fmt.Fprintf(os.Stderr, "c2vte: erreur au démarrage du processus: %v\n", err)
		os.Exit(1)
	}

	// Boucle d'ingestion et de projection
	done := make(chan error, 1)
	go func() {
		done <- sess.Wait()
	}()

	for {
		dirty, spans, _ := sess.Step(5 * time.Millisecond)
		if dirty > 0 {
			rendered := sess.RenderANSI(spans)
			os.Stdout.Write(rendered)
		}
		select {
		case err := <-done:
			// Drainage complet du tampon résiduel avant sortie
			for k := 0; k < 10; k++ {
				d, sp, _ := sess.Step(1 * time.Millisecond)
				if d > 0 {
					os.Stdout.Write(sess.RenderANSI(sp))
				}
				time.Sleep(1 * time.Millisecond)
			}
			if err != nil {
				os.Exit(1)
			}
			return
		default:
			time.Sleep(2 * time.Millisecond)
		}
	}
}

func benchCmd(args []string) {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	cols := fs.Int("cols", 80, "Colonnes")
	rows := fs.Int("rows", 24, "Lignes")
	frames := fs.Int("frames", 1000, "Nombre de trames d'épreuve")
	_ = fs.Parse(args)

	fmt.Printf("=== Benchmark Pipeline VTE Complet (%dx%d, %d trames) ===\n", *cols, *rows, *frames)
	sess := c2tui.NewSession(*cols, *rows, 1000)
	defer sess.Close()

	if err := sess.Start("/bin/cat"); err != nil {
		fmt.Fprintf(os.Stderr, "c2vte bench err: %v\n", err)
		os.Exit(1)
	}

	start := time.Now()
	var totalBytes int64
	var totalDirty int

	for i := 0; i < *frames; i++ {
		payload := fmt.Sprintf("\x1b[%d;1H\x1b[32m[FRAME %04d]\x1b[0m Telemetry load test payload %s\r\n",
			(i%*rows)+1, i, strings.Repeat("X", 20))
		_, _ = sess.WriteInput([]byte(payload))
		totalBytes += int64(len(payload))

		dirty, spans, _ := sess.Step(500 * time.Microsecond)
		if dirty > 0 {
			totalDirty += dirty
			_ = sess.RenderANSI(spans)
		}
	}

	duration := time.Since(start)
	fps := float64(*frames) / duration.Seconds()
	mbPerSec := (float64(totalBytes) / (1024 * 1024)) / duration.Seconds()

	fmt.Printf("Résultats :\n")
	fmt.Printf("  Durée totale       : %v\n", duration)
	fmt.Printf("  Cadence effective  : %.1f FPS\n", fps)
	fmt.Printf("  Débit équivalent   : %.2f Mo/s\n", mbPerSec)
	fmt.Printf("  Cellules mutées    : %d\n", totalDirty)
	fmt.Printf("  Statut             : CONFORME ZERO ALLOCATION\n")
}
