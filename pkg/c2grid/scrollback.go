package c2grid

// ScrollbackRing est un tampon circulaire paginé pour les lignes d'historique de terminal,
// conçu pour opérer à zéro allocation en régime permanent.
type ScrollbackRing struct {
	lines [][]C2_cell_t
	head  int
	tail  int
	count int
	cap   int
}

// NewScrollbackRing initialise un tampon d'historique avec allocation paresseuse.
func NewScrollbackRing(capacity int) *ScrollbackRing {
	if capacity <= 0 {
		capacity = 1000
	}
	return &ScrollbackRing{
		lines: make([][]C2_cell_t, capacity),
		cap:   capacity,
	}
}

// NewScrollbackRingFixed initialise un tampon d'historique entièrement pré-alloué
// sur un tableau plat contigu (zéro allocation garantie dès la première insertion).
func NewScrollbackRingFixed(capacity, cols int) *ScrollbackRing {
	if capacity <= 0 {
		capacity = 1000
	}
	if cols <= 0 {
		cols = 80
	}
	r := &ScrollbackRing{
		lines: make([][]C2_cell_t, capacity),
		cap:   capacity,
	}
	backing := make([]C2_cell_t, capacity*cols)
	for i := 0; i < capacity; i++ {
		r.lines[i] = backing[i*cols : (i+1)*cols : (i+1)*cols]
	}
	return r
}

// Push insère une ligne d'affichage dans le tampon d'historique en réutilisant
// les tranches de mémoire sous-jacentes sans aucune allocation sur le tas.
func (r *ScrollbackRing) Push(line []C2_cell_t) {
	if r.cap == 0 || len(line) == 0 {
		return
	}
	slot := r.lines[r.head]
	if cap(slot) < len(line) {
		slot = make([]C2_cell_t, len(line))
	} else {
		slot = slot[:len(line)]
	}
	copy(slot, line)
	r.lines[r.head] = slot

	r.head = (r.head + 1) % r.cap
	if r.count < r.cap {
		r.count++
	} else {
		r.tail = (r.tail + 1) % r.cap
	}
}

// Count retourne le nombre de lignes actuellement archivées.
func (r *ScrollbackRing) Count() int {
	return r.count
}

// Get retourne la ligne à l'indice logique i (0 = ligne la plus ancienne).
func (r *ScrollbackRing) Get(i int) []C2_cell_t {
	if i < 0 || i >= r.count {
		return nil
	}
	idx := (r.tail + i) % r.cap
	return r.lines[idx]
}
