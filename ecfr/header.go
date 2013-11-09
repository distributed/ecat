package ecfr

import (
	"errors"
)

type Header struct {
	Word uint16
}

func (h *Header) Overlay(b []byte) ([]byte, error) {
	if len(b) < 2 {
		return b, errors.New("not enough bytes for header")
	}

	h.Word, b = getUint16(b)
	return b, nil
}

func (h *Header) FrameLength() uint16 {
	return h.Word & ((1 << 11) - 1)
}

// TODO: data type?
func (h *Header) Type() uint8 {
	return uint8(h.Word>>12) & 0x0f
}
