package c2grid

// C2_cell_t represents a single terminal cell, 8 bytes.
type C2_cell_t struct {
	Rune  uint32 // 4 bytes
	Fg    uint8  // 1 byte
	Bg    uint8  // 1 byte
	Flags uint8  // 1 byte
	Width uint8  // 1 byte
}
