package ecfr

import (
	"errors"
	"fmt"
)

type Datagram struct {
	DatagramHeader
	data           []byte
	WorkingCounter uint16
	buffer         []byte
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

	dg.data = b[:dg.DataLength()]
	b = b[dg.DataLength():]

	if len(b) < 2 {
		err = fmt.Errorf("overlaying ecat dgram: need 2 bytes for working counter, got %d", len(b))
		return
	}

	// guarded by condition above
	dg.WorkingCounter, b = getUint16(b)

	dg.buffer = d
	return
}

func (dg *Datagram) Commit() (err error) {
	err = dg.DatagramHeader.Commit()
	if err != nil {

	}

	// the data is already committed

	wcoffs := datagramHeaderLength + len(dg.data)
	if len(dg.buffer) < (wcoffs + 2) {
		fmt.Errorf("cannot commit data: buffer too short, need %d bytes, have %d", wcoffs+2, len(dg.buffer))
	}

	putUint16(dg.buffer[wcoffs:wcoffs+2], dg.WorkingCounter)

	return
}

type DatagramHeader struct {
	Command   CommandType
	Index     uint8
	Addr32    uint32
	LenWord   uint16
	Interrupt uint16
	buffer    []byte
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

	dh.buffer = d

	return
}

func (dh *DatagramHeader) Commit() (err error) {
	var b []byte

	b = putUint8(b, uint8(dh.Command))
	b = putUint8(b, dh.Index)
	b = putUint32(b, dh.Addr32)
	b = putUint16(b, dh.LenWord)
	b = putUint16(b, dh.Interrupt)

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

func (dh *DatagramHeader) SetLast(last bool) {
	if last {
		dh.LenWord &^= (1 << lastindicatorBit)
	} else {
		dh.LenWord |= (1 << lastindicatorBit)
	}
}

func PointDatagramHeaderTo(d []byte) (dh DatagramHeader, err error) {
	if len(d) < datagramHeaderLength {
		err = fmt.Errorf("datagram header needs %d bytes, only have %d", datagramHeaderLength, len(d))
		return
	}

	dh.buffer = d[0:datagramHeaderLength]
	return
}

func (dg *Datagram) Data() []byte {
	return dg.data
}

func (dg *Datagram) SetDataLen(ndl int) error {
	nl := datagramOverheadLength + ndl
	if (cap(dg.buffer)) < nl {
		return fmt.Errorf("datagram with new size needs %d bytes of space, only %d in buffer", nl, cap(dg.buffer))
	}

	if ndl > datagramMaxDataLength {
		return errors.New("new data length exceeds maximum datagram data length")
	}

	dg.data = dg.data[0:ndl]
	dg.buffer = dg.buffer[0:nl]

	dg.LenWord &^= datagramDataLengthMask
	dg.LenWord |= (uint16(ndl) & datagramDataLengthMask)

	return nil
}

func PointDatagramTo(d []byte) (dg Datagram, err error) {
	if len(d) < datagramOverheadLength {
		err = errors.New("byte slice too short to be pointed to by a datagram")
		return
	}

	dg = Datagram{
		buffer: d,
		data:   d[datagramHeaderLength:datagramHeaderLength],
	}
	dg.DatagramHeader, err = PointDatagramHeaderTo(d)

	return
}

const (
	roundtripBit     = 14
	lastindicatorBit = 15

	datagramHeaderLength   = 10
	datagramFooterLength   = 2
	datagramOverheadLength = datagramHeaderLength + datagramFooterLength

	datagramDataLengthMask = (1 << 12) - 1
	datagramMaxDataLength  = datagramDataLengthMask
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
