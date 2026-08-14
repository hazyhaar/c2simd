// Package checksum fournit les contrôles d'intégrité transpilés de C2Go (fnv1a, crc32).
package checksum

// FNV1a64 calcule le hash FNV-1a 64-bit sur le buffer data.
func FNV1a64(data []byte) uint64 {
	var v4 uint64
	var v8 uint8
	var v9 uint64
	var v2 uint64
	v2 = uint64(14695981039346656037)
	v4 = uint64(0)
	len_ := uint64(len(data))
	for v4 < len_ {
		v8 = data[int(v4)]
		v9 = uint64(v8)
		v2 = v2 ^ v9
		v2 = v2 * uint64(1099511628211)
		v4 = v4 + uint64(1)
	}
	return uint64(v2)
}

// CRC32IEEE calcule le contrôle de redondance cyclique IEEE 802.3.
func CRC32IEEE(data []byte) uint32 {
	var v2 uint32 = uint32(4294967295)
	var v4 uint32 = uint32(0)
	len_ := uint64(len(data))
	for uint64(v4) < len_ {
		v9 := data[int(v4)]
		v2 = v2 ^ uint32(v9)
		v5 := uint32(0)
		for v5 < uint32(8) {
			v17 := v2 & uint32(1)
			v18 := int(v17)
			v20 := -int(v18)
			v21 := uint32(v20)
			v23 := v2 >> uint8(1)
			v25 := uint32(3988292384) & v21
			v2 = v23 ^ v25
			v5 = v5 + uint32(1)
		}
		v4 = v4 + uint32(1)
	}
	v27 := ^v2
	return uint32(v27)
}
