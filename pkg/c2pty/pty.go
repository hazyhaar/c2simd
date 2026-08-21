package c2pty

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

var (
	// ErrClosed indique que l'instance PTY a été fermée.
	ErrClosed = errors.New("c2pty: pseudo-terminal fermé")
	// ErrProcessNotStarted indique qu'aucun processus n'a encore été démarré.
	ErrProcessNotStarted = errors.New("c2pty: aucun sous-processus actif")
	// ErrAlreadyStarted indique qu'un processus a déjà été attaché au PTY.
	ErrAlreadyStarted = errors.New("c2pty: processus déjà démarré")
)

// PTY encapsule une session de pseudo-terminal Unix maître/esclave et son sous-processus associé.
// Il implémente les interfaces io.Reader, io.Writer et io.Closer.
type PTY struct {
	master    *os.File
	slave     *os.File
	ptsName   string
	cmd       *exec.Cmd
	ws        Winsize
	mu        sync.Mutex
	closed    bool
	closeOnce sync.Once
}

// Open instancie un pseudo-terminal avec la géométrie spécifiée.
// Si ws est nul, une taille par défaut de 80 colonnes par 24 lignes est appliquée.
func Open(ws *Winsize) (*PTY, error) {
	targetWs := Winsize{Cols: 80, Rows: 24}
	if ws != nil {
		targetWs = *ws
	}

	master, slave, err := OpenPTY(&targetWs)
	if err != nil {
		return nil, err
	}

	ptsName, err := Ptsname(master)
	if err != nil {
		_ = master.Close()
		_ = slave.Close()
		return nil, err
	}

	return &PTY{
		master:  master,
		slave:   slave,
		ptsName: ptsName,
		ws:      targetWs,
	}, nil
}

// StartCommand alloue un nouveau PTY et y démarre immédiatement la commande spécifiée.
func StartCommand(command string, args ...string) (*PTY, error) {
	return StartWithWinsize(nil, command, args...)
}

// StartWithWinsize alloue un nouveau PTY avec une taille donnée et y démarre la commande.
func StartWithWinsize(ws *Winsize, command string, args ...string) (*PTY, error) {
	p, err := Open(ws)
	if err != nil {
		return nil, err
	}
	if err := p.Start(command, args...); err != nil {
		_ = p.Close()
		return nil, err
	}
	return p, nil
}

// Start configure et lance la commande indiquée dans le pseudo-terminal esclave.
func (p *PTY) Start(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	return p.StartCmd(cmd)
}

// StartCmd attache et lance un *exec.Cmd préconfiguré au pseudo-terminal.
func (p *PTY) StartCmd(cmd *exec.Cmd) error {
	if cmd == nil {
		return os.ErrInvalid
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrClosed
	}
	if p.cmd != nil {
		return ErrAlreadyStarted
	}
	if p.slave == nil {
		return errors.New("c2pty: descripteur esclave indisponible")
	}

	cmd.Stdin = p.slave
	cmd.Stdout = p.slave
	cmd.Stderr = p.slave
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
	cmd.SysProcAttr.Setctty = true
	cmd.SysProcAttr.Ctty = 0

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("c2pty: échec démarrage commande: %w", err)
	}

	p.cmd = cmd

	// Fermeture du descripteur esclave côté parent pour permettre la détection EOF
	// lorsque le sous-processus enfant se termine.
	_ = p.slave.Close()
	p.slave = nil

	return nil
}

// Read lit les données produites par le terminal maître.
// Sous Linux, la fermeture de tous les descripteurs esclaves provoque l'erreur EIO,
// qui est automatiquement traduite en io.EOF.
func (p *PTY) Read(buf []byte) (int, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return 0, ErrClosed
	}
	master := p.master
	p.mu.Unlock()

	if master == nil {
		return 0, ErrClosed
	}

	n, err := master.Read(buf)
	if err != nil {
		if errors.Is(err, syscall.EIO) {
			return n, io.EOF
		}
		return n, err
	}
	return n, nil
}

// Write transmet les données saisies au terminal maître.
func (p *PTY) Write(buf []byte) (int, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return 0, ErrClosed
	}
	master := p.master
	p.mu.Unlock()

	if master == nil {
		return 0, ErrClosed
	}

	return master.Write(buf)
}

// Resize met à jour la dimension de la fenêtre de terminal (colonnes, lignes)
// et déclenche un signal SIGWINCH transmis par le noyau au sous-processus.
func (p *PTY) Resize(cols, rows uint16) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrClosed
	}
	p.ws.Cols = cols
	p.ws.Rows = rows

	return SetWinsize(p.master, &p.ws)
}

// SetWinsize applique directement une structure Winsize au terminal maître.
func (p *PTY) SetWinsize(ws *Winsize) error {
	if ws == nil {
		return os.ErrInvalid
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrClosed
	}
	p.ws = *ws
	return SetWinsize(p.master, &p.ws)
}

// GetWinsize récupère la géométrie courante du pseudo-terminal.
func (p *PTY) GetWinsize() (*Winsize, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, ErrClosed
	}
	return GetWinsize(p.master)
}

// Master retourne le fichier descripteur maître.
func (p *PTY) Master() *os.File {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.master
}

// Slave retourne le fichier descripteur esclave (si encore ouvert).
func (p *PTY) Slave() *os.File {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.slave
}

// PtsName retourne le chemin du pseudo-terminal esclave sous /dev/pts/.
func (p *PTY) PtsName() string {
	return p.ptsName
}

// Fd retourne le descripteur de fichier du maître.
func (p *PTY) Fd() uintptr {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.master != nil {
		return p.master.Fd()
	}
	return ^uintptr(0)
}

// Process retourne le descripteur du sous-processus en cours d'exécution.
func (p *PTY) Process() *os.Process {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd != nil {
		return p.cmd.Process
	}
	return nil
}

// Wait attend la fin d'exécution du sous-processus associé.
func (p *PTY) Wait() error {
	p.mu.Lock()
	cmd := p.cmd
	p.mu.Unlock()

	if cmd == nil {
		return ErrProcessNotStarted
	}
	return cmd.Wait()
}

// Signal transmet un signal au sous-processus.
func (p *PTY) Signal(sig os.Signal) error {
	proc := p.Process()
	if proc == nil {
		return ErrProcessNotStarted
	}
	return proc.Signal(sig)
}

// Kill termine immédiatement le sous-processus par SIGKILL.
func (p *PTY) Kill() error {
	proc := p.Process()
	if proc == nil {
		return ErrProcessNotStarted
	}
	return proc.Kill()
}

// Close libère les ressources allouées par le pseudo-terminal de manière idempotente.
func (p *PTY) Close() error {
	var closeErr error
	p.closeOnce.Do(func() {
		p.mu.Lock()
		p.closed = true
		master := p.master
		slave := p.slave
		p.master = nil
		p.slave = nil
		cmd := p.cmd
		p.mu.Unlock()

		if slave != nil {
			_ = slave.Close()
		}
		if master != nil {
			if err := master.Close(); err != nil {
				closeErr = err
			}
		}
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})
	return closeErr
}
