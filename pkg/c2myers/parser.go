package c2myers

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// ParsePatchSet analyse et découpe un patch unifié en mémoire avec validation de sécurité.
func ParsePatchSet(patchContent string) (*PatchSet, error) {
	scanner := bufio.NewScanner(strings.NewReader(patchContent))
	// Buffer étendu à 16 Mo pour les patchs contenant de très longues lignes (minifiées / JSON)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 16*1024*1024)

	ps := &PatchSet{}

	var curFile *FilePatch
	var curHunk *Hunk

	for scanner.Scan() {
		line := scanner.Text()

		// 1. Détection début de fichier diff
		if strings.HasPrefix(line, "diff --git ") {
			if curHunk != nil && curFile != nil {
				curFile.Hunks = append(curFile.Hunks, *curHunk)
				curHunk = nil
			}
			if curFile != nil {
				ps.Files = append(ps.Files, *curFile)
			}
			curFile = &FilePatch{}
			continue
		}

		// 2. Détection de mode et types spéciaux (création ou mutation de mode)
		if strings.HasPrefix(line, "new file mode ") || strings.HasPrefix(line, "new mode ") || strings.HasPrefix(line, "old mode ") {
			mode := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(line, "new file mode "), "new mode "), "old mode "))
			if C2_validate_patch_mode([]byte(mode)) == 0 {
				return nil, fmt.Errorf("sécurité violée: création ou mutation vers lien symbolique ou gitlink (mode %s) interdite", mode)
			}
			if curFile != nil {
				if strings.HasPrefix(line, "new file mode ") {
					curFile.IsNewFile = true
				}
				curFile.NewMode = mode
			}
			continue
		}

		if strings.HasPrefix(line, "deleted file mode ") {
			mode := strings.TrimSpace(strings.TrimPrefix(line, "deleted file mode "))
			if curFile != nil {
				curFile.IsDeleted = true
				curFile.OldMode = mode
			}
			continue
		}

		// 3. Chemins cibles (--- a/ et +++ b/)
		if strings.HasPrefix(line, "--- a/") || strings.HasPrefix(line, "--- /dev/null") {
			if curFile == nil {
				curFile = &FilePatch{}
			}
			if strings.HasPrefix(line, "--- a/") {
				p := strings.TrimPrefix(line, "--- a/")
				if err := validatePath(p); err != nil {
					return nil, err
				}
				curFile.OldPath = p
			}
			continue
		}

		if strings.HasPrefix(line, "+++ b/") || strings.HasPrefix(line, "+++ /dev/null") {
			if curFile == nil {
				curFile = &FilePatch{}
			}
			if strings.HasPrefix(line, "+++ b/") {
				p := strings.TrimPrefix(line, "+++ b/")
				if err := validatePath(p); err != nil {
					return nil, err
				}
				curFile.NewPath = p
			}
			continue
		}

		// 4. En-tête de Hunk (@@ -start,len +start,len @@)
		if strings.HasPrefix(line, "@@ ") {
			if curHunk != nil && curFile != nil {
				if err := validateHunkLineCounts(curHunk); err != nil {
					return nil, err
				}
				curFile.Hunks = append(curFile.Hunks, *curHunk)
			}
			hunk, err := parseHunkHeader(line)
			if err != nil {
				return nil, err
			}
			curHunk = hunk
			continue
		}

		// 5. Lignes de contenu du hunk
		if curHunk != nil {
			if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ") {
				curHunk.Lines = append(curHunk.Lines, line)
			}
		}
	}

	if curHunk != nil && curFile != nil {
		if err := validateHunkLineCounts(curHunk); err != nil {
			return nil, err
		}
		curFile.Hunks = append(curFile.Hunks, *curHunk)
	}
	if curFile != nil {
		ps.Files = append(ps.Files, *curFile)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("lecture du patch: %w", err)
	}

	return ps, nil
}

func validateHunkLineCounts(h *Hunk) error {
	oldCount := 0
	newCount := 0
	for _, l := range h.Lines {
		if len(l) == 0 {
			continue
		}
		switch l[0] {
		case ' ':
			oldCount++
			newCount++
		case '-':
			oldCount++
		case '+':
			newCount++
		}
	}
	if oldCount != h.OldLines || newCount != h.NewLines {
		return fmt.Errorf("hunk malformé ou tronqué: lignes réelles (old=%d, new=%d) ne correspondent pas au header (old=%d, new=%d)", oldCount, newCount, h.OldLines, h.NewLines)
	}
	return nil
}

func validatePath(p string) error {
	p = strings.TrimSpace(p)
	if p == "" || p == "/dev/null" {
		return nil
	}
	// Normalisation stricte : conversion des antislashes Windows en slashes POSIX
	p = strings.ReplaceAll(p, "\\", "/")

	if filepath.IsAbs(p) || strings.HasPrefix(p, "/") {
		return fmt.Errorf("sécurité: chemin absolu %q interdit dans le patch", p)
	}
	parts := strings.Split(p, "/")
	for _, part := range parts {
		if part == ".." {
			return fmt.Errorf("sécurité: path traversal '..' détecté dans %q", p)
		}
		lower := strings.ToLower(part)
		if lower == ".git" || lower == ".github" || lower == ".gitmodules" || lower == ".gitattributes" || lower == ".gitignore" {
			return fmt.Errorf("sécurité: zone ou fichier protégé %q interdit dans le patch", p)
		}
	}
	return nil
}

func parseHunkHeader(line string) (*Hunk, error) {
	// Format: @@ -1,5 +1,6 @@
	parts := strings.Split(line, "@@")
	if len(parts) < 3 {
		return nil, fmt.Errorf("en-tête de hunk invalide: %q", line)
	}
	header := strings.TrimSpace(parts[1])
	fields := strings.Fields(header)
	if len(fields) != 2 {
		return nil, fmt.Errorf("en-tête de hunk mal formé: %q", header)
	}

	oldSpec := strings.TrimPrefix(fields[0], "-")
	newSpec := strings.TrimPrefix(fields[1], "+")

	oldStart, oldLines, err := parseRange(oldSpec)
	if err != nil {
		return nil, err
	}
	newStart, newLines, err := parseRange(newSpec)
	if err != nil {
		return nil, err
	}

	return &Hunk{
		OldStart: oldStart,
		OldLines: oldLines,
		NewStart: newStart,
		NewLines: newLines,
		Lines:    make([]string, 0, 16),
	}, nil
}

func parseRange(spec string) (int, int, error) {
	parts := strings.Split(spec, ",")
	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("start invalide %q: %w", spec, err)
	}
	lines := 1
	if len(parts) > 1 {
		l, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, fmt.Errorf("lines invalide %q: %w", spec, err)
		}
		lines = l
	}
	return start, lines, nil
}
