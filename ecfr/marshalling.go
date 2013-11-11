package ecfr

// the "native" byte ordering is the little endian encoding scheme
// of ehthercat. big endian routines for the encoding used by ethernet
// are provided below

func getUint8(b []byte) (uint8, []byte) {
	return b[0], b[1:]
}

func getUint16(b []byte) (uint16, []byte) {
	return uint16(b[0]) | uint16(b[1])<<8, b[2:]
}

func getUint32(b []byte) (uint32, []byte) {
	v := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	return v, b[4:]
}

func xgetUint8(b []byte) uint8 {
	return b[0]
}

func xgetUint16(b []byte) uint16 {
	return uint16(b[0]) | uint16(b[1])<<8
}

func xgetUint32(b []byte) uint32 {
	v := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	return v
}

func putUint8(b []byte, v uint8) []byte {
	b[0] = v
	return b[1:]
}

func putUint16(b []byte, v uint16) []byte {
	b[0] = uint8(v)
	b[1] = uint8(v >> 8)
	return b[2:]
}

func putUint32(b []byte, v uint32) []byte {
	b[0] = uint8(v)
	b[1] = uint8(v >> 8)
	b[2] = uint8(v >> 16)
	b[3] = uint8(v >> 24)
	return b[4:]
}

func getUint8BE(b []byte) (uint8, []byte) {
	return b[0], b[1:]
}

func getUint16BE(b []byte) (uint16, []byte) {
	return uint16(b[0])<<8 | uint16(b[1]), b[2:]
}

func getUint32BE(b []byte) (uint32, []byte) {
	v := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	return v, b[4:]
}

func putUint8BE(b []byte, v uint8) []byte {
	b[0] = v
	return b[1:]
}

func putUint16BE(b []byte, v uint16) []byte {
	b[0] = uint8(v >> 8)
	b[1] = uint8(v)
	return b[2:]
}

func putUint32BE(b []byte, v uint32) []byte {
	b[0] = uint8(v >> 24)
	b[1] = uint8(v >> 16)
	b[2] = uint8(v >> 8)
	b[3] = uint8(v)
	return b[4:]
}
