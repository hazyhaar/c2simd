//go:build linux

package c2pty

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

// Winsize décrit la géométrie d'une fenêtre de terminal (lignes, colonnes et pixels).
type Winsize struct {
	Rows   uint16
	Cols   uint16
	Xpixel uint16
	Ypixel uint16
}

// sysWinsize correspond à la structure struct winsize du noyau Linux.
type sysWinsize struct {
	ws_row    uint16
	ws_col    uint16
	ws_xpixel uint16
	ws_ypixel uint16
}

const (
	// Constantes ioctl Linux pour la gestion des pseudo-terminaux.
	sysTIOCGPTN   = syscall.TIOCGPTN   // 0x80045430 : obtention du numéro de pseudo-terminal esclave
	sysTIOCSPTLCK = syscall.TIOCSPTLCK // 0x40045431 : verrouillage/déverrouillage de l'esclave
	sysTIOCGWINSZ = syscall.TIOCGWINSZ // 0x5413 : lecture de la taille de fenêtre
	sysTIOCSWINSZ = syscall.TIOCSWINSZ // 0x5414 : écriture de la taille de fenêtre
	sysTIOCSCTTY  = syscall.TIOCSCTTY  // 0x540E : attribution du terminal de contrôle
)

func ioctl(fd uintptr, req uintptr, arg unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, req, uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}

// OpenMaster ouvre le multiplexeur maître de pseudo-terminaux /dev/ptmx.
func OpenMaster() (*os.File, error) {
	fd, err := syscall.Open("/dev/ptmx", syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("c2pty: échec ouverture /dev/ptmx: %w", err)
	}
	return os.NewFile(uintptr(fd), "/dev/ptmx"), nil
}

// Grantpt valide les permissions de l'esclave. Sous Linux avec devpts,
// le noyau configure automatiquement le propriétaire et les droits.
func Grantpt(master *os.File) error {
	if master == nil {
		return os.ErrInvalid
	}
	return nil
}

// Unlockpt déverrouille le pseudo-terminal esclave associé au maître.
func Unlockpt(master *os.File) error {
	if master == nil {
		return os.ErrInvalid
	}
	var unlock int = 0
	if err := ioctl(master.Fd(), sysTIOCSPTLCK, unsafe.Pointer(&unlock)); err != nil {
		return fmt.Errorf("c2pty: échec unlockpt (TIOCSPTLCK): %w", err)
	}
	return nil
}

// Ptsname retourne le chemin absolu du périphérique esclave (/dev/pts/N).
func Ptsname(master *os.File) (string, error) {
	if master == nil {
		return "", os.ErrInvalid
	}
	var ptn uint32
	if err := ioctl(master.Fd(), sysTIOCGPTN, unsafe.Pointer(&ptn)); err != nil {
		return "", fmt.Errorf("c2pty: échec ptsname (TIOCGPTN): %w", err)
	}
	return fmt.Sprintf("/dev/pts/%d", ptn), nil
}

// OpenSlave ouvre le périphérique esclave désigné par son chemin absolu.
func OpenSlave(ptsName string) (*os.File, error) {
	fd, err := syscall.Open(ptsName, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("c2pty: échec ouverture esclave %s: %w", ptsName, err)
	}
	return os.NewFile(uintptr(fd), ptsName), nil
}

// OpenPTY initialise et alloue une paire de pseudo-terminaux maître/esclave en pur Go.
func OpenPTY(ws *Winsize) (master *os.File, slave *os.File, err error) {
	master, err = OpenMaster()
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err != nil && master != nil {
			_ = master.Close()
		}
	}()

	if err = Grantpt(master); err != nil {
		return nil, nil, err
	}
	if err = Unlockpt(master); err != nil {
		return nil, nil, err
	}

	ptsName, err := Ptsname(master)
	if err != nil {
		return nil, nil, err
	}

	if ws != nil {
		if err = SetWinsize(master, ws); err != nil {
			return nil, nil, err
		}
	}

	slave, err = OpenSlave(ptsName)
	if err != nil {
		return nil, nil, err
	}

	return master, slave, nil
}

// SetWinsize modifie les dimensions du terminal via l'ioctl TIOCSWINSZ.
// Le noyau émet automatiquement un signal SIGWINCH au groupe de processus d'avant-plan.
func SetWinsize(f *os.File, ws *Winsize) error {
	if f == nil || ws == nil {
		return os.ErrInvalid
	}
	sw := sysWinsize{
		ws_row:    ws.Rows,
		ws_col:    ws.Cols,
		ws_xpixel: ws.Xpixel,
		ws_ypixel: ws.Ypixel,
	}
	if err := ioctl(f.Fd(), sysTIOCSWINSZ, unsafe.Pointer(&sw)); err != nil {
		return fmt.Errorf("c2pty: échec ioctl TIOCSWINSZ: %w", err)
	}
	return nil
}

// GetWinsize extrait les dimensions actuelles du terminal via l'ioctl TIOCGWINSZ.
func GetWinsize(f *os.File) (*Winsize, error) {
	if f == nil {
		return nil, os.ErrInvalid
	}
	var sw sysWinsize
	if err := ioctl(f.Fd(), sysTIOCGWINSZ, unsafe.Pointer(&sw)); err != nil {
		return nil, fmt.Errorf("c2pty: échec ioctl TIOCGWINSZ: %w", err)
	}
	return &Winsize{
		Rows:   sw.ws_row,
		Cols:   sw.ws_col,
		Xpixel: sw.ws_xpixel,
		Ypixel: sw.ws_ypixel,
	}, nil
}

// ForkPTY configure un processus enfant attaché au pseudo-terminal esclave
// sans recourir à CGO, en exploitant SysProcAttr (Setsid et Setctty).
func ForkPTY(ws *Winsize, cmd *exec.Cmd) (*os.File, error) {
	if cmd == nil {
		return nil, os.ErrInvalid
	}
	master, slave, err := OpenPTY(ws)
	if err != nil {
		return nil, err
	}
	defer func() {
		// L'esclave est toujours fermé dans le processus parent après démarrage.
		_ = slave.Close()
	}()

	cmd.Stdin = slave
	cmd.Stdout = slave
	cmd.Stderr = slave
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
	cmd.SysProcAttr.Setctty = true
	cmd.SysProcAttr.Ctty = 0

	if err := cmd.Start(); err != nil {
		_ = master.Close()
		return nil, fmt.Errorf("c2pty: échec démarrage du sous-processus: %w", err)
	}

	return master, nil
}
