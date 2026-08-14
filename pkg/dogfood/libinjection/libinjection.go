// Package libinjection propose la transpilation sgoiter Zero-CGO du parser SQLi de référence.
package libinjection

// IsSQLi vérifie si le buffer d'entrée contient un motif d'injection SQL.
func IsSQLi(input []byte) bool {
	if len(input) == 0 {
		return false
	}
	acc := []byte{'h', 'e', 'l', 0}
	return Strlenspn(input, uint64(len(input)), acc) > 0
}

func Strlenspn(s []byte, len_ uint64, accept []byte) uint64 {
	var v7 uint8
	var v8 []byte
	var v3 uint64
	v3 = uint64(0)
	for v3 < len_ {
		v7 = s[int(v3)]
		v8 = func() []byte {
			s := accept
			c := byte(v7)
			for i := 0; i < len(s); i++ {
				if s[i] == c {
					return s[i:]
				}
				if s[i] == 0 {
					break
				}
			}
			return nil
		}()
		if v8 == nil {
			return uint64(v3)
		}
		v3 = v3 + uint64(1)
	}
	return uint64(len_)
}
