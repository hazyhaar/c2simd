package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type BenchRecord struct {
	Timestamp string            `json:"timestamp"`
	Metrics   map[string]Metric `json:"metrics"`
}

type Metric struct {
	Name        string  `json:"name"`
	NsPerOp     float64 `json:"ns_per_op"`
	MBps        float64 `json:"mb_ps"`
	BytesPerOp  uint64  `json:"bytes_per_op"`
	AllocsPerOp uint64  `json:"allocs_per_op"`
}

func main() {
	fmt.Println("=== c2simd Automated Benchmark & Regression Probe (avec Porte Témoin KAT) ===")

	// 1. Validation de la Porte de Témoins KAT (Zero Encryption Foireuse Silencieuse)
	fmt.Println("--> Étape 1/2 : Validation mécanique des témoins KAT (Différentiel & Oracles RFC 8439)...")
	katCmd := exec.Command("go1.27rc1", "test", "-v", "./kat", ".")
	katCmd.Env = append(os.Environ(), "GOEXPERIMENT=simd")
	katCmd.Dir = "/devhoros/c2simd"

	var katOut bytes.Buffer
	katCmd.Stdout = &katOut
	katCmd.Stderr = &katOut

	if err := katCmd.Run(); err != nil {
		fmt.Printf("\n[ÉCHEC CRITIQUE PORTE KAT TÉMOIN] Un ou plusieurs moteurs ont produit un résultat cryptographique foireux !\n")
		fmt.Printf("Sortie du test KAT :\n%s\n", katOut.String())
		fmt.Printf("ABORT : Refus d'exécuter les benchmarks ou d'enregistrer des métriques sur du code divergent.\n")
		os.Exit(1)
	}

	fmt.Println("  ✓ 100 % des témoins KAT sont CONFORMES (Zero Divergence Oracles).")

	// 2. Exécution du benchmark sous sondes
	fmt.Println("--> Étape 2/2 : Exécution des benchmarks de débit et profilage CPU...")
	cmd := exec.Command("go1.27rc1", "test", "-count=1", "-bench=.", "-benchmem", "-cpuprofile=cpu_step.pprof", ".")
	cmd.Env = append(os.Environ(), "GOEXPERIMENT=simd")
	cmd.Dir = "/devhoros/c2simd"

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		fmt.Printf("Erreur lors de l'exécution du benchmark probe : %v\nSortie :\n%s\n", err, out.String())
		os.Exit(1)
	}

	output := out.String()
	fmt.Println("Benchmark exécuté avec succès sous sondes CPU.")

	currentRecord := BenchRecord{
		Timestamp: time.Now().Format(time.RFC3339),
		Metrics:   make(map[string]Metric),
	}

	reName := regexp.MustCompile(`^(Benchmark\w+-\d+)\s+`)
	reNs := regexp.MustCompile(`(\d+(?:\.\d+)?)\s+ns/op`)
	reMBps := regexp.MustCompile(`(\d+(?:\.\d+)?)\s+MB/s`)
	reBytes := regexp.MustCompile(`(\d+)\s+B/op`)
	reAllocs := regexp.MustCompile(`(\d+)\s+allocs/op`)

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if matchName := reName.FindStringSubmatch(line); len(matchName) > 1 {
			name := matchName[1]
			nsMatch := reNs.FindStringSubmatch(line)
			mbMatch := reMBps.FindStringSubmatch(line)
			bMatch := reBytes.FindStringSubmatch(line)
			aMatch := reAllocs.FindStringSubmatch(line)

			if len(nsMatch) > 1 && len(mbMatch) > 1 {
				nsPerOp, _ := strconv.ParseFloat(nsMatch[1], 64)
				mbPs, _ := strconv.ParseFloat(mbMatch[1], 64)
				bytesPerOp, _ := strconv.ParseUint(bMatch[1], 10, 64)
				allocsPerOp, _ := strconv.ParseUint(aMatch[1], 10, 64)

				currentRecord.Metrics[name] = Metric{
					Name:        name,
					NsPerOp:     nsPerOp,
					MBps:        mbPs,
					BytesPerOp:  bytesPerOp,
					AllocsPerOp: allocsPerOp,
				}

				fmt.Printf("  • %-45s | %8.2f MB/s | %9.0f ns/op | %d allocs/op\n", name, mbPs, nsPerOp, allocsPerOp)
			}
		}
	}

	// 3. Sauvegarde de l'historique dans bench_history.json
	historyPath := "/devhoros/c2simd/bench_history.json"
	var history []BenchRecord
	if data, err := os.ReadFile(historyPath); err == nil {
		json.Unmarshal(data, &history)
	}

	history = append(history, currentRecord)
	histData, _ := json.MarshalIndent(history, "", "  ")
	os.WriteFile(historyPath, histData, 0644)
	fmt.Printf("\nHistorique mis à jour dans %s (Total d'enregistrements : %d)\n", historyPath, len(history))

	// 4. Contrôle automatique de régression (> 5%)
	if len(history) > 1 {
		prev := history[len(history)-2]
		fmt.Println("\n--- Contrôle Automatique de Régression vs Run Précédent ---")
		regressions := 0
		for name, currM := range currentRecord.Metrics {
			if prevM, exists := prev.Metrics[name]; exists {
				if prevM.MBps > 0 {
					deltaMBps := ((currM.MBps - prevM.MBps) / prevM.MBps) * 100
					if deltaMBps < -5.0 {
						fmt.Printf("  [REGRESSION ALARM] %s a chuté de %.2f%% (Préc: %.2f MB/s -> Actuel: %.2f MB/s)\n", name, -deltaMBps, prevM.MBps, currM.MBps)
						regressions++
					} else if deltaMBps > 5.0 {
						fmt.Printf("  [SPEEDUP] %s a progressé de +%.2f%% (Préc: %.2f MB/s -> Actuel: %.2f MB/s)\n", name, deltaMBps, prevM.MBps, currM.MBps)
					} else {
						fmt.Printf("  [STABLE] %s (%.2f MB/s)\n", name, currM.MBps)
					}
				}
			}
		}

		if regressions > 0 {
			fmt.Printf("\nATTENTION : %d métrique(s) présente(nt) une régression de débit supérieure au seuil de 5 %% !\n", regressions)
		} else {
			fmt.Println("\nSUCCÈS : Aucune régression de débit n'a été détectée.")
		}
	}
}
