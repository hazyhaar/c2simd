# c2pty

Gestionnaire de pseudo-terminaux Unix POSIX en Go pur (`CGO_ENABLED=0`) et tampon circulaire pour le streaming I/O.

## Composants

- **`pty_linux.go`** : Couche d'appels système POSIX bas niveau (`/dev/ptmx`, ioctl `TIOCGPTN`, `TIOCSPTLCK`, `TIOCGWINSZ`, `TIOCSWINSZ`, `grantpt`/`unlockpt` purs sans CGO, `ForkPTY`).
- **`pty.go`** : Structure `PTY` de haut niveau implémentant les interfaces standard `io.Reader`, `io.Writer` et `io.Closer`, avec gestion des signaux, redimensionnement dynamique (`Resize`) et exécution de commandes (`Start`).
- **`ringbuffer.go`** : Tampon circulaire thread-safe non bloquant optimisé pour le streaming d'entrées/sorties et la rétention de flux VT.

## Utilisation

```go
package main

import (
    "fmt"
    "io"
    "log"

    "code.hazyhaar.fr/devhoros/c2simd/pkg/c2pty"
)

func main() {
    pty, err := c2pty.StartCommand("/bin/sh")
    if err != nil {
        log.Fatal(err)
    }
    defer pty.Close()

    _ = pty.Resize(120, 40)
    _, _ = pty.Write([]byte("uname -a\nexit\n"))

    buf := make([]byte, 1024)
    for {
        n, err := pty.Read(buf)
        if n > 0 {
            fmt.Print(string(buf[:n]))
        }
        if err == io.EOF {
            break
        }
    }
}
```

## Validation

```bash
cd /devhoros/c2simd/pkg/c2pty && go test -race -v ./...
```
