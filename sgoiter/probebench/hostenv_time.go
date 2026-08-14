package probebench

import "time"

func init() {
	unixNano = func() int64 { return time.Now().UnixNano() }
}
