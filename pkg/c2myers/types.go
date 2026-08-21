package c2myers

import "time"

// Hunk décrit un bloc contigu de modifications dans un format diff unifié.
type Hunk struct {
	OldStart int      `json:"old_start"`
	OldLines int      `json:"old_lines"`
	NewStart int      `json:"new_start"`
	NewLines int      `json:"new_lines"`
	Lines    []string `json:"lines"`
}

// FilePatch représente l'ensemble des modifications affectant un fichier cible.
type FilePatch struct {
	OldPath     string `json:"old_path"`
	NewPath     string `json:"new_path"`
	OldMode     string `json:"old_mode,omitempty"`
	NewMode     string `json:"new_mode,omitempty"`
	IsNewFile   bool   `json:"is_new_file"`
	IsDeleted   bool   `json:"is_deleted"`
	IsBinary    bool   `json:"is_binary"`
	Hunks       []Hunk `json:"hunks"`
}

// PatchSet regroupe l'ensemble des fichiers modifiés d'un patch unifié.
type PatchSet struct {
	Files     []FilePatch `json:"files"`
	CreatedAt time.Time   `json:"created_at"`
}

// OpType représente le type d'opération élémentaire d'un diff.
type OpType int

const (
	OpEqual OpType = iota
	OpInsert
	OpDelete
)

// EditOp modélise une modification élémentaire sur une séquence de lignes.
type EditOp struct {
	Type OpType
	Line string
}
