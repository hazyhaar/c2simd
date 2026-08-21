package c2myers

import (
	"fmt"
	"strings"
)

// ApplyPatchText applique un ensemble de Hunks sur le texte d'origine.
// Si checkOnly est vrai, vérifie la faisabilité sans retourner le texte modifié.
func ApplyPatchText(originalText string, hunks []Hunk, checkOnly bool) (string, error) {
	origLines := splitLines(originalText)
	var resultLines []string

	origIdx := 0 // 0-indexed line in origLines

	for _, hunk := range hunks {
		targetOldIdx := hunk.OldStart - 1
		if targetOldIdx < 0 {
			targetOldIdx = 0
		}

		// Copier les lignes inchangées précédant le hunk
		for origIdx < targetOldIdx && origIdx < len(origLines) {
			if !checkOnly {
				resultLines = append(resultLines, origLines[origIdx])
			}
			origIdx++
		}

		// Traiter les lignes du hunk
		for _, hLine := range hunk.Lines {
			if hLine == "" {
				continue
			}
			marker := hLine[0]
			content := hLine[1:]

			switch marker {
			case ' ': // Contexte inchangé
				if origIdx >= len(origLines) {
					return "", fmt.Errorf("conflit de contexte à la ligne %d : fin de fichier inattendue (attendu %q)", origIdx+1, content)
				}
				if origLines[origIdx] != content {
					return "", fmt.Errorf("conflit de contexte à la ligne %d : attendu %q, trouvé %q", origIdx+1, content, origLines[origIdx])
				}
				if !checkOnly {
					resultLines = append(resultLines, content)
				}
				origIdx++

			case '-': // Ligne supprimée
				if origIdx >= len(origLines) {
					return "", fmt.Errorf("conflit de suppression à la ligne %d : fin de fichier inattendue (tentative de suppression de %q)", origIdx+1, content)
				}
				if origLines[origIdx] != content {
					return "", fmt.Errorf("conflit de suppression à la ligne %d : attendu %q, trouvé %q", origIdx+1, content, origLines[origIdx])
				}
				origIdx++

			case '+': // Ligne insérée
				if !checkOnly {
					resultLines = append(resultLines, content)
				}
			}
		}
	}

	// Copier les lignes restantes après le dernier hunk
	for origIdx < len(origLines) {
		if !checkOnly {
			resultLines = append(resultLines, origLines[origIdx])
		}
		origIdx++
	}

	if checkOnly {
		return "", nil
	}

	return strings.Join(resultLines, "\n"), nil
}
