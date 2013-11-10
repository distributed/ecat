package ecfr

import (
	"fmt"
)

type Datagram struct {
	DatagramHeader
	Data           []byte
	WorkingCounter uint16
}

func (dg *Datagram) Overlay(d []byte) (b []byte, err error) {
	//fmt.Printf("overlaying datagram over %s\n", spew.Sdump(d))
	b, err = dg.DatagramHeader.Overlay(d)
	if err != nil {
		return
	}

	if len(b) < int(dg.DataLength()) {
		err = fmt.Errorf("overlaying ecat dgram: need %d bytes of data, have %d", dg.DataLength(), len(b))
		return
	}

	dg.Data = b[:dg.DataLength()]
	b = b[dg.DataLength():]

	if len(b) < 2 {
		err = fmt.Errorf("overlaying ecat dgram: need 2 bytes for working counter, got %d", len(b))
		return
	}

	// guarded by condition above
	dg.WorkingCounter, b = getUint16(b)
	return
}

type DatagramHeader struct {
	Command   CommandType
	Index     uint8
	Addr32    uint32
	LenWord   uint16
	Interrupt uint16
}

const (
	datagramHeaderByteLen = 10
)

func (dh *DatagramHeader) Overlay(d []byte) (b []byte, err error) {
	b = d
	if len(b) < datagramHeaderByteLen {
		err = fmt.Errorf("need %d bytes for dgram header, have %d", datagramHeaderByteLen, len(b))
		return
	}

	var c8 uint8
	c8, b = getUint8(b)
	dh.Command = CommandType(c8)
	dh.Index, b = getUint8(b)
	dh.Addr32, b = getUint32(b)
	dh.LenWord, b = getUint16(b)
	dh.Interrupt, b = getUint16(b)

	return
}

func (dh *DatagramHeader) SlaveAddr() uint16 {
	return uint16(dh.Addr32)
}

func (dh *DatagramHeader) OffsetAddr() uint16 {
	return uint16(dh.Addr32 >> 16)
}

func (dh *DatagramHeader) LogicalAddr() uint32 {
	return dh.Addr32
}

func (dh *DatagramHeader) DataLength() uint16 {
	return dh.LenWord & ((1 << 11) - 1)
}

func (dh *DatagramHeader) Roundtrip() bool {
	return (dh.LenWord & (1 << roundtripBit)) != 0
}

func (dh *DatagramHeader) Last() bool {
	return (dh.LenWord & (1 << lastindicatorBit)) == 0
}

const (
	roundtripBit     = 14
	lastindicatorBit = 15
)

type CommandType uint8

func (ct CommandType) String() string {
	if cts, ok := commandTypeName[ct]; ok {
		return cts
	}
	return fmt.Sprintf("CommandType(%d)", uint(ct))
}

const (
	NOP  CommandType = 0
	APRD CommandType = 1
	APWR CommandType = 2
	APRW CommandType = 3
	FPRD CommandType = 4
	FPWR CommandType = 5
	FPRW CommandType = 6
	BRD  CommandType = 7
	BWR  CommandType = 8
	BRW  CommandType = 9
	LRD  CommandType = 10
	LWR  CommandType = 11
	LRW  CommandType = 12
	ARMW CommandType = 13
	FRMW CommandType = 14
)

var commandTypeName = map[CommandType]string{
	NOP:  "NOP",
	APRD: "APRD",
	APWR: "APWR",
	APRW: "APRW",
	FPRD: "FPRD",
	FPWR: "FPWR",
	FPRW: "FPRW",
	BRD:  "BRD",
	BWR:  "BWR",
	BRW:  "BRW",
	LRD:  "LRD",
	LWR:  "LWR",
	LRW:  "LRW",
	ARMW: "ARMW",
	FRMW: "FRMW",
}
