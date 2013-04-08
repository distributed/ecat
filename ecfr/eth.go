package ecfr

import (
	"errors"
	"fmt"
)

type ETHAddr [6]byte

// bounds need to be checked already
func sliceToETHADDR(e *ETHAddr, s []byte) {
	e[0] = s[0]
	e[1] = s[1]
	e[2] = s[2]
	e[3] = s[3]
	e[4] = s[4]
	e[5] = s[5]
}

type ETHFrame struct {
	Destination, Source ETHAddr
	Type                uint16

	UseVlan bool
	VLANTCI uint16

	framebuf []byte
}

func OverlayETHFrame(fb []byte) (*ETHFrame, error) {
	if len(fb) == 0 {
		fb = make([]byte, 1522)
	}

	if len(fb) < min_headerandpayload {
		return nil, fmt.Errorf("NewETHFrame: buffer too small, need at least %d bytes", min_headerandpayload)
	}

	ef := &ETHFrame{}

	// guarded by len(fb) < min_headerandpayload
	return ef, nil
}

func (ef *ETHFrame) GetHeaderLen() int {
	vlanlen := 0
	if ef.UseVlan {
		vlanlen = 4
	}
	// dest, src, type, (vlan len)
	l := 6 + 6 + 2 + vlanlen

	return l
}

// header contents will be undefined if you do not call WriteDown() before.
func (ef *ETHFrame) GetFrameBuf() []byte {
	return ef.framebuf
}

func (ef *ETHFrame) GetPayload() []byte {
	return ef.framebuf[ef.GetHeaderLen():]
}

func (ef *ETHFrame) SetPayloadLen(npl int) error {
	nl := npl + ef.GetHeaderLen()
	if nl < min_headerandpayload {
		return fmt.Errorf("SetPayloadLen: payload too small, need at least %d bytes", min_headerandpayload-ef.GetHeaderLen())
	}

	maxnl := max_framelen_novlan
	if ef.UseVlan {
		maxnl = max_framelen_vlan
	}

	if nl > maxnl {
		return fmt.Errorf("SetPayloadLen: payload too big, maximum for this configuration is %d bytes", maxnl-ef.GetHeaderLen())
	}

	if nl > cap(ef.framebuf) {
		return fmt.Errorf("SetPayloadLen: payload too big for buffer, buffer can hold a %d bytes maximum", nl-ef.GetHeaderLen())
	}

	ef.framebuf = ef.framebuf[0:nl]
	return nil
}

func (ef *ETHFrame) WriteDown() error {
	// bounds should already be checked
	// TODO: API clarification

	copy(ef.framebuf[0:6], ef.Destination[:])
	copy(ef.framebuf[6:12], ef.Source[:])

	pos := 12
	if ef.UseVlan {
		return errors.New("VLAN tags are not supported")
	}

	ef.framebuf[pos] = uint8(ef.Type >> 8)
	ef.framebuf[pos] = uint8(ef.Type)

	pos += 2

	return nil
}

const (
	min_framelen_with_fcs = 60
	fcs_len               = 4
	min_headerandpayload  = min_framelen_with_fcs - fcs_len

	// excluding fcs
	max_framelen_novlan = 1518
	max_framelen_vlan   = 1522
)
