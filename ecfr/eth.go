package ecfr

import (
	"errors"
	"fmt"
)

type ETHAddr [6]byte

func (ea ETHAddr) String() string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", ea[0], ea[1], ea[2], ea[3], ea[4], ea[5])
}

// bounds need to be checked already
func sliceToETHADDR(s []byte) ETHAddr {
	var e ETHAddr
	e[0] = s[0]
	e[1] = s[1]
	e[2] = s[2]
	e[3] = s[3]
	e[4] = s[4]
	e[5] = s[5]
	return e
}

// payload len is len(ef.GetPayload())
// capacity for payload is: cap(ef.GetPayload()) - ef.GetFooterLen()
type ETHFrame struct {
	Destination, Source ETHAddr
	Type                uint16

	UseVlan bool
	VLANTCI uint16

	framebuf []byte
}

func OverlayETHFrame(fb []byte) (*ETHFrame, error) {
	if len(fb) == 0 {
		fb = make([]byte, max_framelen)
	}

	if len(fb) < min_framelen_with_fcs {
		return nil, fmt.Errorf(
			"NewETHFrame: buffer too small, need at least %d bytes",
			min_headerandpayload)
	}

	ef := &ETHFrame{}
	ef.framebuf = fb

	// TODO: read this information at overlay time?

	// guarded by len(fb) < min_framelen_with_fcs
	ef.Destination = sliceToETHADDR(fb[offsetDestination:offsetSource])
	ef.Source = sliceToETHADDR(fb[offsetSource:offsetVLANOrType])
	ef.Type, _ = getUint16(fb[offsetVLANOrType:])
	if ef.Type == etherTypeVLAN {
		ef.UseVlan = true
		ef.VLANTCI, _ = getUint16(fb[offsetVLANTCI:])

	}
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

func (ef *ETHFrame) GetFooterLen() int {
	return fcs_len
}

// header contents will be undefined if you do not call WriteDown() before.
func (ef *ETHFrame) GetFrameBuf() []byte {
	return ef.framebuf
}

func (ef *ETHFrame) GetPayload() []byte {
	return ef.framebuf[ef.GetHeaderLen() : len(ef.framebuf)-ef.GetFooterLen()]
}

func (ef *ETHFrame) SetPayloadLen(npl int) error {
	nl := npl + ef.GetHeaderLen() + ef.GetFooterLen()
	// TODO: check
	if nl < min_framelen_with_fcs {
		return fmt.Errorf(
			"SetPayloadLen: payload too small, need at least %d bytes",
			min_framelen_with_fcs-ef.GetHeaderLen()-ef.GetFooterLen())
	}

	maxnl := max_framelen_novlan
	if ef.UseVlan {
		maxnl = max_framelen_vlan
	}

	if nl > maxnl {
		return fmt.Errorf(
			"SetPayloadLen: payload too big, maximum for this configuration is %d bytes",
			maxnl-ef.GetHeaderLen())
	}

	if nl > cap(ef.framebuf) {
		return fmt.Errorf(
			"SetPayloadLen: payload  of %d bytes too big for buffer, buffer can hold a %d bytes maximum",
			npl,
			cap(ef.framebuf)-ef.GetFooterLen()-ef.GetHeaderLen())
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

	// TODO: how to deal with the FCS?

	return nil
}

// TODO: +4 bytes of FCS for future compatibility?
const (
	min_framelen_with_fcs = 64
	fcs_len               = 4
	min_headerandpayload  = min_framelen_with_fcs - fcs_len

	// inclduing fcs
	max_framelen_novlan = 1522
	max_framelen_vlan   = 1526
	max_framelen        = max_framelen_vlan

	offsetDestination = 0
	offsetSource      = 6
	offsetVLANOrType  = 12
	offsetVLANTCI     = 14

	etherTypeVLAN = 0x8100
)
