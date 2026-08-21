package c2painter

import "sync"

const (
	NodeTypeRect        uint16 = 1
	NodeTypeRoundedRect uint16 = 2
	NodeTypeCircle      uint16 = 3
	NodeTypeEllipse     uint16 = 4
	NodeTypeLine        uint16 = 5
	NodeTypeTextGlyph   uint16 = 6
	NodeTypeMask        uint16 = 7
	NodeTypeShadow      uint16 = 8
)

// Scene gère un graphe de scène linéaire plat à double tamponnage contigu.
// Les goroutines de l'UI ajoutent des nœuds dans le tampon d'écriture, et la
// passe de rendu capture un instantané instantané (snapshot) zéro-copie sans
// aucun parcours de pointeurs ni risque de course de données.
type Scene struct {
	mu       sync.Mutex
	staging  []C2_node_t
	snapshot []C2_node_t
	capacity int
}

// NewScene initialise un gestionnaire de scène linéaire préalloué.
func NewScene(initialCapacity int) *Scene {
	if initialCapacity <= 0 {
		initialCapacity = 1024
	}
	return &Scene{
		staging:  make([]C2_node_t, 0, initialCapacity),
		snapshot: make([]C2_node_t, 0, initialCapacity),
		capacity: initialCapacity,
	}
}

// Clear réinitialise la liste des nœuds en préparation de la prochaine trame.
func (s *Scene) Clear() {
	s.mu.Lock()
	s.staging = s.staging[:0]
	s.mu.Unlock()
}

// AddRect ajoute un rectangle plein à la scène.
func (s *Scene) AddRect(x, y, w, h int, color uint32) {
	s.mu.Lock()
	s.staging = append(s.staging, C2_node_t{
		Type_:  NodeTypeRect,
		Flags:  0,
		X:      x,
		Y:      y,
		W:      w,
		H:      h,
		Color0: color,
	})
	s.mu.Unlock()
}

// AddStrokeRect ajoute un rectangle bordé à la scène.
func (s *Scene) AddStrokeRect(x, y, w, h, strokeWidth int, color uint32) {
	s.mu.Lock()
	s.staging = append(s.staging, C2_node_t{
		Type_:  NodeTypeRect,
		Flags:  1,
		X:      x,
		Y:      y,
		W:      w,
		H:      h,
		Color0: color,
		Param0: strokeWidth,
	})
	s.mu.Unlock()
}

// AddRoundedRect ajoute un rectangle à coins arrondis à la scène.
func (s *Scene) AddRoundedRect(x, y, w, h, radius int, color uint32) {
	s.mu.Lock()
	s.staging = append(s.staging, C2_node_t{
		Type_:  NodeTypeRoundedRect,
		Flags:  0,
		X:      x,
		Y:      y,
		W:      w,
		H:      h,
		Color0: color,
		Param0: radius,
	})
	s.mu.Unlock()
}

// AddStrokeRoundedRect ajoute un rectangle arrondi avec bordure.
func (s *Scene) AddStrokeRoundedRect(x, y, w, h, radius, strokeWidth int, color uint32) {
	s.mu.Lock()
	s.staging = append(s.staging, C2_node_t{
		Type_:  NodeTypeRoundedRect,
		Flags:  1,
		X:      x,
		Y:      y,
		W:      w,
		H:      h,
		Color0: color,
		Param0: radius,
		Param1: strokeWidth,
	})
	s.mu.Unlock()
}

// AddCircle ajoute un cercle plein à la scène.
func (s *Scene) AddCircle(cx, cy, radius int, color uint32) {
	s.mu.Lock()
	s.staging = append(s.staging, C2_node_t{
		Type_:  NodeTypeCircle,
		Flags:  0,
		X:      cx,
		Y:      cy,
		Color0: color,
		Param0: radius,
	})
	s.mu.Unlock()
}

// AddStrokeCircle ajoute un cercle tracé à la scène.
func (s *Scene) AddStrokeCircle(cx, cy, radius, strokeWidth int, color uint32) {
	s.mu.Lock()
	s.staging = append(s.staging, C2_node_t{
		Type_:  NodeTypeCircle,
		Flags:  1,
		X:      cx,
		Y:      cy,
		Color0: color,
		Param0: radius,
		Param1: strokeWidth,
	})
	s.mu.Unlock()
}

// AddEllipse ajoute une ellipse pleine à la scène.
func (s *Scene) AddEllipse(cx, cy, rx, ry int, color uint32) {
	s.mu.Lock()
	s.staging = append(s.staging, C2_node_t{
		Type_:  NodeTypeEllipse,
		Flags:  0,
		X:      cx,
		Y:      cy,
		Color0: color,
		Param0: rx,
		Param1: ry,
	})
	s.mu.Unlock()
}

// AddLine ajoute un segment de ligne antialiasé à la scène.
func (s *Scene) AddLine(x0, y0, x1, y1, strokeWidth int, color uint32) {
	s.mu.Lock()
	s.staging = append(s.staging, C2_node_t{
		Type_:  NodeTypeLine,
		Flags:  0,
		X:      x0,
		Y:      y0,
		W:      x1,
		H:      y1,
		Color0: color,
		Param0: strokeWidth,
	})
	s.mu.Unlock()
}

// Snapshot capture instantanément la scène courante dans un tampon immuable
// pour le thread de rendu, garantissant une étanchéité totale face aux courses.
func (s *Scene) Snapshot() []C2_node_t {
	s.mu.Lock()
	defer s.mu.Unlock()

	n := len(s.staging)
	if cap(s.snapshot) < n {
		s.snapshot = make([]C2_node_t, n)
	} else {
		s.snapshot = s.snapshot[:n]
	}
	copy(s.snapshot, s.staging)
	return s.snapshot
}

// RenderSnapshot exécute le rendu en lot d'un snapshot de scène vers le peintre.
func RenderSnapshot(p *Painter, nodes []C2_node_t) {
	if p == nil || len(nodes) == 0 {
		return
	}
	C2_scene_render(&p.ctx, nodes, len(nodes))
}
