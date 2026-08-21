package c2pty

import (
	"errors"
	"io"
	"sync"
)

var (
	// ErrBufferFull indique que le tampon circulaire n'a plus d'espace disponible.
	ErrBufferFull = errors.New("c2pty: tampon circulaire plein")
	// ErrBufferEmpty indique que le tampon circulaire est vide.
	ErrBufferEmpty = errors.New("c2pty: tampon circulaire vide")
	// ErrRingClosed indique que le tampon circulaire a été fermé.
	ErrRingClosed = errors.New("c2pty: tampon circulaire fermé")
)

// RingBuffer implémente un tampon circulaire thread-safe non bloquant
// optimisé pour le streaming d'entrées/sorties et la rétention de flux VT/terminal.
type RingBuffer struct {
	buf    []byte
	cap    int
	head   int // Index de lecture
	tail   int // Index d'écriture
	size   int // Nombre d'octets actuellement stockés
	closed bool
	mu     sync.Mutex
}

// NewRingBuffer initialise un tampon circulaire d'une capacité minimale donnée.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = 1024
	}
	return &RingBuffer{
		buf:  make([]byte, capacity),
		cap:  capacity,
		head: 0,
		tail: 0,
		size: 0,
	}
}

// Cap retourne la capacité totale du tampon circulaire en octets.
func (r *RingBuffer) Cap() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cap
}

// Len retourne le nombre d'octets non encore lus dans le tampon.
func (r *RingBuffer) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.size
}

// Free retourne l'espace restant disponible en octets pour une écriture non écrasante.
func (r *RingBuffer) Free() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cap - r.size
}

// IsEmpty indique si le tampon ne contient aucune donnée.
func (r *RingBuffer) IsEmpty() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.size == 0
}

// IsFull indique si le tampon a atteint sa capacité maximale.
func (r *RingBuffer) IsFull() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.size == r.cap
}

// Reset réinitialise les curseurs de lecture et écriture sans réallouer la mémoire.
func (r *RingBuffer) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.head = 0
	r.tail = 0
	r.size = 0
}

// Write insère des données dans le tampon circulaire de manière non bloquante.
// S'il n'y a pas assez de place pour tout écrire, écrit autant que possible
// et retourne le nombre d'octets écrits accompagné de ErrBufferFull.
func (r *RingBuffer) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return 0, io.ErrClosedPipe
	}

	available := r.cap - r.size
	if available == 0 {
		return 0, ErrBufferFull
	}

	toWrite := len(p)
	var err error
	if toWrite > available {
		toWrite = available
		err = ErrBufferFull
	}

	chunk1 := r.cap - r.tail
	if chunk1 > toWrite {
		chunk1 = toWrite
	}
	copy(r.buf[r.tail:r.tail+chunk1], p[:chunk1])

	chunk2 := toWrite - chunk1
	if chunk2 > 0 {
		copy(r.buf[:chunk2], p[chunk1:toWrite])
	}

	r.tail = (r.tail + toWrite) % r.cap
	r.size += toWrite

	return toWrite, err
}

// WriteOverwrite insère des octets dans le tampon circulaire en écrasant les plus anciens
// si la capacité est dépassée. Cette opération ne retourne jamais d'erreur et ne bloque jamais.
func (r *RingBuffer) WriteOverwrite(p []byte) int {
	if len(p) == 0 {
		return 0
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return 0
	}

	// Si l'entrée dépasse ou égale la capacité totale, on ne garde que les derniers r.cap octets.
	if len(p) >= r.cap {
		src := p[len(p)-r.cap:]
		copy(r.buf, src)
		r.head = 0
		r.tail = 0
		r.size = r.cap
		return len(p)
	}

	// Si l'insertion dépasse l'espace libre, on avance la tête pour libérer les plus anciens octets.
	overflow := len(p) - (r.cap - r.size)
	if overflow > 0 {
		r.head = (r.head + overflow) % r.cap
		r.size -= overflow
	}

	chunk1 := r.cap - r.tail
	if chunk1 > len(p) {
		chunk1 = len(p)
	}
	copy(r.buf[r.tail:r.tail+chunk1], p[:chunk1])

	chunk2 := len(p) - chunk1
	if chunk2 > 0 {
		copy(r.buf[:chunk2], p[chunk1:])
	}

	r.tail = (r.tail + len(p)) % r.cap
	r.size += len(p)

	return len(p)
}

// Read extrait des octets du tampon circulaire de manière non bloquante.
// Si le tampon est vide :
// - renvoie (0, io.EOF) si le tampon a été fermé via Close(),
// - renvoie (0, ErrBufferEmpty) si le tampon est ouvert.
func (r *RingBuffer) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.size == 0 {
		if r.closed {
			return 0, io.EOF
		}
		return 0, ErrBufferEmpty
	}

	toRead := len(p)
	if toRead > r.size {
		toRead = r.size
	}

	chunk1 := r.cap - r.head
	if chunk1 > toRead {
		chunk1 = toRead
	}
	copy(p[:chunk1], r.buf[r.head:r.head+chunk1])

	chunk2 := toRead - chunk1
	if chunk2 > 0 {
		copy(p[chunk1:toRead], r.buf[:chunk2])
	}

	r.head = (r.head + toRead) % r.cap
	r.size -= toRead

	return toRead, nil
}

// TryRead est un alias direct non bloquant de Read.
func (r *RingBuffer) TryRead(p []byte) (int, error) {
	return r.Read(p)
}

// TryWrite est un alias direct non bloquant de Write.
func (r *RingBuffer) TryWrite(p []byte) (int, error) {
	return r.Write(p)
}

// PeekInto inspecte jusqu'à len(p) octets non consommés sans modifier l'état du tampon,
// en copiant directement dans la tranche p fournie par l'appelant (zéro allocation).
func (r *RingBuffer) PeekInto(p []byte) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.size == 0 || len(p) == 0 {
		return 0
	}

	toRead := len(p)
	if toRead > r.size {
		toRead = r.size
	}

	chunk1 := r.cap - r.head
	if chunk1 > toRead {
		chunk1 = toRead
	}
	copy(p[:chunk1], r.buf[r.head:r.head+chunk1])

	chunk2 := toRead - chunk1
	if chunk2 > 0 {
		copy(p[chunk1:toRead], r.buf[:chunk2])
	}

	return toRead
}

// Peek inspecte jusqu'à n octets non consommés sans modifier l'état du tampon.
func (r *RingBuffer) Peek(n int) []byte {
	if n <= 0 {
		return nil
	}
	out := make([]byte, n)
	read := r.PeekInto(out)
	if read == 0 {
		return nil
	}
	return out[:read]
}

// Discard consomme et ignore jusqu'à n octets dans le tampon.
func (r *RingBuffer) Discard(n int) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.size == 0 || n <= 0 {
		return 0
	}

	if n > r.size {
		n = r.size
	}

	r.head = (r.head + n) % r.cap
	r.size -= n
	return n
}

// Bytes retourne une copie de l'ensemble des données non encore lues.
func (r *RingBuffer) Bytes() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.size == 0 {
		return nil
	}

	out := make([]byte, r.size)
	chunk1 := r.cap - r.head
	if chunk1 > r.size {
		chunk1 = r.size
	}
	copy(out[:chunk1], r.buf[r.head:r.head+chunk1])

	chunk2 := r.size - chunk1
	if chunk2 > 0 {
		copy(out[chunk1:], r.buf[:chunk2])
	}

	return out
}

// String convertit les octets en attente en chaîne de caractères.
func (r *RingBuffer) String() string {
	return string(r.Bytes())
}

// Close ferme le tampon. Les écritures futures retourneront io.ErrClosedPipe.
// Les lectures continueront jusqu'à épuisement des données résiduelles, puis retourneront io.EOF.
func (r *RingBuffer) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return nil
}
